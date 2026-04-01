#include "infrastructure/discovery/service_registry.hpp"
#include "infrastructure/discovery/etcd_client.hpp"

#include <nlohmann/json.hpp>

#include <mutex>
#include <unordered_map>

namespace infra::discovery {

    // ServiceEndpoint 实现
    std::string ServiceEndpoint::to_json() const {
        nlohmann::json j;
        j["node_id"] = node_id;
        j["host"] = host;
        j["port"] = port;
        j["metadata"] = metadata;
        return j.dump();
    }

    std::optional<ServiceEndpoint> ServiceEndpoint::from_json(const std::string &json) {
        try {
            auto j = nlohmann::json::parse(json);
            ServiceEndpoint endpoint;
            endpoint.node_id = j.at("node_id").get<std::string>();
            endpoint.host = j.at("host").get<std::string>();
            endpoint.port = j.at("port").get<std::uint16_t>();
            if (j.contains("metadata")) {
                endpoint.metadata = j.at("metadata").get<std::map<std::string, std::string>>();
            }
            return endpoint;
        } catch (const std::exception &) {
            return std::nullopt;
        }
    }

    // ServiceRegistry::Impl
    struct ServiceRegistry::Impl {
        RegistryConfig config;
        std::unique_ptr<EtcdClient> client;

        // 当前注册的服务（用于 deregister）
        std::mutex registered_mutex;
        std::unordered_map<std::string, std::int64_t> registered_leases; // key -> lease_id

        // Watch 管理
        std::mutex watch_mutex;
        std::unordered_map<std::string, EtcdClient::WatchId> watch_ids; // service_name -> watch_id
        std::unordered_map<std::string, ChangeCallback> watch_callbacks; // service_name -> callback
    };

    ServiceRegistry::ServiceRegistry(const RegistryConfig &config)
        : impl_(std::make_unique<Impl>()) {
        impl_->config = config;
        impl_->client = std::make_unique<EtcdClient>();
    }

    ServiceRegistry::~ServiceRegistry() {
        // 取消所有 watch
        {
            std::lock_guard lock(impl_->watch_mutex);
            for (const auto &[_, watch_id]: impl_->watch_ids) {
                impl_->client->cancel_watch(watch_id);
            }
        }

        // 注销所有服务
        {
            std::lock_guard lock(impl_->registered_mutex);
            for (const auto &[key, lease_id]: impl_->registered_leases) {
                impl_->client->revoke_lease(lease_id);
            }
        }

        impl_->client->disconnect();
    }

    bool ServiceRegistry::connect() {
        return impl_->client->connect(impl_->config.endpoints);
    }

    bool ServiceRegistry::is_connected() const noexcept {
        return impl_->client->is_connected();
    }

    bool ServiceRegistry::register_service(std::string_view service_name, const ServiceEndpoint &endpoint) {
        if (!impl_->client->is_connected()) {
            return false;
        }

        // 创建 lease
        auto lease_id = impl_->client->grant_lease(impl_->config.ttl_seconds);
        if (lease_id == 0) {
            return false;
        }

        // 构建 key
        auto key = build_key(service_name, endpoint.node_id);
        auto value = endpoint.to_json();

        // 写入 KV
        if (!impl_->client->put(key, value, lease_id)) {
            impl_->client->revoke_lease(lease_id);
            return false;
        }

        // 启动心跳续约
        if (!impl_->client->start_keepalive(lease_id)) {
            impl_->client->revoke_lease(lease_id);
            return false;
        }

        // 记录已注册
        {
            std::lock_guard lock(impl_->registered_mutex);
            impl_->registered_leases[key] = lease_id;
        }

        return true;
    }

    bool ServiceRegistry::deregister_service(std::string_view service_name, std::string_view node_id) {
        if (!impl_->client->is_connected()) {
            return false;
        }

        auto key = build_key(service_name, node_id);

        // 获取 lease_id
        std::int64_t lease_id = 0;
        {
            std::lock_guard lock(impl_->registered_mutex);
            if (auto it = impl_->registered_leases.find(key); it != impl_->registered_leases.end()) {
                lease_id = it->second;
                impl_->registered_leases.erase(it);
            }
        }

        // 撤销 lease（会自动删除 key）
        if (lease_id > 0) {
            return impl_->client->revoke_lease(lease_id);
        }

        // 如果没有 lease_id，直接删除 key
        return impl_->client->del(key);
    }

