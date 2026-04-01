#include "infrastructure/net/channel/codec/length_field_decoder.h"
#include "infrastructure/log/logger.hpp"

#include <algorithm>
#include <cstdint>

namespace infra::net::channel {

    LengthFieldDecoder::LengthFieldDecoder(
        uint32_t max_frame_length,
        uint32_t length_field_offset,
        uint32_t length_field_length
    ) : max_frame_length_(max_frame_length)
      , length_field_offset_(length_field_offset)
      , length_field_length_(length_field_length) {}

    void LengthFieldDecoder::channel_read(ChannelHandlerContext& ctx, InboundMessage&& msg) {
        // 只处理 Bytes 类型
        if (!std::holds_alternative<Bytes>(msg)) {
            ctx.fire_channel_read(std::move(msg));
            return;
        }

        auto& data = std::get<Bytes>(msg);
        
        // 将新数据追加到缓冲区
        buffer_.insert(buffer_.end(), data.begin(), data.end());

        // 尝试解析所有完整的消息
        while (buffer_.size() >= length_field_offset_ + length_field_length_) {
            // 读取长度字段（大端序）
            uint32_t frame_len = read_length(buffer_);

            // 检查长度是否合理（防止恶意数据）
            if (!is_valid_frame(frame_len)) {
                LOG_ERROR("[LengthFieldDecoder] invalid frame length: {}", frame_len);
                ctx.fire_close();
                return;
            }

            // 检查是否收到完整消息
            size_t total_len = length_field_offset_ + length_field_length_ + frame_len;
            if (buffer_.size() < total_len) {
                // 数据不完整，等待更多数据
                break;
            }

            // 提取消息体（不包含长度字段）
            Bytes frame(buffer_.begin() + length_field_offset_ + length_field_length_, buffer_.begin() + total_len);
            
            // 从缓冲区移除已处理的数据
            buffer_.erase(buffer_.begin(), buffer_.begin() + total_len);

            // 传播到下一个 Handler
            ctx.fire_channel_read(Bytes(std::move(frame)));
        }
    }

    uint32_t LengthFieldDecoder::read_length(const Bytes& data) const {
        // 大端序读取 4 字节长度
        uint32_t length = 0;
        for (uint32_t i = 0; i < length_field_length_; ++i) {
            length = (length << 8) | data[length_field_offset_ + i];
        }
        return length;
    }

    bool LengthFieldDecoder::is_valid_frame(uint32_t frame_length) const {
        return frame_length > 0 && frame_length <= max_frame_length_;
    }

} // namespace infra::net::channel
