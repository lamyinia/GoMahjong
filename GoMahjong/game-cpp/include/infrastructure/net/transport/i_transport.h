#pragma once

#include <cstddef>
#include <cstdint>
#include <functional>
#include <memory>
#include <system_error>
#include <vector>

namespace infra::net::transport {

    class ITransport : public std::enable_shared_from_this<ITransport> {
    public:
        using Bytes = std::vector<std::uint8_t>;

        using OnBytes = std::function<void(Bytes &&)>;
        using OnClosed = std::function<void()>;
        using OnError = std::function<void(const std::error_code &)>;

        virtual ~ITransport() = default;

        virtual void start(OnBytes onBytes, OnClosed onClosed, OnError onError) = 0;

        virtual void send(Bytes &&data) = 0;

        virtual void close() = 0;

        virtual bool is_closed() const noexcept = 0;
    };

} // namespace infra::net::transport