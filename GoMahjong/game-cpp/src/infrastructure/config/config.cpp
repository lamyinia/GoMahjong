#include "config.hpp"

#include <fstream>
#include <stdexcept>

#include <nlohmann/json.hpp>

namespace infra::config {

Config Config::load_from_file(const std::string& path) {
    std::ifstream ifs(path);
    if (!ifs.is_open()) {
        throw std::runtime_error("open config failed: " + path);
    }

    nlohmann::json j;
    ifs >> j;

    Config cfg;

    if (j.contains("server")) {
        const auto& s = j.at("server");
        if (s.contains("tcp_port")) {
            cfg.server_.tcp_port = s.at("tcp_port").get<std::uint16_t>();
        }
        if (s.contains("max_frame_bytes")) {
            cfg.server_.max_frame_bytes = s.at("max_frame_bytes").get<std::uint32_t>();
        }
        if (s.contains("idle_timeout_seconds")) {
            cfg.server_.idle_timeout_seconds = s.at("idle_timeout_seconds").get<std::uint32_t>();
        }
    }

    return cfg;
}

Config Config::load_from_file_or_default(const std::string& path) noexcept {
    try {
        return load_from_file(path);
    } catch (...) {
        return Config{};
    }
}

} // namespace infra::config
