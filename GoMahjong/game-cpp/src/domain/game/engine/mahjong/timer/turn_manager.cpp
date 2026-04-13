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

        if (!ticker->start(allocatedTime)) {
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
            if (!ticker->start(timeLimitSec)) {
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

} // namespace domain::game::engine::mahjong::timer
