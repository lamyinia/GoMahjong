#pragma once

#include <string>
#include <unordered_map>

#include "infrastructure/net/channel/i_channel.h"


namespace infra::net::session {
    namespace channel = infra::net::channel;

    class Session {
    private:
        std::string playerID;
        std::unordered_map<channel::ChannelType, channel::IChannel> channels;
    };
}