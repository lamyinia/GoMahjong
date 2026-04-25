#include "bootstrap/server_hub.h"

#include <algorithm>
#include <atomic>
#include <iostream>
#include <chrono>
#include <fstream>

#include "infrastructure/config/config.hpp"
#include "infrastructure/net/listener/tcp_listener.h"
#include "infrastructure/net/reliability/wild_endpoint_manager.h"
#include "infrastructure/net/session/session_manager.h"
#include "infrastructure/persistence/mongo_pool.hpp"
#include "domain/game/service/game_service.hpp"
#include "infrastructure/log/logger.hpp"
#include "domain/game/handler/play_tile_handler.h"
#include "domain/game/room/room_manager.h"
#include "domain/game/outbound/out_dispatcher.h"
#include "infrastructure/util/timing_wheel.h"
#include "infrastructure/util/timer_thread.h"
#include "infrastructure/rpc/grpc_server.hpp"
#include "infrastructure/discovery/service_registry.hpp"

namespace gomahjong::bootstrap {
    ServerHub::ServerHub(const infra::config::Config &cfg) : cfg_(cfg) {}

    ServerHub::~ServerHub() {
        stop();
    }

    void ServerHub::start() {
        if (started_) return;

        build_pools();

        build_services();

        write_back();

        start_listeners();

        started_ = true;
    }

    void ServerHub::stop() {
        if (!started_) return;
        // 停止监听端口

        if (work_guard_) work_guard_->reset();
        ioc_.stop();

        for (auto &t: io_threads_) {
            if (t.joinable()) {
                t.join();
            }
        }
        io_threads_.clear();

        // 停止数据库线程池
        if (mongo_pool_) {
            mongo_pool_->stop();
            LOG_INFO("mongo pool stopped");
        }

        // 停止负载上报线程
        load_report_running_.store(false);
        if (load_report_thread_.joinable()) {
            load_report_thread_.join();
        }

        // 注销服务
        if (service_registry_ && service_registry_->is_connected()) {
            service_registry_->deregister_service(cfg_.server().service_name, cfg_.server().node_id);
            LOG_INFO("service deregistered from etcd");
        }

        // 停止 gRPC 服务
        if (grpc_server_) {
            grpc_server_->stop();
            LOG_INFO("gRPC server stopped");
        }

        // 停止游戏房间管理器（Actor 线程池）
        if (room_manager_) {
            room_manager_->stop();
            LOG_INFO("room manager stopped");
        }

        // 释放资源
        tcp_listener_.reset();
        wild_endpoint_manager_.reset();
        session_manager_.reset();
        mongo_pool_.reset();
        room_manager_.reset();
        out_dispatcher_.reset();
        timer_thread_.reset();
        timing_wheel_.reset();
        grpc_server_.reset();
        service_registry_.reset();
    }

    void ServerHub::build_pools() {
        // 初始化日志系统
        infra::log::init(cfg_.server().log);
        
        // 创建网络 IO 线程池
        work_guard_.emplace(boost::asio::make_work_guard(ioc_));
        unsigned int threads = std::max(2u, std::thread::hardware_concurrency() > 0 ? std::thread::hardware_concurrency() / 2 : 2u);

        io_threads_.reserve(threads);
        for (unsigned int i = 0; i < threads; ++i) {
            io_threads_.emplace_back([this] {
                ioc_.run();
            });
        }
        LOG_INFO("io 线程池启动 threads 数量: {}", threads);

        // 创建数据库线程池
        mongo_pool_ = std::make_shared<infra::persistence::MongoPool>(cfg_.server().mongodb);
        mongo_pool_->start();
        LOG_INFO("数据库线程池启动 threads 数量: {}", cfg_.server().mongodb.thread_count);

        // 创建游戏房间管理器（Actor 线程池）
        auto actorCount = cfg_.server().actor.count;
        auto queueCapacity = cfg_.server().actor.queue_capacity;
        room_manager_ = std::make_unique<domain::game::room::RoomManager>(actorCount, queueCapacity);
        room_manager_->start();
        domain::game::service::GameService::instance().init(*room_manager_);
        LOG_INFO("room manager started with {} actors, queue capacity {}", actorCount, queueCapacity);

        // 创建出站调度器
        out_dispatcher_ = std::make_unique<domain::game::outbound::OutDispatcher>();

        // 创建时间轮 + 定时器线程
        auto twTickMs = cfg_.server().timer_wheel.tick_duration_ms;
        auto twSize = cfg_.server().timer_wheel.wheel_size;
        timing_wheel_ = std::make_unique<infra::util::TimingWheel>(twTickMs, twSize);

        infra::discovery::RegistryConfig registryCfg{
            cfg_.server().etcd.endpoints,
            cfg_.server().etcd.ttl_seconds
        };
        service_registry_ = std::make_unique<infra::discovery::ServiceRegistry>(registryCfg);
        if (service_registry_->connect()) {
            LOG_INFO("connected to etcd: {}", cfg_.server().etcd.endpoints);
        } else {
            LOG_WARN("failed to connect to etcd: {}", cfg_.server().etcd.endpoints);
        }
    }

