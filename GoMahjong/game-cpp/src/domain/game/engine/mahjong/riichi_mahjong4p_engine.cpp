#include "domain/game/engine/mahjong/riichi_mahjong4p_engine.h"

#include "infrastructure/log/logger.hpp"
#include "domain/game/event/mahjong_game_event.h"
#include "generated/game_mahjong.pb.h"

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

            // 超时
            HANDLE_EVENT(PlayerTimeout, event::PlayerTimeoutEvent, handlePlayerTimeout);

            default:
                LOG_WARN("[RiichiMahjong4PEngine] unhandled event type: {}", static_cast<int>(event.type));
                break;
        }
    }

    // === 事件处理函数实现 ===

    void RiichiMahjong4PEngine::handlePlayTile(const event::PlayTileEvent& e) {
        (void)e;

        // TODO: 实现出牌逻辑

        // 测试：广播 GameStatePush
        if (context_) {
            gomahjong::game::GameStatePush push;
            push.set_room_id(0);
            push.set_current_turn(0);
            push.set_remaining_tiles(136);
            context_->broadcast("game.state", push);
        }
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
        phase_ = GamePhase::GameOver;
        if (context_) context_->notifyGameOver();
    }

    void RiichiMahjong4PEngine::handleTsumo(const event::TsumoEvent& e) {
        (void)e;
        // TODO: 实现自摸逻辑
        phase_ = GamePhase::GameOver;
        if (context_) context_->notifyGameOver();
    }

    void RiichiMahjong4PEngine::handleDraw(const event::DrawEvent& e) {
        (void)e;
        // TODO: 实现流局逻辑
        phase_ = GamePhase::GameOver;
        if (context_) context_->notifyGameOver();
    }

    void RiichiMahjong4PEngine::handlePlayerTimeout(const event::PlayerTimeoutEvent& e) {
        LOG_WARN("[RiichiMahjong4PEngine] player {} (seat {}) timeout", e.playerId, e.seatIndex);

        // 在 RoomActor 线程中安全地更新 ticker 状态
        if (turnManager_) {
            if (!turnManager_->applyTimeout(e.seatIndex)) {
                // ticker 已非 Running（玩家在超时前已操作），忽略此超时
                LOG_DEBUG("[RiichiMahjong4PEngine] seat {} timeout ignored, player already acted", e.seatIndex);
                return;
            }
        }

        // TODO: 实现超时逻辑（自动出牌/跳过操作）
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

    void RiichiMahjong4PEngine::setContext(EngineContext* context) {
        context_ = context;
    }

    void RiichiMahjong4PEngine::initTimerSystem(infra::util::TimingWheel* wheel) {
        if (!wheel) {
            LOG_ERROR("[RiichiMahjong4PEngine] cannot init timer system with null wheel");
            return;
        }

        turnManager_ = std::make_unique<mahjong::timer::TurnManager>(wheel);

        // 配置 TurnManager 的超时事件回调：通过 EngineContext 投递到 RoomActor 队列
        if (context_) {
            turnManager_->setRoomId(context_->roomId());
            turnManager_->setPlayerIds(context_->playerIds());
            turnManager_->setTimeoutEventCallback(
                [this](const std::string& roomId, const event::GameEvent& event) {
                    if (context_) {
                        context_->submitEvent(roomId, event);
                    }
                }
            );
        }

        LOG_INFO("[RiichiMahjong4PEngine] timer system initialized for room {}",
                 context_ ? context_->roomId() : "unknown");
    }

} // namespace domain::game::engine
