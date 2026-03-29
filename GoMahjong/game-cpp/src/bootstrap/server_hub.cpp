#include "bootstrap/server_hub.h"

#include <algorithm>
#include <cstdlib>
#include <iostream>

#include "infrastructure/config/config.hpp"
#include "infrastructure/net/listener/tcp_listener.h"
#include "infrastructure/log/logger.hpp"

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

        // 释放资源
        tcp_listener_.reset();
    }

    void ServerHub::build_pools() {
        work_guard_.emplace(boost::asio::make_work_guard(ioc_));
        unsigned int threads = std::max(
                2u,
                std::thread::hardware_concurrency() > 0 ? std::thread::hardware_concurrency() / 2 : 2u);

        io_threads_.reserve(threads);
        for (unsigned int i = 0; i < threads; ++i) {
            io_threads_.emplace_back([this] { ioc_.run(); });
        }
        LOG_INFO("[hub] io threads 数量: {}", threads);
    }

    void ServerHub::build_services() {

    }

    void ServerHub::build_listeners() {
        // 小作用域 using，生命周期在整个函数块内，对运行效率无影响
        using namespace infra::net::listener;
        using namespace infra::net::transport;

        auto port = cfg_.server().net.tcp.port;

        tcp_listener_ = std::make_unique<TcpListener>(ioc_, boost::asio::ip::tcp::endpoint(boost::asio::ip::tcp::v4(), port));

        tcp_listener_->start(
                [this](std::shared_ptr<ITransport> transport) {
                    LOG_INFO("[hub] new connection accepted");
                    // Echo demo: start transport with echo callback
                    transport->start(
                            [transport](ITransport::Bytes &&data) {
                                // Echo back received data
                                LOG_INFO("[hub] received {} bytes, echoing back", data.size());
                                transport->send(std::move(data));
                            },
                            [] {
                                LOG_INFO("[hub] connection closed");
                            },
                            [](const std::error_code &ec) {
                                LOG_ERROR("[hub] connection error: {}", ec.message());
                            });
                },
                [](const std::error_code &ec) {
                    LOG_ERROR("[hub] accept error: {}", ec.message());
                });

        LOG_INFO("[hub] tcp listener started on port {}", port);
    }

    void ServerHub::write_back() {

    }
}