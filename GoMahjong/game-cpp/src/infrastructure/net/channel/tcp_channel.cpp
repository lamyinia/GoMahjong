#include "infrastructure/net/channel/tcp_channel.h"
#include "infrastructure/log/logger.hpp"

#include <sstream>
#include <random>

namespace infra::net::channel {

    static std::atomic<uint64_t> channel_counter{0};

    std::string TcpChannel::generate_id() {
        std::ostringstream oss;
        oss << "tcp-" << ++channel_counter;
        return oss.str();
    }

    TcpChannel::TcpChannel(
        std::shared_ptr<transport::ITransport> transport,
        const Strand& strand
    )
        : transport_(std::move(transport))
        , pipeline_(*this)
        , strand_(strand)  // 拷贝，与 TcpTransport 共享同一个底层 strand 实现
        , id_(generate_id())
        , active_(true) {
        LOG_DEBUG("[TcpChannel] created: {}", id_);
    }

    TcpChannel::~TcpChannel() {
        if (active_.exchange(false)) {
            close();
        }
        LOG_DEBUG("[TcpChannel] destroyed: {}", id_);
    }

    // === Handler 管理 ===

    void TcpChannel::add_inbound(std::shared_ptr<ChannelInboundHandler> handler) {
        pipeline_.add_inbound(std::move(handler));
    }

    void TcpChannel::add_outbound(std::shared_ptr<ChannelOutboundHandler> handler) {
        pipeline_.add_outbound(std::move(handler));
    }

    void TcpChannel::add_duplex(std::shared_ptr<ChannelDuplexHandler> handler) {
        pipeline_.add_duplex(std::move(handler));
    }

    // === 出站事件入口 ===

    void TcpChannel::send(Bytes&& data) {
        if (!active_) {
            LOG_WARN("[TcpChannel] send on closed channel: {}", id_);
            return;
        }
        // strand_ 投递，保证 contexts_ 访问无锁
        boost::asio::post(strand_, [self = shared_from_this(), this, data = std::move(data)]() mutable {
            if (!active_) return;
            pipeline_.fire_write(std::move(data));
        });
    }

    void TcpChannel::flush() {
        if (!active_) return;
        boost::asio::post(strand_, [self = shared_from_this(), this]() {
            if (!active_) return;
            pipeline_.fire_flush();
        });
    }

    void TcpChannel::close() {
        if (!active_.exchange(false)) {
            return;
        }
        LOG_DEBUG("[TcpChannel] closing: {}", id_);
        boost::asio::post(strand_, [self = shared_from_this(), this]() {
            pipeline_.fire_close();
            pipeline_.fire_channel_inactive();
        });
    }

    // === 进站事件入口 ===

    void TcpChannel::start_read() {
        if (reading_.exchange(true)) {
            LOG_WARN("[TcpChannel] start_read already reading: {}", id_);
            return;
        }
        if (!transport_) {
            LOG_ERROR("[TcpChannel] start_read without transport: {}", id_);
            return;
        }
        pipeline_.fire_channel_active();
        auto self = shared_from_this();
        transport_->start(
            [this, self](Bytes&& data) { on_transport_bytes(std::move(data)); },
            [this, self]() { on_transport_closed(); },
            [this, self](const std::error_code& ec) { on_transport_error(ec); }
        );

        LOG_DEBUG("[TcpChannel] starting transport, channel_id={}", id_);
    }

    // === Transport 操作（由 Pipeline 调用）===

    void TcpChannel::transport_write(Bytes&& data) {
        if (transport_) {
            transport_->send(std::move(data));
        }
    }

    void TcpChannel::transport_flush() {
        // TCP Transport 没有 flush 概念，数据立即发送
    }

    void TcpChannel::transport_close() {
        if (transport_) {
            transport_->close();
        }
    }

    void TcpChannel::set_on_error(OnError on_error) {
        on_error_ = std::move(on_error);
    }

    // === Transport 回调 ===

    void TcpChannel::on_transport_bytes(Bytes&& data) {
        if (!active_) return;
        LOG_DEBUG("[TcpChannel] on_transport_bytes, size={}, firing channel_read", data.size());
        pipeline_.fire_channel_read(std::move(data));
    }

    void TcpChannel::on_transport_closed() {
        if (!active_.exchange(false)) return;
        LOG_DEBUG("[TcpChannel] transport closed: {}", id_);
        pipeline_.fire_channel_inactive();
    }

    void TcpChannel::on_transport_error(const std::error_code& ec) {
        LOG_ERROR("[TcpChannel] transport error: {} - {}", id_, ec.message());
        pipeline_.fire_exception_caught(ec);
        if (on_error_) {
            on_error_(ec);
        }
    }

} // namespace infra::net::channel
