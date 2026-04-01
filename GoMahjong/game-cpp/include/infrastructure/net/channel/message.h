#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

namespace infra::net::channel {

    // 前向声明
    class ChannelHandlerContext;

    /**
     * @brief 消息类型
     * 
     * 在 Handler 链中传递的消息对象
     */
    struct Message {
        std::string route;              // 路由（如 "auth.login"）
        std::vector<uint8_t> payload;   // 业务数据
        uint64_t client_seq{0};         // 请求序列号

        // 便捷方法
        bool has_route() const { return !route.empty(); }
        bool has_payload() const { return !payload.empty(); }
    };

    using MessagePtr = std::shared_ptr<Message>;

} // namespace infra::net::channel
