#include "infrastructure/net/channel/codec/protobuf_encoder.h"
#include "infrastructure/log/logger.hpp"

// 包含 protobuf 生成的头文件
#include "generated/envelope.pb.h"

namespace infra::net::channel {

    void ProtobufEncoder::write(ChannelHandlerContext& ctx, OutboundMessage&& msg) {
        // 只处理 MessagePtr 类型
        if (!std::holds_alternative<MessagePtr>(msg)) {
            ctx.fire_write(std::move(msg));
            return;
        }

        auto& message = std::get<MessagePtr>(msg);
        if (!message) {
            LOG_ERROR("[ProtobufEncoder] message is null");
            return;
        }

        // 序列化为 Bytes
        auto bytes = serialize_envelope(*message);
        if (bytes.empty()) {
            LOG_ERROR("[ProtobufEncoder] failed to serialize message");
            return;
        }

        // 传播 Bytes 到下游 Handler
        ctx.fire_write(Bytes(std::move(bytes)));
    }

    Bytes ProtobufEncoder::serialize_envelope(const Message& message) const {
        gomahjong::net::Envelope envelope;

        if (message.route.empty()) {
            LOG_ERROR("[ProtobufEncoder] empty route");
            return {};
        }

        envelope.set_route(message.route);
        if (!message.payload.empty()) {
            envelope.set_payload(message.payload.data(), message.payload.size());
        }
        envelope.set_client_seq(message.client_seq);

        Bytes bytes;
        bytes.resize(envelope.ByteSizeLong());
        
        if (!envelope.SerializeToArray(bytes.data(), static_cast<int>(bytes.size()))) {
            LOG_ERROR("[ProtobufEncoder] failed to serialize envelope");
            return {};
        }

        return bytes;
    }

} // namespace infra::net::channel
