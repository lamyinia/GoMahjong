#include "infrastructure/persistence/mongo_client.hpp"

#include <mongocxx/client.hpp>
#include <mongocxx/instance.hpp>
#include <mongocxx/uri.hpp>
#include <mongocxx/options/client.hpp>
#include <mongocxx/exception/exception.hpp>

#include "infrastructure/log/logger.hpp"

namespace infra::persistence {

    // 全局 MongoDB 实例（每个进程只需一个）
    static mongocxx::instance *g_instance = nullptr;
    static std::mutex g_instance_mutex;

    // 确保全局实例已初始化
    static void ensure_instance() {
        std::lock_guard lock(g_instance_mutex);
        if (!g_instance) {
            g_instance = new mongocxx::instance();
        }
    }

    // MongoClient::Impl
    struct MongoClient::Impl {
        MongoConfig config;
        std::unique_ptr<mongocxx::client> client;
        bool connected{false};

        explicit Impl(const MongoConfig &cfg) : config(cfg) {
        }
    };

    MongoClient::MongoClient(const MongoConfig &config)
        : impl_(std::make_unique<Impl>(config)) {
        ensure_instance();
    }

    MongoClient::~MongoClient() {
        disconnect();
    }

    bool MongoClient::connect() {
        if (impl_->connected) {
            return true;
        }

        try {
            // 构建 URI
            mongocxx::uri uri{impl_->config.uri};

            // 配置连接池选项
            mongocxx::options::client client_opts;
            auto pool_opts = mongocxx::options::pool{};
            pool_opts.min_size(impl_->config.min_pool_size);
            pool_opts.max_size(impl_->config.max_pool_size);
            client_opts.pool_opts(pool_opts);

            // 如果有认证信息
            if (!impl_->config.username.empty() && !impl_->config.password.empty()) {
                // URI 中已包含认证信息时不需要额外设置
                // 如果 URI 不包含认证，可以在这里构建带认证的 URI
            }

            // 创建客户端
            impl_->client = std::make_unique<mongocxx::client>(uri, client_opts);

            // 测试连接
            auto db = impl_->client->database(impl_->config.database);
            db.run_command(bsoncxx::builder::basic::make_document(
                    bsoncxx::builder::basic::kvp("ping", 1)));

            impl_->connected = true;
            LOG_INFO("[MongoClient] 连接成功: {}, 数据库: {}", impl_->config.uri, impl_->config.database);
            return true;
        } catch (const mongocxx::exception &e) {
            LOG_ERROR("[MongoClient] 连接失败: {}", e.what());
            impl_->connected = false;
            return false;
        }
    }

    void MongoClient::disconnect() {
        if (!impl_->connected) {
            return;
        }

        impl_->client.reset();
        impl_->connected = false;
        LOG_INFO("[MongoClient] 已断开连接");
    }

    bool MongoClient::is_connected() const noexcept {
        return impl_->connected;
    }

    mongocxx::database MongoClient::database() {
        if (!impl_->connected || !impl_->client) {
            throw std::runtime_error("[MongoClient] 未连接");
        }
        return impl_->client->database(impl_->config.database);
    }

    mongocxx::collection MongoClient::collection(const std::string &name) {
        return database()[name];
    }

    mongocxx::client &MongoClient::client() {
        if (!impl_->connected || !impl_->client) {
            throw std::runtime_error("[MongoClient] 未连接");
        }
        return *impl_->client;
    }

} // namespace infra::persistence
