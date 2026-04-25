#pragma once

#include "infrastructure/util/timing_wheel.h"

#include <chrono>
#include <cstdint>
#include <functional>
#include <string>

namespace domain::game::mahjong::timer {

    enum class TickerState {
        Idle,       // 空闲
        Running,    // 计时中
        Stopped,    // 已停止（玩家操作）
        Timeout     // 已超时
    };

    // 底层通过 TimingWheel 调度，支持：跨回合累计剩余时间、每回合补偿
    // 超时回调由上层 TurnManager 在 start 时注入，投递 PlayerTimeout 游戏事件
    class PlayerTicker {
    public:
        using TimeoutCallback = std::function<void()>;
        using StateChangeCallback = std::function<void(TickerState oldState, TickerState newState)>;
        using StopCallback = std::function<void()>;

        explicit PlayerTicker(int seatIndex, int totalAvailableTime, infra::util::TimingWheel* wheel = nullptr);

        // 启动计时，duration: 本次分配时间（秒），onTimeout: 定时器到期回调（在 TimerThread 调用）
        bool start(int duration, TimeoutCallback onTimeout);

        // 停止计时（玩家操作），返回已用时间（秒）
        bool stop();

        // 设置跨回合累计剩余时间
        void setAvailable(int available);

        [[nodiscard]] int getAvailable() const { return available_; }
        [[nodiscard]] TickerState getState() const { return state_; }
        [[nodiscard]] int getSeatIndex() const { return seatIndex_; }
        [[nodiscard]] infra::util::TimerHandle currentHandle() const { return currentHandle_; }

        // 回调设置
        void setOnStop(StopCallback cb);
        void setOnStateChange(StateChangeCallback cb);

        // 注入 TimingWheel（延迟注入）
        void setTimingWheel(infra::util::TimingWheel* wheel);

        // 默认配置
        static constexpr int DefaultMaxRoundTime = 30;  // 单回合最大时间（秒）
        static constexpr int DefaultCompensation = 5;    // 每回合补偿时间（秒）

    private:
        void transitionTo(TickerState newState);

    public:
        // 由 TurnManager::onTickerTimeout 调用，更新 ticker 状态为 Timeout
        void onTimerExpired();

        int seatIndex_;
        int available_;           // 跨回合累计剩余时间（秒）
        TickerState state_;

        infra::util::TimingWheel* wheel_;
        infra::util::TimerHandle currentHandle_;  // 当前定时器句柄

        std::chrono::steady_clock::time_point roundStartTime_;  // 本回合开始时间
        int currentDuration_{0};  // 本回合分配的时间（秒）

        StopCallback onStop_;
        StateChangeCallback onStateChange_;
    };

} // namespace domain::game::engine::mahjong::timer
