#include "domain/game/engine/mahjong/riichi_mahjong4p_engine.h"

#include "infrastructure/log/logger.hpp"
#include "domain/game/event/game_event.h"

namespace domain::game::engine {
    using EventType = event::EventType;

    // === 事件处理宏 ===
#define HANDLE_EVENT(type, eventType, handler) \
    case EventType::type: \
        if (const auto* e = std::get_if<eventType>(&event.data); e != nullptr) { \
            handler(*e); \
        } else { \
            LOG_WARN("[RiichiMahjong4PEngine] event.type is " #type " but variant data is not " #eventType); \
        } \
        break

    RiichiMahjong4PEngine::RiichiMahjong4PEngine() = default;

    void RiichiMahjong4PEngine::handleEvent(const event::GameEvent &event) {
        LOG_DEBUG("[RiichiMahjong4PEngine] handle event type: {}", static_cast<int>(event.type));

        switch (event.type) {
            // 出牌相关
            HANDLE_EVENT(PlayTile, event::PlayTileEvent, handlePlayTile);
            HANDLE_EVENT(DrawTile, event::DrawTileEvent, handleDrawTile);

            // 副露
            HANDLE_EVENT(Chi, event::ChiEvent, handleChi);
            HANDLE_EVENT(Pon, event::PonEvent, handlePon);
            HANDLE_EVENT(Kan, event::KanEvent, handleKan);

            // 胡牌
            HANDLE_EVENT(Ron, event::RonEvent, handleRon);
            HANDLE_EVENT(Tsumo, event::TsumoEvent, handleTsumo);
            HANDLE_EVENT(Draw, event::DrawEvent, handleDraw);

            default:
                LOG_WARN("[RiichiMahjong4PEngine] unhandled event type: {}", static_cast<int>(event.type));
                break;
        }
    }

    // === 事件处理函数实现 ===

    void RiichiMahjong4PEngine::handlePlayTile(const event::PlayTileEvent& e) {
        (void)e;
        // TODO: 实现出牌逻辑
    }

    void RiichiMahjong4PEngine::handleDrawTile(const event::DrawTileEvent& e) {
        (void)e;
        // TODO: 实现摸牌逻辑
    }

    void RiichiMahjong4PEngine::handleChi(const event::ChiEvent& e) {
        (void)e;
        // TODO: 实现吃牌逻辑
    }

    void RiichiMahjong4PEngine::handlePon(const event::PonEvent& e) {
        (void)e;
        // TODO: 实现碰牌逻辑
    }

    void RiichiMahjong4PEngine::handleKan(const event::KanEvent& e) {
        (void)e;
        // TODO: 实现杠牌逻辑
    }

    void RiichiMahjong4PEngine::handleRon(const event::RonEvent& e) {
        (void)e;
        // TODO: 实现荣胡逻辑
    }

    void RiichiMahjong4PEngine::handleTsumo(const event::TsumoEvent& e) {
        (void)e;
        // TODO: 实现自摸逻辑
    }

    void RiichiMahjong4PEngine::handleDraw(const event::DrawEvent& e) {
        (void)e;
        // TODO: 实现流局逻辑
    }



    void RiichiMahjong4PEngine::onPlayerJoin(const std::string &userId) {
        players_.insert(userId);
        LOG_INFO("[RiichiMahjong4PEngine] player {} joined, total: {}", userId, players_.size());
    }

    void RiichiMahjong4PEngine::onPlayerLeave(const std::string &userId) {
        players_.erase(userId);
        LOG_INFO("[RiichiMahjong4PEngine] player {} left, total: {}", userId, players_.size());
    }

    bool RiichiMahjong4PEngine::hasPlayer(const std::string &userId) const {
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
        return {};
    }

    std::string RiichiMahjong4PEngine::getPlayerState(const std::string &userId) const {
        (void)userId;
        return {};
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
