#include "infrastructure/net/listener/tcp_listener.h"
#include "infrastructure/net/transport/tcp_transport.h"
#include "infrastructure/log/logger.hpp"

#include <boost/asio.hpp>

namespace infra::net::listener {

    TcpListener::TcpListener(boost::asio::io_context &ioc, boost::asio::ip::tcp::endpoint ep)
        : ioc_(ioc), acceptor_(ioc) {
        boost::system::error_code ec;
        acceptor_.open(ep.protocol(), ec);
        if (ec) {
            LOG_ERROR("[tcp_listener] open failed: {}", ec.message());
            return;
        }
        acceptor_.set_option(boost::asio::socket_base::reuse_address(true), ec);
        if (ec) {
            LOG_ERROR("[tcp_listener] set_option failed: {}", ec.message());
            return;
        }
        acceptor_.bind(ep, ec);
        if (ec) {
            LOG_ERROR("[tcp_listener] bind failed: {}", ec.message());
            return;
        }

        acceptor_.listen(boost::asio::socket_base::max_listen_connections, ec);
        if (ec) {
            LOG_ERROR("[tcp_listener] listen failed: {}", ec.message());
            return;
        }

        LOG_INFO("[tcp_listener] listening on {}:{}", ep.address().to_string(), ep.port());
    }

    void TcpListener::start(OnAccept onAccept, OnError onError) {
        if (started_.exchange(true)) {
            return;
        }

        onAccept_ = std::move(onAccept);
        onError_ = std::move(onError);
        do_accept();
    }

    void TcpListener::stop() {
        if (!started_.exchange(false)) {
            return;
        }

        boost::system::error_code ec;
        acceptor_.close(ec);
        if (ec) {
            LOG_ERROR("[tcp_listener] close failed: {}", ec.message());
        }
    }

    void TcpListener::do_accept() {
        if (!started_) {
            return;
        }
        
        // 这里的 socket 代表一个完成三次握手的 TCP 连接的 fd
        acceptor_.async_accept(
            [this](boost::system::error_code ec, boost::asio::ip::tcp::socket socket) {
                if (!started_) {
                    return;
                }

                if (ec) {
                    if (onError_) {
                        const std::error_code se(ec.value(), std::system_category());
                        onError_(se);
                    }
                    if (started_) {
                        do_accept();
                    }
                    return;
                }

                auto transport = std::make_shared<infra::net::transport::TcpTransport>(std::move(socket));
                if (onAccept_) {
                    onAccept_(transport);
                }

                do_accept();
            });
    }

}