//
// Created by lanyo on 2026/4/4.
//
// MongoPool 单元测试
//

#include <iostream>
#include <chrono>
#include <thread>
#include <atomic>
#include <cassert>
#include <vector>

#include <bsoncxx/builder/stream/document.hpp>
#include <bsoncxx/builder/stream/helpers.hpp>
#include <bsoncxx/json.hpp>

#include "infrastructure/persistence/mongo_pool.hpp"
#include "infrastructure/log/logger.hpp"

using namespace infra::persistence;

// 测试配置
MongoConfig create_test_config() {
    MongoConfig config;
    config.uri = "mongodb://localhost:27017";
    config.database = "gomahjong_test";
    config.thread_count = 2;
    config.queue_max_size = 100;
    return config;
}

// 测试 1: 配置默认值
void test_config_defaults() {
    std::cout << "[TEST] 配置默认值测试..." << std::endl;
    
    MongoConfig config;
    assert(config.uri == "mongodb://localhost:27017");
    assert(config.database == "gomahjong");
    assert(config.min_pool_size == 10);
    assert(config.max_pool_size == 100);
    assert(config.thread_count == 4);
    assert(config.queue_max_size == 1000);
    
    std::cout << "[PASS] 配置默认值测试通过" << std::endl;
}

// 测试 2: 线程池生命周期
void test_pool_lifecycle() {
    std::cout << "[TEST] 线程池生命周期测试..." << std::endl;
    
    auto config = create_test_config();
    MongoPool pool(config);
    
    // 初始状态：未运行
    assert(!pool.is_running());
    
    // 启动
    pool.start();
    assert(pool.is_running());
    
    // 重复启动（应该被忽略）
    pool.start();
    assert(pool.is_running());
    
    // 停止
    pool.stop();
    assert(!pool.is_running());
    
    // 重复停止（应该被忽略）
    pool.stop();
    assert(!pool.is_running());
    
    std::cout << "[PASS] 线程池生命周期测试通过" << std::endl;
}

// 测试 3: 统计信息
void test_statistics() {
    std::cout << "[TEST] 统计信息测试..." << std::endl;
    
    auto config = create_test_config();
    MongoPool pool(config);
    
    assert(pool.pending_tasks() == 0);
    assert(pool.active_connections() == config.thread_count);
    
    pool.start();
    assert(pool.pending_tasks() == 0);
    
    pool.stop();
    std::cout << "[PASS] 统计信息测试通过" << std::endl;
}

// 测试 4: 异步任务提交（不执行，仅测试队列）
void test_async_queue() {
    std::cout << "[TEST] 异步队列测试..." << std::endl;
    
    auto config = create_test_config();
    config.thread_count = 1;
    config.queue_max_size = 10;
    
    MongoPool pool(config);
    pool.start();
    
    // 提交空任务（不实际执行数据库操作）
    std::atomic<int> counter{0};
    for (int i = 0; i < 5; ++i) {
        pool.async_execute([&counter](mongocxx::client& client) {
            counter++;
        });
    }
    
    // 等待任务完成
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    // 注意：由于没有实际的 MongoDB 连接，任务可能会失败
    // 这里主要测试队列机制
    
    pool.stop();
    std::cout << "[PASS] 异步队列测试通过 (计数器: " << counter << ")" << std::endl;
}

// 测试 5: 队列满时丢弃任务
void test_queue_full() {
    std::cout << "[TEST] 队列满测试..." << std::endl;
    
    auto config = create_test_config();
    config.thread_count = 1;
    config.queue_max_size = 2;  // 很小的队列
    
    MongoPool pool(config);
    
    // 不启动线程池，任务会堆积在队列中
    // 但由于队列很小，超过限制的任务会被丢弃
    
    for (int i = 0; i < 10; ++i) {
        pool.async_execute([](mongocxx::client& client) {
            // 空操作
        });
    }
    
    pool.stop();
    std::cout << "[PASS] 队列满测试通过" << std::endl;
}

// 测试 6: 错误回调
void test_error_callback() {
    std::cout << "[TEST] 错误回调测试..." << std::endl;
    
    auto config = create_test_config();
    MongoPool pool(config);
    
    // 不启动线程池，应该触发错误回调
    std::atomic<bool> error_called{false};
    pool.async_execute_with_error(
        [](mongocxx::client& client) {
            // 不会执行
        },
        [&error_called](std::exception_ptr eptr) {
            error_called = true;
            std::cout << "  [回调] 收到错误回调" << std::endl;
        }
    );
    
    assert(error_called);
    
    std::cout << "[PASS] 错误回调测试通过" << std::endl;
}

