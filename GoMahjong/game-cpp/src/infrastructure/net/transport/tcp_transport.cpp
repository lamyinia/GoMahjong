#include "infrastructure/net/transport/tcp_transport.h"
#include "infrastructure/net/transport/i_transport.h"

#include <boost/asio/write.hpp>
#include <boost/asio/post.hpp>

#include <algorithm>
#include <cerrno>
#include <cstring>
#include <system_error>

namespace infra::net::transport {
    void TcpTransport::start(OnBytes onBytes, OnClosed onClosed, OnError onError) {
        boost::asio::post(strand_, [self = shared_from_this(), this,
                onBytes = std::move(onBytes),
                onClosed = std::move(onClosed),
                onError = std::move(onError)]() mutable {
            if (closed_) return;
            if (started_) return;
            started_ = true;

            onBytes_ = std::move(onBytes);
            onClosed_ = std::move(onClosed);
            onError_ = std::move(onError);

            do_read();
        });
    }

    void TcpTransport::send(Bytes &&data) {
        boost::asio::post(strand_, [self = shared_from_this(), this, data = std::move(data)]() mutable {
            if (closed_) return;
            if (!socket_.is_open()) return;

            write_queue_.push_back(std::move(data));
            if (writing_) return;

            writing_ = true;
            do_write();
        });
    }

    void TcpTransport::close() {
        boost::asio::post(strand_, [self = shared_from_this(), this] { do_close(); });
    }

    bool TcpTransport::is_closed() const noexcept { return !socket_.is_open(); }

    void TcpTransport::do_read() {
        if (closed_) return;
        if (!socket_.is_open()) {
            do_close();
            return;
        }

        socket_.async_read_some(
                boost::asio::buffer(read_buf_),
                boost::asio::bind_executor(
                        strand_,
                        [self = shared_from_this(), this](const boost::system::error_code &ec,
                                                          std::size_t n) {
                            if (closed_) return;

                            if (ec) {
                                if (ec == boost::asio::error::eof ||
                                    ec == boost::asio::error::connection_reset ||
                                    ec == boost::asio::error::operation_aborted) {
                                    do_close();
                                    return;
                                }

                                if (onError_) {
                                    const std::error_code se(ec.value(), std::system_category());
                                    onError_(se);
                                }
                                do_close();
                                return;
                            }

                            if (n > 0 && onBytes_) {
                                Bytes bytes;
                                bytes.insert(bytes.end(), read_buf_.begin(), read_buf_.begin() + n);
                                onBytes_(std::move(bytes));
                            }

                            do_read();
                        }));
    }

    void TcpTransport::do_write() {
        if (closed_) {
            writing_ = false;
            return;
        }
        if (!socket_.is_open()) {
            do_close();
            writing_ = false;
            return;
        }
        if (write_queue_.empty()) {
            writing_ = false;
            return;
        }

        boost::asio::async_write(
                socket_,
                boost::asio::buffer(write_queue_.front()),
                boost::asio::bind_executor(strand_, [self = shared_from_this(), this](const boost::system::error_code &ec, std::size_t) {
                            if (closed_) {
                                writing_ = false;
                                return;
                            }

                            if (ec) {
                                if (onError_) {
                                    const std::error_code se(ec.value(), std::system_category());
                                    onError_(se);
                                }
                                do_close();
                                writing_ = false;
                                return;
                            }

                            if (!write_queue_.empty()) {
                                write_queue_.pop_front();
                            }

                            do_write();
                        }));
    }

    void TcpTransport::do_close() {
        if (closed_) return;
        closed_ = true;

        boost::system::error_code ignored;
        if (socket_.is_open()) {
            socket_.shutdown(boost::asio::ip::tcp::socket::shutdown_both, ignored);
            socket_.close(ignored);
        }

        // drop pending writes
        write_queue_.clear();
        writing_ = false;

        // invoke callback once
        if (onClosed_) {
            auto cb = std::move(onClosed_);
            onClosed_ = nullptr;
            cb();
        }
    }


} // namespace infra::net::transport