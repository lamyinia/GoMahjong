#include "infrastructure/net/reliability/wild_endpoint.h"
#include "infrastructure/net/reliability/auth_handler.h"
#include "infrastructure/net/channel/codec/length_field_decoder.h"
#include "infrastructure/net/channel/codec/length_field_encoder.h"
#include "infrastructure/net/channel/codec/protobuf_decoder.h"
#include "infrastructure/net/channel/codec/protobuf_encoder.h"
#include "infrastructure/log/logger.hpp"

#include <boost/asio/io_context.hpp>
#include <boost/system/error_code.hpp>

namespace infra::net::reliability {

    WildEndpoint::WildEndpoint(boost::asio::any_io_executor executor,
                               std::shared_ptr<channel::IChannel> channel,
                               std::chrono::milliseconds timeout)
        : executor_(std::move(executor))
        , channel_(std::move(channel))
        , timer_(executor_)
        , timeout_(timeout) {
        if (channel_) {
            id_ = channel_->id();
        }
    }

    WildEndpoint::~WildEndpoint() {
        cancel_timer();
    }

    void WildEndpoint::start_wait_auth() {
        if (!channel_) {
            LOG_ERROR("[WildEndpoint] channel is null");
            return;
        }

        LOG_DEBUG("[WildEndpoint] start waiting auth, channel_id={}", id_);

        // 1. 添加 Codec Handler 到 Pipeline
        // 进站：Bytes -> Bytes (拆包) -> MessagePtr (反序列化)
        channel_->add_inbound(std::make_shared<channel::LengthFieldDecoder>());
        channel_->add_inbound(std::make_shared<channel::ProtobufDecoder>());

        // 出站：MessagePtr -> Bytes (序列化) -> Bytes (加长度头)
        channel_->add_outbound(std::make_shared<channel::ProtobufEncoder>());
        channel_->add_outbound(std::make_shared<channel::LengthFieldEncoder>());

        // 2. 添加 AuthHandler 到 Pipeline
        channel_->add_inbound(std::make_shared<AuthHandler>(shared_from_this()));

        // 3. 启动超时定时器
        start_timeout_timer();

        // 4. 开始读取数据
        channel_->start_read();
    }

    void WildEndpoint::on_auth_success(const std::string& player_id) {
        if (auth_done_.exchange(true)) {
            return; // 已经处理过
        }

        LOG_INFO("[WildEndpoint] auth success, channel_id={}, player_id={}", id_, player_id);

        cancel_timer();

        if (onAuthSuccess_) {
            onAuthSuccess_(player_id);
        }
    }

    void WildEndpoint::on_auth_failed() {
        if (auth_done_.exchange(true)) {
            return; // 已经处理过
        }

        LOG_WARN("[WildEndpoint] auth failed, channel_id={}", id_);

        cancel_timer();

        if (onAuthFailed_) {
            onAuthFailed_();
        }

        // 关闭连接
        if (channel_) {
            channel_->close();
        }
    }

    void WildEndpoint::start_timeout_timer() {
        timer_.expires_after(timeout_);
        timer_.async_wait([self = shared_from_this()](const boost::system::error_code& ec) {
            self->handle_timeout(ec);
        });
    }

    void WildEndpoint::cancel_timer() {
        boost::system::error_code ec;
        timer_.cancel(ec);
    }

    void WildEndpoint::handle_timeout(const boost::system::error_code& ec) {
        if (ec == boost::asio::error::operation_aborted) {
            return; // 定时器被取消
        }

        if (!ec && !auth_done_.load()) {
            LOG_WARN("[WildEndpoint] auth timeout, channel_id={}", id_);
            on_auth_failed();
        }
    }

} // namespace infra::net::reliability