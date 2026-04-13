#pragma once

#include "domain/game/engine/engine.h"
#include "domain/game/engine/engine_context.h"
#include "domain/game/engine/mahjong/timer/turn_manager.h"
#include "domain/game/event/mahjong_game_event.h"

#include <memory>
#include <set>
#include <string>

namespace domain::game::engine {

    // 日麻 4 人游戏引擎
    class RiichiMahjong4PEngine : public Engine {
    public:
        RiichiMahjong4PEngine();
        ~RiichiMahjong4PEngine() override = default;

        void handleEvent(const event::GameEvent& event) override;

        void onPlayerJoin(const std::string& userId) override;
        void onPlayerLeave(const std::string& userId) override;
        bool hasPlayer(const std::string& userId) const override;
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
        [[nodiscard]] mahjong::timer::TurnManager* getTurnManager() { return turnManager_.get(); }

    private:
        std::set<std::string> players_;
        GamePhase phase_{GamePhase::Waiting};
        bool started_{false};
        EngineContext* context_{nullptr};
        std::unique_ptr<mahjong::timer::TurnManager> turnManager_;

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