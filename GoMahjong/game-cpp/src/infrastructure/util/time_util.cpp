#include "infrastructure/util/time_util.hpp"

#include <ctime>
#include <sstream>

namespace infra::util {

std::uint64_t TimeUtil::nowMillis() {
    return static_cast<std::uint64_t>(
        std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count()
    );
}

std::uint64_t TimeUtil::nowMicros() {
    return static_cast<std::uint64_t>(
        std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count()
    );
}

std::uint64_t TimeUtil::nowSeconds() {
    return static_cast<std::uint64_t>(
        std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count()
    );
}

std::string TimeUtil::format(std::uint64_t timestamp, const std::string& fmt) {
    auto time = static_cast<std::time_t>(timestamp / 1000);
    std::tm* tm = std::localtime(&time);

    char buffer[128] = {0};
    if (tm == nullptr) {
        return "";
    }

    if (std::strftime(buffer, sizeof(buffer), fmt.c_str(), tm) == 0) {
        return "";
    }
    return std::string(buffer);
}

std::uint64_t TimeUtil::parse(const std::string& timeStr, const std::string& fmt) {
    std::tm tm = {};

    if (::strptime(timeStr.c_str(), fmt.c_str(), &tm) == nullptr) {
        return 0;
    }

    std::time_t time = std::mktime(&tm);
    return static_cast<std::uint64_t>(time) * 1000;
}

std::string TimeUtil::today() {
    return format(nowMillis(), "%Y-%m-%d");
}

std::string TimeUtil::nowTime() {
    return format(nowMillis(), "%H:%M:%S");
}

std::int64_t TimeUtil::diffMillis(std::uint64_t start, std::uint64_t end) {
    return static_cast<std::int64_t>(end) - static_cast<std::int64_t>(start);
}

bool TimeUtil::isExpired(std::uint64_t timestamp, std::uint64_t ttlMs) {
    return nowMillis() > timestamp + ttlMs;
}

std::uint64_t TimeUtil::todayZero() {
    std::time_t now = std::time(nullptr);
    std::tm* tm = std::localtime(&now);
    tm->tm_hour = 0;
    tm->tm_min = 0;
    tm->tm_sec = 0;
    return static_cast<std::uint64_t>(std::mktime(tm)) * 1000;
}

std::uint64_t TimeUtil::weekStart() {
    std::time_t now = std::time(nullptr);
    std::tm* tm = std::localtime(&now);
    
    // 计算本周一
    int dayOfWeek = tm->tm_wday;
    if (dayOfWeek == 0) dayOfWeek = 7;  // Sunday = 7
    tm->tm_hour = 0;
    tm->tm_min = 0;
    tm->tm_sec = 0;
    tm->tm_mday -= (dayOfWeek - 1);
    
    return static_cast<std::uint64_t>(std::mktime(tm)) * 1000;
}

std::uint64_t TimeUtil::monthStart() {
    std::time_t now = std::time(nullptr);
    std::tm* tm = std::localtime(&now);
    tm->tm_hour = 0;
    tm->tm_min = 0;
    tm->tm_sec = 0;
    tm->tm_mday = 1;
    
    return static_cast<std::uint64_t>(std::mktime(tm)) * 1000;
}

Timer::Timer() : start_(std::chrono::steady_clock::now()) {}

void Timer::reset() {
    start_ = std::chrono::steady_clock::now();
}

std::uint64_t Timer::elapsedMillis() const {
    return static_cast<std::uint64_t>(
        std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::steady_clock::now() - start_
        ).count()
    );
}

std::uint64_t Timer::elapsedMicros() const {
    return static_cast<std::uint64_t>(
        std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::steady_clock::now() - start_
        ).count()
    );
}

double Timer::elapsedSeconds() const {
    return std::chrono::duration<double>(
        std::chrono::steady_clock::now() - start_
    ).count();
}

bool Timer::isTimeout(std::uint64_t timeoutMs) const {
    return elapsedMillis() >= timeoutMs;
}

} // namespace infra::util
