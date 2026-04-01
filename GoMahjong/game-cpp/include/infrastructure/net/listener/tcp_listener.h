#pragma once

#include <memory>
#include <boost/asio.hpp>
#include <system_error>
#include <functional>
#include <atomic>

#include "infrastructure/net/channel/i_channel.h"

namespace infra::net::listener {

    namespace channel = infra::net::channel;

    class TcpListener {
    public:
        using OnError = std::function<void(const std::error_code&)>;
        using OnNewChannel = std::function<void(std::shared_ptr<channel::IChannel>)>;

        explicit TcpListener(boost::asio::io_context& ioc, boost::asio::ip::tcp::endpoint ep);

        /**
         * @brief 启动监听
         * @param onError 错误回调
         * @param onNewChannel 新连接回调（接收 Channel）
         */
        void start(OnError onError, OnNewChannel onNewChannel);

        void stop();

    private:
        void do_accept();

        boost::asio::io_context& ioc_;
        boost::asio::ip::tcp::acceptor acceptor_;
        OnError onError_;
        OnNewChannel onNewChannel_;
        std::atomic_bool started_{false};
    };

} // namespace infra::net::listener