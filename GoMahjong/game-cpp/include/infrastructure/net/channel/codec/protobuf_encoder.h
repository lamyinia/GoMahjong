#pragma once

#include "infrastructure/net/channel/i_channel_handler.h"

#include <memory>

// 前向声明 protobuf 生成的类
namespace gomahjong::net {
    class Envelope;
}

namespace infra::net::channel {

    /**
     * @brief Protobuf 编码器（出站序列化）
     * 
     * 将 MessagePtr 序列化为 Bytes
     * 
     * 消息类型会变化：
     *   MessagePtr -> Bytes (本 Handler)
     *   Bytes -> 网络传输 (下游 Handler)
     */
    class ProtobufEncoder : public ChannelOutboundHandler {
    public:
        void write(ChannelHandlerContext& ctx, OutboundMessage&& msg) override;

    private:
        Bytes serialize_envelope(const Message& message) const;
    };

} // namespace infra::net::channel
