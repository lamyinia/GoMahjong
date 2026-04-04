#pragma once

#include "domain/game/room/room.h"
#include "domain/game/room/room_actor.h"

#include <cstdint>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <vector>

namespace domain::game::room {

    // 房间管理器
    // 管理所有游戏房间实例，维护玩家到房间的路由
    class RoomManager {
    public:
        RoomManager();
        explicit RoomManager(std::uint32_t actorCount, std::uint32_t queueCapacity = 1024);
        ~RoomManager();

        RoomManager(const RoomManager &) = delete;
        RoomManager &operator=(const RoomManager &) = delete;

        // === 生命周期 ===
        void start();
        void stop();

        // === 事件处理 ===
        // 提交事件到 Actor 池
        void submitEvent(const std::string& roomId, const event::GameEvent& event);

        // 创建房间
        // players: userID 列表
        // engineType: 游戏引擎类型
        // 返回房间指针，失败返回 nullptr
        Room* create_room(const std::vector<std::string> &players, std::int32_t engineType);

        // 获取房间
        Room* get_room(const std::string &roomId);

        // 获取玩家所在房间
        Room* get_player_room(const std::string &userId);

        // 删除房间
        bool delete_room(const std::string &roomId);

        // 获取统计信息
        std::size_t room_count() const;
        std::size_t player_count() const;
        std::size_t actor_count() const;

    private:
        // 事件处理器回调
        void onEvent(const std::string& roomId, const event::GameEvent& event);

    private:
        mutable std::mutex mutex_;
        std::map<std::string, std::unique_ptr<Room>> rooms_;     // roomId -> Room
        std::map<std::string, std::string> playerRoom_;          // userId -> roomId
        std::unique_ptr<RoomActorPool> actorPool_;                // Actor 池
    };

} // namespace domain::game::room