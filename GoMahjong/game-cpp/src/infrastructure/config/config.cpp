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
        const auto& server = j["server"];

        // Parse net.tcp
        if (server.contains("net") && server["net"].contains("tcp")) {
            const auto& tcp = server["net"]["tcp"];
            if (tcp.contains("port")) {
                cfg.server_.net.tcp.port = tcp.at("port").get<std::uint16_t>();
            }
            if (tcp.contains("max_frame_bytes")) {
                cfg.server_.net.tcp.max_frame_bytes = tcp.at("max_frame_bytes").get<std::uint32_t>();
            }
            if (tcp.contains("idle_timeout_seconds")) {
                cfg.server_.net.tcp.idle_timeout_seconds = tcp.at("idle_timeout_seconds").get<std::uint32_t>();
            }
        }

        // Parse log (reserved for future)
        if (server.contains("log")) {
            // Future: parse log configuration
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
