#include "domain/game/room/room_executor.h"
#include "infrastructure/log/logger.hpp"

#include <atomic>

namespace domain::game::room {

    RoomExecutor::RoomExecutor(std::size_t threadCount)
        : threadCount_(threadCount) {
    }

    RoomExecutor::~RoomExecutor() {
        stop();
    }

    void RoomExecutor::start() {
        if (running_.exchange(true)) {
            return;  // 已经在运行
        }

        LOG_INFO("[RoomExecutor] starting with {} threads", threadCount_);

        // 启动工作线程
        for (std::size_t i = 0; i < threadCount_; ++i) {
            workers_.emplace_back(&RoomExecutor::workerThread, this);
        }
    }

    void RoomExecutor::stop() {
        if (!running_.exchange(false)) {
            return;  // 已经停止
        }

        LOG_INFO("[RoomExecutor] stopping");

        // 唤醒所有工作线程
        queueCv_.notify_all();

        // 等待所有工作线程结束
        for (auto& worker : workers_) {
            if (worker.joinable()) {
                worker.join();
            }
        }
        workers_.clear();

        LOG_INFO("[RoomExecutor] stopped");
    }

    void RoomExecutor::submitEvent(const std::string& roomId, const event::GameEvent& event) {
        {
            std::lock_guard lock(queueMutex_);
            eventQueue_.push({roomId, event});
        }
        queueCv_.notify_one();
    }

    void RoomExecutor::setEventHandler(EventHandler handler) {
        eventHandler_ = std::move(handler);
    }

    std::size_t RoomExecutor::queueSize() const {
        std::lock_guard lock(queueMutex_);
        return eventQueue_.size();
    }

    void RoomExecutor::workerThread() {
        while (running_) {
            EventTask task;

            // 等待事件
            {
                std::unique_lock lock(queueMutex_);
                queueCv_.wait(lock, [this] {
                    return !eventQueue_.empty() || !running_;
                });

                if (!running_ && eventQueue_.empty()) {
                    break;
                }

                if (eventQueue_.empty()) {
                    continue;
                }

                task = eventQueue_.front();
                eventQueue_.pop();
            }

            // 处理事件
            processEvent(task);
        }
    }

    void RoomExecutor::processEvent(const EventTask& task) {
        if (!eventHandler_) {
            LOG_WARN("[RoomExecutor] no event handler set, skipping event for room {}", task.roomId);
            return;
        }

        // 获取或创建房间锁
        std::mutex* roomMutex = nullptr;
        {
            std::lock_guard lock(roomMutexMapMutex_);
            auto it = roomMutexes_.find(task.roomId);
            if (it == roomMutexes_.end()) {
                auto [insertedIt, success] = roomMutexes_.emplace(
                    task.roomId, std::make_unique<std::mutex>()
                );
                roomMutex = insertedIt->second.get();
            } else {
                roomMutex = it->second.get();
            }
        }

        // 锁定房间，保证同一房间的事件顺序处理
        std::lock_guard lock(*roomMutex);

        try {
            eventHandler_(task.roomId, task.event);
        } catch (const std::exception& e) {
            LOG_ERROR("[RoomExecutor] error processing event for room {}: {}", task.roomId, e.what());
        } catch (...) {
            LOG_ERROR("[RoomExecutor] unknown error processing event for room {}", task.roomId);
        }
    }

} // namespace domain::game::room
