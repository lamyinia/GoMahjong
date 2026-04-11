#include "infrastructure/util/string_util.hpp"

#include <sstream>
#include <iomanip>
#include <cctype>
#include <cstdlib>

namespace infra::util {

std::string StringUtil::trim(const std::string& str) {
    return ltrim(rtrim(str));
}

std::string StringUtil::ltrim(const std::string& str) {
    std::size_t start = str.find_first_not_of(" \t\n\r\f\v");
    return (start == std::string::npos) ? "" : str.substr(start);
}

std::string StringUtil::rtrim(const std::string& str) {
    std::size_t end = str.find_last_not_of(" \t\n\r\f\v");
    return (end == std::string::npos) ? "" : str.substr(0, end + 1);
}

std::vector<std::string> StringUtil::split(const std::string& str, char delimiter, std::size_t maxSplit) {
    std::vector<std::string> result;
    std::istringstream iss(str);
    std::string token;
    
    std::size_t count = 0;
    while (std::getline(iss, token, delimiter)) {
        if (maxSplit > 0 && count >= maxSplit) {
            // 剩余部分作为最后一个元素
            std::string remaining;
            std::getline(iss, remaining);
            if (!token.empty() || !remaining.empty()) {
                result.push_back(token + delimiter + remaining);
            }
            break;
        }
        result.push_back(token);
        ++count;
    }
    
    return result;
}

std::vector<std::string> StringUtil::split(const std::string& str, const std::string& delimiter) {
    std::vector<std::string> result;
    
    if (delimiter.empty()) {
        result.push_back(str);
        return result;
    }
    
    std::size_t start = 0;
    std::size_t end = str.find(delimiter);
    
    while (end != std::string::npos) {
        result.push_back(str.substr(start, end - start));
        start = end + delimiter.length();
        end = str.find(delimiter, start);
    }
    
    result.push_back(str.substr(start));
    return result;
}

std::string StringUtil::join(const std::vector<std::string>& items, const std::string& delimiter) {
    if (items.empty()) {
        return "";
    }
    
    std::ostringstream oss;
    oss << items[0];
    for (std::size_t i = 1; i < items.size(); ++i) {
        oss << delimiter << items[i];
    }
    return oss.str();
}

std::string StringUtil::toUpper(std::string str) {
    std::transform(str.begin(), str.end(), str.begin(), ::toupper);
    return str;
}

std::string StringUtil::toLower(std::string str) {
    std::transform(str.begin(), str.end(), str.begin(), ::tolower);
    return str;
}

bool StringUtil::startsWith(const std::string& str, const std::string& prefix) {
    if (prefix.size() > str.size()) {
        return false;
    }
    return str.compare(0, prefix.size(), prefix) == 0;
}

bool StringUtil::endsWith(const std::string& str, const std::string& suffix) {
    if (suffix.size() > str.size()) {
        return false;
    }
    return str.compare(str.size() - suffix.size(), suffix.size(), suffix) == 0;
}

bool StringUtil::contains(const std::string& str, const std::string& substr) {
    return str.find(substr) != std::string::npos;
}

std::string StringUtil::replaceAll(std::string str, const std::string& from, const std::string& to) {
    if (from.empty()) {
        return str;
    }
    
    std::size_t pos = 0;
    while ((pos = str.find(from, pos)) != std::string::npos) {
        str.replace(pos, from.length(), to);
        pos += to.length();
    }
    return str;
}

bool StringUtil::isBlank(const std::string& str) {
    return str.empty() || str.find_first_not_of(" \t\n\r\f\v") == std::string::npos;
}

bool StringUtil::isDigit(const std::string& str) {
    if (str.empty()) {
        return false;
    }
    for (char c : str) {
        if (!std::isdigit(static_cast<unsigned char>(c))) {
            return false;
        }
    }
    return true;
}

std::int32_t StringUtil::toInt32(const std::string& str, std::int32_t defaultValue) {
    try {
        return static_cast<std::int32_t>(std::stoi(str));
    } catch (...) {
        return defaultValue;
    }
}

std::int64_t StringUtil::toInt64(const std::string& str, std::int64_t defaultValue) {
    try {
        return static_cast<std::int64_t>(std::stoll(str));
    } catch (...) {
        return defaultValue;
    }
}

std::uint32_t StringUtil::toUInt32(const std::string& str, std::uint32_t defaultValue) {
    try {
        return static_cast<std::uint32_t>(std::stoul(str));
    } catch (...) {
        return defaultValue;
    }
}

std::uint64_t StringUtil::toUInt64(const std::string& str, std::uint64_t defaultValue) {
    try {
        return static_cast<std::uint64_t>(std::stoull(str));
    } catch (...) {
        return defaultValue;
    }
}

double StringUtil::toDouble(const std::string& str, double defaultValue) {
    try {
        return std::stod(str);
    } catch (...) {
        return defaultValue;
    }
}

std::string StringUtil::toString(std::int32_t value) {
    return std::to_string(value);
}

std::string StringUtil::toString(std::int64_t value) {
    return std::to_string(value);
}

std::string StringUtil::toString(std::uint32_t value) {
    return std::to_string(value);
}

std::string StringUtil::toString(std::uint64_t value) {
    return std::to_string(value);
}

std::string StringUtil::toString(double value, int precision) {
    std::ostringstream oss;
    oss << std::fixed << std::setprecision(precision) << value;
    return oss.str();
}

std::string StringUtil::toHex(const std::string& data) {
    std::ostringstream oss;
    oss << std::hex << std::uppercase << std::setfill('0');
    for (unsigned char c : data) {
        oss << std::setw(2) << static_cast<int>(c);
    }
    return oss.str();
}

std::string StringUtil::fromHex(const std::string& hex) {
    std::string result;
    result.reserve(hex.length() / 2);
    
    for (std::size_t i = 0; i < hex.length(); i += 2) {
        std::string byteStr = hex.substr(i, 2);
        unsigned char byte = static_cast<unsigned char>(std::stoul(byteStr, nullptr, 16));
        result.push_back(byte);
    }
    return result;
}

std::string StringUtil::urlEncode(const std::string& str) {
    std::ostringstream oss;
    oss << std::hex << std::uppercase << std::setfill('0');
    
    for (char c : str) {
        if (std::isalnum(static_cast<unsigned char>(c)) || c == '-' || c == '_' || c == '.' || c == '~') {
            oss << c;
        } else {
            oss << '%' << std::setw(2) << static_cast<int>(static_cast<unsigned char>(c));
        }
    }
    return oss.str();
}

std::string StringUtil::urlDecode(const std::string& str) {
    std::ostringstream oss;
    
    for (std::size_t i = 0; i < str.length(); ++i) {
        if (str[i] == '%' && i + 2 < str.length()) {
            std::string hex = str.substr(i + 1, 2);
            char c = static_cast<char>(std::stoul(hex, nullptr, 16));
            oss << c;
            i += 2;
        } else if (str[i] == '+') {
            oss << ' ';
        } else {
            oss << str[i];
        }
    }
    return oss.str();
}

} // namespace infra::util
