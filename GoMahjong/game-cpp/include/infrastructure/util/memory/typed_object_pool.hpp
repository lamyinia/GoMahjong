#pragma once

#include "infrastructure/util/memory/fixed_size_pool.hpp"

#include <cstddef>
#include <memory>
#include <type_traits>
#include <utility>

namespace infra::util::memory {

/**
 * TypedObjectPool - Type-safe object pool based on FixedSizeMemoryPool
 * 
 * Features:
 * - Automatic constructor/destructor calls
 * - Type-safe interface
 * - RAII wrapper for pooled objects
 * 
 * Usage:
 *   TypedObjectPool<GameEvent> pool(1024);
 *   auto obj = pool.acquire(arg1, arg2);
 *   // use obj...
 *   // auto-release when PooledPtr goes out of scope
 */
template <typename T>
class TypedObjectPool {
public:
    // Forward declaration
    class PooledPtr;

    /**
     * Constructor
     * @param initialCapacity Initial number of objects to preallocate
     */
    explicit TypedObjectPool(std::size_t initialCapacity = 64)
        : pool_(sizeof(T), initialCapacity, alignof(T))
    {}

    /**
     * Acquire an object from the pool
     * @tparam Args Constructor argument types
     * @param args Constructor arguments
     * @return PooledPtr managing the object
     */
    template <typename... Args>
    PooledPtr acquire(Args&&... args) {
        void* memory = pool_.allocate();
        if (memory == nullptr) {
            return PooledPtr(nullptr, this);
        }

        // Construct object in-place
        T* obj = new (memory) T(std::forward<Args>(args)...);
        return PooledPtr(obj, this);
    }

    /**
     * Release an object back to the pool
     * @param obj Pointer to object
     */
    void release(T* obj) {
        if (obj == nullptr) return;
        
        // Call destructor
        obj->~T();
        
        // Return memory to pool
        pool_.deallocate(obj);
    }

    /**
     * Get underlying memory pool stats
     */
    [[nodiscard]] std::size_t blockSize() const noexcept { return pool_.blockSize(); }
    [[nodiscard]] std::size_t capacity() const noexcept { return pool_.capacity(); }
    [[nodiscard]] std::size_t used() const noexcept { return pool_.used(); }
    [[nodiscard]] std::size_t available() const noexcept { return pool_.available(); }

    /**
     * Preallocate more objects
     */
    bool expand(std::size_t count) {
        return pool_.expand(count);
    }

    /**
     * RAII wrapper for pooled objects
     */
    class PooledPtr {
    public:
        PooledPtr() noexcept : obj_(nullptr), pool_(nullptr) {}
        PooledPtr(T* obj, TypedObjectPool* pool) noexcept : obj_(obj), pool_(pool) {}
        
        // Move-only
        PooledPtr(PooledPtr&& other) noexcept : obj_(other.obj_), pool_(other.pool_) {
            other.obj_ = nullptr;
            other.pool_ = nullptr;
        }
        
        PooledPtr& operator=(PooledPtr&& other) noexcept {
            if (this != &other) {
                reset();
                obj_ = other.obj_;
                pool_ = other.pool_;
                other.obj_ = nullptr;
                other.pool_ = nullptr;
            }
            return *this;
        }

        PooledPtr(const PooledPtr&) = delete;
        PooledPtr& operator=(const PooledPtr&) = delete;

        ~PooledPtr() {
            reset();
        }

        // Accessors
        T* get() const noexcept { return obj_; }
        T& operator*() const noexcept { return *obj_; }
        T* operator->() const noexcept { return obj_; }
        
        explicit operator bool() const noexcept { return obj_ != nullptr; }

        /**
         * Release ownership (don't return to pool)
         * @return Raw pointer, caller is responsible for calling pool->release()
         */
        T* release() noexcept {
            T* obj = obj_;
            obj_ = nullptr;
            pool_ = nullptr;
            return obj;
        }

        /**
         * Return object to pool
         */
        void reset() {
            if (obj_ != nullptr && pool_ != nullptr) {
                pool_->release(obj_);
            }
            obj_ = nullptr;
            pool_ = nullptr;
        }

    private:
        T* obj_;
        TypedObjectPool* pool_;
    };

private:
    FixedSizeMemoryPool pool_;
};

// ==================== Convenience Alias ====================

template <typename T>
using PooledPtr = typename TypedObjectPool<T>::PooledPtr;

} // namespace infra::util::memory
