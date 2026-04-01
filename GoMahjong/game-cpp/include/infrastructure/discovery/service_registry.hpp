#pragma once

#include <cstdint>
#include <map>
#include <memory>
#include <string>
#include <string_view>
#include <vector>

namespace infra::discovery {
    class EtcdClient;
}

namespace infra::discovery {

    // 服务端点信息
    struct ServiceEndpoint {
        std::string node_id;                              // 节点唯一标识
        std::string host;                                 // 主机地址
        std::uint16_t port{};                             // 端口
        std::map<std::string, std::string> metadata;      // 元数据（如 game_count, player_count）

        // 序列化为 JSON 字符串
        [[nodiscard]] std::string to_json() const;

        // 从 JSON 字符串解析
        static std::optional<ServiceEndpoint> from_json(const std::string &json);
    };

    // 服务注册配置
    struct RegistryConfig {
        std::string endpoints;    // etcd 地址，如 "http://127.0.0.1:2379"
        std::int64_t ttl_seconds; // 注册 TTL（心跳间隔）
    };

    // 服务注册器
    // 提供服务注册、发现、变更监听
    class ServiceRegistry {
    public:
        explicit ServiceRegistry(const RegistryConfig &config);

        ~ServiceRegistry();

        ServiceRegistry(const ServiceRegistry &) = delete;

        ServiceRegistry &operator=(const ServiceRegistry &) = delete;

        // 连接 etcd
        // 返回是否连接成功
        bool connect();

        bool is_connected() const noexcept;

        // 注册服务
        // service_name: 服务名，如 "game-service"
        // endpoint: 服务端点信息
        // 返回是否注册成功
        bool register_service(std::string_view service_name, const ServiceEndpoint &endpoint);

        // 注销服务
        bool deregister_service(std::string_view service_name, std::string_view node_id);

        // 发现服务（返回所有实例）
        std::vector<ServiceEndpoint> discover(std::string_view service_name);

        // 更新元数据（如负载信息）
        // 会更新 key 的 value（需要重新注册）
        bool update_metadata(
                std::string_view service_name,
                std::string_view node_id,
                const std::map<std::string, std::string> &metadata);

        // 监听服务变更
        // service_name: 服务名
        // callback: 变更回调，返回最新的服务列表
        using ChangeCallback = std::function<void(std::string_view service_name, const std::vector<ServiceEndpoint> &endpoints)>;
        void watch_service(std::string_view service_name, ChangeCallback callback);

        // 取消监听
        void cancel_watch(std::string_view service_name);

    private:
        // 构建 etcd key
        [[nodiscard]] static std::string build_key(std::string_view service_name, std::string_view node_id);

        // 构建前缀
        [[nodiscard]] static std::string build_prefix(std::string_view service_name);

        // 处理 watch 事件
        void handle_watch_event(std::string_view service_name, const WatchEvent &event);

    private:
        struct Impl;
        std::unique_ptr<Impl> impl_;
    };

} // namespace infra::discovery
