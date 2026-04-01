#include "infrastructure/net/channel/codec/protobuf_decoder.h"
#include "infrastructure/log/logger.hpp"

// 包含 protobuf 生成的头文件
#include "proto/envelope.pb.h"

namespace infra::net::channel {

    void ProtobufDecoder::channel_read(ChannelHandlerContext& ctx, InboundMessage&& msg) {
        // 只处理 Bytes 类型
        if (!std::holds_alternative<Bytes>(msg)) {
            ctx.fire_channel_read(std::move(msg));
            return;
        }

        auto& data = std::get<Bytes>(msg);
        auto envelope = parse_envelope(data);
        if (!envelope) {
            // fix me 考虑发回错误码
            LOG_ERROR("[ProtobufDecoder] failed to parse envelope");
            ctx.fire_close();
            return;
        }

        // 创建 Message 并填充
        auto message = std::make_shared<Message>();
        // TODO: 从 Envelope 中提取 route 和 payload
        // message->route = envelope->route();
        // message->payload.assign(envelope->payload().begin(), envelope->payload().end());

        // 传播 MessagePtr 到下游 Handler
        ctx.fire_channel_read(message);
    }

    std::shared_ptr<gomahjong::net::Envelope> ProtobufDecoder::parse_envelope(const Bytes& data) const {
        auto envelope = std::make_shared<gomahjong::net::Envelope>();
        if (!envelope->ParseFromArray(data.data(), static_cast<int>(data.size()))) {
            return nullptr;
        }
        return envelope;
    }

} // namespace infra::net::channel
