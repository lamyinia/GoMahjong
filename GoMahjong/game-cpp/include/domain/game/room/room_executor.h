#pragma once

#include "domain/game/event/game_event.h"

#include <condition_variable>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <queue>
#include <string>
#include <thread>
#include <vector>

namespace domain::game::room {

    class Room;  // 前向声明

    // === 事件任务 ===
    struct EventTask {
        std::string roomId;
        event::GameEvent event;
    };

    // === RoomExecutor：房间事件执行器 ===
    // 负责管理线程池和事件队列，按房间 ID 顺序处理事件
    class RoomExecutor {
    public:
        using EventHandler = std::function<void(const std::string& roomId, const event::GameEvent& event)>;

        explicit RoomExecutor(std::size_t threadCount = 4);
        ~RoomExecutor();

        // 禁止拷贝和移动
        RoomExecutor(const RoomExecutor&) = delete;
        RoomExecutor& operator=(const RoomExecutor&) = delete;
        RoomExecutor(RoomExecutor&&) = delete;
        RoomExecutor& operator=(RoomExecutor&&) = delete;

        // === 生命周期 ===
        void start();
        void stop();

        // === 事件提交 ===
        // 提交事件到队列，线程安全
        void submitEvent(const std::string& roomId, const event::GameEvent& event);

        // === 事件处理器注册 ===
        void setEventHandler(EventHandler handler);

        // === 状态查询 ===
        [[nodiscard]] bool isRunning() const { return running_; }
        [[nodiscard]] std::size_t queueSize() const;

    private:
        // === 工作线程函数 ===
        void workerThread();

        // === 事件处理 ===
        void processEvent(const EventTask& task);

    private:
        // 线程池
        std::vector<std::thread> workers_;
        std::size_t threadCount_;

        // 事件队列
        std::queue<EventTask> eventQueue_;
        mutable std::mutex queueMutex_;
        std::condition_variable queueCv_;

        // 房间锁映射（保证同一房间的事件顺序处理）
        std::map<std::string, std::unique_ptr<std::mutex>> roomMutexes_;
        std::mutex roomMutexMapMutex_;

        // 事件处理器
        EventHandler eventHandler_;

        // 运行状态
        std::atomic<bool> running_{false};
    };

} // namespace domain::game::room