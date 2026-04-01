#pragma once

#include "channel_handler_context.h"
#include "message.h"

#include <memory>
#include <variant>

namespace infra::net::channel {

    // 消息类型：可以是原始字节或解析后的消息
    using InboundMessage = std::variant<Bytes, MessagePtr>;
    using OutboundMessage = std::variant<Bytes, MessagePtr>;

    /**
     * @brief 进站 Handler 接口
     * 
     * 处理从底层往上层传播的事件：
     *   channel_active -> channel_read -> channel_inactive -> exception_caught
     * 
     * 通过 ctx.fire_xxx() 传播到下一个 Handler
     */
    class ChannelInboundHandler {
    public:
        virtual ~ChannelInboundHandler() = default;

        virtual void channel_active(ChannelHandlerContext& ctx) {
            ctx.fire_channel_active();
        }

        /**
         * @brief 收到数据时调用
         * @param ctx Handler 上下文
         * @param msg 收到的消息（Bytes 或 MessagePtr）
         */
        virtual void channel_read(ChannelHandlerContext& ctx, InboundMessage&& msg) {
            ctx.fire_channel_read(std::move(msg));
        }

        virtual void channel_inactive(ChannelHandlerContext& ctx) {
            ctx.fire_channel_inactive();
        }

        virtual void exception_caught(ChannelHandlerContext& ctx, const std::error_code& ec) {
            ctx.fire_exception_caught(ec);
        }
    };

    /**
     * @brief 出站 Handler 接口
     * 
     * 处理从上层往底层传播的事件：
     *   write -> flush -> close
     */
    class ChannelOutboundHandler {
    public:
        virtual ~ChannelOutboundHandler() = default;

        virtual void write(ChannelHandlerContext& ctx, OutboundMessage&& msg) {
            ctx.fire_write(std::move(msg));
        }

        virtual void flush(ChannelHandlerContext& ctx) {
            ctx.fire_flush();
        }

        virtual void close(ChannelHandlerContext& ctx) {
            ctx.fire_close();
        }
    };

    /**
     * @brief 双向 Handler（同时处理进站和出站）
     */
    class ChannelDuplexHandler : public ChannelInboundHandler, public ChannelOutboundHandler {
    public:
        ~ChannelDuplexHandler() override = default;
    };

} // namespace infra::net::channel
