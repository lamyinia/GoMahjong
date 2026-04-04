#include "bootstrap/server_hub.h"

#include <algorithm>
#include <iostream>

#include "infrastructure/config/config.hpp"
#include "infrastructure/net/listener/tcp_listener.h"
#include "infrastructure/net/reliability/wild_endpoint_manager.h"
#include "infrastructure/net/session/session_manager.h"
#include "infrastructure/persistence/mongo_pool.hpp"
#include "infrastructure/log/logger.hpp"
#include "domain/game/handler/play_tile_handler.h"
#include "domain/game/room/room_manager.h"

namespace gomahjong::bootstrap {
    ServerHub::ServerHub(const infra::config::Config &cfg) : cfg_(cfg) {}

    ServerHub::~ServerHub() {
        stop();
    }

    void ServerHub::start() {
        if (started_) return;
        build_pools();

        build_services();

        build_listeners();

        write_back();

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
            LOG_INFO("[hub] mongo pool stopped");
        }

        // 停止游戏房间管理器（Actor 线程池）
        if (room_manager_) {
            room_manager_->stop();
            LOG_INFO("[hub] room manager stopped");
        }

        // 释放资源
        tcp_listener_.reset();
        wild_endpoint_manager_.reset();
        session_manager_.reset();
        mongo_pool_.reset();
        room_manager_.reset();
    }

    void ServerHub::build_pools() {
        // 创建网络 IO 线程池
        work_guard_.emplace(boost::asio::make_work_guard(ioc_));
        unsigned int threads = std::max(2u, std::thread::hardware_concurrency() > 0 ? std::thread::hardware_concurrency() / 2 : 2u);

        io_threads_.reserve(threads);
        for (unsigned int i = 0; i < threads; ++i) {
            io_threads_.emplace_back([this] {
                ioc_.run();
            });
        }
        LOG_INFO("[hub] io threads 数量: {}", threads);

        // 创建数据库线程池
        mongo_pool_ = std::make_shared<infra::persistence::MongoPool>(cfg_.server().mongodb);
        mongo_pool_->start();
        LOG_INFO("[hub] mongo pool started with {} threads", cfg_.server().mongodb.thread_count);

        // 创建游戏房间管理器（Actor 线程池）
        auto actorCount = cfg_.server().actor.count;
        auto queueCapacity = cfg_.server().actor.queue_capacity;
        room_manager_ = std::make_unique<domain::game::room::RoomManager>(actorCount, queueCapacity);
        room_manager_->start();
        LOG_INFO("[hub] room manager started with {} actors, queue capacity {}", actorCount, queueCapacity);
    }

    void ServerHub::build_services() {
        // 创建未认证连接管理器
        wild_endpoint_manager_ = std::make_shared<infra::net::reliability::WildEndpointManager>(
            ioc_.get_executor(),
            std::chrono::milliseconds(5000)  // 5秒认证超时
        );
        session_manager_ = std::make_shared<infra::net::session::SessionManager>();

        // 设置回调
        setup_wild_endpoint_callbacks();

        // 注册游戏 Handler
        domain::game::handler::registerGameHandlers();

        LOG_INFO("[hub] services built");
    }

    void ServerHub::build_listeners() {
        // 小作用域 using，生命周期在整个函数块内，对运行效率无影响
        using namespace infra::net::listener;

        auto port = cfg_.server().net.tcp.port;

        tcp_listener_ = std::make_unique<TcpListener>(ioc_,
                                                      boost::asio::ip::tcp::endpoint(boost::asio::ip::tcp::v4(), port));

        // 捕获 wild_endpoint_manager_ 的弱引用，避免循环引用
        auto wild_manager = wild_endpoint_manager_;

        tcp_listener_->start(
                [](const std::error_code &ec) {
                    LOG_ERROR("[hub] accept error: {}", ec.message());
                },
                [wild_manager](const std::shared_ptr<infra::net::channel::IChannel>& channel) {
                    LOG_INFO("[hub] new connection accepted");
                    // 将新连接交给 WildEndpointManager 管理
                    if (wild_manager && channel) {
                        wild_manager->add_channel(channel);
                    }
                });

        LOG_INFO("[hub] tcp listener started on port {}", port);
    }

    void ServerHub::write_back() {

    }

    void ServerHub::setup_wild_endpoint_callbacks() {
        if (!wild_endpoint_manager_ || !session_manager_) {
            LOG_ERROR("[hub] cannot setup callbacks: managers not initialized");
            return;
        }

        auto session_mgr = session_manager_;

        // 设置认证成功回调：创建 Session
        wild_endpoint_manager_->set_on_authenticated(
            [session_mgr](const std::string& player_id, 
                          std::shared_ptr<infra::net::channel::IChannel> channel) {
                LOG_INFO("[hub] player {} authenticated, creating session", player_id);
                // 创建或获取会话
                session_mgr->create_or_get_session(player_id, std::move(channel));
            }
        );

        LOG_INFO("[hub] wild endpoint callbacks setup complete");
    }
}