#include "domain/game/engine/mahjong/timer/turn_manager.h"
#include "infrastructure/log/logger.hpp"

#include <algorithm>

namespace domain::game::mahjong::timer {

    TurnManager::TurnManager(infra::util::TimingWheel* wheel,
                             int playerCount,
                             int totalTime,
                             int compensation,
                             int maxRoundTime,
                             int reactCompensation)
        : playerCount_(playerCount)
        , compensation_(compensation)
        , maxRoundTime_(maxRoundTime)
        , reactCompensation_(reactCompensation)
        , wheel_(wheel) {
        tickers_.reserve(playerCount_);
        for (int i = 0; i < playerCount_; ++i) {
            tickers_.push_back(std::make_unique<PlayerTicker>(i, totalTime, wheel));
        }
    }

    TurnManager::~TurnManager() = default;

    void TurnManager::enterDrawPhase(int seatIndex) {
        if (seatIndex < 0 || seatIndex >= playerCount_) {
            LOG_ERROR("invalid seat index: {}", seatIndex);
            return;
        }

        stopAllTickers();
        turnPointer_ = seatIndex;
        phase_ = TurnPhase::DrawTile;

        LOG_DEBUG("enterDrawPhase seat={}, phase=DrawTile", seatIndex);
    }

    bool TurnManager::enterMainActionPhase(int seatIndex, int roundCompensation) {
        if (seatIndex < 0 || seatIndex >= playerCount_) {
            LOG_ERROR("invalid seat index: {}", seatIndex);
            return false;
        }

        stopAllTickers();
        turnPointer_ = seatIndex;
        phase_ = TurnPhase::MainAction;

        int comp = (roundCompensation > 0) ? roundCompensation : compensation_;
        auto* ticker = tickers_[seatIndex].get();
        int allocatedTime = ticker->getAvailable() + comp;
        allocatedTime = std::min(allocatedTime, maxRoundTime_);
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

        int comp = (timeLimitSec > 0) ? timeLimitSec : reactCompensation_;
        for (int seat : eligibleSeats) {
            if (seat < 0 || seat >= playerCount_) {
                LOG_WARN("invalid seat {} in eligibleSeats, skipping", seat);
                continue;
            }
            auto* ticker = tickers_[seat].get();
            int allocatedTime = ticker->getAvailable() + comp;
            ticker->setAvailable(allocatedTime);
            if (!ticker->start(allocatedTime, [this, seat]() {
                    onTickerTimeout(seat);
                })) {
                LOG_WARN("failed to start ticker for seat {} in reaction phase", seat);
            }
        }

        LOG_DEBUG("enterReactionPhase seats={}, comp={}s, phase=WaitReaction", eligibleSeats.size(), comp);
        return true;
    }

    void TurnManager::enterResolvePhase() {
        stopAllTickers();
        phase_ = TurnPhase::ResolveReaction;
        LOG_DEBUG("enterResolvePhase, phase=ResolveReaction");
    }

    int TurnManager::nextTurn() {
        turnPointer_ = (turnPointer_ + 1) % playerCount_;
        return turnPointer_;
    }

    PlayerTicker* TurnManager::getPlayerTicker(int seatIndex) {
        if (seatIndex < 0 || seatIndex >= playerCount_) {
            return nullptr;
        }
        return tickers_[seatIndex].get();
    }

    std::vector<TickerState> TurnManager::getAllPlayerTimerStates() const {
        std::vector<TickerState> states(playerCount_);
        for (int i = 0; i < playerCount_; ++i) {
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
        if (timeoutEventCallback_) {
            std::string playerId = (seatIndex < static_cast<int>(playerIds_.size())) ? playerIds_[seatIndex] : "";
            auto event = event::GameEvent::playerTimeout(playerId, seatIndex);
            timeoutEventCallback_(roomId_, event);
        }

        LOG_DEBUG("seat {} timer expired, submitted PlayerTimeout event for room {}", seatIndex, roomId_);
    }

    bool TurnManager::applyTimeout(int seatIndex) {
        if (seatIndex < 0 || seatIndex >= playerCount_) {
            return false;
        }
        auto* ticker = tickers_[seatIndex].get();
        // 只有 Running 状态才应用超时（玩家可能已操作，ticker 已 Stopped）
        if (ticker->getState() != TickerState::Running) {
            LOG_DEBUG("seat {} ticker not Running, skip applyTimeout", seatIndex);
            return false;
        }
        ticker->onTimerExpired();
        LOG_DEBUG("seat {} applied timeout", seatIndex);
        return true;
    }

    void TurnManager::stopTickerForSeat(int seatIndex) {
        if (seatIndex < 0 || seatIndex >= playerCount_) {
            LOG_WARN("stopTickerForSeat: invalid seat {}", seatIndex);
            return;
        }
        auto* ticker = tickers_[seatIndex].get();
        if (ticker && ticker->getState() == TickerState::Running) {
            ticker->stop();
        }
    }

} // namespace domain::game::engine::mahjong::timer
