#pragma once

#include <boost/asio.hpp>

#include <cstddef>
#include <cstdint>

namespace gomahjong::net {

class TcpServer {
public:
    using tcp = boost::asio::ip::tcp;

    TcpServer(
        boost::asio::io_context& ioc,
        unsigned short port,
        std::uint32_t max_frame_bytes,
        std::size_t max_accumulated_bytes,
        std::uint32_t idle_timeout_seconds);

    void start();

private:
    void do_accept();

private:
    boost::asio::io_context& ioc_;
    tcp::acceptor acceptor_;

    std::uint32_t max_frame_bytes_{0};
    std::size_t max_accumulated_bytes_{0};
    std::uint32_t idle_timeout_seconds_{0};
};

} // namespace gomahjong::net
