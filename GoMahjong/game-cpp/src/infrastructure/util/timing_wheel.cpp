#include "infrastructure/util/timing_wheel.h"
#include "infrastructure/log/logger.hpp"

#include <algorithm>

namespace infra::util {

    TimingWheel::TimingWheel(std::uint32_t tickDurationMs, std::uint32_t wheelSize)
        : tickDurationMs_(tickDurationMs),
          wheelSize_(wheelSize),
          slots_(wheelSize) {
    }

    void TimingWheel::setExpiredCallback(ExpiredCallback cb) {
        expiredCallback_ = std::move(cb);
    }

    TimerHandle TimingWheel::schedule(std::uint64_t delayMs,
                                       const std::string& roomId,
                                       std::function<void()> cb) {
        auto id = nextTimerId_.fetch_add(1, std::memory_order_relaxed);

        // 计算目标 slot 和轮数
        auto totalTicks = static_cast<std::uint32_t>(
            std::max(std::uint64_t(1), delayMs / tickDurationMs_));
        std::uint32_t rounds = totalTicks / wheelSize_;
        std::uint32_t slotOffset = totalTicks % wheelSize_;
        std::uint32_t targetSlot = (currentSlot_.load(std::memory_order_relaxed) + slotOffset) % wheelSize_;

        auto entry = std::make_shared<TimerEntry>();
        entry->id = id;
        entry->roomId = roomId;
        entry->remainingRounds = rounds;
        entry->callback = std::move(cb);

        // 存入 pending map（供 fire 查找）
        {
            std::lock_guard lock(pendingMutex_);
            pendingEntries_[id] = entry;
        }

        // 插入目标 slot
        {
            std::lock_guard lock(slots_[targetSlot].mutex);
            slots_[targetSlot].entries.push_front(entry);
        }

        LOG_TRACE("scheduled timer {} in room {}, delay={}ms, slot={}, rounds={}",
                  id, roomId, delayMs, targetSlot, rounds);

        return TimerHandle{id};
    }

    void TimingWheel::cancel(const TimerHandle& handle) {
        // 标记取消（延迟删除，tick 时清理）
        std::shared_ptr<TimerEntry> entry;
        {
            std::lock_guard lock(pendingMutex_);
            auto it = pendingEntries_.find(handle.id);
            if (it != pendingEntries_.end()) {
                entry = it->second;
                pendingEntries_.erase(it);
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

        // 通知到期（不在锁内）
        for (auto& entry : expired) {
            LOG_TRACE("timer {} in room {} expired", entry->id, entry->roomId);

            if (expiredCallback_) {
                expiredCallback_(entry->roomId, entry->id);
            }
        }
    }

    void TimingWheel::fire(uint64_t timerId) {
        std::shared_ptr<TimerEntry> entry;
        {
            std::lock_guard lock(pendingMutex_);
            auto it = pendingEntries_.find(timerId);
            if (it != pendingEntries_.end()) {
                entry = it->second;
                pendingEntries_.erase(it);
            }
        }

        if (entry && !entry->cancelled.load(std::memory_order_relaxed)) {
            if (entry->callback) {
                entry->callback();
            }
        }
    }

} // namespace infra::util
