#include "infrastructure/util/timing_wheel.h"
#include "infrastructure/log/logger.hpp"

#include <algorithm>
#include <unordered_map>

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

        // 插入目标 slot + map
        {
            std::lock_guard lock(slots_[targetSlot].mutex);
            slots_[targetSlot].entries.push_front(entry);
        }
        {
            std::lock_guard lock(mapMutex_);
            timerMap_[id] = entry;
        }

        LOG_TRACE("scheduled timer {}, delay={}ms, slot={}, rounds={}",
                  id, delayMs, targetSlot, rounds);

        return TimerHandle{id};
    }

    void TimingWheel::cancel(const TimerHandle& handle) {
        // O(1) cancel：通过 map 直接查找 entry 指针
        std::shared_ptr<TimerEntry> entry;
        {
            std::lock_guard lock(mapMutex_);
            auto it = timerMap_.find(handle.id);
            if (it != timerMap_.end()) {
                entry = it->second;
                timerMap_.erase(it);
            }
        }
        if (entry) {
            entry->cancelled.store(true, std::memory_order_relaxed);
            LOG_TRACE("cancelled timer {}", handle.id);
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
                    auto id = entry->id;  // 捕获 id，erase 后 entry 引用失效
                    it = slot.entries.erase_after(prev);
                    // 从 map 中同步清理
                    {
                        std::lock_guard mapLock(mapMutex_);
                        timerMap_.erase(id);
                    }
                    continue;
                }

                if (entry->remainingRounds > 0) {
                    entry->remainingRounds--;
                    prev = it;
                    ++it;
                } else {
                    // 到期
                    auto id = entry->id;  // 捕获 id，erase 后 entry 引用失效
                    expired.push_back(entry);
                    it = slot.entries.erase_after(prev);
                    // 从 map 中同步清理
                    {
                        std::lock_guard mapLock(mapMutex_);
                        timerMap_.erase(id);
                    }
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
