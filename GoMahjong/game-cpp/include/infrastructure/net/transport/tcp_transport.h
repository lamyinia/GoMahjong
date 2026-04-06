#pragma once

#include "infrastructure/net/transport/i_transport.h"

#include <boost/asio.hpp>
 #include <boost/asio/strand.hpp>

#include <array>
#include <cstddef>
#include <cstdint>
#include <deque>
#include <functional>
#include <memory>
#include <system_error>
#include <utility>

namespace infra::net::transport {

    class TcpTransport final : public ITransport {
    public:
        explicit TcpTransport(boost::asio::ip::tcp::socket socket)
                : socket_(std::move(socket)), strand_(socket_.get_executor()) {}

        void start(OnBytes onBytes, OnClosed onClosed, OnError onError) override;

        void send(Bytes &&data) override;

        void close() override;

        bool is_closed() const noexcept override;

        Strand strand() const override;

    private:
        void do_read();
        void do_write();
        void do_close();

        boost::asio::ip::tcp::socket socket_;

        boost::asio::strand<boost::asio::any_io_executor> strand_;

        OnBytes onBytes_{};
        OnClosed onClosed_{};
        OnError onError_{};

        std::array<std::uint8_t, 4096> read_buf_{};
        std::deque<Bytes> write_queue_{};
        bool writing_{false};
        bool started_{false};
        bool closed_{false};
    };

} // namespace infra::net::transport
