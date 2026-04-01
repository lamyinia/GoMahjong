#pragma once

#include "infrastructure/net/channel/i_channel_handler.h"

#include <cstdint>

namespace infra::net::channel {

    /**
     * @brief 长度字段解码器（进站拆包）
     * 
     * 协议格式：[4字节长度(大端序)][消息数据]
     * 
     * 处理粘包：循环读取，提取完整帧
     * 处理拆包：缓存不完整数据，等待更多数据
     */
    class LengthFieldDecoder : public ChannelInboundHandler {
    public:
        /**
         * @brief 构造函数
         * @param max_frame_length 最大帧长度（默认 4MB）
         * @param length_field_offset 长度字段偏移（默认 0）
         * @param length_field_length 长度字段长度（默认 4 字节）
         */
        explicit LengthFieldDecoder(
            uint32_t max_frame_length = 4 * 1024 * 1024,
            uint32_t length_field_offset = 0,
            uint32_t length_field_length = 4
        );

        void channel_read(ChannelHandlerContext& ctx, InboundMessage&& msg) override;

    private:
        uint32_t read_length(const Bytes& data) const;
        bool is_valid_frame(uint32_t frame_length) const;

    private:
        Bytes buffer_;
        uint32_t max_frame_length_;
        uint32_t length_field_offset_;
        uint32_t length_field_length_;
    };

} // namespace infra::net::channel