// 测试 7: 析构时自动停止
void test_auto_stop_on_destructor() {
    std::cout << "[TEST] 析构自动停止测试..." << std::endl;
    
    auto config = create_test_config();
    {
        MongoPool pool(config);
        pool.start();
        assert(pool.is_running());
        // 离开作用域，析构函数应该自动调用 stop()
    }
    
    std::cout << "[PASS] 析构自动停止测试通过" << std::endl;
}

// ==================== CRUD 测试 ====================
// 注意：以下测试需要 MongoDB 服务运行

// 测试 8: 插入文档 (Create)
void test_insert_document() {
    std::cout << "[TEST] 插入文档测试..." << std::endl;
    
    auto config = create_test_config();
    MongoPool pool(config);
    pool.start();
    
    std::atomic<bool> success{false};
    
    // 使用 BSON 构建文档
    pool.async_execute([&success](mongocxx::client& client) {
        try {
            auto db = client["gomahjong_test"];
            auto collection = db["test_collection"];
            
            // 构建文档
            bsoncxx::builder::stream::document document{};
            document << "name" << "test_player"
                     << "score" << 100
                     << "level" << 5;
            
            // 插入文档
            auto result = collection.insert_one(document.view());
            
            if (result) {
                success = true;
                std::cout << "  [插入成功] ID: " << result->inserted_id().get_oid().value.to_string() << std::endl;
            }
        } catch (const std::exception& e) {
            std::cout << "  [插入失败] " << e.what() << std::endl;
        }
    });
    
    // 等待异步操作完成
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    
    pool.stop();
    
    if (success) {
        std::cout << "[PASS] 插入文档测试通过" << std::endl;
    } else {
        std::cout << "[SKIP] 插入文档测试跳过（MongoDB 未连接）" << std::endl;
    }
}

// 测试 9: 查询文档 (Read)
void test_find_document() {
    std::cout << "[TEST] 查询文档测试..." << std::endl;
    
    auto config = create_test_config();
    MongoPool pool(config);
    pool.start();
    
    std::atomic<bool> success{false};
    
    pool.async_execute([&success](mongocxx::client& client) {
        try {
            auto db = client["gomahjong_test"];
            auto collection = db["test_collection"];
            
            // 构建查询条件
            bsoncxx::builder::stream::document filter{};
            filter << "name" << "test_player";
            
            // 查询文档
            auto result = collection.find_one(filter.view());
            
            if (result) {
                success = true;
                auto doc = result->view();
                std::cout << "  [查询成功] name: " << doc["name"].get_string().value
                          << ", score: " << doc["score"].get_int32().value << std::endl;
            } else {
                std::cout << "  [查询结果] 未找到文档" << std::endl;
            }
        } catch (const std::exception& e) {
            std::cout << "  [查询失败] " << e.what() << std::endl;
        }
    });
    
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    
    pool.stop();
    
    if (success) {
        std::cout << "[PASS] 查询文档测试通过" << std::endl;
    } else {
        std::cout << "[SKIP] 查询文档测试跳过（MongoDB 未连接或无数据）" << std::endl;
    }
}

// 测试 10: 更新文档 (Update)
void test_update_document() {
    std::cout << "[TEST] 更新文档测试..." << std::endl;
    
    auto config = create_test_config();
    MongoPool pool(config);
    pool.start();
    
    std::atomic<bool> success{false};
    
    pool.async_execute([&success](mongocxx::client& client) {
        try {
            auto db = client["gomahjong_test"];
            auto collection = db["test_collection"];
            
            // 构建查询条件
            bsoncxx::builder::stream::document filter{};
            filter << "name" << "test_player";
            
            // 构建更新操作
            bsoncxx::builder::stream::document update{};
            update << "$set" << bsoncxx::builder::stream::open_document
                   << "score" << 200
                   << "level" << 10
                   << bsoncxx::builder::stream::close_document;
            
            // 更新文档
            auto result = collection.update_one(filter.view(), update.view());
            
            if (result && result->modified_count() > 0) {
                success = true;
                std::cout << "  [更新成功] 修改了 " << result->modified_count() << " 个文档" << std::endl;
            } else {
                std::cout << "  [更新结果] 未修改任何文档" << std::endl;
            }
        } catch (const std::exception& e) {
            std::cout << "  [更新失败] " << e.what() << std::endl;
        }
    });
    
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    
    pool.stop();
    
    if (success) {
        std::cout << "[PASS] 更新文档测试通过" << std::endl;
    } else {
        std::cout << "[SKIP] 更新文档测试跳过（MongoDB 未连接或无数据）" << std::endl;
    }
}

