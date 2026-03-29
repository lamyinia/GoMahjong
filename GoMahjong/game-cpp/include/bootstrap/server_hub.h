#pragma once

#include <boost/asio.hpp>
#include <cstddef>
#include <cstdint>
#include <memory>
#include <optional>
#include <thread>
#include <vector>

namespace infra::config { class Config; }

namespace infra::net::listener { class TcpListener; }


// 生命周期管理器，以及依赖的组装
namespace gomahjong::bootstrap {
    class ServerHub {
    public:
        explicit ServerHub(const infra::config::Config &cfg);

        ~ServerHub();

        ServerHub(const ServerHub &) = delete;

        ServerHub &operator=(const ServerHub &) = delete;

        void start();

        void stop();

        boost::asio::io_context &ioc() noexcept { return ioc_; }

    private:
        void build_pools();

        void build_services();

        void build_listeners();

        void write_back();

    private:
        const infra::config::Config &cfg_;

        boost::asio::io_context ioc_;
        std::vector<std::thread> io_threads_;
        std::optional<boost::asio::executor_work_guard<boost::asio::io_context::executor_type>> work_guard_;

        std::unique_ptr<infra::net::listener::TcpListener> tcp_listener_;

        bool started_ = false;
    };
}