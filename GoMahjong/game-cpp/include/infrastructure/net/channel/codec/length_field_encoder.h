#pragma once

#include "infrastructure/net/channel/i_channel_handler.h"

#include <cstdint>

namespace infra::net::channel {

    /**
     * @brief 长度字段编码器（出站加长度头）
     * 
     * 协议格式：[4字节长度(大端序)][消息数据]
     */
    class LengthFieldEncoder : public ChannelOutboundHandler {
    public:
        /**
         * @brief 构造函数
         * @param length_field_length 长度字段长度（默认 4 字节）
         */
        explicit LengthFieldEncoder(uint32_t length_field_length = 4);

        void write(ChannelHandlerContext& ctx, OutboundMessage&& msg) override;

    private:
        void write_length(Bytes& out, uint32_t length) const;

    private:
        uint32_t length_field_length_;
    };

} // namespace infra::net::channel
