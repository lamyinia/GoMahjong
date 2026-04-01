#pragma once

#include <cstdint>
#include <memory>
#include <string>

// 前向声明，避免在头文件中包含 mongo-cxx-driver 头文件
namespace mongocxx {
    class client;
    class database;
    class collection;
}

namespace infra::persistence {

    // MongoDB 配置
    struct MongoConfig {
        std::string uri{"mongodb://localhost:27017"};
        std::string database{"gomahjong"};
        std::uint32_t min_pool_size{10};
        std::uint32_t max_pool_size{100};
        std::string username;
        std::string password;
    };

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
