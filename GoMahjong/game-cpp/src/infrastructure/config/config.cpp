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

        // Parse etcd
        if (server.contains("etcd")) {
            const auto& etcd = server["etcd"];
            if (etcd.contains("endpoints")) {
                cfg.server_.etcd.endpoints = etcd.at("endpoints").get<std::string>();
            }
            if (etcd.contains("ttl_seconds")) {
                cfg.server_.etcd.ttl_seconds = etcd.at("ttl_seconds").get<std::int64_t>();
            }
        }

        // Parse grpc
        if (server.contains("grpc")) {
            const auto& grpc = server["grpc"];
            if (grpc.contains("port")) {
                cfg.server_.grpc.port = grpc.at("port").get<std::uint16_t>();
            }
        }

        // Parse mongodb
        if (server.contains("mongodb")) {
            const auto& mongodb = server["mongodb"];
            if (mongodb.contains("uri")) {
                cfg.server_.mongodb.uri = mongodb.at("uri").get<std::string>();
            }
            if (mongodb.contains("database")) {
                cfg.server_.mongodb.database = mongodb.at("database").get<std::string>();
            }
            if (mongodb.contains("min_pool_size")) {
                cfg.server_.mongodb.min_pool_size = mongodb.at("min_pool_size").get<std::uint32_t>();
            }
            if (mongodb.contains("max_pool_size")) {
                cfg.server_.mongodb.max_pool_size = mongodb.at("max_pool_size").get<std::uint32_t>();
            }
            if (mongodb.contains("username")) {
                cfg.server_.mongodb.username = mongodb.at("username").get<std::string>();
            }
            if (mongodb.contains("password")) {
                cfg.server_.mongodb.password = mongodb.at("password").get<std::string>();
            }
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
