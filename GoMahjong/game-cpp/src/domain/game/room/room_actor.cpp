#include "domain/game/room/room_actor.h"
#include "infrastructure/log/logger.hpp"

#include <algorithm>

namespace domain::game::room {

    // === RoomActor 实现 ===

    RoomActor::RoomActor(std::uint32_t queueCapacity)
            : queueCapacity_(queueCapacity) {
    }

    RoomActor::~RoomActor() {
        stop();
    }

    void RoomActor::start() {
        if (running_.exchange(true)) {
            return;  // 已经在运行
        }

        worker_ = std::thread(&RoomActor::workerThread, this);
        LOG_INFO("[RoomActor] started");
    }

    void RoomActor::stop() {
        if (!running_.exchange(false)) {
            return;  // 已经停止
        }

        // 唤醒工作线程
        queueCv_.notify_all();

        if (worker_.joinable()) {
            worker_.join();
        }

        LOG_INFO("[RoomActor] stopped, processed {} rooms", rooms_.size());
    }

    bool RoomActor::submitEvent(const std::string &roomId, const event::GameEvent &event) {
        if (!running_) {
            return false;
        }

        std::lock_guard lock(queueMutex_);
        if (eventQueue_.size() >= queueCapacity_) {
            LOG_WARN("[RoomActor] queue full, dropping event for room {}", roomId);
            return false;
        }
        eventQueue_.push({roomId, event});
        queueCv_.notify_one();
        return true;
    }

    void RoomActor::addRoom(const std::string &roomId) {
        std::lock_guard lock(roomsMutex_);
        rooms_[roomId] = true;
        LOG_DEBUG("[RoomActor] added room {}, total: {}", roomId, rooms_.size());
    }

    void RoomActor::removeRoom(const std::string &roomId) {
        std::lock_guard lock(roomsMutex_);
        rooms_.erase(roomId);
        LOG_DEBUG("[RoomActor] removed room {}, total: {}", roomId, rooms_.size());
    }

    std::size_t RoomActor::roomCount() const {
        std::lock_guard lock(roomsMutex_);
        return rooms_.size();
    }

    std::size_t RoomActor::pendingEvents() const {
        std::lock_guard lock(queueMutex_);
        return eventQueue_.size();
    }

    void RoomActor::setEventHandler(EventHandler handler) {
        eventHandler_ = std::move(handler);
    }

    void RoomActor::workerThread() {
        LOG_INFO("[RoomActor] worker thread started");

        while (running_) {
            RoomEvent roomEvent;
            {
                std::unique_lock lock(queueMutex_);
                queueCv_.wait(lock, [this] { return !eventQueue_.empty() || !running_; });

                if (!running_ && eventQueue_.empty()) {
                    break;
                }

                if (eventQueue_.empty()) {
                    continue;
                }

                roomEvent = eventQueue_.front();
                eventQueue_.pop();
            }
            processEvent(roomEvent);
        }

        LOG_INFO("[RoomActor] worker thread exited");
    }

    void RoomActor::processEvent(const RoomEvent &roomEvent) {
        if (eventHandler_) {
            eventHandler_(roomEvent.roomId, roomEvent.event);
        } else {
            LOG_WARN("[RoomActor] no event handler set, dropping event for room {}", roomEvent.roomId);
        }
    }

    // === RoomActorPool 实现 ===

    RoomActorPool::RoomActorPool(std::uint32_t actorCount, std::uint32_t queueCapacity) {
        actors_.reserve(actorCount);
        for (std::uint32_t i = 0; i < actorCount; ++i) {
            actors_.push_back(std::make_unique<RoomActor>(queueCapacity));
        }
    }

    RoomActorPool::~RoomActorPool() {
        stop();
    }

    void RoomActorPool::start() {
        for (auto &actor: actors_) {
            actor->start();
        }
        LOG_INFO("[RoomActorPool] started with {} actors", actors_.size());
    }

