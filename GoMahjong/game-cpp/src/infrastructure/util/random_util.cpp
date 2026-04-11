#include "infrastructure/util/random_util.hpp"

#include <random>
#include <numeric>

namespace infra::util {

RandomUtil& RandomUtil::instance() {
    static RandomUtil inst;
    return inst;
}

RandomUtil::RandomUtil() {
    // 使用随机设备初始化
    std::random_device rd;
    generator_.seed(rd());
}

std::int32_t RandomUtil::randomInt(std::int32_t min, std::int32_t max) {
    if (min > max) {
        std::swap(min, max);
    }
    std::uniform_int_distribution<std::int32_t> dist(min, max);
    return dist(generator_);
}

double RandomUtil::randomDouble() {
    std::uniform_real_distribution<double> dist(0.0, 1.0);
    return dist(generator_);
}

double RandomUtil::randomDouble(double min, double max) {
    if (min > max) {
        std::swap(min, max);
    }
    std::uniform_real_distribution<double> dist(min, max);
    return dist(generator_);
}

std::int32_t RandomUtil::randomByWeight(const std::vector<std::uint32_t>& weights) {
    if (weights.empty()) {
        return -1;
    }

    // 计算总权重
    std::uint64_t totalWeight = 0;
    for (std::uint32_t w : weights) {
        totalWeight += w;
    }

    if (totalWeight == 0) {
        return -1;
    }

    // 生成随机值
    std::uniform_int_distribution<std::uint64_t> dist(1, totalWeight);
    std::uint64_t randomValue = dist(generator_);

    // 根据权重选择索引
    std::uint64_t cumulative = 0;
    for (std::size_t i = 0; i < weights.size(); ++i) {
        cumulative += weights[i];
        if (randomValue <= cumulative) {
            return static_cast<std::int32_t>(i);
        }
    }

    return static_cast<std::int32_t>(weights.size() - 1);
}

void RandomUtil::setSeed(std::uint32_t seed) {
    generator_.seed(seed);
}

} // namespace infra::util
