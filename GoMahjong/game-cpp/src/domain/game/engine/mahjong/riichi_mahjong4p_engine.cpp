//
// Created by lanyo on 2026/4/5.
//

#include "domain/game/engine/mahjong/riichi_mahjong4p_engine.h"
#include "infrastructure/log/logger.hpp"

namespace domain::game::engine {

    RiichiMahjong4PEngine::RiichiMahjong4PEngine() = default;

    void RiichiMahjong4PEngine::handleEvent(const event::GameEvent& event) {
        LOG_DEBUG("[RiichiMahjong4PEngine] handle event type: {}", static_cast<int>(event.type));
        // TODO: 实现具体的事件处理逻辑
    }

    void RiichiMahjong4PEngine::onPlayerJoin(const std::string& userId) {
        players_.insert(userId);
        LOG_INFO("[RiichiMahjong4PEngine] player {} joined, total: {}", userId, players_.size());
    }

    void RiichiMahjong4PEngine::onPlayerLeave(const std::string& userId) {
        players_.erase(userId);
        LOG_INFO("[RiichiMahjong4PEngine] player {} left, total: {}", userId, players_.size());
    }

    bool RiichiMahjong4PEngine::hasPlayer(const std::string& userId) const {
        return players_.contains(userId);
    }

    std::size_t RiichiMahjong4PEngine::playerCount() const {
        return players_.size();
    }

    GamePhase RiichiMahjong4PEngine::getPhase() const {
        return phase_;
    }

    bool RiichiMahjong4PEngine::isGameOver() const {
        return phase_ == GamePhase::GameOver;
    }

    bool RiichiMahjong4PEngine::canStart() const {

        return players_.size() == 4 && !started_;
    }

    std::string RiichiMahjong4PEngine::getGameState() const {

    }

    std::string RiichiMahjong4PEngine::getPlayerState(const std::string& userId) const {

    }

    void RiichiMahjong4PEngine::start() {
        if (canStart()) {
            started_ = true;
            phase_ = GamePhase::Playing;
            LOG_INFO("[RiichiMahjong4PEngine] game started with {} players", players_.size());
        } else {
            LOG_WARN("[RiichiMahjong4PEngine] cannot start: players={}, started={}", 
                     players_.size(), started_);
        }
    }

    void RiichiMahjong4PEngine::reset() {
        phase_ = GamePhase::Waiting;
        started_ = false;
        LOG_INFO("[RiichiMahjong4PEngine] game reset");
    }

    void RiichiMahjong4PEngine::destroy() {
        players_.clear();
        phase_ = GamePhase::GameOver;
        started_ = false;
        LOG_INFO("[RiichiMahjong4PEngine] game destroyed");
    }

} // namespace domain::game::engine
