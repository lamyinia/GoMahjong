#include "tcp_connection.hpp"

#include <google/protobuf/stubs/common.h>

#include <cstring>
#include <iostream>

#include "envelope.pb.h"

namespace gomahjong::net {

    static constexpr std::size_t kMaxWriteQueueBytes = 1024 * 1024; // 1MB

    TcpConnection::TcpConnection(
        tcp::socket socket,
        boost::asio::any_io_executor executor,
        std::uint32_t max_frame_bytes,
        std::size_t max_accumulated_bytes,
        std::uint32_t idle_timeout_seconds)
        : socket_(std::move(socket)),
          idle_timer_(executor),
          idle_timeout_(std::chrono::seconds(idle_timeout_seconds)),
          decoder_(max_frame_bytes, max_accumulated_bytes) {}

    void TcpConnection::start() {
        refresh_idle_timer();
        do_read();

        try {
            auto ep = socket_.remote_endpoint();
            std::cout << "[tcp] open " << ep.address().to_string() << ":" << ep.port() << "\n";
        } catch (...) {
            std::cout << "[tcp] open (unknown endpoint)\n";
        }
    }

    void TcpConnection::stop() {
        if (stopped_) {
            return;
        }
        stopped_ = true;

        boost::system::error_code ignored;
        idle_timer_.cancel(ignored);
        socket_.shutdown(tcp::socket::shutdown_both, ignored);
        socket_.close(ignored);

        std::cout << "[tcp] close\n";
    }

    void TcpConnection::do_read() {
        auto self = shared_from_this();
        socket_.async_read_some(
                boost::asio::buffer(read_buf_), [self](const boost::system::error_code &ec, std::size_t n) {
                    self->on_read(ec, n);
                });
    }

    void TcpConnection::on_read(const boost::system::error_code &ec, std::size_t n) {
        if (stopped_) {
            return;
        }

        if (ec) {
            std::cout << "[tcp] read error: " << ec.message() << "\n";
            stop();
            return;
        }

        refresh_idle_timer();

        if (!decoder_.append(std::span<const std::uint8_t>(read_buf_.data(), n))) {
            std::cout << "[proto] accumulated bytes exceed limit\n";
            stop();
            return;
        }

        for (;;) {
            //  还能拿出一帧就继续处理 —— 同一次读里有多帧时会被连续弹出 —— 这就是 粘包（多帧粘在一起）的处理。
            if (decoder_.invalid_length()) {
                std::cout << "[proto] invalid frame length\n";
                stop();
                return;
            }

            auto frameOpt = decoder_.try_pop();
            if (!frameOpt.has_value()) {
                break;
            }

            handle_frame(*frameOpt);
            if (stopped_) {
                return;
            }
        }

        do_read();
    }

    void TcpConnection::refresh_idle_timer() {
        idle_timer_.expires_after(idle_timeout_);

        auto self = shared_from_this();
        idle_timer_.async_wait([self](const boost::system::error_code &ec) {
            self->on_idle_timeout(ec);
        });
    }

    void TcpConnection::on_idle_timeout(const boost::system::error_code &ec) {
        if (stopped_) {
            return;
        }
        if (ec == boost::asio::error::operation_aborted) {
            return;
        }

        std::cout << "[tcp] idle timeout\n";
        stop();
    }

    void TcpConnection::handle_frame(const Frame &frame) {
        gomahjong::net::Envelope env;
        if (!env.ParseFromArray(frame.body.data(), static_cast<int>(frame.body.size()))) {
            std::cout << "[proto] decode failed size=" << frame.body.size() << "\n";
        } else {
            std::cout << "[msg] route=" << env.route() << " size=" << frame.body.size() << "\n";
        }

        // Echo original frame (length-prefix + body) back.
        auto len_prefix = encode_length_be(static_cast<std::uint32_t>(frame.body.size()));
        auto out = std::make_shared<std::string>();
        out->resize(4 + frame.body.size());
        std::memcpy(out->data(), len_prefix.data(), 4);
        if (!frame.body.empty()) {
            std::memcpy(out->data() + 4, frame.body.data(), frame.body.size());
        }

        async_write(std::move(out));
    }

    void TcpConnection::async_write(std::shared_ptr<std::string> data) {
        if (stopped_) {
            return;
        }

        if (write_queued_bytes_ + data->size() > kMaxWriteQueueBytes) {
            std::cout << "[tcp] write queue overflow, closing\n";
            stop();
            return;
        }

        bool writing = !write_queue_.empty();
        write_queued_bytes_ += data->size();
        write_queue_.push_back(std::move(data));

        if (!writing) {
            do_write();
        }
    }

    void TcpConnection::do_write() {
        if (stopped_ || write_queue_.empty()) {
            return;
        }

        auto self = shared_from_this();
        auto &front = write_queue_.front();
        boost::asio::async_write(
                socket_,
                boost::asio::buffer(*front),
                [self](const boost::system::error_code &ec, std::size_t n) {
                    self->on_write(ec, n);
                });
    }

    void TcpConnection::on_write(const boost::system::error_code &ec, std::size_t /*n*/) {
        if (stopped_) {
            return;
        }

        if (ec) {
            std::cout << "[tcp] write error: " << ec.message() << "\n";
            stop();
            return;
        }

        if (!write_queue_.empty()) {
            write_queued_bytes_ -= write_queue_.front()->size();
            write_queue_.pop_front();
        }

        if (!write_queue_.empty()) {
            do_write();
        }
    }

} // namespace gomahjong::net
