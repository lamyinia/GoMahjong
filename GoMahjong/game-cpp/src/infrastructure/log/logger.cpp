#include "infrastructure/log/logger.hpp"

#include <spdlog/sinks/stdout_color_sinks.h>
#include <spdlog/spdlog.h>

#include "infrastructure/config/config.hpp"

namespace infra::log {

static std::shared_ptr<spdlog::logger> g_logger;

static spdlog::level::level_enum parse_level(const std::string& level) {
    if (level == "trace") return spdlog::level::trace;
    if (level == "debug") return spdlog::level::debug;
    if (level == "info") return spdlog::level::info;
    if (level == "warn") return spdlog::level::warn;
    if (level == "error") return spdlog::level::err;
    if (level == "critical") return spdlog::level::critical;
    if (level == "off") return spdlog::level::off;
    return spdlog::level::info;  // default
}

void init(const config::LogConfig& cfg) {
    if (g_logger) {
        return;
    }
    g_logger = spdlog::get("gomahjong");
    if (g_logger) {
        return;
    }
    // 全局注册表中也不存在，创建新的
    // stdout_color_mt 会自动注册到全局注册表，如果已存在会抛异常
    try {
        g_logger = spdlog::stdout_color_mt("gomahjong");
        g_logger->set_pattern("[%Y-%m-%d %H:%M:%S.%e] [%^%l%$] [%t] [%s:%# %!] %v");
        g_logger->set_level(parse_level(cfg.level));
    } catch (const spdlog::spdlog_ex&) {
        // 如果抛异常说明其他线程刚创建了，再次获取
        g_logger = spdlog::get("gomahjong");
    }
}

std::shared_ptr<spdlog::logger> get() {
    if (!g_logger) {
        init(config::LogConfig{});  // use default config (info level)
    }
    return g_logger;
}

} // namespace infra::log
