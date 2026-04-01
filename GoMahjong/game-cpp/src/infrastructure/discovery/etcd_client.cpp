#include "infrastructure/discovery/etcd_client.hpp"

#include <etcd/Client.hpp>
#include <etcd/Watcher.hpp>
#include <etcd/KeepAlive.hpp>

#include <atomic>
#include <mutex>
#include <unordered_map>

namespace infra::discovery {

    // Pimpl 实现类
    struct EtcdClient::Impl {
        std::unique_ptr<etcd::Client> client;
        std::atomic<bool> connected{false};

        // KeepAlive 管理
        std::mutex keepalive_mutex;
        std::unordered_map<std::int64_t, std::unique_ptr<etcd::KeepAlive>> keepalives;

        // Watch 管理
        std::mutex watch_mutex;
        std::atomic<std::uint64_t> next_watch_id{1};
        std::unordered_map<WatchId, std::unique_ptr<etcd::Watcher>> watchers;
        std::unordered_map<WatchId, WatchCallback> watch_callbacks;
    };

    EtcdClient::EtcdClient() : impl_(std::make_unique<Impl>()) {
    }

    EtcdClient::~EtcdClient() {
        disconnect();
    }

    bool EtcdClient::connect(const std::string &endpoints) {
        if (impl_->connected) {
            return true;
        }

        try {
            impl_->client = std::make_unique<etcd::Client>(endpoints);
            // 测试连接：尝试获取一个不存在的 key
            auto response = impl_->client->get("/__etcd_client_test__").get();
            impl_->connected = true;
            return true;
        } catch (const std::exception &e) {
            impl_->connected = false;
            return false;
        }
    }

    bool EtcdClient::is_connected() const noexcept {
        return impl_->connected;
    }

    void EtcdClient::disconnect() {
        if (!impl_->connected) {
            return;
        }

        // 停止所有 watch
        cancel_all_watches();

        // 停止所有 keepalive
        {
            std::lock_guard lock(impl_->keepalive_mutex);
            impl_->keepalives.clear();
        }

        impl_->client.reset();
        impl_->connected = false;
    }

    bool EtcdClient::put(std::string_view key, std::string_view value, std::int64_t lease_id) {
        if (!impl_->connected || !impl_->client) {
            return false;
        }

        try {
            auto response = lease_id > 0
                                ? impl_->client->set(std::string(key), std::string(value), lease_id).get()
                                : impl_->client->set(std::string(key), std::string(value)).get();
            return response.is_ok();
        } catch (const std::exception &) {
            return false;
        }
    }

    std::optional<std::string> EtcdClient::get(std::string_view key) {
        if (!impl_->connected || !impl_->client) {
            return std::nullopt;
        }

        try {
            auto response = impl_->client->get(std::string(key)).get();
            if (response.is_ok() && !response.value().key().empty()) {
                return response.value().as_string();
            }
            return std::nullopt;
        } catch (const std::exception &) {
            return std::nullopt;
        }
    }

    bool EtcdClient::del(std::string_view key) {
        if (!impl_->connected || !impl_->client) {
            return false;
        }

        try {
            auto response = impl_->client->rm(std::string(key)).get();
            return response.is_ok();
        } catch (const std::exception &) {
            return false;
        }
    }

    std::vector<std::pair<std::string, std::string>> EtcdClient::get_by_prefix(std::string_view prefix) {
        std::vector<std::pair<std::string, std::string>> result;

        if (!impl_->connected || !impl_->client) {
            return result;
        }

        try {
            auto response = impl_->client->ls(std::string(prefix)).get();
            if (response.is_ok()) {
                for (const auto &kv: response.values()) {
                    result.emplace_back(kv.key(), kv.as_string());
                }
            }
        } catch (const std::exception &) {
            // 返回空列表
        }

        return result;
    }

    std::int64_t EtcdClient::grant_lease(std::int64_t ttl_seconds) {
        if (!impl_->connected || !impl_->client) {
            return 0;
        }

        try {
            auto response = impl_->client->leasegrant(ttl_seconds).get();
            if (response.is_ok()) {
                return response.value().lease();
            }
            return 0;
        } catch (const std::exception &) {
            return 0;
        }
    }

    bool EtcdClient::start_keepalive(std::int64_t lease_id) {
        if (!impl_->connected || !impl_->client || lease_id == 0) {
            return false;
        }

        try {
            std::lock_guard lock(impl_->keepalive_mutex);

            // 已存在则跳过
            if (impl_->keepalives.contains(lease_id)) {
                return true;
            }

            // 创建 KeepAlive，后台线程自动续约
            auto keepalive = std::make_unique<etcd::KeepAlive>(*impl_->client, lease_id);
            impl_->keepalives.emplace(lease_id, std::move(keepalive));
            return true;
        } catch (const std::exception &) {
            return false;
        }
    }

    void EtcdClient::stop_keepalive(std::int64_t lease_id) {
        std::lock_guard lock(impl_->keepalive_mutex);
        impl_->keepalives.erase(lease_id);
    }

    bool EtcdClient::revoke_lease(std::int64_t lease_id) {
        if (!impl_->connected || !impl_->client || lease_id == 0) {
            return false;
        }

        // 先停止 keepalive
        stop_keepalive(lease_id);

        try {
            auto response = impl_->client->revoke(lease_id).get();
            return response.is_ok();
        } catch (const std::exception &) {
            return false;
        }
    }

    EtcdClient::WatchId EtcdClient::watch(std::string_view prefix, WatchCallback callback) {
        if (!impl_->connected || !impl_->client || !callback) {
            return 0;
        }

        std::lock_guard lock(impl_->watch_mutex);

        WatchId watch_id = impl_->next_watch_id++;

        try {
            auto watcher = std::make_unique<etcd::Watcher>(
                    *impl_->client,
                    std::string(prefix),
                    [this, watch_id, callback](etcd::Response response) {
                        if (response.is_ok()) {
                            for (const auto &event: response.events()) {
                                WatchEvent evt;
                                switch (event.event_type()) {
                                    case etcd::Event::EventType::SET:
                                        evt.type = WatchEventType::Put;
                                        break;
                                    case etcd::Event::EventType::DELETE_:
                                        evt.type = WatchEventType::Delete;
                                        break;
                                    default:
                                        continue;
                                }
                                evt.key = event.kv().key();
                                evt.value = event.kv().as_string();
                                callback(evt);
                            }
                        }
                    });

            impl_->watchers.emplace(watch_id, std::move(watcher));
            impl_->watch_callbacks.emplace(watch_id, std::move(callback));
            return watch_id;
        } catch (const std::exception &) {
            return 0;
        }
    }

    void EtcdClient::cancel_watch(WatchId watch_id) {
        std::lock_guard lock(impl_->watch_mutex);
        impl_->watchers.erase(watch_id);
        impl_->watch_callbacks.erase(watch_id);
    }

    void EtcdClient::cancel_all_watches() {
        std::lock_guard lock(impl_->watch_mutex);
        impl_->watchers.clear();
        impl_->watch_callbacks.clear();
    }

} // namespace infra::discovery
