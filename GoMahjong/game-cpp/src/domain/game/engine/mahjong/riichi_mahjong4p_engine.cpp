#include "domain/game/engine/mahjong/riichi_mahjong4p_engine.h"

#include "infrastructure/log/logger.hpp"
#include "infrastructure/util/timing_wheel.h"
#include "infrastructure/config/config.hpp"
#include "domain/game/event/mahjong_game_event.h"
#include "generated/game_mahjong.pb.h"

#include <random>

namespace domain::game::engine {
    using EventType = event::EventType;
    using TickerState = mj::timer::TickerState;

#define HANDLE_EVENT(type, eventType, handler) \
    case EventType::type: \
        if (const auto* e = std::get_if<eventType>(&event.data); e != nullptr) { \
            handler(*e); \
        } else { \
            LOG_WARN(" event.type is " #type " but variant data is not " #eventType); \
        } \
        break

    RiichiMahjong4PEngine::RiichiMahjong4PEngine() = default;

    void RiichiMahjong4PEngine::handleEvent(const event::GameEvent &event) {
        LOG_DEBUG("handle event type: {}", static_cast<int>(event.type));

        switch (event.type) {
            // 局流程
            HANDLE_EVENT(RoundStart, event::RoundStartEvent, handleRoundStart);
            HANDLE_EVENT(RoundEnd, event::RoundEndEvent, handleRoundEnd);

            // 出牌相关
            HANDLE_EVENT(PlayTile, event::PlayTileEvent, handlePlayTile);
            HANDLE_EVENT(Riichi, event::RiichiEvent, handleRiichi);

            // 副露
            HANDLE_EVENT(Chi, event::ChiEvent, handleChi);
            HANDLE_EVENT(Pon, event::PonEvent, handlePon);
            HANDLE_EVENT(Kan, event::KanEvent, handleKan);

            // 胡牌
            HANDLE_EVENT(Ron, event::RonEvent, handleRon);
            HANDLE_EVENT(Tsumo, event::TsumoEvent, handleTsumo);

            // 反应
            HANDLE_EVENT(Skip, event::SkipEvent, handleSkip);
            HANDLE_EVENT(KyuushuKyuukai, event::KyuushuKyuukaiEvent, handleKyuushuKyuukai);
            HANDLE_EVENT(Snapshoot, event::SnapshootEvent, handleSnapshoot);

            // 超时
            HANDLE_EVENT(PlayerTimeout, event::PlayerTimeoutEvent, handlePlayerTimeout);

            default:
                LOG_WARN("unhandled event type: {}", static_cast<int>(event.type));
                break;
        }
    }

    void RiichiMahjong4PEngine::handleRoundStart(const event::RoundStartEvent& e) {
        (void)e;
        startRound();
    }

    void RiichiMahjong4PEngine::handleRoundEnd(const event::RoundEndEvent& e) {
        
    }

    void RiichiMahjong4PEngine::startRound() {
        LOG_DEBUG("round start: wind={}, round={}, dealer={}, honba={}, sticks={}",
                 static_cast<int>(situation_.round_wind), situation_.round_number,
                 situation_.dealer_index, situation_.honba, situation_.riichi_sticks);

        deck_manager_.initRound();
        deck_manager_.revealDoraIndicator();

        distributeCards();
        broadcastRoundStart();

        // 庄家摸第14张牌并推送
        auto dealer = situation_.dealer_index;
        auto [tile, ok] = deck_manager_.draw();
        if (!ok) {
            LOG_ERROR("dealer draw failed, wall empty");
            return;
        }
        auto* p = getPlayer(dealer);
        if (p) {
            p->drawTile(tile);
            pushDrawTile(dealer, tile);
        }

        // 庄家进入出牌阶段
        dropTurn(dealer, false);
    }

    void RiichiMahjong4PEngine::distributeCards() {
        // 重置所有玩家
        for (int i = 0; i < 4; ++i) {
            auto* p = getPlayer(i);
            if (p) p->resetForRound();
        }

        // 每人13张
        for (int r = 0; r < 13; ++r) {
            for (int i = 0; i < 4; ++i) {
                auto [tile, ok] = deck_manager_.deal();
                if (!ok) {
                    LOG_ERROR("deal failed at round {} seat {}", r, i);
                    return;
                }
                auto* p = getPlayer(i);
                if (p) p->addTile(tile);
            }
        }
    }

    void RiichiMahjong4PEngine::dropTurn(int seatIndex, bool needDraw) {
        if (needDraw) {
            auto [tile, ok] = deck_manager_.draw();
            if (!ok) {
                // 牌墙空，流局
                if (context_) {
                    auto ev = event::GameEvent::draw();
                    context_->submitEvent(context_->roomId(), ev);
                }
                return;
            }
            auto* p = getPlayer(seatIndex);
            if (p) {
                p->drawTile(tile);
                pushDrawTile(seatIndex, tile);
            }
        }
        if (turnManager_) {
            turnManager_->enterMainActionPhase(seatIndex);
        }
        // 计算可选操作并推送
        computeMainActions(seatIndex);
        pushOperations(seatIndex);
    }

    void RiichiMahjong4PEngine::handlePlayTile(const event::PlayTileEvent& e) {
        int seat = getSeatIndex(e.playerId);
        if (seat < 0) {
            LOG_WARN("playTile: unknown player {}", e.playerId);
            return;
        }

        auto* p = getPlayer(seat);
        if (!p) return;
        if (turnManager_) {
            turnManager_->stopTickerForSeat(seat);
        }

        // 出牌：从手牌移到弃牌堆
        if (!p->discardTile(e.tile)) {
            LOG_WARN("playTile: player {} cannot discard tile type={} id={}",
                     e.playerId, static_cast<int>(e.tile.type), e.tile.id);
            return;
        }

        // 记录最后出牌
        last_discard_.seat = seat;
        last_discard_.tile = e.tile;
        last_discard_.valid = true;

        // 广播出牌
        broadcastDiscardTile(seat, e.tile);

        // 进入反应阶段
        enterReactionPhase();
    }

    void RiichiMahjong4PEngine::handleRiichi(const event::RiichiEvent& e) {
        (void)e;
        // TODO: 实现立直逻辑
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

        if (context_) context_->notifyGameOver();
    }

    void RiichiMahjong4PEngine::handleTsumo(const event::TsumoEvent& e) {
        (void)e;
        // TODO: 实现自摸逻辑

        if (context_) context_->notifyGameOver();
    }

    void RiichiMahjong4PEngine::handleSkip(const event::SkipEvent& e) {
        LOG_DEBUG("player {} skip", e.playerId);

        int seat = getSeatIndex(e.playerId);
        if (seat < 0) {
            LOG_WARN("skip: unknown player {}", e.playerId);
            return;
        }

        // 停止该玩家的 ticker
        if (turnManager_) {
            turnManager_->stopTickerForSeat(seat);
        }

        // 记录跳过反应
        mj::PlayerOperation skipOp;
        skipOp.type = "SKIP";
        recordReaction(seat, skipOp);

        // 检查反应是否全部收集完毕
        if (isReactionComplete()) {
            if (turnManager_) {
                turnManager_->enterResolvePhase();
            }
            resolveReactions();
        }
    }

    void RiichiMahjong4PEngine::handleKyuushuKyuukai(const event::KyuushuKyuukaiEvent& e) {
        LOG_DEBUG("player {} declares kyuushu kyuukai", e.playerId);

        int seat = getSeatIndex(e.playerId);
        if (seat < 0) {
            LOG_WARN("kyuushu kyuukai: unknown player {}", e.playerId);
            return;
        }

        // TODO: 验证该玩家手牌是否满足九种九牌条件，满足则触发流局
    }

    void RiichiMahjong4PEngine::handleSnapshoot(const event::SnapshootEvent& e) {
        LOG_DEBUG("player {} snapshoot", e.playerId);

        int seat = getSeatIndex(e.playerId);
        if (seat < 0) {
            LOG_WARN("snapshoot: unknown player {}", e.playerId);
            return;
        }

        // TODO: 构建 GameStatePush protobuf 并通过 context_->send 推送给该玩家
        // gomahjong::game::GameStatePush push;
        // ... 填充快照数据 ...
        // context_->send(e.playerId, handler::route::kGameState, push);
    }

    void RiichiMahjong4PEngine::handlePlayerTimeout(const event::PlayerTimeoutEvent& e) {
        LOG_WARN("player {} (seat {}) timeout", e.playerId, e.seatIndex);

        // 在 RoomActor 线程中安全地更新 ticker 状态
        if (turnManager_) {
            if (!turnManager_->applyTimeout(e.seatIndex)) {
                // ticker 已非 Running（玩家在超时前已操作），忽略此超时
                LOG_DEBUG("seat {} timeout ignored, player already acted", e.seatIndex);
                return;
            }
        }

        // TODO: 实现超时逻辑（自动出牌/跳过操作）
    }

    void RiichiMahjong4PEngine::onPlayerJoin(const std::string &playerId) {
        players_.insert(playerId);
        LOG_DEBUG("player {} joined, total: {}", playerId, players_.size());
    }

    void RiichiMahjong4PEngine::onPlayerLeave(const std::string &playerId) {
        players_.erase(playerId);
        LOG_DEBUG("player {} left, total: {}", playerId, players_.size());
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

        return players_.size() == infra::config::Config::instance().server().game.riichi_mahjong_4p.player_count && !started_;
    }

    std::string RiichiMahjong4PEngine::getGameState() const {
        return {};
    }

    std::string RiichiMahjong4PEngine::getPlayerState(const std::string &playerId) const {
        (void)playerId;
        return {};
    }

    void RiichiMahjong4PEngine::start() {
        if (!canStart()) {
            LOG_WARN("cannot start: players={}, started={}",
                     players_.size(), started_);
            return;
        }

        started_ = true;
        phase_ = GamePhase::Playing;

        // 初始化座位映射和 PlayerImage
        int seat = 0;
        for (const auto& playerId : players_) {
            player_seat_map_[playerId] = seat;
            players_image_[seat] = std::make_unique<mj::PlayerImage>(playerId, seat, infra::config::Config::instance().server().game.riichi_mahjong_4p.initial_points);
            ++seat;
        }

        deck_manager_ = mj::DeckManager(std::random_device{}());

        if (context_ && turnManager_) {
            auto* wheel = turnManager_->getTimingWheel();
            if (wheel) {
                auto roomId = context_->roomId();
                auto playerIds = context_->playerIds();
                wheel->schedule(infra::config::Config::instance().server().game.riichi_mahjong_4p.round_start_delay_ms, [this, roomId, playerIds]() {
                    auto ev = event::GameEvent::roundStart();
                    if (context_) {
                        context_->submitEvent(roomId, ev);
                    }
                });
            }
        }
    }

    void RiichiMahjong4PEngine::reset() {
        phase_ = GamePhase::Waiting;
        started_ = false;
        situation_ = mj::Situation{};
        reactions_.clear();
        last_discard_ = mj::LastDiscard{};
        for (auto& p : players_image_) {
            p.reset();
        }
        player_seat_map_.clear();
        LOG_INFO(" game reset");
    }

    void RiichiMahjong4PEngine::setContext(EngineContext* context) {
        context_ = context;
    }

    void RiichiMahjong4PEngine::initTimerSystem(infra::util::TimingWheel* wheel) {
        if (!wheel) {
            LOG_ERROR("cannot init timer system with null wheel");
            return;
        }

        const auto& rm4p = infra::config::Config::instance().server().game.riichi_mahjong_4p;
        turnManager_ = std::make_unique<mj::timer::TurnManager>(
            wheel,
            static_cast<int>(rm4p.player_count),
            rm4p.total_time,
            rm4p.compensation,
            rm4p.max_round_time,
            rm4p.react_compensation);

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

        LOG_DEBUG("timer system initialized for room {}",context_ ? context_->roomId() : "unknown");
    }

    int RiichiMahjong4PEngine::getSeatIndex(const std::string& playerId) const {
        auto it = player_seat_map_.find(playerId);
        if (it != player_seat_map_.end()) return it->second;
        return -1;
    }

    mj::PlayerImage* RiichiMahjong4PEngine::getPlayer(int seatIndex) {
        if (seatIndex < 0 || seatIndex >= 4) return nullptr;
        return players_image_[seatIndex].get();
    }

    const mj::PlayerImage* RiichiMahjong4PEngine::getPlayer(int seatIndex) const {
        if (seatIndex < 0 || seatIndex >= 4) return nullptr;
        return players_image_[seatIndex].get();
    }
} // namespace domain::game::engine
