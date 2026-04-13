#pragma once

#include "infrastructure/util/timing_wheel.h"

#include <atomic>
#include <thread>

namespace infra::util {

    // 定时器线程：独立线程驱动 TimingWheel tick
    // 到期后通过 ExpiredCallback 通知（通常路由到 RoomActor 队列）
    class TimerThread {
    public:
        explicit TimerThread(TimingWheel& wheel, std::uint32_t tickIntervalMs = 50);

        ~TimerThread();

        TimerThread(const TimerThread&) = delete;
        TimerThread& operator=(const TimerThread&) = delete;

        void start();
        void stop();

        [[nodiscard]] bool isRunning() const { return running_.load(std::memory_order_relaxed); }

    private:
        void run();

        TimingWheel& wheel_;
        std::uint32_t tickIntervalMs_;
        std::atomic<bool> running_{false};
        std::thread thread_;
    };

} // namespace infra::util
