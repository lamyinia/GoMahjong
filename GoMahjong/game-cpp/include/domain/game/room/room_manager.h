#pragma once

#include <cstdint>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <string>

namespace domain::game::room {

    // 房间信息
    struct Room {
        std::string id;
        std::map<std::string, std::string> players; // userID -> connectorTopic
        std::int32_t engine_type{};
    };

    // 房间管理器
    // 管理所有游戏房间实例，维护玩家到房间的路由
    class RoomManager {
    public:
        RoomManager();

        ~RoomManager() = default;

        RoomManager(const RoomManager &) = delete;

        RoomManager &operator=(const RoomManager &) = delete;

        // 创建房间
        // players: userID -> connectorTopic
        // engine_type: 游戏引擎类型
        // 返回房间指针，失败返回 nullptr
        const Room *create_room(const std::map<std::string, std::string> &players, std::int32_t engine_type);

        // 获取房间
        const Room *get_room(const std::string &room_id) const;

        // 获取玩家所在房间
        const Room *get_player_room(const std::string &user_id) const;

        // 删除房间
        bool delete_room(const std::string &room_id);

        // 获取统计信息
        std::size_t room_count() const;
        std::size_t player_count() const;

    private:
        mutable std::mutex mutex_;
        std::map<std::string, std::unique_ptr<Room>> rooms_;     // roomID -> Room
        std::map<std::string, std::string> player_room_;         // userID -> roomID
    };

} // namespace domain::game::room