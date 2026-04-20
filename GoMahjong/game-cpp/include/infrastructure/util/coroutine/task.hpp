#pragma once

#include <coroutine>
#include <exception>
#include <utility>
#include <memory>
#include <atomic>
#include <mutex>
#include <condition_variable>
#include <optional>

namespace infra::util::coro {

// Forward declarations
template <typename T>
class Task;

template <typename T>
class TaskPromise;

namespace detail {

// Shared state for Task result storage
template <typename T>
struct TaskState {
    std::atomic<bool> ready{false};
    std::exception_ptr exception;
    std::optional<T> value;
    std::coroutine_handle<> continuation;
    std::mutex mutex;
    std::condition_variable cv;
};

template <>
struct TaskState<void> {
    std::atomic<bool> ready{false};
    std::exception_ptr exception;
    std::coroutine_handle<> continuation;
    std::mutex mutex;
    std::condition_variable cv;
};

} // namespace detail

// Promise type for Task<T>
template <typename T>
class TaskPromise {
public:
    TaskPromise() = default;

    Task<T> get_return_object() noexcept;

    std::suspend_always initial_suspend() noexcept { return {}; }

    auto final_suspend() noexcept {
        struct FinalAwaiter {
            bool await_ready() noexcept { return false; }
            
            std::coroutine_handle<> await_suspend(std::coroutine_handle<TaskPromise> h) noexcept {
                auto& promise = h.promise();
                if (promise.state_) {
                    std::lock_guard<std::mutex> lock(promise.state_->mutex);
                    promise.state_->ready.store(true, std::memory_order_release);
                    if (promise.state_->continuation) {
                        return promise.state_->continuation;
                    }
                    promise.state_->cv.notify_all();
                }
                return std::noop_coroutine();
            }
            
            void await_resume() noexcept {}
        };
        return FinalAwaiter{};
    }

    void return_value(T value) {
        if (state_) {
            std::lock_guard<std::mutex> lock(state_->mutex);
            state_->value = std::move(value);
        }
    }

    void unhandled_exception() {
        if (state_) {
            std::lock_guard<std::mutex> lock(state_->mutex);
            state_->exception = std::current_exception();
        }
    }

    void setState(std::shared_ptr<detail::TaskState<T>> state) {
        state_ = std::move(state);
    }

private:
    std::shared_ptr<detail::TaskState<T>> state_;
};

// Promise type for Task<void>
template <>
class TaskPromise<void> {
public:
    TaskPromise() = default;

    Task<void> get_return_object() noexcept;

    std::suspend_always initial_suspend() noexcept { return {}; }

    auto final_suspend() noexcept {
        struct FinalAwaiter {
            bool await_ready() noexcept { return false; }
            
            std::coroutine_handle<> await_suspend(std::coroutine_handle<TaskPromise> h) noexcept {
                auto& promise = h.promise();
                if (promise.state_) {
                    std::lock_guard<std::mutex> lock(promise.state_->mutex);
                    promise.state_->ready.store(true, std::memory_order_release);
                    if (promise.state_->continuation) {
                        return promise.state_->continuation;
                    }
                    promise.state_->cv.notify_all();
                }
                return std::noop_coroutine();
            }
            
            void await_resume() noexcept {}
        };
        return FinalAwaiter{};
    }

    void return_void() {
        if (state_) {
            std::lock_guard<std::mutex> lock(state_->mutex);
        }
    }

    void unhandled_exception() {
        if (state_) {
            std::lock_guard<std::mutex> lock(state_->mutex);
            state_->exception = std::current_exception();
        }
    }

    void setState(std::shared_ptr<detail::TaskState<void>> state) {
        state_ = std::move(state);
    }

private:
    std::shared_ptr<detail::TaskState<void>> state_;
};

// Task<T> - main coroutine return type
template <typename T>
class Task {
public:
    using promise_type = TaskPromise<T>;
    using value_type = T;

    Task() noexcept : handle_(nullptr) {}

    explicit Task(std::coroutine_handle<promise_type> h) noexcept : handle_(h) {}

    Task(std::coroutine_handle<promise_type> h, std::shared_ptr<detail::TaskState<T>> state) noexcept
        : handle_(h), state_(std::move(state)) {}

    Task(Task&& other) noexcept : handle_(other.handle_), state_(std::move(other.state_)) {
        other.handle_ = nullptr;
    }

    Task& operator=(Task&& other) noexcept {
        if (this != &other) {
            destroy();
            handle_ = other.handle_;
            state_ = std::move(other.state_);
            other.handle_ = nullptr;
        }
        return *this;
    }

    Task(const Task&) = delete;
    Task& operator=(const Task&) = delete;

    ~Task() { destroy(); }

    explicit operator bool() const noexcept { return handle_ != nullptr; }

    bool done() const noexcept { return handle_ && handle_.done(); }

