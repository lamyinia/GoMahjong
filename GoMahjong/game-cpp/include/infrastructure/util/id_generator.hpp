#pragma once

#include <cstdint>
#include <chrono>
#include <mutex>

namespace infra::util {

/**
 * 雪花算法 ID 生成器
 * 结构：64 位 = 1 位符号位 + 41 位时间戳 + 10 位机器 ID + 12 位序列号
 * 
 * 优点：
 * - 分布式唯一
 * - 趋势递增
 * - 高性能（每毫秒可生成 4096 个 ID）
 */
class IdGenerator {
public:
    /**
     * 获取单例实例
     * @param machineId 机器 ID (0-1023)，默认从配置获取
     */
    static IdGenerator& instance(std::uint16_t machineId = 0);

    /**
     * 生成唯一 ID
     * @return 64 位唯一 ID
     */
    std::uint64_t nextId();

    /**
     * 设置机器 ID
     * @param machineId 机器 ID (0-1023)
     */
    void setMachineId(std::uint16_t machineId);

private:
    IdGenerator(std::uint16_t machineId);
    
    // 禁止拷贝
    IdGenerator(const IdGenerator&) = delete;
    IdGenerator& operator=(const IdGenerator&) = delete;

    std::uint64_t waitNextMillis(std::uint64_t lastTimestamp);

    std::mutex mutex_;
    std::uint16_t machineId_;      // 机器 ID (10 位)
    std::uint64_t sequence_;        // 序列号 (12 位)
    std::uint64_t lastTimestamp_;   // 上次生成 ID 的时间戳

    // 雪花算法常量
    static constexpr std::uint64_t EPOCH = 1700000000000ULL;  // 起始时间戳 (2023-11-15)
    static constexpr std::uint8_t MACHINE_ID_BITS = 10;       // 机器 ID 位数
    static constexpr std::uint8_t SEQUENCE_BITS = 12;         // 序列号位数
    static constexpr std::uint64_t MAX_MACHINE_ID = (1ULL << MACHINE_ID_BITS) - 1;
    static constexpr std::uint64_t MAX_SEQUENCE = (1ULL << SEQUENCE_BITS) - 1;
    static constexpr std::uint8_t MACHINE_ID_SHIFT = SEQUENCE_BITS;
    static constexpr std::uint8_t TIMESTAMP_SHIFT = SEQUENCE_BITS + MACHINE_ID_BITS;
};

} // namespace infra::util
