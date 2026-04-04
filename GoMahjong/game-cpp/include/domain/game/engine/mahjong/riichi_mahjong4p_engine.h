#pragma once

#include "domain/game/engine/engine.h"

#include <set>
#include <string>

namespace domain::game::engine {

    // 日本麻将 4 人游戏引擎
    class RiichiMahjong4PEngine : public Engine {
    public:
        RiichiMahjong4PEngine();
        ~RiichiMahjong4PEngine() override = default;

        // === 核心事件处理 ===
        void handleEvent(const event::GameEvent& event) override;

        // === 玩家管理 ===
        void onPlayerJoin(const std::string& userId) override;
        void onPlayerLeave(const std::string& userId) override;
        bool hasPlayer(const std::string& userId) const override;
        std::size_t playerCount() const override;

        // === 游戏状态 ===
        [[nodiscard]] GamePhase getPhase() const override;
        [[nodiscard]] bool isGameOver() const override;
        [[nodiscard]] bool canStart() const override;

        // === 状态序列化 ===
        [[nodiscard]] std::string getGameState() const override;
        [[nodiscard]] std::string getPlayerState(const std::string& userId) const override;

        // === 游戏控制 ===
        void start() override;
        void reset() override;
        void destroy() override;

    private:
        std::set<std::string> players_;
        GamePhase phase_{GamePhase::Waiting};
        bool started_{false};
    };

} // namespace domain::game::engine