    void ServerHub::build_services() {
        // 创建未认证连接管理器
        wild_endpoint_manager_ = std::make_shared<infra::net::reliability::WildEndpointManager>(
            ioc_.get_executor(),
            std::chrono::milliseconds(5000)  // 30 秒认证超时
        );
        session_manager_ = std::make_shared<infra::net::session::SessionManager>();

        // 注入 OutDispatcher 依赖
        if (out_dispatcher_) {
            out_dispatcher_->setSessionManager(session_manager_);
            room_manager_->setOutDispatcher(out_dispatcher_.get());
        }

        // 配置时间轮：注入到 RoomManager，启动 TimerThread
        // expiredCallback 由各房间的 TurnManager 在 initTimerSystem 时设置
        if (timing_wheel_) {
            room_manager_->setTimingWheel(timing_wheel_.get());
            timer_thread_ = std::make_unique<infra::util::TimerThread>(*timing_wheel_, cfg_.server().timer_wheel.tick_duration_ms);
            timer_thread_->start();
        }

        // 注册游戏 Handler
        domain::game::handler::registerGameHandlers();

        // 初始化 gRPC 服务（但不启动）
        auto grpcPort = cfg_.server().grpc.port;
        grpc_server_ = std::make_unique<infra::rpc::GrpcServer>(grpcPort);
        
        // 注册 GameService
        grpc_server_->register_service(
            domain::game::service::GameService::instance().get_grpc_service()
        );
    }

    // 额外的注入逻辑（回调设置、服务注册、负载上报）
    void ServerHub::write_back() {
        if (wild_endpoint_manager_ && session_manager_) {
            auto session_mgr = session_manager_;
            wild_endpoint_manager_->set_on_authenticated(
                    [session_mgr](const std::string& player_id,
                                  std::shared_ptr<infra::net::channel::IChannel> channel) {
                        LOG_INFO("player {} authenticated, creating session", player_id);
                        session_mgr->create_or_get_session(player_id, std::move(channel));
                    }
            );
            LOG_DEBUG("wild endpoint 回调完成");
        }

        if (service_registry_ && service_registry_->is_connected()) {
            infra::discovery::ServiceEndpoint endpoint;
            endpoint.node_id = cfg_.server().node_id;
            endpoint.host = cfg_.server().host;
            endpoint.port = cfg_.server().grpc.port;

            if (service_registry_->register_service(cfg_.server().service_name, endpoint)) {
                LOG_INFO("service registered to etcd: {}", cfg_.server().service_name);
            } else {
                LOG_WARN("failed to register service to etcd");
            }

            start_load_reporter();
        }
    }

    void ServerHub::start_listeners() {
        using namespace infra::net::listener;

        auto tcpPort = cfg_.server().net.tcp.port;

        tcp_listener_ = std::make_unique<TcpListener>(ioc_, boost::asio::ip::tcp::endpoint(boost::asio::ip::tcp::v4(), tcpPort));

        auto wild_manager = wild_endpoint_manager_;

        tcp_listener_->start(
                [](const std::error_code &ec) {
                    LOG_ERROR("tcp_listener 的 OnError 回调触发, accept error: {}", ec.message());
                },
                [wild_manager](const std::shared_ptr<infra::net::channel::IChannel>& channel) {
                    LOG_INFO("tcp_listener 的 onNewChannel 回调触发");
                    // 将新连接交给 WildEndpointManager 管理
                    if (wild_manager && channel) {
                        wild_manager->add_channel(channel);
                    }
                });

        LOG_INFO("tcp listener started on port {}", tcpPort);

        if (grpc_server_) {
            if (grpc_server_->start(false)) {
                LOG_INFO("gRPC server started on port {}", cfg_.server().grpc.port);
            } else {
                LOG_ERROR("gRPC server failed to start on port {}", cfg_.server().grpc.port);
            }
        }
    }

