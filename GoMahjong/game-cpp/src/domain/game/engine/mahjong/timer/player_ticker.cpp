#include "domain/game/engine/mahjong/timer/player_ticker.h"
#include "infrastructure/log/logger.hpp"

#include <algorithm>

namespace domain::game::mahjong::timer {

    PlayerTicker::PlayerTicker(int seatIndex, int totalAvailableTime, infra::util::TimingWheel* wheel)
        : seatIndex_(seatIndex),
          available_(totalAvailableTime),
          state_(TickerState::Idle),
          wheel_(wheel) {
    }

    bool PlayerTicker::start(int duration, TimeoutCallback onTimeout) {
        if (state_ == TickerState::Running) {
            LOG_WARN("[PlayerTicker] seat {} already running", seatIndex_);
            return false;
        }

        if (available_ < duration) {
            LOG_WARN("[PlayerTicker] seat {} available {} < duration {}", seatIndex_, available_, duration);
            return false;
        }

        if (!wheel_) {
            LOG_ERROR("[PlayerTicker] seat {} no timing wheel", seatIndex_);
            return false;
        }

        currentDuration_ = duration;
        roundStartTime_ = std::chrono::steady_clock::now();

        // 调度定时器，传入超时回调（在 TimerThread 线程调用）
        currentHandle_ = wheel_->schedule(
            static_cast<uint64_t>(duration) * 1000,  // 秒 → 毫秒
            std::move(onTimeout)
        );

        transitionTo(TickerState::Running);
        LOG_DEBUG("[PlayerTicker] seat {} started, duration={}s, available={}s", seatIndex_, duration, available_);
        return true;
    }

    bool PlayerTicker::stop() {
        if (state_ != TickerState::Running) {
            return false;
        }

        // 取消定时器
        if (wheel_) {
            wheel_->cancel(currentHandle_);
        }

        // 计算已用时间（毫秒精度，向上取整到秒）
        auto now = std::chrono::steady_clock::now();
        auto usedMs = std::chrono::duration_cast<std::chrono::milliseconds>(now - roundStartTime_).count();
        int usedSeconds = static_cast<int>((usedMs + 999) / 1000);  // 向上取整
        usedSeconds = std::min(usedSeconds, currentDuration_);

        // 扣减剩余时间
        available_ = std::max(0, available_ - usedSeconds);

        transitionTo(TickerState::Stopped);

        if (onStop_) {
            onStop_();
        }

        LOG_DEBUG("[PlayerTicker] seat {} stopped, used={}ms ({}s), available={}s", seatIndex_, usedMs, usedSeconds, available_);
        return true;
    }

    void PlayerTicker::setAvailable(int available) {
        available_ = available;
    }

    void PlayerTicker::setOnStop(StopCallback cb) {
        onStop_ = std::move(cb);
    }

    void PlayerTicker::setOnStateChange(StateChangeCallback cb) {
        onStateChange_ = std::move(cb);
    }

    void PlayerTicker::setTimingWheel(infra::util::TimingWheel* wheel) {
        wheel_ = wheel;
    }

    void PlayerTicker::transitionTo(TickerState newState) {
        auto oldState = state_;
        state_ = newState;
        if (onStateChange_) {
            onStateChange_(oldState, newState);
        }
    }

    void PlayerTicker::onTimerExpired() {
        if (state_ != TickerState::Running) {
            return;  // 可能已被 stop
        }

        available_ = 0;
        transitionTo(TickerState::Timeout);

        LOG_DEBUG("[PlayerTicker] seat {} timeout", seatIndex_);
    }

} // namespace domain::game::engine::mahjong::timer
