#include "domain/game/room/room_manager.h"
#include "domain/game/room/room.h"

#include "infrastructure/log/logger.hpp"

#include <chrono>
#include <random>
#include <sstream>
#include <iomanip>
#include <atomic>

namespace domain::game::room {

    static std::atomic<std::uint64_t> roomCounter{0};

    static std::string generate_room_id() {
        auto now = std::chrono::system_clock::now();
        auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(now.time_since_epoch()).count();
        auto counter = roomCounter.fetch_add(1, std::memory_order_relaxed);

        std::ostringstream oss;
        oss << "room_" << ms << "_" << std::hex << std::setw(4) << std::setfill('0') << (counter % 65536);
        return oss.str();
    }

    RoomManager::RoomManager() 
        : actorPool_(std::make_unique<RoomActorPool>(4, 1024)) {
        actorPool_->setLifecycleNotifier(this);
    }

    RoomManager::RoomManager(std::uint32_t actorCount, std::uint32_t queueCapacity)
        : actorPool_(std::make_unique<RoomActorPool>(actorCount, queueCapacity)) {
        actorPool_->setLifecycleNotifier(this);
    }

    RoomManager::~RoomManager() {
        stop();
    }

    void RoomManager::start() {
        if (actorPool_) {
            actorPool_->start();
            LOG_INFO("started with {} actors", actorPool_->actorCount());
        }
    }

    void RoomManager::stop() {
        if (actorPool_) {
            actorPool_->stop();
            LOG_INFO("stopped");
        }
    }

    void RoomManager::submitEvent(const std::string& roomId, const event::GameEvent& event) {
        if (actorPool_) {
            actorPool_->submitEvent(roomId, event);
        }
    }

    std::string RoomManager::create_room(const std::vector<std::string> &players, std::int32_t engineType) {
        std::lock_guard lock(mutex_);

        for (const auto &playerId: players) {
            if (playerRoom_.contains(playerId)) {
                LOG_WARN("player {} already in a room", playerId);
                return {};
            }
        }

        auto roomId = generate_room_id();

        auto room = std::make_unique<Room>(roomId, engineType);
        for (const auto &userId: players) {
            room->addPlayer(userId);
        }

        for (const auto &playId: players) {
            playerRoom_[playId] = roomId;
        }
        roomPlayers_[roomId] = players;

        // 所有权转移给 Actor
        if (actorPool_) {
            bool ok = actorPool_->assignRoom(std::move(room));
            if (!ok) {
                // 回滚路由
                for (const auto &userId: players) {
                    playerRoom_.erase(userId);
                }
                roomPlayers_.erase(roomId);
                LOG_ERROR("failed to assign room {} to actor", roomId);
                return {};
            }
        }

        LOG_DEBUG("created room {} with {} players", roomId, players.size());
        return roomId;
    }

    std::optional<std::string> RoomManager::get_player_room_id(const std::string &playerId) {
        std::lock_guard lock(mutex_);
        auto it = playerRoom_.find(playerId);
        if (it != playerRoom_.end()) {
            return it->second;
        }
        return std::nullopt;
    }

    bool RoomManager::delete_room(const std::string &roomId) {
        std::lock_guard lock(mutex_);

        auto it = roomPlayers_.find(roomId);
        if (it == roomPlayers_.end()) {
            LOG_WARN("room {} not found", roomId);
            return false;
        }

        for (const auto &userId: it->second) {
            playerRoom_.erase(userId);
        }
        roomPlayers_.erase(it);

        if (actorPool_) {
            actorPool_->removeRoom(roomId);
        }

        LOG_INFO("deleted room {}", roomId);
        return true;
    }

    void RoomManager::onGameEnd(const std::string& roomId) {
        std::lock_guard lock(mutex_);

        auto it = roomPlayers_.find(roomId);
        if (it == roomPlayers_.end()) {
            LOG_WARN("onGameEnd: room {} not found in routing", roomId);
            return;
        }

        for (const auto &userId: it->second) {
            playerRoom_.erase(userId);
        }
        roomPlayers_.erase(it);

        if (actorPool_) {
            actorPool_->removeRoom(roomId);
        }

        LOG_INFO("game ended, cleaned up room {}", roomId);
    }

    void RoomManager::setOutDispatcher(outbound::OutDispatcher* dispatcher) {
        if (actorPool_) {
            actorPool_->setOutDispatcher(dispatcher);
        }
    }

    void RoomManager::setTimingWheel(infra::util::TimingWheel* wheel) {
        if (actorPool_) {
            actorPool_->setTimingWheel(wheel);
        }
    }

    bool RoomManager::submitTimerEvent(const std::string& roomId, uint64_t timerId) {
        if (actorPool_) {
            return actorPool_->submitTimerEvent(roomId, timerId);
        }
        return false;
    }

    std::size_t RoomManager::room_count() const {
        std::lock_guard lock(mutex_);
        return roomPlayers_.size();
    }

    std::size_t RoomManager::player_count() const {
        std::lock_guard lock(mutex_);
        return playerRoom_.size();
    }

    std::size_t RoomManager::actor_count() const {
        return actorPool_ ? actorPool_->actorCount() : 0;
    }

} // namespace domain::game::room
