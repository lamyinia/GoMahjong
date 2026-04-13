#include "domain/game/engine/mahjong/timer/turn_manager.h"
#include "infrastructure/log/logger.hpp"

#include <algorithm>

namespace domain::game::engine::mahjong::timer {

    TurnManager::TurnManager(infra::util::TimingWheel* wheel, int totalTime)
        : wheel_(wheel) {
        for (int i = 0; i < TurnManager::PlayerCount; ++i) {
            tickers_[i] = std::make_unique<PlayerTicker>(i, totalTime, wheel);
        }
    }

    TurnManager::~TurnManager() = default;

    void TurnManager::enterDrawPhase(int seatIndex) {
        if (seatIndex < 0 || seatIndex >= TurnManager::PlayerCount) {
            LOG_ERROR("invalid seat index: {}", seatIndex);
            return;
        }

        stopAllTickers();
        turnPointer_ = seatIndex;
        phase_ = TurnPhase::DrawTile;

        LOG_DEBUG("enterDrawPhase seat={}, phase=DrawTile", seatIndex);
    }

    bool TurnManager::enterMainActionPhase(int seatIndex, int roundCompensation) {
        if (seatIndex < 0 || seatIndex >= TurnManager::PlayerCount) {
            LOG_ERROR("invalid seat index: {}", seatIndex);
            return false;
        }

        stopAllTickers();
        turnPointer_ = seatIndex;
        phase_ = TurnPhase::MainAction;

        // 分配时间 = 玩家剩余时间 + 补偿，上限 maxRoundTime
        auto* ticker = tickers_[seatIndex].get();
        int allocatedTime = ticker->getAvailable() + roundCompensation;
        allocatedTime = std::min(allocatedTime, DefaultMaxRoundTime);
        ticker->setAvailable(allocatedTime);

        if (!ticker->start(allocatedTime, [this, seatIndex]() {
                onTickerTimeout(seatIndex);
            })) {
            LOG_ERROR("failed to start ticker for seat {}", seatIndex);
            return false;
        }

        LOG_DEBUG("enterMainActionPhase seat={}, allocated={}s, phase=MainAction", seatIndex, allocatedTime);
        return true;
    }

    bool TurnManager::enterReactionPhase(const std::vector<int>& eligibleSeats, int timeLimitSec) {
        stopAllTickers();
        phase_ = TurnPhase::WaitReaction;

        // 给每个可反应玩家启动计时器
        for (int seat : eligibleSeats) {
            if (seat < 0 || seat >= TurnManager::PlayerCount) {
                LOG_WARN("invalid seat {} in eligibleSeats, skipping", seat);
                continue;
            }
            auto* ticker = tickers_[seat].get();
            if (!ticker->start(timeLimitSec, [this, seat]() {
                    onTickerTimeout(seat);
                })) {
                LOG_WARN("failed to start ticker for seat {} in reaction phase", seat);
            }
        }

        LOG_DEBUG("enterReactionPhase seats={}, timeLimit={}s, phase=WaitReaction",
                  eligibleSeats.size(), timeLimitSec);
        return true;
    }

    void TurnManager::enterResolvePhase() {
        stopAllTickers();
        phase_ = TurnPhase::ResolveReaction;
        LOG_DEBUG("enterResolvePhase, phase=ResolveReaction");
    }

    int TurnManager::nextTurn() {
        turnPointer_ = (turnPointer_ + 1) % TurnManager::PlayerCount;
        return turnPointer_;
    }

    PlayerTicker* TurnManager::getPlayerTicker(int seatIndex) {
        if (seatIndex < 0 || seatIndex >= TurnManager::PlayerCount) {
            return nullptr;
        }
        return tickers_[seatIndex].get();
    }

    std::array<TickerState, TurnManager::PlayerCount> TurnManager::getAllPlayerTimerStates() const {
        std::array<TickerState, TurnManager::PlayerCount> states;
        for (int i = 0; i < TurnManager::PlayerCount; ++i) {
            states[i] = tickers_[i]->getState();
        }
        return states;
    }

    void TurnManager::setTimingWheel(infra::util::TimingWheel* wheel) {
        wheel_ = wheel;
        for (auto& ticker : tickers_) {
            ticker->setTimingWheel(wheel);
        }
    }

    void TurnManager::stopAllTickers() {
        for (auto& ticker : tickers_) {
            if (ticker->getState() == TickerState::Running) {
                ticker->stop();
            }
        }
    }

    void TurnManager::setRoomId(const std::string& roomId) {
        roomId_ = roomId;
    }

    void TurnManager::setPlayerIds(const std::vector<std::string>& playerIds) {
        playerIds_ = playerIds;
    }

    void TurnManager::setTimeoutEventCallback(TimeoutEventCallback cb) {
        timeoutEventCallback_ = std::move(cb);
    }

    void TurnManager::onTickerTimeout(int seatIndex) {
        // 只投递 PlayerTimeout 游戏事件，不修改 ticker 状态
        // ticker 状态由 RoomActor 线程在处理事件时通过 applyTimeout 更新，保证线程安全
        if (timeoutEventCallback_) {
            std::string playerId = (seatIndex < static_cast<int>(playerIds_.size())) ? playerIds_[seatIndex] : "";
            auto event = event::GameEvent::playerTimeout(playerId, seatIndex);
            timeoutEventCallback_(roomId_, event);
        }

        LOG_DEBUG("[TurnManager] seat {} timer expired, submitted PlayerTimeout event for room {}", seatIndex, roomId_);
    }

    bool TurnManager::applyTimeout(int seatIndex) {
        if (seatIndex < 0 || seatIndex >= TurnManager::PlayerCount) {
            return false;
        }
        auto* ticker = tickers_[seatIndex].get();
        // 只有 Running 状态才应用超时（玩家可能已操作，ticker 已 Stopped）
        if (ticker->getState() != TickerState::Running) {
            LOG_DEBUG("[TurnManager] seat {} ticker not Running, skip applyTimeout", seatIndex);
            return false;
        }
        ticker->onTimerExpired();
        LOG_DEBUG("[TurnManager] seat {} applied timeout", seatIndex);
        return true;
    }

} // namespace domain::game::engine::mahjong::timer
