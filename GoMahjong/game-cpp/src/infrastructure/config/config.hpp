#pragma once

#include <cstdint>
#include <string>

namespace infra::config {

struct TcpConfig {
    std::uint16_t port{7000};
    std::uint32_t max_frame_bytes{64 * 1024};
    std::uint32_t idle_timeout_seconds{60};
};

struct NetConfig {
    TcpConfig tcp;
};

struct LogConfig {
    // Reserved for future log settings
};

struct ServerConfig {
    NetConfig net;
    LogConfig log;
};

class Config {
public:
    static Config load_from_file(const std::string& path);
    static Config load_from_file_or_default(const std::string& path) noexcept;

    const ServerConfig& server() const noexcept { return server_; }

private:
    ServerConfig server_;
};

} // namespace infra::config
