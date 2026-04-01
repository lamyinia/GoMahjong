#pragma once

#include "i_channel_handler.h"
#include "channel_handler_context.h"

#include <deque>
#include <memory>
#include <mutex>
#include <vector>

namespace infra::net::channel {

    // 前向声明
    class IChannel;

    /**
     * @brief 默认的 ChannelHandlerContext 实现
     */
    class DefaultChannelHandlerContext : public ChannelHandlerContext {
    public:
        DefaultChannelHandlerContext(
            IChannel& channel,
            ChannelPipeline& pipeline,
            size_t index,
            std::shared_ptr<ChannelInboundHandler> inbound,
            std::shared_ptr<ChannelOutboundHandler> outbound
        );

        IChannel& channel() override { return channel_; }
        ChannelPipeline& pipeline() override { return pipeline_; }

        void fire_channel_active() override;
        void fire_channel_read(InboundMessage&& msg) override;
        void fire_channel_inactive() override;
        void fire_exception_caught(const std::error_code& ec) override;

        void fire_write(OutboundMessage&& msg) override;
        void fire_flush() override;
        void fire_close() override;

        ChannelInboundHandler* inbound_handler() const { return inbound_.get(); }
        ChannelOutboundHandler* outbound_handler() const { return outbound_.get(); }
        size_t index() const { return index_; }

    private:
        IChannel& channel_;
        ChannelPipeline& pipeline_;
        size_t index_;
        std::shared_ptr<ChannelInboundHandler> inbound_;
        std::shared_ptr<ChannelOutboundHandler> outbound_;
    };

    /**
     * @brief Channel Pipeline（Handler 链管理）
     */
    class ChannelPipeline {
    public:
        explicit ChannelPipeline(IChannel& channel);

        void add_inbound(std::shared_ptr<ChannelInboundHandler> handler);
        void add_outbound(std::shared_ptr<ChannelOutboundHandler> handler);
        void add_duplex(std::shared_ptr<ChannelDuplexHandler> handler);
        void clear();

        // 进站事件入口
        void fire_channel_active();
        void fire_channel_read(InboundMessage&& msg);
        void fire_channel_inactive();
        void fire_exception_caught(const std::error_code& ec);

        // 出站事件入口
        void fire_write(OutboundMessage&& msg);
        void fire_flush();
        void fire_close();

        // Context 传播
        void fire_channel_active_from(size_t index);
        void fire_channel_read_from(size_t index, InboundMessage&& msg);
        void fire_channel_inactive_from(size_t index);
        void fire_exception_caught_from(size_t index, const std::error_code& ec);

        void fire_write_from(size_t index, OutboundMessage&& msg);
        void fire_flush_from(size_t index);
        void fire_close_from(size_t index);

    private:
        IChannel& channel_;
        std::mutex mutex_;
        std::vector<std::shared_ptr<DefaultChannelHandlerContext>> contexts_;
    };

} // namespace infra::net::channel
