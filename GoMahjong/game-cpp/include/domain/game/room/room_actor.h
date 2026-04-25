#pragma once

#include "domain/game/event/mahjong_game_event.h"
#include "domain/game/outbound/out_dispatcher.h"
#include "infrastructure/util/timing_wheel.h"

#include <atomic>
#include <condition_variable>
#include <cstdint>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <queue>
#include <set>
#include <string>
#include <thread>
#include <variant>
#include <vector>

namespace domain::game::room {
    class Room;

    class RoomLifecycleNotifier {
    public:
        virtual ~RoomLifecycleNotifier() = default;
        virtual void onGameEnd(const std::string& roomId) = 0;
    };


    struct GameEventData {
        std::string roomId;
        event::GameEvent event;
    };

    struct AddRoomData {
        std::string roomId;
        std::unique_ptr<Room> room;
    };

    struct RemoveRoomData {
        std::string roomId;
    };

    using ActorEvent = std::variant<GameEventData, AddRoomData, RemoveRoomData>;

    // 单个 Actor：独占线程，管理多个房间，rooms_ 只在 worker 线程读写，完全无锁
    class RoomActor {
    public:
        explicit RoomActor(std::uint32_t queueCapacity = 1024);
        ~RoomActor();

        RoomActor(const RoomActor&) = delete;
        RoomActor& operator=(const RoomActor&) = delete;
        RoomActor(RoomActor&&) = delete;
        RoomActor& operator=(RoomActor&&) = delete;

        void start();
        void stop();

        bool submitEvent(const std::string& roomId, const event::GameEvent& event);

        bool submitAddRoom(std::unique_ptr<Room> room);
        bool submitRemoveRoom(const std::string& roomId);
        [[nodiscard]] std::size_t roomCount() const { return roomCount_.load(std::memory_order_relaxed); }
        [[nodiscard]] std::size_t pendingEvents() const;

        void setLifecycleNotifier(RoomLifecycleNotifier* notifier);
        void setOutDispatcher(outbound::OutDispatcher* dispatcher);
        void setTimingWheel(infra::util::TimingWheel* wheel);

    private:
        void workerThread();
        void processEvent(ActorEvent& evt);

        void handleGameEvent(GameEventData& data);
        void handleAddRoom(AddRoomData& data);
        void handleRemoveRoom(RemoveRoomData& data);
    private:
        std::atomic<bool> running_{false};
        std::thread worker_;

        mutable std::mutex queueMutex_;
        std::queue<ActorEvent> eventQueue_;
        std::condition_variable queueCv_;
        std::uint32_t queueCapacity_;

        std::map<std::string, std::unique_ptr<Room>> rooms_;
        std::atomic<std::size_t> roomCount_{0};
        std::set<std::string> gameOverRooms_;  // 待清理的房间（回调中记录，handleGameEvent返回后清理）

        RoomLifecycleNotifier* lifecycleNotifier_{nullptr};
        outbound::OutDispatcher* outDispatcher_{nullptr};
        infra::util::TimingWheel* timingWheel_{nullptr};
    };

    class RoomActorPool {
    public:
        explicit RoomActorPool(std::uint32_t actorCount = 4, std::uint32_t queueCapacity = 1024);
        ~RoomActorPool();

        RoomActorPool(const RoomActorPool&) = delete;
        RoomActorPool& operator=(const RoomActorPool&) = delete;

        void start();
        void stop();

        bool submitEvent(const std::string& roomId, const event::GameEvent& event) const;

        bool assignRoom(std::unique_ptr<Room> room);
        bool removeRoom(const std::string& roomId);
        [[nodiscard]] RoomActor* getActorForRoom(const std::string& roomId) const;

        void setLifecycleNotifier(RoomLifecycleNotifier* notifier);
        void setOutDispatcher(outbound::OutDispatcher* dispatcher);
        void setTimingWheel(infra::util::TimingWheel* wheel);

        [[nodiscard]] std::size_t actorCount() const { return actors_.size(); }
        [[nodiscard]] std::size_t totalRooms() const;
        [[nodiscard]] std::size_t totalPendingEvents() const;

    private:
        [[nodiscard]] RoomActor* selectLeastLoadedActor() const;
        [[nodiscard]] std::size_t hashRoomId(const std::string& roomId) const;

    private:
        std::vector<std::unique_ptr<RoomActor>> actors_;
        
        mutable std::mutex roomActorMapMutex_;
        std::map<std::string, RoomActor*> roomActorMap_;
        
        mutable std::atomic<std::size_t> nextActorIndex_{0};
    };

} // namespace domain::game::room
