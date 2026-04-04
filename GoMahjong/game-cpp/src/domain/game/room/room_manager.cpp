#include "domain/game/room/room_manager.h"

#include "infrastructure/log/logger.hpp"

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

    RoomManager::RoomManager() 
        : actorPool_(std::make_unique<RoomActorPool>(4, 1024)) {
        // 设置事件处理器回调
        actorPool_->setEventHandler([this](const std::string& roomId, const event::GameEvent& event) {
            this->onEvent(roomId, event);
        });
    }

    RoomManager::RoomManager(std::uint32_t actorCount, std::uint32_t queueCapacity)
        : actorPool_(std::make_unique<RoomActorPool>(actorCount, queueCapacity)) {
        // 设置事件处理器回调
        actorPool_->setEventHandler([this](const std::string& roomId, const event::GameEvent& event) {
            this->onEvent(roomId, event);
        });
    }

    RoomManager::~RoomManager() {
        stop();
    }

    void RoomManager::start() {
        if (actorPool_) {
            actorPool_->start();
            LOG_INFO("[RoomManager] started with {} actors", actorPool_->actorCount());
        }
    }

    void RoomManager::stop() {
        if (actorPool_) {
            actorPool_->stop();
            LOG_INFO("[RoomManager] stopped");
        }
    }

    void RoomManager::submitEvent(const std::string& roomId, const event::GameEvent& event) {
        if (actorPool_) {
            actorPool_->submitEvent(roomId, event);
        }
    }

    void RoomManager::onEvent(const std::string& roomId, const event::GameEvent& event) {
        auto* room = get_room(roomId);
        if (!room) {
            LOG_WARN("[RoomManager] room {} not found for event", roomId);
            return;
        }

        room->handleEvent(event);
    }

    Room* RoomManager::create_room(const std::vector<std::string> &players, std::int32_t engineType) {
        std::lock_guard lock(mutex_);

        // 检查玩家是否已在其他房间
        for (const auto &userId: players) {
            if (playerRoom_.contains(userId)) {
                LOG_WARN("[RoomManager] player {} already in a room", userId);
                return nullptr;
            }
        }

        auto roomId = generate_room_id();
        auto room = std::make_unique<Room>(roomId, engineType);
        for (const auto &userId: players) {
            room->addPlayer(userId);
        }
        auto* result = room.get();
        for (const auto &userId: players) {
            playerRoom_[userId] = roomId;
        }

        rooms_[roomId] = std::move(room);
        
        // 分配房间到 Actor 池
        if (actorPool_) {
            actorPool_->assignRoom(roomId);
        }
        
        LOG_INFO("[RoomManager] created room {} with {} players", roomId, players.size());
        
        // 初始化游戏
        if (result) {
            result->initGame();
        }
        
        return result;
    }

    Room* RoomManager::get_room(const std::string &roomId) {
        std::lock_guard lock(mutex_);
        if (auto it = rooms_.find(roomId); it != rooms_.end()) {
            return it->second.get();
        }
        return nullptr;
    }

    Room* RoomManager::get_player_room(const std::string &userId) {
        std::lock_guard lock(mutex_);
        if (auto it = playerRoom_.find(userId); it != playerRoom_.end()) {
            if (auto roomIt = rooms_.find(it->second); roomIt != rooms_.end()) {
                return roomIt->second.get();
            }
        }
        return nullptr;
    }

    bool RoomManager::delete_room(const std::string &roomId) {
        std::lock_guard lock(mutex_);

        auto it = rooms_.find(roomId);
        if (it == rooms_.end()) {
            LOG_WARN("[RoomManager] room {} not found", roomId);
            return false;
        }

        // 清理玩家路由
        for (const auto &userId: it->second->getPlayers()) {
            playerRoom_.erase(userId);
        }

        // 从 Actor 池移除房间
        if (actorPool_) {
            actorPool_->removeRoom(roomId);
        }

        rooms_.erase(it);
        LOG_INFO("[RoomManager] deleted room {}", roomId);
        return true;
    }

    std::size_t RoomManager::room_count() const {
        std::lock_guard lock(mutex_);
        return rooms_.size();
    }

    std::size_t RoomManager::player_count() const {
        std::lock_guard lock(mutex_);
        return playerRoom_.size();
    }

    std::size_t RoomManager::actor_count() const {
        return actorPool_ ? actorPool_->actorCount() : 0;
    }

} // namespace domain::game::room