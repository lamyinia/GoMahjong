#pragma once

#include <vector>
#include <queue>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <functional>
#include <future>
#include <atomic>

namespace infra::util {

/**
 * 线程池
 * 用于异步任务处理，提高并发性能
 * 
 * 适用场景：
 * - 异步数据库操作
 * - 异步网络请求
 * - 并行计算任务
 */
class ThreadPool {
public:
    /**
     * 构造线程池
     * @param threadCount 线程数量（默认为 CPU 核心数）
     */
    explicit ThreadPool(std::size_t threadCount = 0);

    /**
     * 析构，等待所有任务完成
     */
    ~ThreadPool();

    /**
     * 提交任务
     * @tparam F 任务函数类型
     * @tparam Args 参数类型
     * @param f 任务函数
     * @param args 参数
     * @return future 对象，用于获取结果
     */
    template <typename F, typename... Args>
    auto submit(F&& f, Args&&... args) -> std::future<typename std::invoke_result<F, Args...>::type>;

    /**
     * 提交任务（无返回值）
     * @param task 任务函数
     */
    void submitVoid(std::function<void()> task);

    /**
     * 等待所有任务完成
     */
    void waitAll();

    /**
     * 获取线程数量
     */
    std::size_t threadCount() const { return threads_.size(); }

    /**
     * 获取待处理任务数量
     */
    std::size_t pendingTasks() const;

    /**
     * 是否已停止
     */
    bool isStopped() const { return stopped_; }

    /**
     * 停止线程池
     * @param wait 是否等待任务完成
     */
    void stop(bool wait = true);

private:
    void workerThread();

    std::vector<std::thread> threads_;
    std::queue<std::function<void()>> tasks_;
    
    mutable std::mutex mutex_;
    std::condition_variable cv_;
    std::condition_variable cvDone_;
    
    std::atomic<bool> stopped_;
    std::atomic<std::size_t> activeTasks_;
    std::size_t waitingForDone_;
};

// ==================== 模板实现 ====================

template <typename F, typename... Args>
auto ThreadPool::submit(F&& f, Args&&... args) -> std::future<typename std::invoke_result<F, Args...>::type> {
    using ReturnType = typename std::invoke_result<F, Args...>::type;

    auto task = std::make_shared<std::packaged_task<ReturnType()>>(
        std::bind(std::forward<F>(f), std::forward<Args>(args)...)
    );

    std::future<ReturnType> result = task->get_future();

    {
        std::lock_guard<std::mutex> lock(mutex_);
        if (stopped_) {
            throw std::runtime_error("ThreadPool is stopped");
        }
        tasks_.emplace([task]() { (*task)(); });
    }

    cv_.notify_one();
    return result;
}

} // namespace infra::util
