#pragma once

#include "domain/game/engine/engine.h"
#include "domain/game/engine/engine_context.h"

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

namespace domain::game::room {

    // 房间信息
    class Room {
    public:
        Room(std::string id, std::int32_t engineType);
        ~Room();

        // 禁止拷贝
        Room(const Room&) = delete;
        Room& operator=(const Room&) = delete;

        // 允许移动
        Room(Room&&) noexcept;
        Room& operator=(Room&&) noexcept;

        [[nodiscard]] const std::string& getId() const { return id_; }
        [[nodiscard]] std::int32_t getEngineType() const { return engineType_; }
        [[nodiscard]] const std::vector<std::string>& getPlayers() const { return players_; }
        [[nodiscard]] engine::Engine* getEngine() const { return engine_.get(); }
        [[nodiscard]] engine::EngineContext* getEngineContext() const { return engineContext_.get(); }

        void addPlayer(const std::string& userId);
        void removePlayer(const std::string& userId);
        [[nodiscard]] bool hasPlayer(const std::string& userId) const;
        [[nodiscard]] std::size_t playerCount() const { return players_.size(); }

        void initGame();

        void handleEvent(const event::GameEvent& event);

        [[nodiscard]] bool isGameOver() const;

    private:
        std::string id_;
        std::int32_t engineType_{};
        std::vector<std::string> players_;  // userId list
        std::unique_ptr<engine::Engine> engine_;  // 游戏状态机
        std::unique_ptr<engine::EngineContext> engineContext_;  // Engine 与外界的桥梁
    };

} // namespace domain::game::room
