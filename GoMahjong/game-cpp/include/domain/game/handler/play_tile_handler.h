#pragma once

#include "infrastructure/net/channel/channel_handler_context.h"
#include "infrastructure/net/channel/message.h"

#include <functional>
#include <memory>

namespace domain::game::handler {

    namespace channel = infra::net::channel;

    /**
     * @brief 注册游戏相关的 Handler
     */
    void registerGameHandlers();

    /**
     * @brief 处理出牌请求
     * @param ctx Handler 上下文
     * @param msg 消息
     */
    void handlePlayTile(channel::ChannelHandlerContext& ctx, 
                        const channel::MessagePtr& msg);

} // namespace domain::game::handler