    void start() {
        if (handle_ && !handle_.done()) {
            handle_.resume();
        }
    }

    T get() {
        if (!handle_) {
            throw std::runtime_error("Task is empty");
        }

        {
            std::unique_lock<std::mutex> lock(state_->mutex);
            state_->cv.wait(lock, [this] {
                return state_->ready.load(std::memory_order_acquire);
            });
        }

        if (state_->exception) {
            std::rethrow_exception(state_->exception);
        }

        return std::move(*state_->value);
    }

    auto operator co_await() {
        struct Awaiter {
            std::coroutine_handle<promise_type> handle;
            std::shared_ptr<detail::TaskState<T>> state;

            bool await_ready() {
                return state->ready.load(std::memory_order_acquire);
            }

            std::coroutine_handle<> await_suspend(std::coroutine_handle<> continuation) {
                std::lock_guard<std::mutex> lock(state->mutex);
                state->continuation = continuation;
                if (state->ready.load(std::memory_order_acquire)) {
                    return continuation;
                }
                return handle;
            }

            T await_resume() {
                if (state->exception) {
                    std::rethrow_exception(state->exception);
                }
                return std::move(*state->value);
            }
        };

        return Awaiter{handle_, state_};
    }

    std::coroutine_handle<promise_type> handle() const noexcept { return handle_; }

private:
    void destroy() {
        if (handle_) {
            if (!handle_.done()) {
                handle_.destroy();
            }
            handle_ = nullptr;
        }
    }

    std::coroutine_handle<promise_type> handle_;
    std::shared_ptr<detail::TaskState<T>> state_;
};

// Task<void> specialization
template <>
class Task<void> {
public:
    using promise_type = TaskPromise<void>;
    using value_type = void;

    Task() noexcept : handle_(nullptr) {}

    explicit Task(std::coroutine_handle<promise_type> h) noexcept : handle_(h) {}

    Task(std::coroutine_handle<promise_type> h, std::shared_ptr<detail::TaskState<void>> state) noexcept
        : handle_(h), state_(std::move(state)) {}

    Task(Task&& other) noexcept : handle_(other.handle_), state_(std::move(other.state_)) {
        other.handle_ = nullptr;
    }

    Task& operator=(Task&& other) noexcept {
        if (this != &other) {
            destroy();
            handle_ = other.handle_;
            state_ = std::move(other.state_);
            other.handle_ = nullptr;
        }
        return *this;
    }

    Task(const Task&) = delete;
    Task& operator=(const Task&) = delete;

    ~Task() { destroy(); }

    explicit operator bool() const noexcept { return handle_ != nullptr; }

    bool done() const noexcept { return handle_ && handle_.done(); }

    void start() {
        if (handle_ && !handle_.done()) {
            handle_.resume();
        }
    }

    void get() {
        if (!handle_) {
            throw std::runtime_error("Task is empty");
        }

        {
            std::unique_lock<std::mutex> lock(state_->mutex);
            state_->cv.wait(lock, [this] {
                return state_->ready.load(std::memory_order_acquire);
            });
        }

        if (state_->exception) {
            std::rethrow_exception(state_->exception);
        }
    }

    auto operator co_await() {
        struct Awaiter {
            std::coroutine_handle<promise_type> handle;
            std::shared_ptr<detail::TaskState<void>> state;

            bool await_ready() {
                return state->ready.load(std::memory_order_acquire);
            }

            std::coroutine_handle<> await_suspend(std::coroutine_handle<> continuation) {
                std::lock_guard<std::mutex> lock(state->mutex);
                state->continuation = continuation;
                if (state->ready.load(std::memory_order_acquire)) {
                    return continuation;
                }
                return handle;
            }

            void await_resume() {
                if (state->exception) {
                    std::rethrow_exception(state->exception);
                }
            }
        };

        return Awaiter{handle_, state_};
    }

    std::coroutine_handle<promise_type> handle() const noexcept { return handle_; }

private:
    void destroy() {
        if (handle_) {
            if (!handle_.done()) {
                handle_.destroy();
            }
            handle_ = nullptr;
        }
    }

    std::coroutine_handle<promise_type> handle_;
    std::shared_ptr<detail::TaskState<void>> state_;
};

// Implement get_return_object
template <typename T>
Task<T> TaskPromise<T>::get_return_object() noexcept {
    auto state = std::make_shared<detail::TaskState<T>>();
    state_ = state;
    return Task<T>(std::coroutine_handle<TaskPromise>::from_promise(*this), std::move(state));
}

inline Task<void> TaskPromise<void>::get_return_object() noexcept {
    auto state = std::make_shared<detail::TaskState<void>>();
    state_ = state;
    return Task<void>(std::coroutine_handle<TaskPromise>::from_promise(*this), std::move(state));
}

} // namespace infra::util::coro