    void ServerHub::start_load_reporter() {
        if (!service_registry_ || !room_manager_) {
            return;
        }

        load_report_running_.store(true);
        auto interval = std::chrono::seconds(cfg_.server().etcd.report_interval_seconds);
        auto serviceName = cfg_.server().service_name;
        auto nodeId = cfg_.server().node_id;

        load_report_thread_ = std::thread([this, interval, serviceName, nodeId]() {
            LOG_INFO("load reporter started, interval: {}s", interval.count());

            while (load_report_running_.load()) {
                auto metadata = collect_load_metadata();

                // 上报到 etcd
                if (service_registry_->update_metadata(serviceName, nodeId, metadata)) {
//                    LOG_DEBUG("load reported: rooms={}, players={}", metadata["room_count"], metadata["player_count"]);
                } else {
                    LOG_WARN("load reported failure");
                }

                // 分段睡眠以支持快速停止
                for (int i = 0; i < interval.count() && load_report_running_.load(); ++i) {
                    std::this_thread::sleep_for(std::chrono::seconds(1));
                }
            }

            LOG_INFO("load reporter stopped");
        });
    }

    std::map<std::string, std::string> ServerHub::collect_load_metadata() {
        std::map<std::string, std::string> metadata;

        // 业务指标
        metadata["room_count"] = std::to_string(room_manager_->room_count());
        metadata["player_count"] = std::to_string(room_manager_->player_count());
        metadata["actor_count"] = std::to_string(room_manager_->actor_count());

        // 系统指标
        metadata["cpu_percent"] = std::to_string(static_cast<int>(get_cpu_percent() * 10) / 10.0);
        metadata["memory_mb"] = std::to_string(static_cast<int>(get_memory_mb() * 10) / 10.0);
        metadata["uptime_seconds"] = std::to_string(get_uptime_seconds());

        return metadata;
    }

    // ==================== Platform-Specific System Metrics ====================

#ifdef _WIN32
    double ServerHub::get_cpu_percent() {
        static FILETIME prevIdle = {0, 0};
        static FILETIME prevKernel = {0, 0};
        static FILETIME prevUser = {0, 0};

        FILETIME idle, kernel, user;
        GetSystemTimes(&idle, &kernel, &user);

        auto toUll = [](const FILETIME& ft) {
            return ((unsigned long long)ft.dwHighDateTime << 32) | ft.dwLowDateTime;
        };

        auto idleDiff = toUll(idle) - toUll(prevIdle);
        auto kernelDiff = toUll(kernel) - toUll(prevKernel);
        auto userDiff = toUll(user) - toUll(prevUser);
        auto total = kernelDiff + userDiff;

        prevIdle = idle;
        prevKernel = kernel;
        prevUser = user;

        if (total == 0) return 0.0;
        return 100.0 * (1.0 - (double)idleDiff / (double)total);
    }

    double ServerHub::get_memory_mb() {
        PROCESS_MEMORY_COUNTERS_EX pmc;
        if (GetProcessMemoryInfo(GetCurrentProcess(), (PROCESS_MEMORY_COUNTERS*)&pmc, sizeof(pmc))) {
            return static_cast<double>(pmc.WorkingSetSize) / (1024.0 * 1024.0);
        }
        return 0.0;
    }
#else
    double ServerHub::get_cpu_percent() {
        // Linux: read /proc/stat
        static unsigned long long prevIdle = 0, prevTotal = 0;
        std::ifstream procStat("/proc/stat");
        if (!procStat.is_open()) return 0.0;

        std::string line;
        std::getline(procStat, line);
        unsigned long long user, nice, system, idle, iowait, irq, softirq;
        if (sscanf(line.c_str(), "cpu %llu %llu %llu %llu %llu %llu %llu",
                   &user, &nice, &system, &idle, &iowait, &irq, &softirq) != 7) {
            return 0.0;
        }

        auto total = user + nice + system + idle + iowait + irq + softirq;
        auto idleDiff = idle - prevIdle;
        auto totalDiff = total - prevTotal;
        prevIdle = idle;
        prevTotal = total;

        if (totalDiff == 0) return 0.0;
        return 100.0 * (1.0 - (double)idleDiff / (double)totalDiff);
    }

    double ServerHub::get_memory_mb() {
        std::ifstream status("/proc/self/status");
        if (!status.is_open()) return 0.0;
        std::string line;
        while (std::getline(status, line)) {
            if (line.find("VmRSS:") == 0) {
                unsigned long kb = 0;
                if (sscanf(line.c_str(), "VmRSS: %lu kB", &kb) == 1) {
                    return static_cast<double>(kb) / 1024.0;
                }
            }
        }
        return 0.0;
    }
#endif

    std::int64_t ServerHub::get_uptime_seconds() {
        static auto startTime = std::chrono::steady_clock::now();
        return std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::steady_clock::now() - startTime
        ).count();
    }
}