#include "tcp_server.hpp"

#include <iostream>

#include "tcp_connection.hpp"

namespace gomahjong::net {

TcpServer::TcpServer(
    boost::asio::io_context& ioc,
    unsigned short port,
    std::uint32_t max_frame_bytes,
    std::size_t max_accumulated_bytes,
    std::uint32_t idle_timeout_seconds)
    : ioc_(ioc),
      acceptor_(ioc),
      max_frame_bytes_(max_frame_bytes),
      max_accumulated_bytes_(max_accumulated_bytes),
      idle_timeout_seconds_(idle_timeout_seconds) {
    tcp::endpoint ep{tcp::v4(), port};

    boost::system::error_code ec;
    acceptor_.open(ep.protocol(), ec);
    if (ec) {
        throw std::runtime_error("acceptor open failed: " + ec.message());
    }

    acceptor_.set_option(boost::asio::socket_base::reuse_address(true), ec);
    if (ec) {
        throw std::runtime_error("acceptor set_option failed: " + ec.message());
    }

    acceptor_.bind(ep, ec);
    if (ec) {
        throw std::runtime_error("acceptor bind failed: " + ec.message());
    }

    acceptor_.listen(boost::asio::socket_base::max_listen_connections, ec);
    if (ec) {
        throw std::runtime_error("acceptor listen failed: " + ec.message());
    }
}

void TcpServer::start() {
    std::cout << "[tcp] listen :" << acceptor_.local_endpoint().port() << "\n";
    do_accept();
}

void TcpServer::do_accept() {
    acceptor_.async_accept(
        boost::asio::make_strand(ioc_),
        [this](const boost::system::error_code& ec, tcp::socket socket) {
            if (!ec) {
                auto conn = std::make_shared<TcpConnection>(
                    std::move(socket),
                    ioc_.get_executor(),
                    max_frame_bytes_,
                    max_accumulated_bytes_,
                    idle_timeout_seconds_);
                conn->start();
            } else {
                std::cout << "[tcp] accept error: " << ec.message() << "\n";
            }

            do_accept();
        });
}

} // namespace gomahjong::net
