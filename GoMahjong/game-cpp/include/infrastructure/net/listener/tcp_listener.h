#pragma once

#include <memory>
#include <boost/asio.hpp>
#include <system_error>
#include <functional>
#include <atomic>

#include "infrastructure/net/transport/i_transport.h"

namespace infra::net::listener {
    class TcpListener {
    public:
        using OnAccept = std::function<void(std::shared_ptr<infra::net::transport::ITransport>)>;
        using OnError = std::function<void(const std::error_code &)>;

        explicit TcpListener(boost::asio::io_context &ioc, boost::asio::ip::tcp::endpoint ep);

        void start(OnAccept onAccept, OnError onError);

        void stop();

    private:
        void do_accept();

        boost::asio::io_context &ioc_;
        boost::asio::ip::tcp::acceptor acceptor_;
        OnAccept onAccept_;
        OnError onError_;
        std::atomic_bool started_{false};
    };
}