#pragma once

#include "domain/game/event/game_event.h"

#include <atomic>
#include <condition_variable>
#include <cstdint>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <queue>
#include <string>
#include <thread>
#include <vector>

namespace domain::game::room {

    // 前向声明
    class Room;

    // 房间事件包装
    struct RoomEvent {
        std::string roomId;
        event::GameEvent event;
    };

    // 单个 Actor：独占线程，管理多个房间
    class RoomActor {
    public:
        using EventHandler = std::function<void(const std::string& roomId, const event::GameEvent& event)>;

        explicit RoomActor(std::uint32_t queueCapacity = 1024);
        ~RoomActor();

        // 禁止拷贝和移动
        RoomActor(const RoomActor&) = delete;
        RoomActor& operator=(const RoomActor&) = delete;
        RoomActor(RoomActor&&) = delete;
        RoomActor& operator=(RoomActor&&) = delete;

        // === 生命周期 ===
        void start();
        void stop();

        // === 事件提交 ===
        bool submitEvent(const std::string& roomId, const event::GameEvent& event);

        // === 房间管理 ===
        void addRoom(const std::string& roomId);
        void removeRoom(const std::string& roomId);
        [[nodiscard]] std::size_t roomCount() const;
        [[nodiscard]] std::size_t pendingEvents() const;

        // === 回调设置 ===
        void setEventHandler(EventHandler handler);

    private:
        void workerThread();
        void processEvent(const RoomEvent& roomEvent);

    private:
        std::atomic<bool> running_{false};
        std::thread worker_;
        
        // 事件队列（单生产者-单消费者，使用 mutex 保护）
        mutable std::mutex queueMutex_;
        std::queue<RoomEvent> eventQueue_;
        std::condition_variable queueCv_;
        std::uint32_t queueCapacity_;
        
        // 房间集合（用于统计）
        mutable std::mutex roomsMutex_;
        std::map<std::string, bool> rooms_;
        
        // 事件处理器
        EventHandler eventHandler_;
    };

    // Actor 池：管理多个 Actor，提供负载均衡
    class RoomActorPool {
    public:
        explicit RoomActorPool(std::uint32_t actorCount = 4, std::uint32_t queueCapacity = 1024);
        ~RoomActorPool();

        RoomActorPool(const RoomActorPool&) = delete;
        RoomActorPool& operator=(const RoomActorPool&) = delete;

        // === 生命周期 ===
        void start();
        void stop();

        // === 事件提交（自动路由到对应 Actor）===
        bool submitEvent(const std::string& roomId, const event::GameEvent& event);

        // === 房间管理 ===
        // 分配房间到负载最低的 Actor
        void assignRoom(const std::string& roomId);
        // 移除房间
        void removeRoom(const std::string& roomId);
        // 获取房间所在的 Actor
        [[nodiscard]] RoomActor* getActorForRoom(const std::string& roomId) const;

        // === 统计 ===
        [[nodiscard]] std::size_t actorCount() const { return actors_.size(); }
        [[nodiscard]] std::size_t totalRooms() const;
        [[nodiscard]] std::size_t totalPendingEvents() const;

        // === 回调设置 ===
        void setEventHandler(RoomActor::EventHandler handler);

    private:
        // 选择负载最低的 Actor
        [[nodiscard]] RoomActor* selectLeastLoadedActor() const;
        // 计算房间 ID 的哈希值（用于一致性）
        [[nodiscard]] std::size_t hashRoomId(const std::string& roomId) const;

    private:
        std::vector<std::unique_ptr<RoomActor>> actors_;
        
        // 房间到 Actor 的映射
        mutable std::mutex roomActorMapMutex_;
        std::map<std::string, RoomActor*> roomActorMap_;
        
        // 下一个分配的 Actor 索引（轮询）
        mutable std::atomic<std::size_t> nextActorIndex_{0};
        
        // 事件处理器
        RoomActor::EventHandler eventHandler_;
    };

} // namespace domain::game::room
