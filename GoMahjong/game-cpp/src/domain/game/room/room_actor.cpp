#include "domain/game/room/room_actor.h"
#include "domain/game/room/room.h"
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
            return;
        }

        worker_ = std::thread(&RoomActor::workerThread, this);
    }

    void RoomActor::stop() {
        if (!running_.exchange(false)) {
            return;
        }

        queueCv_.notify_all();

        if (worker_.joinable()) {
            worker_.join();
        }

        LOG_INFO("stopped, remaining rooms: {}", rooms_.size());
    }

    bool RoomActor::submitEvent(const std::string &roomId, const event::GameEvent &event) {
        if (!running_) {
            return false;
        }

        std::lock_guard lock(queueMutex_);
        if (eventQueue_.size() >= queueCapacity_) {
            LOG_WARN("queue full, dropping event for room {}", roomId);
            return false;
        }
        eventQueue_.push(GameEventData{roomId, event});
        queueCv_.notify_one();
        return true;
    }

    bool RoomActor::submitAddRoom(std::unique_ptr<Room> room) {
        if (!running_) {
            return false;
        }

        auto roomId = room->getId();
        std::lock_guard lock(queueMutex_);
        if (eventQueue_.size() >= queueCapacity_) {
            LOG_WARN("queue full, dropping addRoom for room {}", roomId);
            return false;
        }
        eventQueue_.push(AddRoomData{std::move(roomId), std::move(room)});
        queueCv_.notify_one();
        return true;
    }

    bool RoomActor::submitRemoveRoom(const std::string &roomId) {
        if (!running_) {
            return false;
        }

        std::lock_guard lock(queueMutex_);
        if (eventQueue_.size() >= queueCapacity_) {
            LOG_WARN("[RoomActor] queue full, dropping removeRoom for room {}", roomId);
            return false;
        }
        eventQueue_.push(RemoveRoomData{roomId});
        queueCv_.notify_one();
        return true;
    }

    bool RoomActor::submitTimerEvent(const std::string &roomId, uint64_t timerId) {
        if (!running_) {
            return false;
        }

        std::lock_guard lock(queueMutex_);
        if (eventQueue_.size() >= queueCapacity_) {
            LOG_WARN("[RoomActor] queue full, dropping timerEvent for room {}", roomId);
            return false;
        }
        eventQueue_.push(TimerEventData{roomId, timerId});
        queueCv_.notify_one();
        return true;
    }

    std::size_t RoomActor::pendingEvents() const {
        std::lock_guard lock(queueMutex_);
        return eventQueue_.size();
    }

    void RoomActor::setLifecycleNotifier(RoomLifecycleNotifier *notifier) {
        lifecycleNotifier_ = notifier;
    }

    void RoomActor::setOutDispatcher(outbound::OutDispatcher *dispatcher) {
        outDispatcher_ = dispatcher;
    }

    void RoomActor::setTimingWheel(infra::util::TimingWheel *wheel) {
        timingWheel_ = wheel;
    }

    void RoomActor::workerThread() {
        LOG_DEBUG("worker thread started");

        while (running_) {
            ActorEvent evt;
            {
                std::unique_lock lock(queueMutex_);
                queueCv_.wait(lock, [this] { return !eventQueue_.empty() || !running_; });

                if (!running_ && eventQueue_.empty()) {
                    break;
                }

                if (eventQueue_.empty()) {
                    continue;
                }

                evt = std::move(eventQueue_.front());
                eventQueue_.pop();
            }
            processEvent(evt);
        }

        LOG_DEBUG("[RoomActor] worker thread exited");
    }

    void RoomActor::processEvent(ActorEvent &evt) {
        std::visit([this](auto &&arg) {
            using T = std::decay_t<decltype(arg)>;
            if constexpr (std::is_same_v<T, GameEventData>) {
                handleGameEvent(arg);
            } else if constexpr (std::is_same_v<T, AddRoomData>) {
                handleAddRoom(arg);
            } else if constexpr (std::is_same_v<T, RemoveRoomData>) {
                handleRemoveRoom(arg);
            } else if constexpr (std::is_same_v<T, TimerEventData>) {
                handleTimerEvent(arg);
            }
        }, evt);
    }

    void RoomActor::handleGameEvent(GameEventData &data) {
        auto it = rooms_.find(data.roomId);
        if (it == rooms_.end()) {
            LOG_WARN("room {} not found, dropping event", data.roomId);
            return;
        }

        auto *room = it->second.get();
        room->handleEvent(data.event);

        // Engine 通过回调将 roomId 记录到 gameOverRooms_，此处安全清理
        for (const auto& id : gameOverRooms_) {
            auto roomIt = rooms_.find(id);
            if (roomIt != rooms_.end()) {
                rooms_.erase(roomIt);
                roomCount_.store(rooms_.size(), std::memory_order_relaxed);
                LOG_DEBUG("[RoomActor] game over, removed room {}, total: {}", id, rooms_.size());
            }
            if (lifecycleNotifier_) {
                lifecycleNotifier_->onGameEnd(id);
            }
        }
        gameOverRooms_.clear();
    }

    void RoomActor::handleAddRoom(AddRoomData &data) {
        auto roomId = data.roomId;
        auto *roomPtr = data.room.get();

        roomPtr->initGame();

        rooms_[std::move(data.roomId)] = std::move(data.room);

        // 配置 EngineContext 回调：Engine 通知游戏结束时，触发清理链
        auto it = rooms_.find(roomId);
        if (it != rooms_.end()) {
            auto* ctx = it->second->getEngineContext();
            if (ctx) {
                // 注入 OutDispatcher
                if (outDispatcher_) {
                    ctx->setOutDispatcher(outDispatcher_);
                }
                ctx->setGameOverCallback([this](const std::string& id) {
                    // 仅记录，不直接删除（避免在 Room::handleEvent 调用栈中销毁 Room）
                    gameOverRooms_.insert(id);
                });
            }
        }

        roomCount_.store(rooms_.size(), std::memory_order_relaxed);
        LOG_DEBUG("[RoomActor] added room {}, total: {}", roomId, rooms_.size());
    }

    void RoomActor::handleRemoveRoom(RemoveRoomData &data) {
        auto it = rooms_.find(data.roomId);
        if (it != rooms_.end()) {
            rooms_.erase(it);
            roomCount_.store(rooms_.size(), std::memory_order_relaxed);
            LOG_DEBUG("[RoomActor] removed room {}, total: {}", data.roomId, rooms_.size());
        }
    }

    void RoomActor::handleTimerEvent(TimerEventData &data) {
        // 检查房间是否还存在（可能已 gameOver 被移除）
        if (rooms_.find(data.roomId) == rooms_.end()) {
            LOG_DEBUG("[RoomActor] timer {} for room {} dropped (room not found)", data.timerId, data.roomId);
            return;
        }

        // 在 Actor 线程中安全执行回调
        if (timingWheel_) {
            timingWheel_->fire(data.timerId);
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
        LOG_INFO("[RoomActorPool] started {} actors", actors_.size());
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

    bool RoomActorPool::assignRoom(std::unique_ptr<Room> room) {
        auto roomId = room->getId();

        std::lock_guard lock(roomActorMapMutex_);

        if (roomActorMap_.contains(roomId)) {
            LOG_DEBUG("room {} already assigned", roomId);
            return false;
        }

        // 选择负载最低的 Actor
        auto *actor = selectLeastLoadedActor();
        if (!actor) {
            LOG_ERROR("no available actor for room {}", roomId);
            return false;
        }

        roomActorMap_[roomId] = actor;
        bool ok = actor->submitAddRoom(std::move(room));
        if (!ok) {
            roomActorMap_.erase(roomId);
            LOG_ERROR("failed to submit addRoom for room {}", roomId);
            return false;
        }

        LOG_DEBUG("assigned room {} to actor, total rooms: {}", roomId, roomActorMap_.size());
        return true;
    }

    bool RoomActorPool::removeRoom(const std::string &roomId) {
        std::lock_guard lock(roomActorMapMutex_);

        auto it = roomActorMap_.find(roomId);
        if (it == roomActorMap_.end()) {
            LOG_DEBUG("room {} not found in mapping", roomId);
            return false;
        }

        auto *actor = it->second;
        bool ok = actor->submitRemoveRoom(roomId);
        roomActorMap_.erase(it);
        LOG_DEBUG("removed room {}, total rooms: {}", roomId, roomActorMap_.size());
        return ok;
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

    void RoomActorPool::setLifecycleNotifier(RoomLifecycleNotifier *notifier) {
        for (auto &actor: actors_) {
            actor->setLifecycleNotifier(notifier);
        }
    }

    void RoomActorPool::setOutDispatcher(outbound::OutDispatcher *dispatcher) {
        for (auto &actor: actors_) {
            actor->setOutDispatcher(dispatcher);
        }
    }

    void RoomActorPool::setTimingWheel(infra::util::TimingWheel *wheel) {
        for (auto &actor: actors_) {
            actor->setTimingWheel(wheel);
        }
    }

    bool RoomActorPool::submitTimerEvent(const std::string &roomId, uint64_t timerId) {
        auto *actor = getActorForRoom(roomId);
        if (!actor) {
            LOG_WARN("[RoomActorPool] no actor for room {}, dropping timer {}", roomId, timerId);
            return false;
        }
        return actor->submitTimerEvent(roomId, timerId);
    }

    RoomActor *RoomActorPool::selectLeastLoadedActor() const {
        if (actors_.empty()) {
            return nullptr;
        }

        std::size_t startIndex = nextActorIndex_.fetch_add(1) % actors_.size();

        auto *selected = actors_[startIndex].get();
        auto minLoad = selected->roomCount();

        if (minLoad < 10) {
            return selected;
        }
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
        return std::hash<std::string>{}(roomId) % actors_.size();
    }

} // namespace domain::game::room
