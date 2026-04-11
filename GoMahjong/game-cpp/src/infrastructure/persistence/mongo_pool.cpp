#include "infrastructure/persistence/mongo_pool.hpp"

#include <mongocxx/exception/exception.hpp>
#include <mongocxx/instance.hpp>
#include <sstream>
#include <iomanip>

#include "infrastructure/log/logger.hpp"

namespace infra::persistence {

    // 全局 MongoDB 实例（每个进程只需一个）
    static mongocxx::instance* g_instance = nullptr;
    static std::mutex g_instance_mutex;

    // 确保全局实例已初始化
    static void ensure_instance() {
        std::lock_guard lock(g_instance_mutex);
        if (!g_instance) {
            g_instance = new mongocxx::instance();
        }
    }

    // === 构造函数和析构函数 ===

    MongoPool::MongoPool(const MongoConfig& config)
        : config_(config) {
        // 确保 mongocxx::instance 已初始化（必须在使用任何 mongocxx 对象前调用）
        ensure_instance();

        // 创建 MongoDB 连接池
        try {
            mongocxx::uri uri{config_.uri};
            mongo_pool_ = std::make_unique<mongocxx::pool>(uri);
            LOG_INFO("[MongoPool] 连接池创建成功: {}", config_.uri);
        } catch (const mongocxx::exception& e) {
            LOG_ERROR("[MongoPool] 连接池创建失败: {}", e.what());
            throw;
        }
    }

    MongoPool::~MongoPool() {
        stop();
    }

    // === 生命周期管理 ===

    void MongoPool::start() {
        if (running_) {
            LOG_WARN("[MongoPool] 线程池已在运行");
            return;
        }

        running_ = true;

        // 启动工作线程
        for (size_t i = 0; i < config_.thread_count; ++i) {
            workers_.emplace_back(&MongoPool::worker_thread, this);
        }

        LOG_INFO("[MongoPool] 线程池启动成功，线程数: {}", config_.thread_count);
    }

    void MongoPool::stop() {
        if (!running_) {
            return;
        }

        running_ = false;

        // 通知所有工作线程
        queue_cv_.notify_all();

        // 等待所有工作线程结束
        for (auto& worker : workers_) {
            if (worker.joinable()) {
                worker.join();
            }
        }
        workers_.clear();

        LOG_INFO("[MongoPool] 线程池已停止");
    }

    bool MongoPool::is_running() const noexcept {
        return running_;
    }

    // === 异步操作 ===

    void MongoPool::async_execute(DbTask operation) {
        if (!running_) {
            LOG_ERROR("[MongoPool] 线程池未启动");
            return;
        }

        {
            std::lock_guard lock(queue_mutex_);

            // 检查队列容量
            if (task_queue_.size() >= config_.queue_max_size) {
                LOG_ERROR("[MongoPool] 任务队列已满，丢弃任务");
                return;
            }

            task_queue_.push(std::move(operation));
            pending_count_++;
        }

        queue_cv_.notify_one();
    }

    void MongoPool::async_execute_with_error(
        DbTask operation,
        std::function<void(std::exception_ptr)> error_callback
    ) {
        if (!running_) {
            LOG_ERROR("[MongoPool] 线程池未启动");
            if (error_callback) {
                error_callback(std::make_exception_ptr(std::runtime_error("线程池未启动")));
            }
            return;
        }

        // 包装任务，添加错误处理
        auto wrapped_task = [op = std::move(operation), cb = std::move(error_callback)](mongocxx::client& client) {
            try {
                op(client);
            } catch (...) {
                if (cb) {
                    cb(std::current_exception());
                }
            }
        };

        {
            std::lock_guard lock(queue_mutex_);

            // 检查队列容量
            if (task_queue_.size() >= config_.queue_max_size) {
                LOG_ERROR("[MongoPool] 任务队列已满，丢弃任务");
                if (error_callback) {
                    error_callback(std::make_exception_ptr(std::runtime_error("任务队列已满")));
                }
                return;
            }

            task_queue_.push(std::move(wrapped_task));
            pending_count_++;
        }

        queue_cv_.notify_one();
    }

    // === 同步操作 ===

    void MongoPool::execute(DbTask operation) {
        if (!running_) {
            LOG_ERROR("[MongoPool] 线程池未启动");
            throw std::runtime_error("线程池未启动");
        }

        try {
            // 从连接池获取连接
            auto conn = mongo_pool_->acquire();
            
            // 执行操作
            operation(*conn);
        } catch (const mongocxx::exception& e) {
            LOG_ERROR("[MongoPool] 数据库操作失败: {}", e.what());
            throw;
        } catch (const std::exception& e) {
            LOG_ERROR("[MongoPool] 操作失败: {}", e.what());
            throw;
        }
    }

    // === 统计信息 ===

    size_t MongoPool::pending_tasks() const {
        return pending_count_.load();
    }

    size_t MongoPool::active_connections() const {
        // mongocxx::pool 没有提供获取活跃连接数的接口
        // 返回线程数作为近似值
        return config_.thread_count;
    }

    // === 私有方法 ===

    // 辅助函数：将 thread::id 转换为字符串
    static std::string thread_id_to_string(std::thread::id id) {
        std::ostringstream oss;
        oss << id;
        return oss.str();
    }

    void MongoPool::worker_thread() {
        LOG_DEBUG("Mongodb 工作线程启动: {}", thread_id_to_string(std::this_thread::get_id()));

        while (running_) {
            DbTask task;

            // 等待任务
            {
                std::unique_lock lock(queue_mutex_);
                queue_cv_.wait(lock, [this] {
                    return !task_queue_.empty() || !running_;
                });

                // 检查是否需要退出
                if (!running_ && task_queue_.empty()) {
                    break;
                }

                // 取出任务
                if (!task_queue_.empty()) {
                    task = std::move(task_queue_.front());
                    task_queue_.pop();
                    pending_count_--;
                }
            }

            // 执行任务
            if (task) {
                try {
                    // 从连接池获取连接
                    auto conn = mongo_pool_->acquire();

                    // 执行数据库操作
                    task(*conn);

                } catch (const mongocxx::exception& e) {
                    LOG_ERROR("[MongoPool] 数据库操作失败: {}", e.what());
                } catch (const std::exception& e) {
                    LOG_ERROR("[MongoPool] 任务执行失败: {}", e.what());
                } catch (...) {
                    LOG_ERROR("[MongoPool] 未知异常");
                }
            }
        }

        LOG_DEBUG("[MongoPool] 工作线程退出: {}", thread_id_to_string(std::this_thread::get_id()));
    }

} // namespace infra::persistence
