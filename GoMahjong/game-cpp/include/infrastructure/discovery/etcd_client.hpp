#pragma once

#include <cstdint>
#include <functional>
#include <map>
#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

namespace etcd {
    class Client;
    class Watcher;
    class KeepAlive;
}

namespace infra::discovery {

    enum class WatchEventType {
        Put,    // key 创建或更新
        Delete, // key 删除
    };

    struct WatchEvent {
        WatchEventType type;
        std::string key;
        std::string value;
    };

    // etcd 客户端封装
    // 提供同步 KV 操作、Lease 管理、Watch 监听
    class EtcdClient {
    public:
        using WatchCallback = std::function<void(const WatchEvent &event)>;
        using WatchId = std::uint64_t;

        EtcdClient();

        ~EtcdClient();

        EtcdClient(const EtcdClient &) = delete;

        EtcdClient &operator=(const EtcdClient &) = delete;

        // 连接管理
        // endpoints: etcd 地址列表，如 "http://127.0.0.1:2379"
        // 返回是否连接成功
        bool connect(const std::string &endpoints);

        bool is_connected() const noexcept;

        void disconnect();

        // KV 操作（同步）
        // lease_id: 可选，绑定 lease 实现自动过期
        bool put(std::string_view key, std::string_view value, std::int64_t lease_id = 0);

        std::optional<std::string> get(std::string_view key);

        bool del(std::string_view key);

        // 范围查询（前缀匹配）
        // prefix: key 前缀
        // 返回 key-value 列表
        std::vector<std::pair<std::string, std::string>> get_by_prefix(std::string_view prefix);

        // Lease 管理
        // ttl_seconds: 租约 TTL
        // 返回 lease_id，失败返回 0
        std::int64_t grant_lease(std::int64_t ttl_seconds);

        // 启动后台心跳续约
        // 成功返回 true，后台线程会自动续约直到 revoke 或析构
        bool start_keepalive(std::int64_t lease_id);

        // 停止指定 lease 的续约
        void stop_keepalive(std::int64_t lease_id);

        // 撤销租约（立即删除关联的 key）
        bool revoke_lease(std::int64_t lease_id);

        // Watch 监听（异步）
        // prefix: 监听的 key 前缀
        // callback: 变更回调
        // 返回 watch_id，用于取消监听
        WatchId watch(std::string_view prefix, WatchCallback callback);

        // 取消监听
        void cancel_watch(WatchId watch_id);

        // 取消所有监听
        void cancel_all_watches();

    private:
        struct Impl;
        std::unique_ptr<Impl> impl_;
    };

} // namespace infra::discovery
