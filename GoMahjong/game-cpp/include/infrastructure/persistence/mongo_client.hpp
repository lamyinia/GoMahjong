#pragma once

#include <cstdint>
#include <memory>
#include <string>

// mongo-cxx-driver types are returned by value in this header; include the full definitions.
#include <mongocxx/client.hpp>
#include <mongocxx/collection.hpp>
#include <mongocxx/database.hpp>

// 使用权威配置
#include "infrastructure/config/config.hpp"

namespace infra::persistence {

    // 使用 config 命名空间中的权威 MongoConfig
    using MongoConfig = infra::config::MongoConfig;

    // MongoDB 客户端封装
    // 提供连接管理、数据库和集合访问
    class MongoClient {
    public:
        explicit MongoClient(const MongoConfig &config);

        ~MongoClient();

        MongoClient(const MongoClient &) = delete;

        MongoClient &operator=(const MongoClient &) = delete;

        // 连接/断开
        bool connect();

        void disconnect();

        [[nodiscard]] bool is_connected() const noexcept;

        // 获取数据库
        [[nodiscard]] mongocxx::database database();

        // 获取集合
        [[nodiscard]] mongocxx::collection collection(const std::string &name);

        // 获取原始客户端（高级用法）
        [[nodiscard]] mongocxx::client &client();

    private:
        struct Impl;
        std::unique_ptr<Impl> impl_;
    };

} // namespace infra::persistence
