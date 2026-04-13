#include "infrastructure/util/timing_wheel.h"
#include "infrastructure/log/logger.hpp"

#include <algorithm>

namespace infra::util {

    TimingWheel::TimingWheel(std::uint32_t tickDurationMs, std::uint32_t wheelSize)
        : tickDurationMs_(tickDurationMs),
          wheelSize_(wheelSize),
          slots_(wheelSize) {
    }

    TimerHandle TimingWheel::schedule(std::uint64_t delayMs, TimerCallback callback) {
        auto id = nextTimerId_.fetch_add(1, std::memory_order_relaxed);

        // 计算目标 slot 和轮数
        auto totalTicks = static_cast<std::uint32_t>(
            std::max(std::uint64_t(1), delayMs / tickDurationMs_));
        std::uint32_t rounds = totalTicks / wheelSize_;
        std::uint32_t slotOffset = totalTicks % wheelSize_;
        std::uint32_t targetSlot = (currentSlot_.load(std::memory_order_relaxed) + slotOffset) % wheelSize_;

        auto entry = std::make_shared<TimerEntry>();
        entry->id = id;
        entry->remainingRounds = rounds;
        entry->callback = std::move(callback);

        // 插入目标 slot
        {
            std::lock_guard lock(slots_[targetSlot].mutex);
            slots_[targetSlot].entries.push_front(entry);
        }

        LOG_TRACE("scheduled timer {}, delay={}ms, slot={}, rounds={}",
                  id, delayMs, targetSlot, rounds);

        return TimerHandle{id};
    }

    void TimingWheel::cancel(const TimerHandle& handle) {
        // 标记取消（延迟删除，tick 时清理）
        // 遍历所有 slot 查找并标记（cancel 不高频，可接受）
        for (auto& slot : slots_) {
            std::lock_guard lock(slot.mutex);
            for (auto& entry : slot.entries) {
                if (entry->id == handle.id) {
                    entry->cancelled.store(true, std::memory_order_relaxed);
                    LOG_TRACE("cancelled timer {}", handle.id);
                    return;
                }
            }
        }
    }

    void TimingWheel::tick() {
        auto slotIndex = currentSlot_.fetch_add(1, std::memory_order_relaxed) % wheelSize_;
        auto& slot = slots_[slotIndex];

        std::vector<std::shared_ptr<TimerEntry>> expired;

        {
            std::lock_guard lock(slot.mutex);
            auto prev = slot.entries.before_begin();
            auto it = slot.entries.begin();
            while (it != slot.entries.end()) {
                auto& entry = *it;

                // 已取消，删除
                if (entry->cancelled.load(std::memory_order_relaxed)) {
                    it = slot.entries.erase_after(prev);
                    continue;
                }

                if (entry->remainingRounds > 0) {
                    entry->remainingRounds--;
                    prev = it;
                    ++it;
                } else {
                    // 到期
                    expired.push_back(entry);
                    it = slot.entries.erase_after(prev);
                }
            }
        }

        // 通知到期（不在 slot 锁内）
        // 检查 cancelled 标志：cancel 可能在此期间被调用
        for (auto& entry : expired) {
            if (entry->cancelled.load(std::memory_order_relaxed)) {
                LOG_TRACE("timer {} expired but cancelled, skipping", entry->id);
                continue;
            }
            LOG_TRACE("timer {} expired, invoking callback", entry->id);
            entry->callback();
        }
    }

} // namespace infra::util
