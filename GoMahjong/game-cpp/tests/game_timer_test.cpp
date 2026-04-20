#include "infrastructure/util/timing_wheel.h"
#include "infrastructure/util/timer_thread.h"
#include "domain/game/engine/mahjong/timer/player_ticker.h"
#include "domain/game/engine/mahjong/timer/turn_manager.h"
#include "infrastructure/config/config.hpp"
#include "infrastructure/log/logger.hpp"

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdio>
#include <mutex>
#include <thread>

using namespace infra::util;
using namespace domain::game::engine::mahjong::timer;

static int tests_passed = 0;
static int tests_failed = 0;

#define ASSERT_TRUE(expr) \
    do { if (!(expr)) { printf("  FAIL: %s (line %d)\n", #expr, __LINE__); tests_failed++; return; } } while(0)

#define ASSERT_EQ(a, b) \
    do { if ((a) != (b)) { printf("  FAIL: %s == %s (line %d)\n", #a, #b, __LINE__); tests_failed++; return; } } while(0)

#define TEST(name) \
    static void test_##name(); \
    struct TestRegistrar_##name { TestRegistrar_##name() { test_funcs.push_back({#name, test_##name}); } } reg_##name; \
    static void test_##name()

struct TestFunc { const char* name; void (*fn)(); };
static std::vector<TestFunc> test_funcs;

// === TimingWheel 基础测试 ===

TEST(timing_wheel_schedule_and_fire) {
    TimingWheel wheel(50, 512);

    std::atomic<int> expireCount{0};
    auto handle = wheel.schedule(50, [&]() {
        expireCount.fetch_add(1);
    });

    // 手动 tick 到期
    for (int i = 0; i < 4; ++i) {
        wheel.tick();
    }

    ASSERT_EQ(expireCount.load(), 1);  // callback 触发 1 次
}

TEST(timing_wheel_cancel) {
    TimingWheel wheel(50, 512);

    std::atomic<int> expireCount{0};
    auto handle = wheel.schedule(100, [&]() {
        expireCount.fetch_add(1);
    });

    wheel.cancel(handle);

    for (int i = 0; i < 4; ++i) {
        wheel.tick();
    }

    ASSERT_EQ(expireCount.load(), 0);
}

TEST(timing_wheel_multiple_timers) {
    TimingWheel wheel(50, 512);

    std::atomic<int> expireCount{0};
    wheel.schedule(50, [&]() { expireCount.fetch_add(1); });
    wheel.schedule(100, [&]() { expireCount.fetch_add(1); });
    wheel.schedule(150, [&]() { expireCount.fetch_add(1); });

    // tick 4 次：50ms, 100ms, 150ms
    for (int i = 0; i < 4; ++i) {
        wheel.tick();
    }

    ASSERT_EQ(expireCount.load(), 3);
}

TEST(timing_wheel_long_delay_rounds) {
    TimingWheel wheel(50, 8);  // 8 slots × 50ms = 400ms 一轮

    std::atomic<int> expireCount{0};
    wheel.schedule(500, [&]() { expireCount.fetch_add(1); });

    // tick 11 次 (超过 10 ticks)
    for (int i = 0; i < 12; ++i) {
        wheel.tick();
    }

    ASSERT_EQ(expireCount.load(), 1);
}

// === TimerThread 集成测试 ===

TEST(timer_thread_basic) {
    TimingWheel wheel(50, 512);
    TimerThread thread(wheel, 50);

    std::atomic<int> expireCount{0};
    std::mutex mtx;
    std::condition_variable cv;
    bool callbackExecuted = false;

    thread.start();

    wheel.schedule(200, [&]() {
        expireCount.fetch_add(1);
        std::lock_guard lock(mtx);
        callbackExecuted = true;
        cv.notify_one();
    });

    // 等待回调执行（最多 2 秒）
    std::unique_lock lock(mtx);
    bool ok = cv.wait_for(lock, std::chrono::seconds(2), [&] { return callbackExecuted; });
    ASSERT_TRUE(ok);

    thread.stop();
    ASSERT_EQ(expireCount.load(), 1);
}

// === PlayerTicker 测试 ===

TEST(player_ticker_start_and_stop) {
    TimingWheel wheel(50, 512);
    TimerThread thread(wheel, 50);

    std::atomic<int> expireCount{0};

    thread.start();

    PlayerTicker ticker(0, 300, &wheel);

    // 启动 10 秒计时
    ASSERT_TRUE(ticker.start(10, [&]() { expireCount.fetch_add(1); }));
    ASSERT_EQ(ticker.getState(), TickerState::Running);

    // 等待 500ms 后停止
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    ASSERT_TRUE(ticker.stop());
    ASSERT_EQ(ticker.getState(), TickerState::Stopped);

    // 剩余时间应该被扣减
    ASSERT_TRUE(ticker.getAvailable() < 300);

    thread.stop();
    ASSERT_EQ(expireCount.load(), 0);  // 未超时
}

TEST(player_ticker_timeout) {
    TimingWheel wheel(50, 512);
    TimerThread thread(wheel, 50);

    std::mutex mtx;
    std::condition_variable cv;
    bool timeoutFired = false;

    thread.start();

    PlayerTicker ticker(0, 300, &wheel);

    // 启动 1 秒计时，传入超时回调
    ASSERT_TRUE(ticker.start(1, [&]() {
        // 模拟 TurnManager::onTickerTimeout：只投递事件，不修改 ticker 状态
        // ticker 状态由 RoomActor 线程通过 applyTimeout 更新
        std::lock_guard lock(mtx);
        timeoutFired = true;
        cv.notify_one();
    }));

    std::unique_lock lock(mtx);
    bool ok = cv.wait_for(lock, std::chrono::seconds(3), [&] { return timeoutFired; });
    ASSERT_TRUE(ok);

    // callback 不改 ticker 状态，ticker 仍为 Running
    // 模拟 RoomActor 线程处理 PlayerTimeout 事件时调用 applyTimeout
    ticker.onTimerExpired();
    ASSERT_EQ(ticker.getState(), TickerState::Timeout);

    thread.stop();
}

// === TurnManager 测试 ===

TEST(turn_manager_enter_draw_phase) {
    TimingWheel wheel(50, 512);
    TurnManager tm(&wheel);

    tm.enterDrawPhase(2);
    ASSERT_EQ(tm.getPhase(), TurnPhase::DrawTile);
    ASSERT_EQ(tm.getCurrentPlayer(), 2);
}

TEST(turn_manager_enter_main_action_phase) {
    TimingWheel wheel(50, 512);
    TimerThread thread(wheel, 50);

    thread.start();

    TurnManager tm(&wheel);

    ASSERT_TRUE(tm.enterMainActionPhase(0, 5));
    ASSERT_EQ(tm.getPhase(), TurnPhase::MainAction);
    ASSERT_EQ(tm.getCurrentPlayer(), 0);

    auto* ticker = tm.getPlayerTicker(0);
    ASSERT_TRUE(ticker != nullptr);
    ASSERT_EQ(ticker->getState(), TickerState::Running);

    thread.stop();
}

TEST(turn_manager_enter_reaction_phase) {
    TimingWheel wheel(50, 512);
    TimerThread thread(wheel, 50);

    thread.start();

    TurnManager tm(&wheel);

    // 先进入出牌阶段
    ASSERT_TRUE(tm.enterMainActionPhase(0, 5));

    // 出牌后进入反应阶段，座位 1,2,3 可反应
    ASSERT_TRUE(tm.enterReactionPhase({1, 2, 3}, 5));
    ASSERT_EQ(tm.getPhase(), TurnPhase::WaitReaction);

    // 检查各座位计时器状态
    ASSERT_EQ(tm.getPlayerTicker(0)->getState(), TickerState::Stopped);  // 出牌者被 stopAllTickers 停止
    ASSERT_EQ(tm.getPlayerTicker(1)->getState(), TickerState::Running);
    ASSERT_EQ(tm.getPlayerTicker(2)->getState(), TickerState::Running);
    ASSERT_EQ(tm.getPlayerTicker(3)->getState(), TickerState::Running);

    thread.stop();
}

TEST(turn_manager_enter_resolve_phase) {
    TimingWheel wheel(50, 512);
    TurnManager tm(&wheel);

    tm.enterMainActionPhase(0, 5);
    tm.enterReactionPhase({1, 2}, 5);
    tm.enterResolvePhase();

    ASSERT_EQ(tm.getPhase(), TurnPhase::ResolveReaction);

    // 所有计时器应停止
    auto states = tm.getAllPlayerTimerStates();
    for (int i = 0; i < TurnManager::PlayerCount; ++i) {
        ASSERT_TRUE(states[i] != TickerState::Running);
    }
}

TEST(turn_manager_next_turn) {
    TimingWheel wheel(50, 512);
    TurnManager tm(&wheel);

    ASSERT_EQ(tm.getCurrentPlayer(), 0);
    ASSERT_EQ(tm.nextTurn(), 1);
    ASSERT_EQ(tm.nextTurn(), 2);
    ASSERT_EQ(tm.nextTurn(), 3);
    ASSERT_EQ(tm.nextTurn(), 0);  // 循环
}

TEST(turn_manager_full_flow) {
    TimingWheel wheel(50, 512);
    TimerThread thread(wheel, 50);

    std::atomic<int> timeoutEventCount{0};

    thread.start();

    TurnManager tm(&wheel);
    tm.setRoomId("test_room");
    tm.setPlayerIds({"p0", "p1", "p2", "p3"});
    tm.setTimeoutEventCallback([&](const std::string& roomId, const domain::game::event::GameEvent& event) {
        timeoutEventCount.fetch_add(1);
    });

    // 1. 摸牌
    tm.enterDrawPhase(0);
    ASSERT_EQ(tm.getPhase(), TurnPhase::DrawTile);

    // 2. 出牌
    ASSERT_TRUE(tm.enterMainActionPhase(0, 5));
    ASSERT_EQ(tm.getPhase(), TurnPhase::MainAction);

    // 3. 停止出牌者计时（模拟出牌）
    tm.getPlayerTicker(0)->stop();

    // 4. 等待反应
    ASSERT_TRUE(tm.enterReactionPhase({1, 2, 3}, 1));  // 1秒反应时间
    ASSERT_EQ(tm.getPhase(), TurnPhase::WaitReaction);

    // 5. 裁决
    tm.enterResolvePhase();
    ASSERT_EQ(tm.getPhase(), TurnPhase::ResolveReaction);

    thread.stop();
}

int main() {
    infra::log::init({"debug", true});

    printf("\n=== Game Timer Tests ===\n\n");

    for (const auto& t : test_funcs) {
        printf("  [RUN] %s\n", t.name);
        int fails_before = tests_failed;
        t.fn();
        if (tests_failed == fails_before) {
            tests_passed++;
        }
    }

    printf("\n  %d passed, %d failed\n", tests_passed, tests_failed);
    return tests_failed > 0 ? 1 : 0;
}
