#pragma once

#include "infrastructure/util/thread_pool.hpp"
#include "infrastructure/util/timing_wheel.h"

#include <coroutine>
#include <chrono>
#include <thread>
#include <future>

namespace infra::util::coro {

// ==================== ThreadPool Awaiter ====================

// Awaiter for submitting callable to ThreadPool
template <typename F>
class ThreadPoolAwaiter {
public:
    using ResultType = std::invoke_result_t<F>;

    ThreadPoolAwaiter(ThreadPool& pool, F&& func)
        : pool_(pool), func_(std::forward<F>(func)) {}

    bool await_ready() const noexcept { return false; }

    void await_suspend(std::coroutine_handle<> handle) {
        if constexpr (std::is_void_v<ResultType>) {
            pool_.submitVoid([this, handle]() {
                try {
                    func_();
                } catch (...) {
                    exception_ = std::current_exception();
                }
                handle.resume();
            });
        } else {
            future_ = pool_.submit([this, handle]() mutable -> ResultType {
                try {
                    return func_();
                } catch (...) {
                    exception_ = std::current_exception();
                    return ResultType{};
                }
            });
            // Wait for result in a separate thread to avoid blocking
            std::thread([this, handle]() mutable {
                try {
                    result_ = future_.get();
                } catch (...) {
                    exception_ = std::current_exception();
                }
                handle.resume();
            }).detach();
        }
    }

    ResultType await_resume() {
        if (exception_) {
            std::rethrow_exception(exception_);
        }
        if constexpr (!std::is_void_v<ResultType>) {
            return std::move(result_);
        }
    }

private:
    ThreadPool& pool_;
    F func_;
    std::exception_ptr exception_;
    std::future<ResultType> future_;
    ResultType result_{};
};

// Helper function to create ThreadPool awaiter
template <typename F>
auto submit_to(ThreadPool& pool, F&& func) {
    return ThreadPoolAwaiter<F>(pool, std::forward<F>(func));
}

// ==================== Sleep Awaiter ====================

// Awaiter for sleeping current coroutine (blocking - for simple use)
class SleepAwaiter {
public:
    explicit SleepAwaiter(std::chrono::milliseconds duration)
        : duration_(duration) {}

    bool await_ready() const noexcept { return duration_.count() == 0; }

    void await_suspend(std::coroutine_handle<> handle) {
        std::this_thread::sleep_for(duration_);
        handle.resume();
    }

    void await_resume() const noexcept {}

private:
    std::chrono::milliseconds duration_;
};

// Helper function
inline SleepAwaiter sleep_for(std::chrono::milliseconds duration) {
    return SleepAwaiter(duration);
}

// ==================== TimingWheel Sleep Awaiter ====================

// Async sleep using TimingWheel (non-blocking)
class TimingWheelSleepAwaiter {
public:
    TimingWheelSleepAwaiter(TimingWheel& wheel, std::chrono::milliseconds duration)
        : wheel_(wheel), duration_(duration) {}

    bool await_ready() const noexcept { return duration_.count() == 0; }

    void await_suspend(std::coroutine_handle<> handle) {
        wheel_.schedule(static_cast<std::uint64_t>(duration_.count()), [handle]() {
            handle.resume();
        });
    }

    void await_resume() const noexcept {}

private:
    TimingWheel& wheel_;
    std::chrono::milliseconds duration_;
};

// Helper function
inline TimingWheelSleepAwaiter async_sleep_for(TimingWheel& wheel, std::chrono::milliseconds duration) {
    return TimingWheelSleepAwaiter(wheel, duration);
}

// ==================== Sync Wait ====================

// Convert async Task to sync (blocking)
template <typename T>
T sync_wait(Task<T>& task) {
    task.start();
    return task.get();
}

inline void sync_wait(Task<void>& task) {
    task.start();
    task.get();
}

} // namespace infra::util::coro
