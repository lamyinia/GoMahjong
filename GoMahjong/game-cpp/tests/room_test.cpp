#include "domain/game/room/room_manager.h"
#include "domain/game/event/mahjong_game_event.h"
#include "domain/game/engine/engine.h"

#include <iostream>
#include <thread>
#include <vector>
#include <atomic>

using namespace domain::game;

// 测试辅助宏
#define TEST_ASSERT(condition, message) \
    do { \
        if (!(condition)) { \
            std::cerr << "[FAIL] " << message << std::endl; \
            return false; \
        } \
        std::cout << "[PASS] " << message << std::endl; \
    } while(0)

// 测试 1：创建房间
bool test_create_room() {
    room::RoomManager manager(2, 1024);
    manager.start();

    std::vector<std::string> players = {"player1", "player2", "player3", "player4"};
    auto roomId = manager.create_room(players, static_cast<std::int32_t>(engine::EngineType::RiichiMahjong4P));

    TEST_ASSERT(!roomId.empty(), "Room should be created");
    TEST_ASSERT(manager.room_count() == 1, "Room count should be 1");
    TEST_ASSERT(manager.player_count() == 4, "Player count should be 4");

    // 验证玩家路由
    auto playerRoomId = manager.get_player_room_id("player1");
    TEST_ASSERT(playerRoomId.has_value(), "Player1 should have a room");
    TEST_ASSERT(*playerRoomId == roomId, "Player1's room should be the created room");

    manager.stop();
    return true;
}

// 测试 2：重复玩家创建失败
bool test_create_room_duplicate_player() {
    room::RoomManager manager(2, 1024);
    manager.start();

    std::vector<std::string> players1 = {"player1", "player2", "player3", "player4"};
    auto roomId1 = manager.create_room(players1, static_cast<std::int32_t>(engine::EngineType::RiichiMahjong4P));
    TEST_ASSERT(!roomId1.empty(), "First room should be created");

    // 尝试用已存在的玩家创建第二个房间
    std::vector<std::string> players2 = {"player1", "player5", "player6", "player7"};
    auto roomId2 = manager.create_room(players2, static_cast<std::int32_t>(engine::EngineType::RiichiMahjong4P));
    TEST_ASSERT(roomId2.empty(), "Second room should not be created (duplicate player)");
    TEST_ASSERT(manager.room_count() == 1, "Room count should still be 1");

    manager.stop();
    return true;
}

// 测试 3：获取房间 ID
bool test_get_room() {
    room::RoomManager manager(2, 1024);
    manager.start();

    std::vector<std::string> players = {"player1", "player2", "player3", "player4"};
    auto roomId = manager.create_room(players, static_cast<std::int32_t>(engine::EngineType::RiichiMahjong4P));
    TEST_ASSERT(!roomId.empty(), "Room should be created");

    // 通过玩家 ID 获取房间 ID
    auto foundRoomId = manager.get_player_room_id("player1");
    TEST_ASSERT(foundRoomId.has_value(), "Room should be found by player ID");
    TEST_ASSERT(*foundRoomId == roomId, "Found room ID should match created room ID");

    // 获取不存在的玩家
    auto notFound = manager.get_player_room_id("non_existent_player");
    TEST_ASSERT(!notFound.has_value(), "Non-existent player should return nullopt");

    manager.stop();
    return true;
}

// 测试 4：删除房间
bool test_delete_room() {
    room::RoomManager manager(2, 1024);
    manager.start();

    std::vector<std::string> players = {"player1", "player2", "player3", "player4"};
    auto roomId = manager.create_room(players, static_cast<std::int32_t>(engine::EngineType::RiichiMahjong4P));
    TEST_ASSERT(!roomId.empty(), "Room should be created");

    // 删除房间
    bool deleted = manager.delete_room(roomId);
    TEST_ASSERT(deleted, "Room should be deleted successfully");
    TEST_ASSERT(manager.room_count() == 0, "Room count should be 0 after deletion");
    TEST_ASSERT(manager.player_count() == 0, "Player count should be 0 after deletion");

    // 验证玩家路由已清理
    auto playerRoomId = manager.get_player_room_id("player1");
    TEST_ASSERT(!playerRoomId.has_value(), "Player1 should not have a room after deletion");

    // 删除不存在的房间
    bool deletedAgain = manager.delete_room(roomId);
    TEST_ASSERT(!deletedAgain, "Deleting non-existent room should return false");

    manager.stop();
    return true;
}

