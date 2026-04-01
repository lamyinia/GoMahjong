#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <system_error>
#include <vector>
#include <variant>


namespace infra::net::channel {

    // 前向声明
    class IChannel;
    class ChannelPipeline;
    struct Message;
    using MessagePtr = std::shared_ptr<Message>;

    // 消息类型
    using Bytes = std::vector<std::uint8_t>;
    using InboundMessage = std::variant<Bytes, MessagePtr>;
    using OutboundMessage = std::variant<Bytes, MessagePtr>;

    /**
     * @brief ChannelHandlerContext - Handler 的上下文
     * 
     * 提供：
     * - 访问 Channel 和 Pipeline
     * - 传播事件到下一个 Handler
     */
    class ChannelHandlerContext {
    public:
        virtual ~ChannelHandlerContext() = default;

        // === 访问器 ===

        virtual IChannel& channel() = 0;
        virtual ChannelPipeline& pipeline() = 0;

        // === 进站事件传播 ===

        virtual void fire_channel_active() = 0;
        virtual void fire_channel_read(InboundMessage&& msg) = 0;
        virtual void fire_channel_inactive() = 0;
        virtual void fire_exception_caught(const std::error_code& ec) = 0;

        // === 出站事件传播 ===

        virtual void fire_write(OutboundMessage&& msg) = 0;
        virtual void fire_flush() = 0;
        virtual void fire_close() = 0;
    };

} // namespace infra::net::channel
