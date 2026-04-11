#pragma once

#include <cstdint>
#include <cstdio>
#include <string>
#include <vector>
#include <sstream>
#include <algorithm>
#include <cctype>
#include <iomanip>

namespace infra::util {

/**
 * 字符串工具类
 */
class StringUtil {
public:
    /**
     * 去除字符串两端的空白字符
     */
    static std::string trim(const std::string& str);

    /**
     * 去除字符串左端的空白字符
     */
    static std::string ltrim(const std::string& str);

    /**
     * 去除字符串右端的空白字符
     */
    static std::string rtrim(const std::string& str);

    /**
     * 分割字符串
     * @param str 要分割的字符串
     * @param delimiter 分隔符
     * @param maxSplit 最大分割次数（0 表示不限制）
     * @return 分割后的字符串数组
     */
    static std::vector<std::string> split(const std::string& str, char delimiter, std::size_t maxSplit = 0);

    /**
     * 分割字符串（支持多字符分隔符）
     */
    static std::vector<std::string> split(const std::string& str, const std::string& delimiter);

    /**
     * 连接字符串数组
     * @param items 字符串数组
     * @param delimiter 分隔符
     * @return 连接后的字符串
     */
    static std::string join(const std::vector<std::string>& items, const std::string& delimiter);

    /**
     * 转换为大写
     */
    static std::string toUpper(std::string str);

    /**
     * 转换为小写
     */
    static std::string toLower(std::string str);

    /**
     * 判断是否以指定前缀开头
     */
    static bool startsWith(const std::string& str, const std::string& prefix);

    /**
     * 判断是否以指定后缀结尾
     */
    static bool endsWith(const std::string& str, const std::string& suffix);

    /**
     * 判断是否包含子串
     */
    static bool contains(const std::string& str, const std::string& substr);

    /**
     * 替换所有匹配的子串
     * @param str 原字符串
     * @param from 要替换的子串
     * @param to 替换后的子串
     * @return 替换后的字符串
     */
    static std::string replaceAll(std::string str, const std::string& from, const std::string& to);

    /**
     * 判断字符串是否为空或全是空白字符
     */
    static bool isBlank(const std::string& str);

    /**
     * 判断字符串是否为数字
     */
    static bool isDigit(const std::string& str);

    /**
     * 格式化字符串（类似 printf）
     */
    template <typename... Args>
    static std::string format(const std::string& fmt, Args... args);

    /**
     * 字符串转数字
     * @param str 字符串
     * @param defaultValue 转换失败时的默认值
     * @return 转换后的数字
     */
    static std::int32_t toInt32(const std::string& str, std::int32_t defaultValue = 0);
    static std::int64_t toInt64(const std::string& str, std::int64_t defaultValue = 0);
    static std::uint32_t toUInt32(const std::string& str, std::uint32_t defaultValue = 0);
    static std::uint64_t toUInt64(const std::string& str, std::uint64_t defaultValue = 0);
    static double toDouble(const std::string& str, double defaultValue = 0.0);

    /**
     * 数字转字符串
     */
    static std::string toString(std::int32_t value);
    static std::string toString(std::int64_t value);
    static std::string toString(std::uint32_t value);
    static std::string toString(std::uint64_t value);
    static std::string toString(double value, int precision = 6);

    /**
     * 十六进制编码/解码
     */
    static std::string toHex(const std::string& data);
    static std::string fromHex(const std::string& hex);

    /**
     * URL 编码/解码
     */
    static std::string urlEncode(const std::string& str);
    static std::string urlDecode(const std::string& str);
};

// ==================== 模板实现 ====================

template <typename... Args>
std::string StringUtil::format(const std::string& fmt, Args... args) {
    std::ostringstream oss;
    // 简化实现，完整实现需要更复杂的格式化逻辑
    // 这里使用 snprintf 风格
    std::size_t size = snprintf(nullptr, 0, fmt.c_str(), args...) + 1;
    std::vector<char> buffer(size);
    snprintf(buffer.data(), size, fmt.c_str(), args...);
    return std::string(buffer.data(), buffer.data() + size - 1);
}

} // namespace infra::util
