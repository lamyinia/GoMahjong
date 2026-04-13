#pragma once

#include "domain/game/event/mahjong_game_event.h"

#include <memory>
#include <string>
#include <vector>

namespace domain::game::engine {
    class EngineContext;
}

namespace domain::game::engine {

    enum class EngineType : std::int32_t {
        Unknown = 0,
        RiichiMahjong4P = 1,
        RiichiMahjong3P = 2,
        RiichiMahjong2P = 3,
    };

    enum class GamePhase {
        Waiting,
        Ready,
        Playing,
        Finished,
        GameOver
    };

    class Engine {
    public:
        virtual ~Engine() = default;

        // === 核心事件处理 ===
        // 统一入口，所有游戏事件都通过此方法处理
        virtual void handleEvent(const event::GameEvent& event) = 0;

        virtual void onPlayerJoin(const std::string& userId) = 0;
        virtual void onPlayerLeave(const std::string& userId) = 0;
        virtual bool hasPlayer(const std::string& userId) const = 0;
        virtual std::size_t playerCount() const = 0;

        [[nodiscard]] virtual GamePhase getPhase() const = 0;
        [[nodiscard]] virtual bool isGameOver() const = 0;
        [[nodiscard]] virtual bool canStart() const = 0;

        [[nodiscard]] virtual std::string getGameState() const = 0;
        
        [[nodiscard]] virtual std::string getPlayerState(const std::string& userId) const = 0;

        virtual void start() = 0;
        virtual void reset() = 0;

        // 注入 EngineContext，Engine 通过它通知生命周期和广播
        virtual void setContext(EngineContext* context) = 0;

        static std::unique_ptr<Engine> create(EngineType type);
    };

} // namespace domain::game::engine