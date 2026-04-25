#pragma once

#include "domain/game/engine/engine.h"
#include "domain/game/engine/engine_context.h"
#include "domain/game/engine/mahjong/material.h"
#include "domain/game/engine/mahjong/player_image.h"
#include "domain/game/engine/mahjong/hu_searcher.h"
#include "domain/game/engine/mahjong/timer/turn_manager.h"
#include "domain/game/event/mahjong_game_event.h"

#include <array>
#include <memory>
#include <set>
#include <string>
#include <unordered_map>

namespace domain::game::engine {

    namespace mj = ::domain::game::mahjong;

    // 日麻 4 人游戏引擎
    class RiichiMahjong4PEngine : public Engine {
    public:
        RiichiMahjong4PEngine();
        ~RiichiMahjong4PEngine() override = default;

        void handleEvent(const event::GameEvent& event) override;

        void onPlayerJoin(const std::string& userId) override;
        void onPlayerLeave(const std::string& userId) override;
        std::size_t playerCount() const override;

        [[nodiscard]] GamePhase getPhase() const override;
        [[nodiscard]] bool isGameOver() const override;
        [[nodiscard]] bool canStart() const override;

        [[nodiscard]] std::string getGameState() const override;
        [[nodiscard]] std::string getPlayerState(const std::string& userId) const override;

        void start() override;
        void reset() override;
        void setContext(EngineContext* context) override;

        // 初始化计时系统（由 RoomActor 在创建房间后调用）
        void initTimerSystem(infra::util::TimingWheel* wheel);

        // 获取 TurnManager（供外部查询状态）
        [[nodiscard]] mj::timer::TurnManager* getTurnManager() { return turnManager_.get(); }

    private:
        // ---- 基础设施 ----
        std::set<std::string> players_;
        std::unordered_map<std::string, int> player_seat_map_;  // playerId → seatIndex
        GamePhase phase_{GamePhase::Waiting};
        bool started_{false};
        EngineContext* context_{nullptr};
        std::unique_ptr<mj::timer::TurnManager> turnManager_;

        // ---- 麻将域模型 ----
        mj::Situation situation_;
        mj::DeckManager deck_manager_;
        std::array<std::unique_ptr<mj::PlayerImage>, 4> players_image_;
        mj::Searcher searcher_;

        // ---- 反应阶段管理 ----
        std::unordered_map<int, mj::PlayerReaction> reactions_; // seatIndex → PlayerReaction

        // ---- 状态追踪 ----
        mj::LastDiscard last_discard_;

        // ---- 辅助方法 ----
        [[nodiscard]] int getSeatIndex(const std::string& playerId) const;
        [[nodiscard]] mj::PlayerImage* getPlayer(int seatIndex);
        [[nodiscard]] const mj::PlayerImage* getPlayer(int seatIndex) const;

        // ---- 局流程 ----
        void startRound();
        void distributeCards();
        void dropTurn(int seatIndex, bool needDraw);

        // ---- 推送方法（定义在 riichi_mahjong4p_push.cpp）----
        void broadcastRoundStart();
        void pushDrawTile(int seatIndex, const mj::Tile& tile, bool isKanDraw = false);

        // ---- 事件处理 ----
        void handleRoundStart(const event::RoundStartEvent& e);
        void handleRoundEnd(const event::RoundEndEvent& e);
        void handlePlayTile(const event::PlayTileEvent& e);
        void handleDrawTile(const event::DrawTileEvent& e);
        void handleChi(const event::ChiEvent& e);
        void handlePon(const event::PonEvent& e);
        void handleKan(const event::KanEvent& e);
        void handleRon(const event::RonEvent& e);
        void handleTsumo(const event::TsumoEvent& e);
        void handleDraw(const event::DrawEvent& e);
        void handlePlayerTimeout(const event::PlayerTimeoutEvent& e);
    };

} // namespace domain::game::engine