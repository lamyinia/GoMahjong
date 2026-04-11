#include "infrastructure/util/thread_pool.hpp"

#include <stdexcept>

namespace infra::util {

ThreadPool::ThreadPool(std::size_t threadCount)
    : stopped_(false)
    , activeTasks_(0)
    , waitingForDone_(0) {
    
    if (threadCount == 0) {
        threadCount = std::thread::hardware_concurrency();
        if (threadCount == 0) {
            threadCount = 4;  // 默认 4 个线程
        }
    }

    threads_.reserve(threadCount);
    for (std::size_t i = 0; i < threadCount; ++i) {
        threads_.emplace_back(&ThreadPool::workerThread, this);
    }
}

ThreadPool::~ThreadPool() {
    stop(true);
}

void ThreadPool::workerThread() {
    while (true) {
        std::function<void()> task;

        {
            std::unique_lock<std::mutex> lock(mutex_);
            cv_.wait(lock, [this] {
                return stopped_ || !tasks_.empty();
            });

            if (stopped_ && tasks_.empty()) {
                return;
            }

            task = std::move(tasks_.front());
            tasks_.pop();
            ++activeTasks_;
        }

        task();

        {
            std::lock_guard<std::mutex> lock(mutex_);
            --activeTasks_;
            if (waitingForDone_ > 0 && tasks_.empty() && activeTasks_ == 0) {
                cvDone_.notify_all();
            }
        }
    }
}

void ThreadPool::submitVoid(std::function<void()> task) {
    {
        std::lock_guard<std::mutex> lock(mutex_);
        if (stopped_) {
            throw std::runtime_error("ThreadPool is stopped");
        }
        tasks_.push(std::move(task));
    }
    cv_.notify_one();
}

void ThreadPool::waitAll() {
    std::unique_lock<std::mutex> lock(mutex_);
    ++waitingForDone_;
    cvDone_.wait(lock, [this] {
        return tasks_.empty() && activeTasks_ == 0;
    });
    --waitingForDone_;
}

std::size_t ThreadPool::pendingTasks() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return tasks_.size();
}

void ThreadPool::stop(bool wait) {
    if (stopped_.exchange(true)) {
        return;  // 已经停止
    }

    if (wait) {
        waitAll();
    }

    cv_.notify_all();
    
    for (auto& thread : threads_) {
        if (thread.joinable()) {
            thread.join();
        }
    }
    
    threads_.clear();
}

} // namespace infra::util
