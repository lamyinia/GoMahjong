#pragma once

#include <cstddef>
#include <cstdint>
#include <functional>
#include <memory>
#include <span>
#include <vector>

namespace infra::net::channel {

    enum class ChannelType {
        Tcp, Websocket, Udp, Kcp
    };

    class IChannel {
    public:
        using Bytes = std::vector<std::uint8_t>;

        virtual ~IChannel() = default;

        virtual ChannelType type() const noexcept = 0;

        virtual void close() = 0;
    };

} // namespace infra::net::channel