// 测试 5：提交事件
bool test_submit_event() {
    room::RoomManager manager(2, 1024);
    manager.start();

    std::vector<std::string> players = {"player1", "player2", "player3", "player4"};
    auto roomId = manager.create_room(players, static_cast<std::int32_t>(engine::EngineType::RiichiMahjong4P));
    TEST_ASSERT(!roomId.empty(), "Room should be created");

    auto playerRoomId = manager.get_player_room_id("player1");
    TEST_ASSERT(playerRoomId.has_value(), "Player1 should have a room");
    TEST_ASSERT(*playerRoomId == roomId, "Player1's room should be the created room");

    // 提交出牌事件
    event::Tile tile{event::TileType::Wan1, 0};
    auto gameEvent = event::GameEvent::playTile("player1", tile);
    manager.submitEvent(roomId, gameEvent);

    // 等待事件处理
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // 提交流局事件
    auto drawEvent = event::GameEvent::draw(false);
    manager.submitEvent(roomId, drawEvent);

    // 等待事件处理
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    manager.stop();
    return true;
}

// 测试 6：多房间并发
bool test_multiple_rooms() {
    room::RoomManager manager(4, 2048);
    manager.start();

    // 创建多个房间
    std::vector<std::string> roomIds;
    for (int i = 0; i < 10; ++i) {
        std::vector<std::string> players = {
            "room" + std::to_string(i) + "_p1",
            "room" + std::to_string(i) + "_p2",
            "room" + std::to_string(i) + "_p3",
            "room" + std::to_string(i) + "_p4"
        };
        auto roomId = manager.create_room(players, static_cast<std::int32_t>(engine::EngineType::RiichiMahjong4P));
        TEST_ASSERT(!roomId.empty(), "Room " + std::to_string(i) + " should be created");
        roomIds.push_back(roomId);
    }

    TEST_ASSERT(manager.room_count() == 10, "Room count should be 10");
    TEST_ASSERT(manager.player_count() == 40, "Player count should be 40");

    // 并发提交事件
    std::atomic<int> eventCount{0};
    std::vector<std::thread> threads;
    for (int i = 0; i < 10; ++i) {
        threads.emplace_back([&manager, &roomIds, &eventCount, i]() {
            for (int j = 0; j < 10; ++j) {
                event::Tile tile{event::TileType::Wan1, static_cast<std::int8_t>(j % 4)};
                auto gameEvent = event::GameEvent::playTile(
                    "room" + std::to_string(i) + "_p" + std::to_string(j % 4 + 1), 
                    tile
                );
                manager.submitEvent(roomIds[i], gameEvent);
                eventCount++;
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }

    // 等待所有事件处理完成
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    TEST_ASSERT(eventCount == 100, "Should have submitted 100 events");

    manager.stop();
    return true;
}

// 测试 7：统计信息
bool test_statistics() {
    room::RoomManager manager(2, 1024);
    manager.start();

    TEST_ASSERT(manager.actor_count() == 2, "Actor count should be 2");
    TEST_ASSERT(manager.room_count() == 0, "Initial room count should be 0");
    TEST_ASSERT(manager.player_count() == 0, "Initial player count should be 0");

    std::vector<std::string> players = {"p1", "p2", "p3", "p4"};
    manager.create_room(players, static_cast<std::int32_t>(engine::EngineType::RiichiMahjong4P));

    TEST_ASSERT(manager.room_count() == 1, "Room count should be 1");
    TEST_ASSERT(manager.player_count() == 4, "Player count should be 4");

    manager.stop();
    return true;
}

// 主测试运行器
int main() {
    std::cout << "=== RoomManager Tests ===" << std::endl << std::endl;

    int passed = 0;
    int failed = 0;

    auto runTest = [&](const char* name, bool (*testFunc)()) {
        std::cout << "--- " << name << " ---" << std::endl;
        if (testFunc()) {
            passed++;
            std::cout << "[OK] " << name << std::endl << std::endl;
        } else {
            failed++;
            std::cout << "[FAILED] " << name << std::endl << std::endl;
        }
    };

    runTest("Create Room", test_create_room);
    runTest("Create Room with Duplicate Player", test_create_room_duplicate_player);
    runTest("Get Room", test_get_room);
    runTest("Delete Room", test_delete_room);
    runTest("Submit Event", test_submit_event);
    runTest("Multiple Rooms Concurrent", test_multiple_rooms);
    runTest("Statistics", test_statistics);

    std::cout << "=== Summary ===" << std::endl;
    std::cout << "Passed: " << passed << std::endl;
    std::cout << "Failed: " << failed << std::endl;

    return failed == 0 ? 0 : 1;
}