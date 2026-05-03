#include "infrastructure/discovery/etcd_client.hpp"

#include <grpcpp/grpcpp.h>
#include <etcdserverpb/rpc.grpc.pb.h>
#include <mvccpb/kv.pb.h>

#include "infrastructure/log/logger.hpp"

#include <atomic>
#include <mutex>
#include <thread>
#include <unordered_map>

namespace infra::discovery {

    using KVStub = etcdserverpb::KV::Stub;
    using LeaseStub = etcdserverpb::Lease::Stub;
    using WatchStub = etcdserverpb::Watch::Stub;

    // Pimpl 实现类
    struct EtcdClient::Impl {
        std::shared_ptr<grpc::Channel> channel;
        std::unique_ptr<KVStub> kv_stub;
        std::unique_ptr<LeaseStub> lease_stub;
        std::unique_ptr<WatchStub> watch_stub;
        std::atomic<bool> connected{false};

        // KeepAlive 管理
        std::mutex keepalive_mutex;
        std::unordered_map<std::int64_t, std::unique_ptr<std::thread>> keepalive_threads;
        std::unordered_map<std::int64_t, std::atomic<bool>> keepalive_running;

        // Watch 管理
        std::mutex watch_mutex;
        std::atomic<std::uint64_t> next_watch_id{1};
        std::unordered_map<WatchId, std::unique_ptr<std::thread>> watch_threads;
        std::unordered_map<WatchId, std::atomic<bool>> watch_running;
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
            // Parse endpoint: "http://host:port" -> "host:port"
            std::string target = endpoints;
            if (target.find("http://") == 0) {
                target = target.substr(7);
            } else if (target.find("https://") == 0) {
                target = target.substr(8);
            }

            grpc::ChannelArguments args;
            args.SetMaxReceiveMessageSize(64 * 1024 * 1024);
            impl_->channel = grpc::CreateCustomChannel(target, grpc::InsecureChannelCredentials(), args);

            // Test connectivity
            auto deadline = std::chrono::system_clock::now() + std::chrono::seconds(5);
            bool connected = impl_->channel->WaitForConnected(deadline);
            if (!connected) {
                LOG_WARN("[EtcdClient] failed to connect to: {}", target);
                return false;
            }

            impl_->kv_stub = etcdserverpb::KV::NewStub(impl_->channel);
            impl_->lease_stub = etcdserverpb::Lease::NewStub(impl_->channel);
            impl_->watch_stub = etcdserverpb::Watch::NewStub(impl_->channel);

