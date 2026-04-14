#include "infrastructure/util/memory.hpp"

#include <iostream>
#include <cassert>
#include <chrono>
#include <vector>
#include <thread>
#include <random>

using namespace infra::util::memory;

// ==================== Test Types ====================

struct TestObject {
    int x;
    double y;
    char data[32];

    TestObject() : x(0), y(0.0) {}
    TestObject(int a, double b) : x(a), y(b) {}
    
    ~TestObject() {
        // Track destructor calls
        x = -1;
    }
};

// ==================== Tests ====================

void test_fixed_size_pool_basic() {
    std::cout << "--- Test: FixedSizeMemoryPool Basic ---\n";

    FixedSizeMemoryPool pool(64, 16);
    
    assert(pool.blockSize() >= 64);
    assert(pool.capacity() >= 16);
    
    // Allocate some blocks
    void* blocks[10];
    for (int i = 0; i < 10; ++i) {
        blocks[i] = pool.allocate();
        assert(blocks[i] != nullptr);
    }
    
    assert(pool.used() == 10);
    
    // Deallocate
    for (int i = 0; i < 10; ++i) {
        pool.deallocate(blocks[i]);
    }
    
    assert(pool.used() == 0);
    
    std::cout << "Test passed!\n\n";
}

void test_fixed_size_pool_expand() {
    std::cout << "--- Test: FixedSizeMemoryPool Expand ---\n";

    FixedSizeMemoryPool pool(64, 4);  // Small initial capacity
    
    std::vector<void*> blocks;
    
    // Allocate more than initial capacity
    for (int i = 0; i < 20; ++i) {
        void* ptr = pool.allocate();
        assert(ptr != nullptr);
        blocks.push_back(ptr);
    }
    
    assert(pool.used() == 20);
    
    // Deallocate all
    for (void* ptr : blocks) {
        pool.deallocate(ptr);
    }
    
    assert(pool.used() == 0);
    
    std::cout << "Test passed!\n\n";
}

void test_typed_object_pool_basic() {
    std::cout << "--- Test: TypedObjectPool Basic ---\n";

    TypedObjectPool<TestObject> pool(16);
    
    {
        auto obj = pool.acquire(42, 3.14);
        assert(obj);
        assert(obj->x == 42);
        assert(obj->y == 3.14);
        
        // Auto-release when out of scope
    }
    
    assert(pool.used() == 0);
    
    std::cout << "Test passed!\n\n";
}

void test_typed_object_pool_multiple() {
    std::cout << "--- Test: TypedObjectPool Multiple Objects ---\n";

    TypedObjectPool<TestObject> pool(8);
    
    std::vector<TypedObjectPool<TestObject>::PooledPtr> objects;
    
    // Acquire multiple objects
    for (int i = 0; i < 10; ++i) {
        auto obj = pool.acquire(i, i * 1.5);
        assert(obj);
        assert(obj->x == i);
        objects.push_back(std::move(obj));
    }
    
    assert(pool.used() == 10);
    
    // Release all
    objects.clear();
    
    assert(pool.used() == 0);
    
    std::cout << "Test passed!\n\n";
}

void test_typed_object_pool_reuse() {
    std::cout << "--- Test: TypedObjectPool Object Reuse ---\n";

    TypedObjectPool<TestObject> pool(4);
    
    // Acquire and release
    {
        auto obj1 = pool.acquire(100, 2.0);
        auto obj2 = pool.acquire(200, 4.0);
        
        // Memory should be reused
        void* ptr1 = obj1.get();
        void* ptr2 = obj2.get();
        
        // obj1, obj2 released here
    }
    
    // Re-acquire - should get same memory
    {
        auto obj1 = pool.acquire(300, 6.0);
        auto obj2 = pool.acquire(400, 8.0);
        
        // Objects should be constructed correctly
        assert(obj1->x == 300);
        assert(obj2->x == 400);
    }
    
    std::cout << "Test passed!\n\n";
}

void test_performance_comparison() {
    std::cout << "--- Test: Performance Comparison ---\n";

    constexpr int ITERATIONS = 100000;
    
    // Test with new/delete
    {
        auto start = std::chrono::high_resolution_clock::now();
        
        for (int i = 0; i < ITERATIONS; ++i) {
            auto* obj = new TestObject(i, i * 1.0);
            delete obj;
        }
        
        auto end = std::chrono::high_resolution_clock::now();
        auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);
        std::cout << "new/delete: " << duration.count() << " us\n";
    }
    
    // Test with pool
    {
        TypedObjectPool<TestObject> pool(1024);
        
        auto start = std::chrono::high_resolution_clock::now();
        
        for (int i = 0; i < ITERATIONS; ++i) {
            auto obj = pool.acquire(i, i * 1.0);
            obj.reset();
        }
        
        auto end = std::chrono::high_resolution_clock::now();
        auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);
        std::cout << "TypedObjectPool: " << duration.count() << " us\n";
    }
    
    std::cout << "Test passed!\n\n";
}

void test_multithreaded() {
    std::cout << "--- Test: Multithreaded Access ---\n";

    TypedObjectPool<TestObject> pool(256);
    constexpr int THREADS = 4;
    constexpr int ITERATIONS_PER_THREAD = 1000;
    
    std::vector<std::thread> threads;
    
    for (int t = 0; t < THREADS; ++t) {
        threads.emplace_back([&pool, t]() {
            for (int i = 0; i < ITERATIONS_PER_THREAD; ++i) {
                auto obj = pool.acquire(t * 1000 + i, i * 0.5);
                assert(obj);
                // Simulate work
                std::this_thread::sleep_for(std::chrono::microseconds(1));
                // Auto-release
            }
        });
    }
    
    for (auto& thread : threads) {
        thread.join();
    }
    
    assert(pool.used() == 0);
    
    std::cout << "Test passed!\n\n";
}

// ==================== Main ====================

int main() {
    std::cout << "=== Memory Pool Tests ===\n\n";

    test_fixed_size_pool_basic();
    test_fixed_size_pool_expand();
    test_typed_object_pool_basic();
    test_typed_object_pool_multiple();
    test_typed_object_pool_reuse();
    test_performance_comparison();
    test_multithreaded();

    std::cout << "=== All Tests Passed! ===\n";
    return 0;
}
