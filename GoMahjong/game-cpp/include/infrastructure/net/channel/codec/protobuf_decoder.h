#pragma once

#include "infrastructure/net/channel/i_channel_handler.h"

#include <memory>

// 前向声明 protobuf 生成的类
namespace gomahjong::net {
    class Envelope;
}

namespace infra::net::channel {

    /**
     * @brief Protobuf 解码器（进站反序列化）
     * 
     * 将 Bytes 反序列化为 Envelope 对象
     * 
     * 消息类型会变化：
     *   Bytes -> Envelope (本 Handler)
     *   Envelope -> 业务消息 (下游 Handler)
     */
    class ProtobufDecoder : public ChannelInboundHandler {
    public:
        void channel_read(ChannelHandlerContext& ctx, InboundMessage&& msg) override;

    private:
        std::shared_ptr<gomahjong::net::Envelope> parse_envelope(const Bytes& data) const;
    };

} // namespace infra::net::channel
