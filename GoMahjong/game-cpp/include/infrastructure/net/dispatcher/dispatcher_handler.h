#pragma once

#include "infrastructure/net/channel/i_channel_handler.h"
#include "infrastructure/net/channel/message.h"

#include <functional>
#include <memory>
#include <string>
#include <unordered_map>
#include <mutex>
#include <atomic>

namespace infra::net::dispatcher {

    namespace channel = infra::net::channel;

    /**
     * @brief 业务 Handler 类型
     * @param ctx Handler 上下文
     * @param msg 消息
     */
    using BusinessHandler = std::function<void(channel::ChannelHandlerContext& ctx, 
                                                const channel::MessagePtr& msg)>;

    /**
     * @brief 消息分发器
     * 
     * 负责：
     * - 检查 Channel 是否已授权
     * - 根据 route 分发消息到对应的业务 Handler
     * - 支持动态注册 Handler
     */
    class DispatcherHandler : public channel::ChannelInboundHandler {
    public:
        DispatcherHandler() = default;

        // === Handler 注册 ===

        /**
         * @brief 注册业务 Handler
         * @param route 路由字符串（如 "game.playTile"）
         * @param handler 处理函数
         */
        void register_handler(const std::string& route, BusinessHandler handler);

        /**
         * @brief 移除 Handler
         */
        void unregister_handler(const std::string& route);

        /**
         * @brief 检查是否有对应的 Handler
         */
        bool has_handler(const std::string& route) const;

        /**
         * @brief 标记初始化完成，之后查找不再加锁
         * 
         * 调用此方法后，handlers_ 不应再被修改
         */
        void mark_initialized();

        /**
         * @brief 检查是否已初始化完成
         */
        bool is_initialized() const;

        // === ChannelInboundHandler 接口 ===

        void channel_read(channel::ChannelHandlerContext& ctx, 
                          channel::InboundMessage&& msg) override;

    private:
        void handle_authorized_message(channel::ChannelHandlerContext& ctx,
                                        const channel::MessagePtr& msg);

        void handle_unauthorized_message(channel::ChannelHandlerContext& ctx,
                                          const channel::MessagePtr& msg);

        mutable std::mutex mutex_;
        std::unordered_map<std::string, BusinessHandler> handlers_;
        std::atomic<bool> initialized_{false};  // 初始化完成后，查找无需加锁
    };

} // namespace infra::net::dispatcher
