#pragma once

#include <coroutine>
#include <exception>
#include <iterator>
#include <memory>

namespace infra::util::coro {

// Generator<T> - for lazy iteration with co_yield
template <typename T>
class Generator {
public:
    class Promise;
    using promise_type = Promise;

    class Promise {
    public:
        T value_;
        std::exception_ptr exception_;

        Generator get_return_object() {
            return Generator(std::coroutine_handle<Promise>::from_promise(*this));
        }

        std::suspend_always initial_suspend() { return {}; }
        std::suspend_always final_suspend() noexcept { return {}; }

        std::suspend_always yield_value(T value) {
            value_ = std::move(value);
            return {};
        }

        void return_void() {}

        void unhandled_exception() {
            exception_ = std::current_exception();
        }
    };

    class Iterator {
    public:
        using iterator_category = std::input_iterator_tag;
        using value_type = T;
        using difference_type = std::ptrdiff_t;
        using pointer = T*;
        using reference = T&;

        Iterator() noexcept : handle_(nullptr) {}
        explicit Iterator(std::coroutine_handle<Promise> h) : handle_(h) {
            advance();
        }

        reference operator*() const { return handle_.promise().value_; }
        pointer operator->() const { return &handle_.promise().value_; }

        Iterator& operator++() {
            handle_.resume();
            advance();
            return *this;
        }

        Iterator operator++(int) {
            Iterator tmp = *this;
            ++(*this);
            return tmp;
        }

        bool operator==(const Iterator& other) const {
            return handle_ == other.handle_;
        }

        bool operator!=(const Iterator& other) const {
            return !(*this == other);
        }

    private:
        void advance() {
            if (handle_ && handle_.done()) {
                handle_ = nullptr;
            }
        }

        std::coroutine_handle<Promise> handle_;
    };

    Generator() noexcept : handle_(nullptr) {}

    Generator(std::coroutine_handle<Promise> h) : handle_(h) {}

    Generator(Generator&& other) noexcept : handle_(other.handle_) {
        other.handle_ = nullptr;
    }

    Generator& operator=(Generator&& other) noexcept {
        if (this != &other) {
            if (handle_) {
                handle_.destroy();
            }
            handle_ = other.handle_;
            other.handle_ = nullptr;
        }
        return *this;
    }

    Generator(const Generator&) = delete;
    Generator& operator=(const Generator&) = delete;

    ~Generator() {
        if (handle_) {
            handle_.destroy();
        }
    }

    Iterator begin() {
        if (handle_) {
            handle_.resume();
            if (handle_.done()) {
                return end();
            }
        }
        return Iterator(handle_);
    }

    Iterator end() {
        return Iterator{};
    }

private:
    std::coroutine_handle<Promise> handle_;
};

} // namespace infra::util::coro
