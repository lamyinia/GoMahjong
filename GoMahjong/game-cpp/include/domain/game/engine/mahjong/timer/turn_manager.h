#pragma once

#include "domain/game/engine/mahjong/timer/player_ticker.h"
#include "domain/game/event/mahjong_game_event.h"

#include <array>
#include <cstdint>
#include <functional>
#include <memory>
#include <string>
#include <vector>

namespace domain::game::mahjong::timer {

    enum class TurnPhase {
        Idle,              // 等待游戏开始
        DrawTile,          // 摸牌阶段（自动，无需计时）
        MainAction,        // 出牌/立直/杠/自摸 选择（当前玩家计时）
        WaitReaction,      // 等待吃碰杠荣和反应（多个玩家各自计时）
        ResolveReaction    // 反应优先级裁决（自动，无需计时）
    };

    // 管理当前出牌座位、回合状态、4 个玩家的计时器
    // 超时回调直接捕获 seat，投递 PlayerTimeout 游戏事件到 RoomActor
    class TurnManager {
    public:
        using TimeoutEventCallback = std::function<void(const std::string& roomId, const event::GameEvent& event)>;

        static constexpr int DefaultPlayerCount = 4;
        static constexpr int DefaultTotalTime = 300;    // 玩家总时间（秒）
        static constexpr int DefaultCompensation = 5;   // 每回合补偿（秒）
        static constexpr int DefaultMaxRoundTime = 30;  // 单回合最大时间（秒）
        static constexpr int DefaultReactCompensation = 5; // 反应补偿时间（秒）

        explicit TurnManager(infra::util::TimingWheel* wheel = nullptr,
                             int playerCount = DefaultPlayerCount,
                             int totalTime = DefaultTotalTime,
                             int compensation = DefaultCompensation,
                             int maxRoundTime = DefaultMaxRoundTime,
                             int reactCompensation = DefaultReactCompensation);
        ~TurnManager();

        // 进入摸牌阶段（自动，无需计时）
        void enterDrawPhase(int seatIndex);

        // 进入出牌/立直/杠/自摸 选择阶段（当前玩家计时）
        bool enterMainActionPhase(int seatIndex, int roundCompensation = 0);

        // 进入等待反应阶段（多个可反应玩家各自计时）
        // eligibleSeats: 可反应的座位列表
        // timeLimitSec: 每个玩家的反应时间（秒）
        bool enterReactionPhase(const std::vector<int>& eligibleSeats, int timeLimitSec = 0);

        // 进入裁决阶段（自动，无需计时）
        void enterResolvePhase();

        int nextTurn();

        [[nodiscard]] int getCurrentPlayer() const { return turnPointer_; }

        [[nodiscard]] TurnPhase getPhase() const { return phase_; }

        PlayerTicker* getPlayerTicker(int seatIndex);

        [[nodiscard]] std::vector<TickerState> getAllPlayerTimerStates() const;

        void setTimingWheel(infra::util::TimingWheel* wheel);
        [[nodiscard]] infra::util::TimingWheel* getTimingWheel() const { return wheel_; }
        void setRoomId(const std::string& roomId);
        void setPlayerIds(const std::vector<std::string>& playerIds);
        void setTimeoutEventCallback(TimeoutEventCallback cb);

        // 在 RoomActor 线程中调用：应用超时状态到指定 seat 的 ticker
        // 返回 true 表示成功应用（ticker 仍为 Running），false 表示已非 Running（玩家已操作）
        bool applyTimeout(int seatIndex);

        // 停止指定座位的 ticker（玩家已操作）
        void stopTickerForSeat(int seatIndex);

    private:
        void stopAllTickers();
        void onTickerTimeout(int seatIndex);  // 定时器到期回调（TimerThread 线程）

        int playerCount_;
        int compensation_;
        int maxRoundTime_;
        int reactCompensation_;

        int turnPointer_{0};
        TurnPhase phase_{TurnPhase::Idle};
        std::vector<std::unique_ptr<PlayerTicker>> tickers_;
        infra::util::TimingWheel* wheel_;

        std::string roomId_;
        std::vector<std::string> playerIds_;
        TimeoutEventCallback timeoutEventCallback_;
    };

} // namespace domain::game::engine::mahjong::timer
