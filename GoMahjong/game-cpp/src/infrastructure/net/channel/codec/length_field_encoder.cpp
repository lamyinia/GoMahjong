#include "infrastructure/net/channel/codec/length_field_encoder.h"

#include <cstdint>

namespace infra::net::channel {

    LengthFieldEncoder::LengthFieldEncoder(uint32_t length_field_length)
        : length_field_length_(length_field_length) {}

    void LengthFieldEncoder::write(ChannelHandlerContext& ctx, OutboundMessage&& msg) {
        // 只处理 Bytes 类型
        if (!std::holds_alternative<Bytes>(msg)) {
            ctx.fire_write(std::move(msg));
            return;
        }

        auto& data = std::get<Bytes>(msg);
        
        // 创建带长度头的数据帧
        Bytes frame;
        frame.reserve(length_field_length_ + data.size());

        // 写入长度字段（大端序）
        uint32_t len = static_cast<uint32_t>(data.size());
        for (int i = static_cast<int>(length_field_length_) - 1; i >= 0; --i) {
            frame.push_back(static_cast<uint8_t>((len >> (i * 8)) & 0xFF));
        }

        // 写入消息体
        frame.insert(frame.end(), data.begin(), data.end());

        // 传播到下一个 Handler
        ctx.fire_write(Bytes(std::move(frame)));
    }

    void LengthFieldEncoder::write_length(Bytes& out, uint32_t length) const {
        // 大端序写入
        for (int i = static_cast<int>(length_field_length_) - 1; i >= 0; --i) {
            out.push_back(static_cast<uint8_t>((length >> (i * 8)) & 0xFF));
        }
    }

} // namespace infra::net::channel
