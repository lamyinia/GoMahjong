#pragma once

#include <cstdint>
#include <string>
#include <chrono>
#include <ctime>

namespace infra::util {

/**
 * 时间工具类
 * 提供时间戳、格式化、计时等功能
 */
class TimeUtil {
public:
    /**
     * 获取当前时间戳（毫秒）
     * @return 毫秒级时间戳
     */
    static std::uint64_t nowMillis();

    /**
     * 获取当前时间戳（微秒）
     * @return 微秒级时间戳
     */
    static std::uint64_t nowMicros();

    /**
     * 获取当前时间戳（秒）
     * @return 秒级时间戳
     */
    static std::uint64_t nowSeconds();

    /**
     * 格式化时间戳为字符串
     * @param timestamp 毫秒级时间戳
     * @param format 格式字符串（默认 "%Y-%m-%d %H:%M:%S"）
     * @return 格式化后的时间字符串
     */
    static std::string format(std::uint64_t timestamp, const std::string& fmt = "%Y-%m-%d %H:%M:%S");

    /**
     * 解析时间字符串为时间戳
     * @param timeStr 时间字符串
     * @param format 格式字符串
     * @return 毫秒级时间戳，解析失败返回 0
     */
    static std::uint64_t parse(const std::string& timeStr, const std::string& fmt = "%Y-%m-%d %H:%M:%S");

    /**
     * 获取当前日期字符串
     * @return 日期字符串（格式：YYYY-MM-DD）
     */
    static std::string today();

    /**
     * 获取当前时间字符串
     * @return 时间字符串（格式：HH:MM:SS）
     */
    static std::string nowTime();

    /**
     * 计算两个时间戳之间的毫秒数
     * @param start 开始时间戳
     * @param end 结束时间戳
     * @return 毫秒数
     */
    static std::int64_t diffMillis(std::uint64_t start, std::uint64_t end);

    /**
     * 判断时间戳是否过期
     * @param timestamp 毫秒级时间戳
     * @param ttlMs 过期时间（毫秒）
     * @return true 表示已过期
     */
    static bool isExpired(std::uint64_t timestamp, std::uint64_t ttlMs);

    /**
     * 获取今天零点的时间戳
     * @return 毫秒级时间戳
     */
    static std::uint64_t todayZero();

    /**
     * 获取本周一零点的时间戳
     * @return 毫秒级时间戳
     */
    static std::uint64_t weekStart();

    /**
     * 获取本月一号零点的时间戳
     * @return 毫秒级时间戳
     */
    static std::uint64_t monthStart();
};

/**
 * 简单计时器
 * 用于性能测量和超时检测
 */
class Timer {
public:
    Timer();

    /**
     * 重置计时器
     */
    void reset();

    /**
     * 获取经过的毫秒数
     */
    std::uint64_t elapsedMillis() const;

    /**
     * 获取经过的微秒数
     */
    std::uint64_t elapsedMicros() const;

    /**
     * 获取经过的秒数
     */
    double elapsedSeconds() const;

    /**
     * 检查是否超时
     * @param timeoutMs 超时时间（毫秒）
     * @return true 表示已超时
     */
    bool isTimeout(std::uint64_t timeoutMs) const;

private:
    std::chrono::steady_clock::time_point start_;
};

} // namespace infra::util
