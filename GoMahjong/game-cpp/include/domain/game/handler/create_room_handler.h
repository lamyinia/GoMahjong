#pragma once

#include "infrastructure/net/channel/channel_handler_context.h"
#include "infrastructure/net/channel/message.h"

#include <functional>
#include <memory>

namespace domain::game::handler {

    namespace channel = infra::net::channel;

    void handleDebugCreateRoom(channel::ChannelHandlerContext& ctx,
                               const channel::MessagePtr& msg);

} // namespace domain::game::handler
