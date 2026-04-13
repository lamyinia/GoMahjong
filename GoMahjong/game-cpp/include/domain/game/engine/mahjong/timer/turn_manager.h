#pragma once

#include "domain/game/engine/mahjong/timer/player_ticker.h"

#include <array>
#include <cstdint>
#include <memory>
#include <string>
#include <vector>

namespace domain::game::engine::mahjong::timer {

    enum class TurnPhase {
        Idle,              // 等待游戏开始
        DrawTile,          // 摸牌阶段（自动，无需计时）
        MainAction,        // 出牌/立直/杠/自摸 选择（当前玩家计时）
        WaitReaction,      // 等待吃碰杠荣和反应（多个玩家各自计时）
        ResolveReaction    // 反应优先级裁决（自动，无需计时）
    };

    // 管理当前出牌座位、回合状态、4 个玩家的计时器
    class TurnManager {
    public:
        static constexpr int PlayerCount = 4;
        static constexpr int DefaultTotalTime = 300;    // 玩家总时间（秒）
        static constexpr int DefaultCompensation = 5;   // 每回合补偿（秒）
        static constexpr int DefaultMaxRoundTime = 30;  // 单回合最大时间（秒）

        explicit TurnManager(infra::util::TimingWheel* wheel = nullptr,
                             int totalTime = DefaultTotalTime);

        // 进入摸牌阶段（自动，无需计时）
        void enterDrawPhase(int seatIndex);

        // 进入出牌/立直/杠/自摸 选择阶段（当前玩家计时）
        bool enterMainActionPhase(int seatIndex, int roundCompensation = DefaultCompensation);

        // 进入等待反应阶段（多个可反应玩家各自计时）
        // eligibleSeats: 可反应的座位列表
        // timeLimitSec: 每个玩家的反应时间（秒）
        bool enterReactionPhase(const std::vector<int>& eligibleSeats, int timeLimitSec = 5);

        // 进入裁决阶段（自动，无需计时）
        void enterResolvePhase();

        int nextTurn();

        [[nodiscard]] int getCurrentPlayer() const { return turnPointer_; }

        [[nodiscard]] TurnPhase getPhase() const { return phase_; }

        PlayerTicker* getPlayerTicker(int seatIndex);

        std::array<TickerState, PlayerCount> getAllPlayerTimerStates() const;

        void setTimingWheel(infra::util::TimingWheel* wheel);

    private:
        void stopAllTickers();

        int turnPointer_{0};
        TurnPhase phase_{TurnPhase::Idle};
        std::array<std::unique_ptr<PlayerTicker>, PlayerCount> tickers_;
        infra::util::TimingWheel* wheel_;
    };

} // namespace domain::game::engine::mahjong::timer
