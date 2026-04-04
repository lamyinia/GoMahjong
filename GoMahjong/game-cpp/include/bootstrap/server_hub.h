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

namespace infra::net::reliability { class WildEndpointManager; }

namespace infra::net::session { class SessionManager; }

namespace infra::persistence { class MongoPool; }

namespace domain::game::room { class RoomManager; }

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

        void setup_wild_endpoint_callbacks();

    private:
        const infra::config::Config &cfg_;

        boost::asio::io_context ioc_;
        std::vector<std::thread> io_threads_;
        std::optional<boost::asio::executor_work_guard<boost::asio::io_context::executor_type>> work_guard_;

        std::unique_ptr<infra::net::listener::TcpListener> tcp_listener_;

        // 未认证连接管理
        std::shared_ptr<infra::net::reliability::WildEndpointManager> wild_endpoint_manager_;

        // 已认证会话管理
        std::shared_ptr<infra::net::session::SessionManager> session_manager_;

        // 数据库线程池
        std::shared_ptr<infra::persistence::MongoPool> mongo_pool_;

        // 游戏房间管理器（包含 Actor 线程池）
        std::unique_ptr<domain::game::room::RoomManager> room_manager_;

        bool started_ = false;
    };
}