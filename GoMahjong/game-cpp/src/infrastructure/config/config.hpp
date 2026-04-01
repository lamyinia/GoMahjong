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

struct EtcdConfig {
    std::string endpoints{"http://127.0.0.1:2379"};
    std::int64_t ttl_seconds{10};
};

struct GrpcConfig {
    std::uint16_t port{9010};
};

struct MongoConfig {
    std::string uri{"mongodb://localhost:27017"};
    std::string database{"gomahjong"};
    std::uint32_t min_pool_size{10};
    std::uint32_t max_pool_size{100};
    std::string username;
    std::string password;
};

struct ServerConfig {
    NetConfig net;
    LogConfig log;
    EtcdConfig etcd;
    GrpcConfig grpc;
    MongoConfig mongodb;
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
