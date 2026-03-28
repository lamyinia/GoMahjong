#include "logger.hpp"

#include <spdlog/sinks/stdout_color_sinks.h>
#include <spdlog/spdlog.h>

namespace infra::log {

static std::shared_ptr<spdlog::logger> g_logger;

void init() {
    if (g_logger) {
        return;
    }

    g_logger = spdlog::stdout_color_mt("gomahjong");
    g_logger->set_pattern("[%Y-%m-%d %H:%M:%S.%e] [%^%l%$] [%t] [%s:%# %!] %v");
    g_logger->set_level(spdlog::level::info);
}

std::shared_ptr<spdlog::logger> get() {
    if (!g_logger) {
        init();
    }
    return g_logger;
}

} // namespace infra::log
