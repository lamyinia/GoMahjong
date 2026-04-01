#include "domain/game/room/room_manager.h"

#include <chrono>
#include <random>
#include <sstream>
#include <iomanip>

namespace domain::game::room {

    // 生成房间 ID
    static std::string generate_room_id() {
        auto now = std::chrono::system_clock::now();
        auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(now.time_since_epoch()).count();

        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 65535);

        std::ostringstream oss;
        oss << "room_" << ms << "_" << std::hex << std::setw(4) << std::setfill('0') << dis(gen);
        return oss.str();
    }

    RoomManager::RoomManager() = default;

    const Room *RoomManager::create_room(const std::map<std::string, std::string> &players, std::int32_t engine_type) {
        // 验证玩家数量（日本麻将需要 4 人）
        if (players.size() != 4) {
            return nullptr;
        }

        std::lock_guard lock(mutex_);

        // 检查玩家是否已在其他房间
        for (const auto &[user_id, _]: players) {
            if (player_room_.contains(user_id)) {
                return nullptr;
            }
        }

        // 创建房间
        auto room = std::make_unique<Room>();
        room->id = generate_room_id();
        room->players = players;
        room->engine_type = engine_type;

        const Room *result = room.get();

        // 更新路由
        for (const auto &[user_id, _]: players) {
            player_room_[user_id] = room->id;
        }

        rooms_[room->id] = std::move(room);
        return result;
    }

    const Room *RoomManager::get_room(const std::string &room_id) const {
        std::lock_guard lock(mutex_);
        if (auto it = rooms_.find(room_id); it != rooms_.end()) {
            return it->second.get();
        }
        return nullptr;
    }

    const Room *RoomManager::get_player_room(const std::string &user_id) const {
        std::lock_guard lock(mutex_);
        if (auto it = player_room_.find(user_id); it != player_room_.end()) {
            if (auto room_it = rooms_.find(it->second); room_it != rooms_.end()) {
                return room_it->second.get();
            }
        }
        return nullptr;
    }

    bool RoomManager::delete_room(const std::string &room_id) {
        std::lock_guard lock(mutex_);

        auto it = rooms_.find(room_id);
        if (it == rooms_.end()) {
            return false;
        }

        // 清理玩家路由
        for (const auto &[user_id, _]: it->second->players) {
            player_room_.erase(user_id);
        }

        rooms_.erase(it);
        return true;
    }

    std::size_t RoomManager::room_count() const {
        std::lock_guard lock(mutex_);
        return rooms_.size();
    }

    std::size_t RoomManager::player_count() const {
        std::lock_guard lock(mutex_);
        return player_room_.size();
    }

} // namespace domain::game::room