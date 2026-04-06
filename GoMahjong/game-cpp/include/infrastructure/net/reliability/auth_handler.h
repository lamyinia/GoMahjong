#pragma once

#include "infrastructure/net/channel/i_channel_handler.h"
#include "infrastructure/net/channel/message.h"

#include <memory>
#include <string>

namespace infra::net::reliability {

    // 前向声明
    class WildEndpoint;

    /**
     * @brief 认证 Handler
     * 
     * 处理客户端的认证请求：
     * - 解析 AuthRequest 消息
     * - 验证 token
     * - 通知 WildEndpoint 认证结果
     */
    class AuthHandler : public channel::ChannelInboundHandler {
    public:
        explicit AuthHandler(std::shared_ptr<WildEndpoint> endpoint);

        void channel_read(channel::ChannelHandlerContext& ctx, channel::InboundMessage&& msg) override;

    private:
        void handle_auth_request(channel::ChannelHandlerContext& ctx, 
                                 const channel::MessagePtr& msg);

        // virtual bool verify_token(const std::string& token, std::string& player_id);

        std::weak_ptr<WildEndpoint> endpoint_;
    };

} // namespace infra::net::reliability
