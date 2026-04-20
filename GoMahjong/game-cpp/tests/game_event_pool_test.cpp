#include "domain/game/event/game_event_pool.hpp"

#include <iostream>
#include <cassert>
#include <chrono>
#include <vector>

using namespace domain::game::event;

void test_basic_acquisition() {
    std::cout << "--- Test: Basic Acquisition ---\n";

    GameEventPool pool(64);

    // Test template-based acquisition
    {
        auto event = pool.acquire<PlayTileEvent>("player1", Tile{TileType::Wan1, 0});
        assert(event);
        assert(event->type == EventType::PlayTile);
        
        auto& data = std::get<PlayTileEvent>(event->data);
        assert(data.playerId == "player1");
        assert(data.tile.type == TileType::Wan1);
    }
    
    assert(pool.used() == 0);  // Auto-released
    
    std::cout << "Test passed!\n\n";
}

void test_convenience_methods() {
    std::cout << "--- Test: Convenience Methods ---\n";

    GameEventPool pool(64);

    // Test all convenience methods
    {
        auto e1 = pool.playTile("p1", Tile{TileType::Wan2, 1});
        assert(e1->type == EventType::PlayTile);
        
        auto e2 = pool.drawTile("p2", Tile{TileType::Tiao3, 2});
        assert(e2->type == EventType::DrawTile);
        
        auto e3 = pool.chi("p3", Tile{TileType::Tong4, 0}, TileType::Tong4);
        assert(e3->type == EventType::Chi);
        
        auto e4 = pool.pon("p4", Tile{TileType::FengDong, 3});
        assert(e4->type == EventType::Pon);
        
        auto e5 = pool.kan("p5", Tile{TileType::SanYuanZhong, 0}, true, false);
        assert(e5->type == EventType::Kan);
        
        auto e6 = pool.ron("p6", Tile{TileType::Wan5, 2}, "p1");
        assert(e6->type == EventType::Ron);
        
        auto e7 = pool.tsumo("p7", Tile{TileType::Tiao7, 1});
        assert(e7->type == EventType::Tsumo);
        
        auto e8 = pool.draw(true);
        assert(e8->type == EventType::Draw);
        
        auto e9 = pool.playerTimeout("p8", 2);
        assert(e9->type == EventType::PlayerTimeout);
        
        auto e10 = pool.turnStart("p1", 30);
        assert(e10->type == EventType::TurnStart);
        
        auto e11 = pool.turnEnd("p1");
        assert(e11->type == EventType::TurnEnd);
        
        auto e12 = pool.roundStart(1, "p1");
        assert(e12->type == EventType::RoundStart);
        
        auto e13 = pool.roundEnd(1);
        assert(e13->type == EventType::RoundEnd);
        
        auto e14 = pool.gameStart("room123");
        assert(e14->type == EventType::GameStart);
        
        auto e15 = pool.gameEnd("room123");
        assert(e15->type == EventType::GameEnd);
    }
    
    assert(pool.used() == 0);
    
    std::cout << "Test passed!\n\n";
}

void test_pool_reuse() {
    std::cout << "--- Test: Pool Reuse ---\n";

    GameEventPool pool(4);

    void* ptr1 = nullptr;
    void* ptr2 = nullptr;
    
    // First acquisition
    {
        auto e1 = pool.playTile("p1", Tile{TileType::Wan1, 0});
        auto e2 = pool.drawTile("p2", Tile{TileType::Wan2, 1});
        ptr1 = e1.get();
        ptr2 = e2.get();
    }
    
    // Re-acquire - should reuse memory
    {
        auto e1 = pool.playTile("p3", Tile{TileType::Wan3, 2});
        auto e2 = pool.drawTile("p4", Tile{TileType::Wan4, 3});
        
        // Memory should be reused (same addresses)
        assert(e1.get() == ptr1 || e1.get() == ptr2);
        assert(e2.get() == ptr1 || e2.get() == ptr2);
        
        // But data should be correct
        auto& data = std::get<PlayTileEvent>(e1->data);
        assert(data.playerId == "p3");
    }
    
    std::cout << "Test passed!\n\n";
}

void test_pool_expansion() {
    std::cout << "--- Test: Pool Expansion ---\n";

    GameEventPool pool(4);  // Small initial capacity
    
    std::vector<GameEventPool::PooledEvent> events;
    
    // Acquire more than initial capacity
    for (int i = 0; i < 20; ++i) {
        auto e = pool.playTile("p" + std::to_string(i), Tile{TileType::Wan1, 0});
        assert(e);
        events.push_back(std::move(e));
    }
    
    assert(pool.used() == 20);
    
    // Release all
    events.clear();
    
    assert(pool.used() == 0);
    
    std::cout << "Test passed!\n\n";
}

void test_performance() {
    std::cout << "--- Test: Performance Comparison ---\n";

    constexpr int ITERATIONS = 100000;
    
    // Test with new/delete
    {
        auto start = std::chrono::high_resolution_clock::now();
        
        for (int i = 0; i < ITERATIONS; ++i) {
            auto* event = new GameEvent(GameEvent::playTile("player", Tile{TileType::Wan1, 0}));
            delete event;
        }
        
        auto end = std::chrono::high_resolution_clock::now();
        auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);
        std::cout << "new/delete: " << duration.count() << " us\n";
    }
    
    // Test with pool
    {
        GameEventPool pool(1024);
        
        auto start = std::chrono::high_resolution_clock::now();
        
        for (int i = 0; i < ITERATIONS; ++i) {
            auto event = pool.playTile("player", Tile{TileType::Wan1, 0});
            event.reset();
        }
        
        auto end = std::chrono::high_resolution_clock::now();
        auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);
        std::cout << "GameEventPool: " << duration.count() << " us\n";
    }
    
    std::cout << "Test passed!\n\n";
}

void test_game_simulation() {
    std::cout << "--- Test: Game Simulation ---\n";

    GameEventPool pool(1024);
    
    // Simulate a typical game round
    {
        // Round start
        auto roundStart = pool.roundStart(1, "player1");
        
        // Turn start
        auto turnStart = pool.turnStart("player1", 30);
        
        // Draw tile
        auto draw = pool.drawTile("player1", Tile{TileType::Wan5, 2});
        
        // Play tile
        auto play = pool.playTile("player1", Tile{TileType::Wan3, 1});
        
        // Turn end
        auto turnEnd = pool.turnEnd("player1");
        
        // ... more events
        
        // Round end
        auto roundEnd = pool.roundEnd(1);
        
        // All events auto-released when out of scope
    }
    
    assert(pool.used() == 0);
    
    std::cout << "Test passed!\n\n";
}

int main() {
    std::cout << "=== GameEventPool Tests ===\n\n";

    test_basic_acquisition();
    test_convenience_methods();
    test_pool_reuse();
    test_pool_expansion();
    test_performance();
    test_game_simulation();

    std::cout << "=== All Tests Passed! ===\n";
    return 0;
}
