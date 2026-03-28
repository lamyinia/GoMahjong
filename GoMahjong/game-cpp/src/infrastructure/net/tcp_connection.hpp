#pragma once

#include <boost/asio.hpp>

#include <chrono>
#include <cstddef>
#include <cstdint>
#include <deque>
#include <memory>
#include <string>
#include <vector>

#include "frame_codec.hpp"

namespace gomahjong::net {

class TcpConnection : public std::enable_shared_from_this<TcpConnection> {
public:
    using tcp = boost::asio::ip::tcp;

    TcpConnection(
        tcp::socket socket,
        boost::asio::any_io_executor executor,
        std::uint32_t max_frame_bytes,
        std::size_t max_accumulated_bytes,
        std::uint32_t idle_timeout_seconds);

    void start();
    void stop();

private:
    void do_read();
    void on_read(const boost::system::error_code& ec, std::size_t n);

    void refresh_idle_timer();
    void on_idle_timeout(const boost::system::error_code& ec);

    void handle_frame(const Frame& frame);

    void async_write(std::shared_ptr<std::string> data);
    void do_write();
    void on_write(const boost::system::error_code& ec, std::size_t n);

private:
    tcp::socket socket_;
    boost::asio::steady_timer idle_timer_;
    std::chrono::seconds idle_timeout_{0};

    std::array<std::uint8_t, 4096> read_buf_{};
    FrameDecoder decoder_;

    std::deque<std::shared_ptr<std::string>> write_queue_;
    std::size_t write_queued_bytes_{0};

    bool stopped_{false};
};

} // namespace gomahjong::net
