#include "infrastructure/util/id_generator.hpp"
 #include "infrastructure/util/time_util.hpp"

#include <stdexcept>

namespace infra::util {

IdGenerator& IdGenerator::instance(std::uint16_t machineId) {
    static IdGenerator inst(machineId);
    return inst;
}

IdGenerator::IdGenerator(std::uint16_t machineId)
    : machineId_(machineId), sequence_(0), lastTimestamp_(0) {
    if (machineId > MAX_MACHINE_ID) {
        throw std::invalid_argument("machine id must be between 0 and 1023");
    }
}

void IdGenerator::setMachineId(std::uint16_t machineId) {
    std::lock_guard<std::mutex> lock(mutex_);
    if (machineId > MAX_MACHINE_ID) {
        throw std::invalid_argument("machine id must be between 0 and 1023");
    }
    machineId_ = machineId;
}

std::uint64_t IdGenerator::nextId() {
    std::lock_guard<std::mutex> lock(mutex_);

    std::uint64_t timestamp = TimeUtil::nowMillis();

    // 时钟回拨检测
    if (timestamp < lastTimestamp_) {
        throw std::runtime_error("clock moved backwards");
    }

    // 同一毫秒内
    if (timestamp == lastTimestamp_) {
        sequence_ = (sequence_ + 1) & MAX_SEQUENCE;
        // 序列号溢出，等待下一毫秒
        if (sequence_ == 0) {
            timestamp = waitNextMillis(lastTimestamp_);
        }
    } else {
        // 不同毫秒，序列号重置
        sequence_ = 0;
    }

    lastTimestamp_ = timestamp;

    // 组装 ID
    return ((timestamp - EPOCH) << TIMESTAMP_SHIFT)
         | (static_cast<std::uint64_t>(machineId_) << MACHINE_ID_SHIFT)
         | sequence_;
}

std::uint64_t IdGenerator::waitNextMillis(std::uint64_t lastTimestamp) {
    std::uint64_t timestamp = TimeUtil::nowMillis();
    while (timestamp <= lastTimestamp) {
        timestamp = TimeUtil::nowMillis();
    }
    return timestamp;
}

} // namespace infra::util
