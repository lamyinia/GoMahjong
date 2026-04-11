#pragma once

#include <memory>
#include <stack>
#include <mutex>
#include <functional>
#include <cassert>

namespace infra::util {

/**
 * 对象池
 * 用于复用对象，减少内存分配和释放开销
 * 
 * 适用场景：
 * - 频繁创建/销毁的对象（如麻将牌、游戏消息）
 * - 内存分配开销大的对象
 */
template <typename T>
class ObjectPool {
public:
    using Factory = std::function<std::unique_ptr<T>()>;

    /**
     * 构造对象池
     * @param initialSize 初始对象数量
     * @param factory 对象工厂函数（默认使用 new T）
     */
    explicit ObjectPool(std::size_t initialSize = 0, Factory factory = nullptr);

    /**
     * 获取对象
     * @return 对象智能指针，归还时自动放回池中
     */
    std::unique_ptr<T, std::function<void(T*)>> acquire();

    /**
     * 释放对象到池中
     * @param obj 要释放的对象
     */
    void release(std::unique_ptr<T> obj);

    /**
     * 预分配对象
     * @param count 数量
     */
    void preallocate(std::size_t count);

    /**
     * 获取池中可用对象数量
     */
    std::size_t available() const;

    /**
     * 清空对象池
     */
    void clear();

    /**
     * 获取总创建数量
     */
    std::size_t totalCreated() const { return totalCreated_; }

private:
    mutable std::mutex mutex_;
    std::stack<std::unique_ptr<T>> pool_;
    Factory factory_;
    std::size_t totalCreated_;
};

// ==================== 实现 ====================

template <typename T>
ObjectPool<T>::ObjectPool(std::size_t initialSize, Factory factory)
    : factory_(factory ? factory : []() { return std::make_unique<T>(); })
    , totalCreated_(0) {
    preallocate(initialSize);
}

template <typename T>
std::unique_ptr<T, std::function<void(T*)>> ObjectPool<T>::acquire() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (pool_.empty()) {
        ++totalCreated_;
        auto obj = factory_();
        // 返回带自定义删除器的智能指针
        return std::unique_ptr<T, std::function<void(T*)>>(
            obj.release(),
            [this](T* p) {
                this->release(std::unique_ptr<T>(p));
            }
        );
    }
    
    auto obj = std::move(pool_.top());
    pool_.pop();
    
    return std::unique_ptr<T, std::function<void(T*)>>(
        obj.release(),
        [this](T* p) {
            this->release(std::unique_ptr<T>(p));
        }
    );
}

template <typename T>
void ObjectPool<T>::release(std::unique_ptr<T> obj) {
    if (!obj) return;
    
    std::lock_guard<std::mutex> lock(mutex_);
    pool_.push(std::move(obj));
}

template <typename T>
void ObjectPool<T>::preallocate(std::size_t count) {
    std::lock_guard<std::mutex> lock(mutex_);
    for (std::size_t i = 0; i < count; ++i) {
        ++totalCreated_;
        pool_.push(factory_());
    }
}

template <typename T>
std::size_t ObjectPool<T>::available() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return pool_.size();
}

template <typename T>
void ObjectPool<T>::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    while (!pool_.empty()) {
        pool_.pop();
    }
}

} // namespace infra::util
