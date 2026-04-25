#pragma once

#include <atomic>
#include <cstdint>
#include <forward_list>
#include <functional>
#include <mutex>
#include <vector>
#include <memory>
#include <unordered_map>

namespace infra::util {

    // 定时器句柄，用于 cancel/pause 操作
    struct TimerHandle {
        uint64_t id{0};
    };

    // 时间轮：O(1) 插入/删除/触发
    // 单层 512 slots × 50ms = 25.6s 一轮，超过的通过 remainingRounds 递减
    // 线程安全：per-slot mutex，tick 只锁当前 slot
    class TimingWheel {
    public:
        // 每个定时器自带回调，到期时直接调用（O(1) 路由）
        using TimerCallback = std::function<void()>;

        explicit TimingWheel(std::uint32_t tickDurationMs = 50,
                             std::uint32_t wheelSize = 512);

        ~TimingWheel() = default;

        TimerHandle schedule(std::uint64_t delayMs, TimerCallback callback);

        void cancel(const TimerHandle& handle);

        void tick();

        [[nodiscard]] std::uint32_t tickDurationMs() const { return tickDurationMs_; }
        [[nodiscard]] std::uint32_t wheelSize() const { return wheelSize_; }

    private:
        struct TimerEntry {
            uint64_t id;
            uint32_t remainingRounds;
            std::atomic<bool> cancelled{false};
            TimerCallback callback;
        };

        struct Slot {
            std::mutex mutex;
            std::forward_list<std::shared_ptr<TimerEntry>> entries;
        };

        std::uint32_t tickDurationMs_;
        std::uint32_t wheelSize_;
        std::vector<Slot> slots_;
        std::atomic<std::uint32_t> currentSlot_{0};
        std::atomic<uint64_t> nextTimerId_{1};

        // O(1) cancel 查找表
        std::unordered_map<uint64_t, std::shared_ptr<TimerEntry>> timerMap_;
        std::mutex mapMutex_;
    };

} // namespace infra::util
