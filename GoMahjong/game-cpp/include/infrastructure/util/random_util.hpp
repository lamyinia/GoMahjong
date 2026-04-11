#pragma once

#include <cstdint>
#include <vector>
#include <random>
#include <algorithm>

namespace infra::util {

/**
 * 随机数工具类
 * 用于游戏服务器的随机数生成、洗牌等操作
 */
class RandomUtil {
public:
    /**
     * 获取单例实例
     */
    static RandomUtil& instance();

    /**
     * 生成随机整数 [min, max]
     * @param min 最小值（包含）
     * @param max 最大值（包含）
     * @return 随机整数
     */
    std::int32_t randomInt(std::int32_t min, std::int32_t max);

    /**
     * 生成随机浮点数 [0.0, 1.0)
     * @return 随机浮点数
     */
    double randomDouble();

    /**
     * 生成随机浮点数 [min, max)
     * @param min 最小值
     * @param max 最大值
     * @return 随机浮点数
     */
    double randomDouble(double min, double max);

    /**
     * 按权重随机选择索引
     * @param weights 权重数组
     * @return 选中的索引，-1 表示失败
     */
    std::int32_t randomByWeight(const std::vector<std::uint32_t>& weights);

    /**
     * 洗牌（Fisher-Yates 算法）
     * @tparam T 元素类型
     * @param items 要洗牌的数组
     */
    template <typename T>
    void shuffle(std::vector<T>& items);

    /**
     * 从数组中随机选择一个元素
     * @tparam T 元素类型
     * @param items 数组
     * @return 随机选择的元素引用
     */
    template <typename T>
    const T& randomChoice(const std::vector<T>& items);

    /**
     * 从数组中随机选择 N 个不重复的元素
     * @tparam T 元素类型
     * @param items 数组
     * @param count 选择数量
     * @return 选中的元素数组
     */
    template <typename T>
    std::vector<T> randomSample(const std::vector<T>& items, std::size_t count);

    /**
     * 设置随机种子
     * @param seed 种子值
     */
    void setSeed(std::uint32_t seed);

private:
    RandomUtil();
    
    // 禁止拷贝
    RandomUtil(const RandomUtil&) = delete;
    RandomUtil& operator=(const RandomUtil&) = delete;

    std::mt19937 generator_;  // Mersenne Twister 随机数生成器
};

// ==================== 模板实现 ====================

template <typename T>
void RandomUtil::shuffle(std::vector<T>& items) {
    std::shuffle(items.begin(), items.end(), generator_);
}

template <typename T>
const T& RandomUtil::randomChoice(const std::vector<T>& items) {
    if (items.empty()) {
        static T empty{};
        return empty;
    }
    std::uniform_int_distribution<std::size_t> dist(0, items.size() - 1);
    return items[dist(generator_)];
}

template <typename T>
std::vector<T> RandomUtil::randomSample(const std::vector<T>& items, std::size_t count) {
    if (count >= items.size()) {
        return items;
    }
    
    std::vector<T> result;
    result.reserve(count);
    
    std::vector<std::size_t> indices(items.size());
    for (std::size_t i = 0; i < items.size(); ++i) {
        indices[i] = i;
    }
    
    shuffle(indices);
    
    for (std::size_t i = 0; i < count; ++i) {
        result.push_back(items[indices[i]]);
    }
    
    return result;
}

} // namespace infra::util
