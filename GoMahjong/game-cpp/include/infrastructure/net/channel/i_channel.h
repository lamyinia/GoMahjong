#pragma once

namespace infra::net::channel {
    enum class ChannelType {
        Tcp, Websocket, Udp, Kcp
    };

    class IChannel {

    };
}