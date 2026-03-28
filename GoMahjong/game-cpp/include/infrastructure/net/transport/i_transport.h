#pragma once

#include <memory>
#include <functional>
#include <vector>

namespace infra::net::transport {
    class ITransport : public std::enable_shared_from_this<ITransport> {
    public:
        virtual ~ITransport() = default;

        virtual void send(std::vector<uint8_t> &&data) = 0;

        virtual void close() = 0;

    };
}