    void RoomActorPool::stop() {
        for (auto &actor: actors_) {
            actor->stop();
        }
        LOG_INFO("[RoomActorPool] stopped");
    }

    bool RoomActorPool::submitEvent(const std::string &roomId, const event::GameEvent &event) {
        auto *actor = getActorForRoom(roomId);
        if (!actor) {
            LOG_WARN("[RoomActorPool] room {} not assigned to any actor", roomId);
            return false;
        }
        return actor->submitEvent(roomId, event);
    }

    void RoomActorPool::assignRoom(const std::string &roomId) {
        std::lock_guard lock(roomActorMapMutex_);

        // 检查是否已分配
        if (roomActorMap_.contains(roomId)) {
            LOG_DEBUG("[RoomActorPool] room {} already assigned", roomId);
            return;
        }

        // 选择负载最低的 Actor
        auto *actor = selectLeastLoadedActor();
        if (!actor) {
            LOG_ERROR("[RoomActorPool] no available actor for room {}", roomId);
            return;
        }

        actor->addRoom(roomId);
        roomActorMap_[roomId] = actor;
        // 直接访问 roomActorMap_.size()，避免调用 totalRooms() 导致死锁
        LOG_INFO("[RoomActorPool] assigned room {} to actor, total rooms: {}", roomId, roomActorMap_.size());
    }

    void RoomActorPool::removeRoom(const std::string &roomId) {
        std::lock_guard lock(roomActorMapMutex_);

        auto it = roomActorMap_.find(roomId);
        if (it == roomActorMap_.end()) {
            LOG_DEBUG("[RoomActorPool] room {} not found in mapping", roomId);
            return;
        }

        auto *actor = it->second;
        actor->removeRoom(roomId);
        roomActorMap_.erase(it);
        // 直接访问 roomActorMap_.size()，避免调用 totalRooms() 导致死锁
        LOG_INFO("[RoomActorPool] removed room {}, total rooms: {}", roomId, roomActorMap_.size());
    }

    RoomActor *RoomActorPool::getActorForRoom(const std::string &roomId) const {
        std::lock_guard lock(roomActorMapMutex_);
        auto it = roomActorMap_.find(roomId);
        return it != roomActorMap_.end() ? it->second : nullptr;
    }

    std::size_t RoomActorPool::totalRooms() const {
        std::lock_guard lock(roomActorMapMutex_);
        return roomActorMap_.size();
    }

    std::size_t RoomActorPool::totalPendingEvents() const {
        std::size_t total = 0;
        for (const auto &actor: actors_) {
            total += actor->pendingEvents();
        }
        return total;
    }

    void RoomActorPool::setEventHandler(RoomActor::EventHandler handler) {
        eventHandler_ = std::move(handler);
        for (auto &actor: actors_) {
            actor->setEventHandler(eventHandler_);
        }
    }

    RoomActor *RoomActorPool::selectLeastLoadedActor() const {
        if (actors_.empty()) {
            return nullptr;
        }

        // 使用轮询 + 负载检查的策略
        // 先检查下一个 Actor，如果负载合理就直接使用
        // 否则遍历所有 Actor 找到负载最低的
        std::size_t startIndex = nextActorIndex_.fetch_add(1) % actors_.size();

        auto *selected = actors_[startIndex].get();
        auto minLoad = selected->roomCount();

        // 如果当前 Actor 负载较低（< 10 个房间），直接使用
        if (minLoad < 10) {
            return selected;
        }

        // 否则遍历所有 Actor 找到负载最低的
        for (const auto &actor: actors_) {
            auto load = actor->roomCount();
            if (load < minLoad) {
                minLoad = load;
                selected = actor.get();
            }
        }

        return selected;
    }

    std::size_t RoomActorPool::hashRoomId(const std::string &roomId) const {
        // 简单的哈希函数，用于一致性分配
        return std::hash<std::string>{}(roomId) % actors_.size();
    }

} // namespace domain::game::room
