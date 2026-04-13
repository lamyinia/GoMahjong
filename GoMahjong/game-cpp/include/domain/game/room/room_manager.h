#pragma once

#include "domain/game/room/room_actor.h"
#include "domain/game/outbound/out_dispatcher.h"
#include "infrastructure/util/timing_wheel.h"

#include <cstdint>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <vector>

namespace domain::game::room {

    // 房间管理器：薄路由层，维护玩家到房间的映射
    // Room 所有权在 RoomActor，RoomManager 只管路由
    class RoomManager : public RoomLifecycleNotifier {
    public:
        RoomManager();
        explicit RoomManager(std::uint32_t actorCount, std::uint32_t queueCapacity = 1024);
        ~RoomManager();

        RoomManager(const RoomManager &) = delete;
        RoomManager &operator=(const RoomManager &) = delete;

        void start();
        void stop();

        void submitEvent(const std::string& roomId, const event::GameEvent& event);

        std::string create_room(const std::vector<std::string> &players, std::int32_t engineType);

        [[nodiscard]] std::optional<std::string> get_player_room_id(const std::string &playerId);

        bool delete_room(const std::string &roomId);

        void onGameEnd(const std::string& roomId) override;

        void setOutDispatcher(outbound::OutDispatcher* dispatcher);
        void setTimingWheel(infra::util::TimingWheel* wheel);
        bool submitTimerEvent(const std::string& roomId, uint64_t timerId);

        [[nodiscard]] std::size_t room_count() const;
        [[nodiscard]] std::size_t player_count() const;
        [[nodiscard]] std::size_t actor_count() const;

    private:
        mutable std::mutex mutex_;
        std::map<std::string, std::string> playerRoom_;              // userId -> roomId
        std::map<std::string, std::vector<std::string>> roomPlayers_; // roomId -> playerIds
        std::unique_ptr<RoomActorPool> actorPool_;
    };

} // namespace domain::game::room