            // Verify with a Get request for a non-existent key
            etcdserverpb::RangeRequest req;
            req.set_key("/__etcd_client_test__");
            req.set_limit(1);
            etcdserverpb::RangeResponse resp;
            grpc::ClientContext ctx;
            ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(3));
            auto status = impl_->kv_stub->Range(&ctx, req, &resp);
            impl_->connected = status.ok() || status.error_code() == grpc::StatusCode::NOT_FOUND;
            if (!impl_->connected) {
                LOG_WARN("[EtcdClient] Range verify failed: error_code={} error_msg={}", static_cast<int>(status.error_code()), status.error_message());
            }
            return impl_->connected.load();
        } catch (const std::exception &e) {
            LOG_WARN("[EtcdClient] connect exception: {}", e.what());
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

        cancel_all_watches();

        // Stop all keepalive threads
        {
            std::lock_guard lock(impl_->keepalive_mutex);
            for (auto &[lease_id, running]: impl_->keepalive_running) {
                running.store(false);
            }
            for (auto &[lease_id, thread]: impl_->keepalive_threads) {
                if (thread && thread->joinable()) {
                    thread->join();
                }
            }
            impl_->keepalive_threads.clear();
            impl_->keepalive_running.clear();
        }

        impl_->kv_stub.reset();
        impl_->lease_stub.reset();
        impl_->watch_stub.reset();
        impl_->channel.reset();
        impl_->connected = false;
    }

    bool EtcdClient::put(std::string_view key, std::string_view value, std::int64_t lease_id) {
        if (!impl_->connected || !impl_->kv_stub) {
            return false;
        }

        etcdserverpb::PutRequest req;
        req.set_key(std::string(key));
        req.set_value(std::string(value));
        if (lease_id > 0) {
            req.set_lease(lease_id);
        }

        etcdserverpb::PutResponse resp;
        grpc::ClientContext ctx;
        ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));

        auto status = impl_->kv_stub->Put(&ctx, req, &resp);
        if (!status.ok()) {
            LOG_WARN("[EtcdClient] put failed: {}", status.error_message());
        }
        return status.ok();
    }

    std::optional<std::string> EtcdClient::get(std::string_view key) {
        if (!impl_->connected || !impl_->kv_stub) {
            return std::nullopt;
        }

        etcdserverpb::RangeRequest req;
        req.set_key(std::string(key));

        etcdserverpb::RangeResponse resp;
        grpc::ClientContext ctx;
        ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));

        auto status = impl_->kv_stub->Range(&ctx, req, &resp);
        if (!status.ok()) {
            LOG_WARN("[EtcdClient] get failed: {}", status.error_message());
            return std::nullopt;
        }
        if (resp.kvs_size() > 0) {
            return resp.kvs(0).value();
        }
        return std::nullopt;
    }

    bool EtcdClient::del(std::string_view key) {
        if (!impl_->connected || !impl_->kv_stub) {
            return false;
        }

        etcdserverpb::DeleteRangeRequest req;
        req.set_key(std::string(key));

        etcdserverpb::DeleteRangeResponse resp;
        grpc::ClientContext ctx;
        ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));

        auto status = impl_->kv_stub->DeleteRange(&ctx, req, &resp);
        return status.ok();
    }

    std::vector<std::pair<std::string, std::string>> EtcdClient::get_by_prefix(std::string_view prefix) {
        std::vector<std::pair<std::string, std::string>> result;

        if (!impl_->connected || !impl_->kv_stub) {
            return result;
        }

        etcdserverpb::RangeRequest req;
        req.set_key(std::string(prefix));
        // range_end: prefix + 1 (next key after prefix) for prefix scan
        std::string range_end(prefix);
        if (!range_end.empty()) {
            range_end.back() += 1;
        }
        req.set_range_end(range_end);

        etcdserverpb::RangeResponse resp;
        grpc::ClientContext ctx;
        ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));

        auto status = impl_->kv_stub->Range(&ctx, req, &resp);
        if (status.ok()) {
            for (const auto &kv: resp.kvs()) {
                result.emplace_back(kv.key(), kv.value());
            }
        }

        return result;
    }

    std::int64_t EtcdClient::grant_lease(std::int64_t ttl_seconds) {
        if (!impl_->connected || !impl_->lease_stub) {
            return 0;
        }

        etcdserverpb::LeaseGrantRequest req;
        req.set_ttl(ttl_seconds);
        // id = 0 lets etcd auto-assign

        etcdserverpb::LeaseGrantResponse resp;
        grpc::ClientContext ctx;
        ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));

        auto status = impl_->lease_stub->LeaseGrant(&ctx, req, &resp);
        if (status.ok() && resp.id() != 0) {
            return resp.id();
        }
        return 0;
    }

    bool EtcdClient::start_keepalive(std::int64_t ttl_seconds, std::int64_t lease_id) {
        if (!impl_->connected || !impl_->lease_stub || lease_id == 0) {
            return false;
        }

        std::lock_guard lock(impl_->keepalive_mutex);

        if (impl_->keepalive_threads.contains(lease_id)) {
            return true;
        }

        // Create running flag
        impl_->keepalive_running.emplace(lease_id, false);
        auto &running = impl_->keepalive_running.at(lease_id);
        running.store(true);

        // Capture raw pointer for the thread (stub lifetime > thread lifetime)
        auto lease_stub_ptr = impl_->lease_stub.get();

        impl_->keepalive_threads.emplace(lease_id, std::make_unique<std::thread>(
                [lease_stub_ptr, lease_id, ttl_seconds, &running]() {
                    // Open a bidirectional stream for keepalive
                    grpc::ClientContext ctx;
                    auto stream = lease_stub_ptr->LeaseKeepAlive(&ctx);

                    while (running.load()) {
                        etcdserverpb::LeaseKeepAliveRequest ka_req;
                        ka_req.set_id(lease_id);

                        if (!stream->Write(ka_req)) {
                            break;
                        }

                        etcdserverpb::LeaseKeepAliveResponse resp;
                        if (!stream->Read(&resp)) {
                            break;
                        }

                        // Sleep for 1/3 of TTL
                        auto sleep_duration = std::chrono::seconds(std::max(1L, ttl_seconds / 3));
                        for (int i = 0; i < sleep_duration.count() * 10 && running.load(); ++i) {
                            std::this_thread::sleep_for(std::chrono::milliseconds(100));
                        }
                    }

                    stream->WritesDone();
                    stream->Finish();
                }
        ));

        return true;
    }

    void EtcdClient::stop_keepalive(std::int64_t lease_id) {
        std::lock_guard lock(impl_->keepalive_mutex);

        if (auto it = impl_->keepalive_running.find(lease_id); it != impl_->keepalive_running.end()) {
            it->second.store(false);
        }
        if (auto it = impl_->keepalive_threads.find(lease_id); it != impl_->keepalive_threads.end()) {
            if (it->second && it->second->joinable()) {
                it->second->join();
            }
            impl_->keepalive_threads.erase(it);
        }
        impl_->keepalive_running.erase(lease_id);
    }

    bool EtcdClient::revoke_lease(std::int64_t lease_id) {
        if (!impl_->connected || !impl_->lease_stub || lease_id == 0) {
            return false;
        }

        stop_keepalive(lease_id);

        etcdserverpb::LeaseRevokeRequest req;
        req.set_id(lease_id);

        etcdserverpb::LeaseRevokeResponse resp;
        grpc::ClientContext ctx;
        ctx.set_deadline(std::chrono::system_clock::now() + std::chrono::seconds(5));

        auto status = impl_->lease_stub->LeaseRevoke(&ctx, req, &resp);
        return status.ok();
    }

    EtcdClient::WatchId EtcdClient::watch(std::string_view prefix, WatchCallback callback) {
        if (!impl_->connected || !impl_->watch_stub || !callback) {
            return 0;
        }

        std::lock_guard lock(impl_->watch_mutex);

        WatchId watch_id = impl_->next_watch_id++;

        impl_->watch_running.emplace(watch_id, true);
        auto &running = impl_->watch_running.at(watch_id);

        impl_->watch_callbacks.emplace(watch_id, std::move(callback));

        auto watch_stub_ptr = impl_->watch_stub.get();
        auto &watch_callbacks = impl_->watch_callbacks;

        std::string prefix_str(prefix);
        // range_end: prefix + 1
        std::string range_end(prefix);
        if (!range_end.empty()) {
            range_end.back() += 1;
        }

        impl_->watch_threads.emplace(watch_id, std::make_unique<std::thread>(
                [watch_stub_ptr, watch_id, prefix_str, range_end, &running, &watch_callbacks]() {
                    grpc::ClientContext ctx;
                    auto stream = watch_stub_ptr->Watch(&ctx);

                    // Send create watch request
                    etcdserverpb::WatchRequest req;
                    auto *create = req.mutable_create_request();
                    create->set_key(prefix_str);
                    create->set_range_end(range_end);

                    if (!stream->Write(req)) {
                        running.store(false);
                        stream->WritesDone();
                        stream->Finish();
                        return;
                    }

                    // Read watch events
                    etcdserverpb::WatchResponse resp;
                    while (running.load() && stream->Read(&resp)) {
                        for (const auto &event: resp.events()) {
                            WatchEvent evt;
                            switch (event.type()) {
                                case mvccpb::Event::PUT:
                                    evt.type = WatchEventType::Put;
                                    break;
                                case mvccpb::Event::DELETE:
                                    evt.type = WatchEventType::Delete;
                                    break;
                                default:
                                    continue;
                            }
                            if (event.has_kv()) {
                                evt.key = event.kv().key();
                                evt.value = event.kv().value();
                            }

                            // Look up callback
                            if (auto it = watch_callbacks.find(watch_id); it != watch_callbacks.end()) {
                                it->second(evt);
                            }
                        }
                    }

                    stream->WritesDone();
                    stream->Finish();
                }
        ));

        return watch_id;
    }

    void EtcdClient::cancel_watch(WatchId watch_id) {
        std::lock_guard lock(impl_->watch_mutex);

        if (auto it = impl_->watch_running.find(watch_id); it != impl_->watch_running.end()) {
            it->second.store(false);
        }
        if (auto it = impl_->watch_threads.find(watch_id); it != impl_->watch_threads.end()) {
            if (it->second && it->second->joinable()) {
                it->second->join();
            }
            impl_->watch_threads.erase(it);
        }
        impl_->watch_running.erase(watch_id);
        impl_->watch_callbacks.erase(watch_id);
    }

    void EtcdClient::cancel_all_watches() {
        std::lock_guard lock(impl_->watch_mutex);

        for (auto &[id, running]: impl_->watch_running) {
            running.store(false);
        }
        for (auto &[id, thread]: impl_->watch_threads) {
            if (thread && thread->joinable()) {
                thread->join();
            }
        }
        impl_->watch_threads.clear();
        impl_->watch_running.clear();
        impl_->watch_callbacks.clear();
    }

} // namespace infra::discovery
