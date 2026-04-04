#pragma once

#include <mongocxx/client.hpp>
#include <mongocxx/pool.hpp>
#include <functional>
#include <queue>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <atomic>
#include <memory>
#include <vector>
#include <exception>

#include "infrastructure/persistence/mongo_client.hpp"

namespace infra::persistence {

    // 数据库任务类型
    using DbTask = std::function<void(mongocxx::client&)>;

    // MongoDB 线程池
    // 职责：
    // 1. 管理 MongoDB 连接池（mongocxx::pool）
    // 2. 管理工作线程池
    // 3. 异步执行数据库任务
    // 4. 不阻塞游戏逻辑线程
    class MongoPool {
    public:
        explicit MongoPool(const MongoConfig& config);
        
        ~MongoPool();
        
        // 禁止拷贝
        MongoPool(const MongoPool&) = delete;
        MongoPool& operator=(const MongoPool&) = delete;
        
        // === 生命周期管理 ===
        
        // 启动线程池
        void start();
        
        // 停止线程池（优雅关闭）
        void stop();
        
        // 是否运行中
        [[nodiscard]] bool is_running() const noexcept;
        
        // === 异步操作 ===
        
        // 异步执行数据库操作（无返回值）
        // operation: 数据库操作函数
        void async_execute(DbTask operation);
        
        // 异步执行数据库操作（带错误回调）
        // operation: 数据库操作函数
        // error_callback: 错误回调
        void async_execute_with_error(
            DbTask operation,
            std::function<void(std::exception_ptr)> error_callback
        );
        
        // === 同步操作（阻塞，慎用）===
        
        // 同步执行数据库操作
        // 注意：会阻塞当前线程，仅在特殊场景使用
        void execute(DbTask operation);
        
        // === 统计信息 ===
        
        // 待处理任务数
        [[nodiscard]] size_t pending_tasks() const;
        
        // 活跃连接数（近似值）
        [[nodiscard]] size_t active_connections() const;
        
    private:
        // 工作线程函数
        void worker_thread();
        
        // 从队列中取出任务
        bool pop_task(DbTask& task);
        
    private:
        MongoConfig config_;
        
        // MongoDB 连接池（官方实现）
        std::unique_ptr<mongocxx::pool> mongo_pool_;
        
        // 任务队列
        std::queue<DbTask> task_queue_;
        mutable std::mutex queue_mutex_;
        std::condition_variable queue_cv_;
        
        // 工作线程
        std::vector<std::thread> workers_;
        std::atomic<bool> running_{false};
        
        // 统计信息
        std::atomic<size_t> pending_count_{0};
    };

} // namespace infra::persistence