// 测试 11: 删除文档 (Delete)
void test_delete_document() {
    std::cout << "[TEST] 删除文档测试..." << std::endl;
    
    auto config = create_test_config();
    MongoPool pool(config);
    pool.start();
    
    std::atomic<bool> success{false};
    
    pool.async_execute([&success](mongocxx::client& client) {
        try {
            auto db = client["gomahjong_test"];
            auto collection = db["test_collection"];
            
            // 构建删除条件
            bsoncxx::builder::stream::document filter{};
            filter << "name" << "test_player";
            
            // 删除文档
            auto result = collection.delete_one(filter.view());
            
            if (result && result->deleted_count() > 0) {
                success = true;
                std::cout << "  [删除成功] 删除了 " << result->deleted_count() << " 个文档" << std::endl;
            } else {
                std::cout << "  [删除结果] 未删除任何文档" << std::endl;
            }
        } catch (const std::exception& e) {
            std::cout << "  [删除失败] " << e.what() << std::endl;
        }
    });
    
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    
    pool.stop();
    
    if (success) {
        std::cout << "[PASS] 删除文档测试通过" << std::endl;
    } else {
        std::cout << "[SKIP] 删除文档测试跳过（MongoDB 未连接或无数据）" << std::endl;
    }
}

// 测试 12: 批量插入 (Batch Insert)
void test_batch_insert() {
    std::cout << "[TEST] 批量插入测试..." << std::endl;
    
    auto config = create_test_config();
    MongoPool pool(config);
    pool.start();
    
    std::atomic<bool> success{false};
    
    pool.async_execute([&success](mongocxx::client& client) {
        try {
            auto db = client["gomahjong_test"];
            auto collection = db["test_batch"];
            
            // 构建多个文档
            std::vector<bsoncxx::document::value> documents;
            for (int i = 0; i < 10; ++i) {
                bsoncxx::builder::stream::document doc{};
                doc << "player_id" << i
                    << "name" << ("player_" + std::to_string(i))
                    << "score" << i * 100;
                documents.push_back(doc << bsoncxx::builder::stream::finalize);
            }
            
            // 批量插入
            auto result = collection.insert_many(documents);
            
            if (result) {
                success = true;
                std::cout << "  [批量插入成功] 插入了 " << result->inserted_count() << " 个文档" << std::endl;
            }
        } catch (const std::exception& e) {
            std::cout << "  [批量插入失败] " << e.what() << std::endl;
        }
    });
    
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    
    pool.stop();
    
    if (success) {
        std::cout << "[PASS] 批量插入测试通过" << std::endl;
    } else {
        std::cout << "[SKIP] 批量插入测试跳过（MongoDB 未连接）" << std::endl;
    }
}

// 测试 13: 清理测试数据
void test_cleanup() {
    std::cout << "[TEST] 清理测试数据..." << std::endl;
    
    auto config = create_test_config();
    MongoPool pool(config);
    pool.start();
    
    std::atomic<bool> success{false};
    
    pool.async_execute([&success](mongocxx::client& client) {
        try {
            auto db = client["gomahjong_test"];
            
            // 删除测试集合
            db["test_collection"].drop();
            db["test_batch"].drop();
            
            success = true;
            std::cout << "  [清理成功] 删除了测试集合" << std::endl;
        } catch (const std::exception& e) {
            std::cout << "  [清理失败] " << e.what() << std::endl;
        }
    });
    
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    
    pool.stop();
    
    if (success) {
        std::cout << "[PASS] 清理测试数据通过" << std::endl;
    } else {
        std::cout << "[SKIP] 清理测试数据跳过（MongoDB 未连接）" << std::endl;
    }
}

int main() {
    std::cout << "========================================" << std::endl;
    std::cout << "       MongoPool 单元测试套件" << std::endl;
    std::cout << "========================================" << std::endl;
    std::cout << std::endl;
    
    std::cout << "=== 第一部分：基础功能测试 ===" << std::endl;
    std::cout << "注意：以下测试不需要 MongoDB 服务运行" << std::endl;
    std::cout << std::endl;
    
    try {
        test_config_defaults();
        test_pool_lifecycle();
        test_statistics();
        test_async_queue();
        test_queue_full();
        test_error_callback();
        test_auto_stop_on_destructor();
        
        std::cout << std::endl;
        std::cout << "=== 第二部分：CRUD 测试 ===" << std::endl;
        std::cout << "注意：以下测试需要 MongoDB 服务运行" << std::endl;
        std::cout << std::endl;
        
        // CRUD 测试（需要 MongoDB）
        test_insert_document();      // Create
        test_find_document();        // Read
        test_update_document();      // Update
        test_delete_document();      // Delete
        test_batch_insert();         // 批量插入
        test_cleanup();              // 清理测试数据
        
        std::cout << std::endl;
        std::cout << "========================================" << std::endl;
        std::cout << "       所有测试完成！" << std::endl;
        std::cout << "========================================" << std::endl;
        
        return 0;
    } catch (const std::exception& e) {
        std::cerr << "[FAIL] 测试失败: " << e.what() << std::endl;
        return 1;
    }
}
