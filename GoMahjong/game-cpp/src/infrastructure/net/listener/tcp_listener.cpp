#include "infrastructure/net/listener/tcp_listener.h"
#include "infrastructure/net/transport/tcp_transport.h"
#include "infrastructure/net/channel/tcp_channel.h"
#include "infrastructure/log/logger.hpp"

#include <boost/asio.hpp>

namespace infra::net::listener {

    TcpListener::TcpListener(boost::asio::io_context& ioc, boost::asio::ip::tcp::endpoint ep)
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

        LOG_DEBUG("listening on {}:{}", ep.address().to_string(), ep.port());
    }

    void TcpListener::start(OnError onError, OnNewChannel onNewChannel) {
        if (started_.exchange(true)) {
            return;
        }

        onError_ = std::move(onError);
        onNewChannel_ = std::move(onNewChannel);
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

                auto transport = std::make_shared<transport::TcpTransport>(std::move(socket));
                auto channel = std::make_shared<channel::TcpChannel>(transport, transport->strand());

                LOG_DEBUG("tcp_listener 新连接: {}", channel->id());

                if (onNewChannel_){
                    onNewChannel_(channel);
                }
                // 开始读取数据（由上层决定何时开始），这里不自动调用 start_read()，让上层有机会先添加 Handler
                do_accept();
            });
    }

} // namespace infra::net::listener