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
    std::string level{"info"};          // trace, debug, info, warn, error, critical, off
    bool short_source{false};            // true: 只显示 file:line，不显示函数签名
};

struct EtcdConfig {
    std::string endpoints{"http://127.0.0.1:2379"};
    std::int64_t ttl_seconds{10};
    std::int64_t report_interval_seconds{5};  // Load report interval
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
    // 数据库线程池配置
    std::uint32_t thread_count{4};        // 工作线程数量
    std::uint32_t queue_max_size{1000};   // 任务队列最大容量
};

struct ActorConfig {
    std::uint32_t count{4};              // Actor 数量（建议 = CPU 核心数）
    std::uint32_t queue_capacity{1024 * 64};  // 每个 Actor 的队列容量
};

struct TimerWheelConfig {
    std::uint32_t tick_duration_ms{50};  // tick 间隔（毫秒）
    std::uint32_t wheel_size{512};        // 时间轮槽数
};

struct DebugConfig {
    bool enabled{false};                 // 调试模式开关
};

struct ServerConfig {
    std::string node_id;           // Node unique identifier
    std::string host;               // External access host
    std::string service_name;       // Service name in etcd
    NetConfig net;
    LogConfig log;
    EtcdConfig etcd;
    GrpcConfig grpc;
    MongoConfig mongodb;
    ActorConfig actor;
    TimerWheelConfig timer_wheel;
    DebugConfig debug;
};

class Config {
public:
    // 单例访问
    static Config& instance() {
        static Config inst;
        return inst;
    }

    // 初始化配置（从文件加载）
    static void init(const std::string& path);

    // 初始化配置（文件不存在则使用默认值）
    static void init_or_default(const std::string& path) noexcept;

    // 从文件加载（返回新实例，用于特殊场景）
    static Config load_from_file(const std::string& path);

    const ServerConfig& server() const noexcept { return server_; }

    // 禁止拷贝，允许移动（用于 init）
    Config(const Config&) = delete;
    Config(Config&&) = default;
    Config& operator=(const Config&) = delete;
    Config& operator=(Config&&) = default;

private:
    Config() = default;
    ServerConfig server_;
};

} // namespace infra::config
