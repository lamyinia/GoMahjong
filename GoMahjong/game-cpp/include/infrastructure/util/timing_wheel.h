#pragma once

#include <atomic>
#include <cstdint>
#include <forward_list>
#include <functional>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>
#include <memory>

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
        // expiredCallback: 定时器到期时调用，参数为 roomId + timerId
        // 由 TimerThread 调用，用于路由到对应的 RoomActor
        using ExpiredCallback = std::function<void(const std::string& roomId, uint64_t timerId)>;

        explicit TimingWheel(std::uint32_t tickDurationMs = 50,
                             std::uint32_t wheelSize = 512);

        ~TimingWheel() = default;

        // 设置到期回调
        void setExpiredCallback(ExpiredCallback cb);

        // 调度定时器，返回句柄
        // delayMs: 延迟毫秒数
        // roomId: 关联房间（用于路由到 Actor）
        // cb: 回调函数（在 Actor 线程执行）
        TimerHandle schedule(std::uint64_t delayMs,
                             const std::string& roomId,
                             std::function<void()> cb);

        // 取消定时器
        void cancel(const TimerHandle& handle);

        // 驱动时间轮前进一格（由 TimerThread 调用）
        void tick();

        // 触发指定定时器的回调（由 RoomActor 调用）
        void fire(uint64_t timerId);

        [[nodiscard]] std::uint32_t tickDurationMs() const { return tickDurationMs_; }
        [[nodiscard]] std::uint32_t wheelSize() const { return wheelSize_; }

    private:
        struct TimerEntry {
            uint64_t id;
            std::string roomId;
            uint32_t remainingRounds;
            std::function<void()> callback;
            std::atomic<bool> cancelled{false};
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

        ExpiredCallback expiredCallback_;

        // 存储待 fire 的 entry（tick 时收集，fire 时查找）
        mutable std::mutex pendingMutex_;
        std::unordered_map<uint64_t, std::shared_ptr<TimerEntry>> pendingEntries_;
    };

} // namespace infra::util
