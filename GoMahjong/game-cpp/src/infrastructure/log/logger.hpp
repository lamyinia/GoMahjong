#pragma once

#include <memory>

#include <spdlog/spdlog.h>
#include <spdlog/logger.h>

namespace infra::log {

void init();
std::shared_ptr<spdlog::logger> get();

} // namespace infra::log

#define INFRA_LOG_TRACE(...) SPDLOG_LOGGER_CALL(::infra::log::get(), spdlog::level::trace, __VA_ARGS__)
#define INFRA_LOG_DEBUG(...) SPDLOG_LOGGER_CALL(::infra::log::get(), spdlog::level::debug, __VA_ARGS__)
#define INFRA_LOG_INFO(...) SPDLOG_LOGGER_CALL(::infra::log::get(), spdlog::level::info, __VA_ARGS__)
#define INFRA_LOG_WARN(...) SPDLOG_LOGGER_CALL(::infra::log::get(), spdlog::level::warn, __VA_ARGS__)
#define INFRA_LOG_ERROR(...) SPDLOG_LOGGER_CALL(::infra::log::get(), spdlog::level::err, __VA_ARGS__)
#define INFRA_LOG_CRITICAL(...) SPDLOG_LOGGER_CALL(::infra::log::get(), spdlog::level::critical, __VA_ARGS__)

#define LOG_TRACE(...) INFRA_LOG_TRACE(__VA_ARGS__)
#define LOG_DEBUG(...) INFRA_LOG_DEBUG(__VA_ARGS__)
#define LOG_INFO(...) INFRA_LOG_INFO(__VA_ARGS__)
#define LOG_WARN(...) INFRA_LOG_WARN(__VA_ARGS__)
#define LOG_ERROR(...) INFRA_LOG_ERROR(__VA_ARGS__)
#define LOG_CRITICAL(...) INFRA_LOG_CRITICAL(__VA_ARGS__)
