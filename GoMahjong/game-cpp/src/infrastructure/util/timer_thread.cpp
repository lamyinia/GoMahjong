#include "infrastructure/util/timer_thread.h"
#include "infrastructure/log/logger.hpp"

#include <chrono>

namespace infra::util {

    TimerThread::TimerThread(TimingWheel& wheel, std::uint32_t tickIntervalMs)
        : wheel_(wheel),
          tickIntervalMs_(tickIntervalMs) {
    }

    TimerThread::~TimerThread() {
        stop();
    }

    void TimerThread::start() {
        if (running_.exchange(true, std::memory_order_acq_rel)) {
            return; // already running
        }
        thread_ = std::thread(&TimerThread::run, this);
        LOG_INFO("[TimerThread] started, tick interval={}ms", tickIntervalMs_);
    }

    void TimerThread::stop() {
        if (!running_.exchange(false, std::memory_order_acq_rel)) {
            return; // already stopped
        }
        if (thread_.joinable()) {
            thread_.join();
        }
        LOG_INFO("[TimerThread] stopped");
    }

    void TimerThread::run() {
        auto interval = std::chrono::milliseconds(tickIntervalMs_);

        while (running_.load(std::memory_order_relaxed)) {
            auto start = std::chrono::steady_clock::now();

            wheel_.tick();

            // 精确睡眠：扣除 tick 执行时间
            auto elapsed = std::chrono::steady_clock::now() - start;
            auto sleepTime = interval - std::chrono::duration_cast<std::chrono::milliseconds>(elapsed);
            if (sleepTime.count() > 0) {
                std::this_thread::sleep_for(sleepTime);
            }
        }
    }

} // namespace infra::util