    std::vector<ServiceEndpoint> ServiceRegistry::discover(std::string_view service_name) {
        std::vector<ServiceEndpoint> endpoints;

        if (!impl_->client->is_connected()) {
            return endpoints;
        }

        auto prefix = build_prefix(service_name);
        auto kvs = impl_->client->get_by_prefix(prefix);

        for (const auto &[key, value]: kvs) {
            auto endpoint = ServiceEndpoint::from_json(value);
            if (endpoint) {
                endpoints.push_back(std::move(*endpoint));
            }
        }

        return endpoints;
    }

    bool ServiceRegistry::update_metadata(
            std::string_view service_name,
            std::string_view node_id,
            const std::map<std::string, std::string> &metadata) {
        if (!impl_->client->is_connected()) {
            return false;
        }

        auto key = build_key(service_name, node_id);

        // 获取当前值
        auto current_value = impl_->client->get(key);
        if (!current_value) {
            return false;
        }

        // 解析并更新 metadata
        auto endpoint = ServiceEndpoint::from_json(*current_value);
        if (!endpoint) {
            return false;
        }

        endpoint->metadata = metadata;

        // 获取 lease_id（如果存在）
        std::int64_t lease_id = 0;
        {
            std::lock_guard lock(impl_->registered_mutex);
            if (auto it = impl_->registered_leases.find(key); it != impl_->registered_leases.end()) {
                lease_id = it->second;
            }
        }

        // 更新 KV
        return impl_->client->put(key, endpoint->to_json(), lease_id);
    }

    void ServiceRegistry::watch_service(std::string_view service_name, ChangeCallback callback) {
        if (!impl_->client->is_connected() || !callback) {
            return;
        }

        std::lock_guard lock(impl_->watch_mutex);

        // 已存在则先取消
        if (auto it = impl_->watch_ids.find(std::string(service_name)); it != impl_->watch_ids.end()) {
            impl_->client->cancel_watch(it->second);
        }

        auto prefix = build_prefix(service_name);

        // 启动 watch
        auto watch_id = impl_->client->watch(
                prefix,
                [this, service_name = std::string(service_name)](const WatchEvent &event) {
                    handle_watch_event(service_name, event);
                });

        if (watch_id > 0) {
            impl_->watch_ids[std::string(service_name)] = watch_id;
            impl_->watch_callbacks[std::string(service_name)] = std::move(callback);
        }
    }

    void ServiceRegistry::cancel_watch(std::string_view service_name) {
        std::lock_guard lock(impl_->watch_mutex);

        if (auto it = impl_->watch_ids.find(std::string(service_name)); it != impl_->watch_ids.end()) {
            impl_->client->cancel_watch(it->second);
            impl_->watch_ids.erase(it);
            impl_->watch_callbacks.erase(std::string(service_name));
        }
    }

    std::string ServiceRegistry::build_key(std::string_view service_name, std::string_view node_id) {
        return std::string("/service/") + std::string(service_name) + "/" + std::string(node_id);
    }

    std::string ServiceRegistry::build_prefix(std::string_view service_name) {
        return std::string("/service/") + std::string(service_name) + "/";
    }

    void ServiceRegistry::handle_watch_event(std::string_view service_name, const WatchEvent &event) {
        ChangeCallback callback;
        {
            std::lock_guard lock(impl_->watch_mutex);
            if (auto it = impl_->watch_callbacks.find(std::string(service_name));
                it != impl_->watch_callbacks.end()) {
                callback = it->second;
            }
        }

        if (callback) {
            // 获取最新服务列表并回调
            auto endpoints = discover(service_name);
            callback(service_name, endpoints);
        }
    }

} // namespace infra::discovery
