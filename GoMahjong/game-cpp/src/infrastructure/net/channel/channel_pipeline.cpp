#include "infrastructure/net/channel/channel_pipeline.h"
#include "infrastructure/net/channel/i_channel.h"
#include "infrastructure/net/transport/i_transport.h"

namespace infra::net::channel {

    // === DefaultChannelHandlerContext 实现 ===
    // 包含错误处理、连接发生、连接关闭、读事件、写事件

    DefaultChannelHandlerContext::DefaultChannelHandlerContext(
        IChannel& channel,
        ChannelPipeline& pipeline,
        size_t index,
        std::shared_ptr<ChannelInboundHandler> inbound,
        std::shared_ptr<ChannelOutboundHandler> outbound
    ) : channel_(channel)
      , pipeline_(pipeline)
      , index_(index)
      , inbound_(std::move(inbound))
      , outbound_(std::move(outbound)) {}

    void DefaultChannelHandlerContext::set_authorized(const std::string& player_id) {
        authorized_ = true;
        player_id_ = player_id;
    }

    // 进站事件传播
    void DefaultChannelHandlerContext::fire_channel_active() {
        pipeline_.fire_channel_active_from(index_ + 1);
    }

    void DefaultChannelHandlerContext::fire_channel_read(InboundMessage&& msg) {
        pipeline_.fire_channel_read_from(index_ + 1, std::move(msg));
    }

    void DefaultChannelHandlerContext::fire_channel_inactive() {
        pipeline_.fire_channel_inactive_from(index_ + 1);
    }

    void DefaultChannelHandlerContext::fire_exception_caught(const std::error_code& ec) {
        pipeline_.fire_exception_caught_from(index_ + 1, ec);
    }

    // 出站事件传播（反向）
    void DefaultChannelHandlerContext::fire_write(OutboundMessage&& msg) {
        pipeline_.fire_write_from(index_ - 1, std::move(msg));
    }

    void DefaultChannelHandlerContext::fire_flush() {
        pipeline_.fire_flush_from(index_ - 1);
    }

    void DefaultChannelHandlerContext::fire_close() {
        pipeline_.fire_close_from(index_ - 1);
    }

    // === ChannelPipeline 实现 ===

    ChannelPipeline::ChannelPipeline(IChannel& channel)
        : channel_(channel) {}

    // === Handler 管理 ===

    void ChannelPipeline::add_inbound(std::shared_ptr<ChannelInboundHandler> handler) {
        auto ctx = std::make_shared<DefaultChannelHandlerContext>(
            channel_, *this, contexts_.size(), std::move(handler), nullptr);
        contexts_.push_back(std::move(ctx));
    }

    void ChannelPipeline::add_outbound(std::shared_ptr<ChannelOutboundHandler> handler) {
        auto ctx = std::make_shared<DefaultChannelHandlerContext>(
            channel_, *this, contexts_.size(), nullptr, std::move(handler));
        contexts_.push_back(std::move(ctx));
    }

    void ChannelPipeline::add_duplex(std::shared_ptr<ChannelDuplexHandler> handler) {
        auto inbound_ptr = std::static_pointer_cast<ChannelInboundHandler>(handler);
        auto outbound_ptr = std::static_pointer_cast<ChannelOutboundHandler>(handler);
        auto ctx = std::make_shared<DefaultChannelHandlerContext>(
            channel_, *this, contexts_.size(), inbound_ptr, outbound_ptr);
        contexts_.push_back(std::move(ctx));
    }

    void ChannelPipeline::clear() {
        contexts_.clear();
    }

    // === 进站事件入口 ===

    void ChannelPipeline::fire_channel_active() {
        fire_channel_active_from(0);
    }

    void ChannelPipeline::fire_channel_read(InboundMessage&& msg) {
        fire_channel_read_from(0, std::move(msg));
    }

    void ChannelPipeline::fire_channel_inactive() {
        fire_channel_inactive_from(0);
    }

    void ChannelPipeline::fire_exception_caught(const std::error_code& ec) {
        fire_exception_caught_from(0, ec);
    }

    // === 出站事件入口（从最后一个开始反向传播）===

    void ChannelPipeline::fire_write(OutboundMessage&& msg) {
        if (contexts_.empty()) {
            // 没有 Handler，直接写入 Transport
            if (std::holds_alternative<Bytes>(msg)) {
                channel_.transport_write(std::move(std::get<Bytes>(msg)));
            }
            return;
        }
        fire_write_from(contexts_.size() - 1, std::move(msg));
    }

    void ChannelPipeline::fire_flush() {
        if (contexts_.empty()) {
            channel_.transport_flush();
            return;
        }
        fire_flush_from(contexts_.size() - 1);
    }

    void ChannelPipeline::fire_close() {
        if (contexts_.empty()) {
            channel_.transport_close();
            return;
        }
        fire_close_from(contexts_.size() - 1);
    }

    // === Context 传播实现 ===

    void ChannelPipeline::fire_channel_active_from(size_t index) {
        for (size_t i = index; i < contexts_.size(); ++i) {
            if (auto* handler = contexts_[i]->inbound_handler()) {
                handler->channel_active(*contexts_[i]);
            }
        }
    }

    void ChannelPipeline::fire_channel_read_from(size_t index, InboundMessage&& msg) {
        for (size_t i = index; i < contexts_.size(); ++i) {
            if (auto* handler = contexts_[i]->inbound_handler()) {
                handler->channel_read(*contexts_[i], std::move(msg));
                return;  // Handler 决定是否继续传播
            }
        }
    }

    void ChannelPipeline::fire_channel_inactive_from(size_t index) {
        for (size_t i = index; i < contexts_.size(); ++i) {
            if (auto* handler = contexts_[i]->inbound_handler()) {
                handler->channel_inactive(*contexts_[i]);
            }
        }
    }

    void ChannelPipeline::fire_exception_caught_from(size_t index, const std::error_code& ec) {
        for (size_t i = index; i < contexts_.size(); ++i) {
            if (auto* handler = contexts_[i]->inbound_handler()) {
                handler->exception_caught(*contexts_[i], ec);
            }
        }
    }

    void ChannelPipeline::fire_write_from(size_t index, OutboundMessage&& msg) {
        // 反向遍历
        for (int i = static_cast<int>(index); i >= 0; --i) {
            if (auto* handler = contexts_[i]->outbound_handler()) {
                handler->write(*contexts_[i], std::move(msg));
                return;  // Handler 决定是否继续传播
            }
        }
        // 所有 Outbound Handler 处理完毕，写入 Transport
        if (std::holds_alternative<Bytes>(msg)) {
            channel_.transport_write(std::move(std::get<Bytes>(msg)));
        }
    }

    void ChannelPipeline::fire_flush_from(size_t index) {
        for (int i = static_cast<int>(index); i >= 0; --i) {
            if (auto* handler = contexts_[i]->outbound_handler()) {
                handler->flush(*contexts_[i]);
            }
        }
        channel_.transport_flush();
    }

    void ChannelPipeline::fire_close_from(size_t index) {
        for (int i = static_cast<int>(index); i >= 0; --i) {
            if (auto* handler = contexts_[i]->outbound_handler()) {
                handler->close(*contexts_[i]);
            }
        }
        channel_.transport_close();
    }

} // namespace infra::net::channel
