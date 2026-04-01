#pragma once

#include <cstddef>
#include <cstdint>
#include <functional>
#include <memory>
#include <span>
#include <string>
#include <system_error>
#include <vector>

namespace infra::net::channel {

    // 前向声明
    class ChannelInboundHandler;
    class ChannelOutboundHandler;
    class ChannelDuplexHandler;
    class ChannelPipeline;

    enum class ChannelType {
        Tcp, Websocket, Udp, Kcp
    };

    /**
     * @brief Channel 接口
     * 
     * Channel 是对 Transport 的更高层抽象，提供：
     * - Handler 链管理（Pipeline）
     * - 进站/出站事件传播
     * - 统一的读写接口
     */
    class IChannel : public std::enable_shared_from_this<IChannel> {
    public:
        using Bytes = std::vector<std::uint8_t>;
        using OnError = std::function<void(const std::error_code&)>;

        virtual ~IChannel() = default;

        // === 基本信息 ===

        virtual ChannelType type() const noexcept = 0;
        virtual std::string id() const = 0;
        virtual bool is_active() const = 0;

        // === Handler 管理 ===

        virtual void add_inbound(std::shared_ptr<ChannelInboundHandler> handler) = 0;
        virtual void add_outbound(std::shared_ptr<ChannelOutboundHandler> handler) = 0;
        virtual void add_duplex(std::shared_ptr<ChannelDuplexHandler> handler) = 0;
        virtual ChannelPipeline& pipeline() = 0;

        // === 出站事件入口（由用户调用）===

        virtual void send(Bytes&& data) = 0;
        virtual void flush() = 0;
        virtual void close() = 0;

        // === 进站事件入口（由实现类调用）===

        virtual void start_read() = 0;

        // === Transport 操作（由 Pipeline 调用）===

        virtual void transport_write(Bytes&& data) = 0;
        virtual void transport_flush() = 0;
        virtual void transport_close() = 0;

        // === 错误处理 ===

        virtual void set_on_error(OnError on_error) = 0;
    };

} // namespace infra::net::channel