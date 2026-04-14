#include "infrastructure/util/coroutine.hpp"
#include "infrastructure/util/thread_pool.hpp"

#include <iostream>
#include <cassert>
#include <string>

using namespace infra::util;
using namespace infra::util::coro;

// ==================== Test Cases ====================

// Test 1: Basic Task<void>
Task<void> simple_void_task() {
    std::cout << "Hello from coroutine!\n";
    co_return;
}

// Test 2: Task with return value
Task<int> simple_int_task() {
    std::cout << "Computing value...\n";
    co_return 42;
}

// Test 3: Chained coroutines
Task<int> chained_task() {
    std::cout << "Chain step 1\n";
    auto task = simple_int_task();
    int result = co_await task;
    std::cout << "Chain step 2, got: " << result << "\n";
    co_return result * 2;
}

// Test 4: Exception handling
Task<int> throwing_task() {
    std::cout << "About to throw...\n";
    throw std::runtime_error("Test exception");
    co_return 0;
}

Task<void> test_exception() {
    try {
        co_await throwing_task();
    } catch (const std::runtime_error& e) {
        std::cout << "Caught exception: " << e.what() << "\n";
    }
}

// Test 5: ThreadPool integration
Task<int> thread_pool_task(ThreadPool& pool) {
    std::cout << "Submitting to thread pool...\n";
    int result = co_await submit_to(pool, []() {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
        return 100;
    });
    std::cout << "Got result from thread pool: " << result << "\n";
    co_return result;
}

// Test 6: Generator
Generator<int> fibonacci(int n) {
    int a = 0, b = 1;
    for (int i = 0; i < n; ++i) {
        co_yield a;
        auto tmp = a;
        a = b;
        b = tmp + b;
    }
}

// ==================== Main ====================

int main() {
    std::cout << "=== Coroutine Library Tests ===\n\n";

    // Test 1: Void task
    {
        std::cout << "--- Test 1: Void Task ---\n";
        auto task = simple_void_task();
        task.start();
        task.get();
        std::cout << "Test 1 passed!\n\n";
    }

    // Test 2: Int task
    {
        std::cout << "--- Test 2: Int Task ---\n";
        auto task = simple_int_task();
        task.start();
        int result = task.get();
        assert(result == 42);
        std::cout << "Result: " << result << "\n";
        std::cout << "Test 2 passed!\n\n";
    }

    // Test 3: Chained task
    {
        std::cout << "--- Test 3: Chained Task ---\n";
        auto task = chained_task();
        task.start();
        int result = task.get();
        assert(result == 84);
        std::cout << "Result: " << result << "\n";
        std::cout << "Test 3 passed!\n\n";
    }

    // Test 4: Exception handling
    {
        std::cout << "--- Test 4: Exception Handling ---\n";
        auto task = test_exception();
        task.start();
        task.get();
        std::cout << "Test 4 passed!\n\n";
    }

    // Test 5: ThreadPool integration
    {
        std::cout << "--- Test 5: ThreadPool Integration ---\n";
        ThreadPool pool(4);
        auto task = thread_pool_task(pool);
        task.start();
        int result = task.get();
        assert(result == 100);
        std::cout << "Test 5 passed!\n\n";
    }

    // Test 6: Generator
    {
        std::cout << "--- Test 6: Generator ---\n";
        std::cout << "Fibonacci(10): ";
        for (int val : fibonacci(10)) {
            std::cout << val << " ";
        }
        std::cout << "\nTest 6 passed!\n\n";
    }

    std::cout << "=== All Tests Passed! ===\n";
    return 0;
}
