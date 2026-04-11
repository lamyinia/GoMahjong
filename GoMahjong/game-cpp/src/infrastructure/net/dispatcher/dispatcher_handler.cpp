#include "infrastructure/net/dispatcher/dispatcher_handler.h"
#include "infrastructure/log/logger.hpp"

namespace infra::net::dispatcher {

    void DispatcherHandler::register_handler(const std::string& route, BusinessHandler handler) {
        std::lock_guard<std::mutex> lock(mutex_);
        handlers_[route] = std::move(handler);
        LOG_DEBUG("registered 处理器 for 路由: {}", route);
    }

    void DispatcherHandler::unregister_handler(const std::string& route) {
        std::lock_guard<std::mutex> lock(mutex_);
        handlers_.erase(route);
        LOG_DEBUG("unregistered 处理器 for 路由: {}", route);
    }

    bool DispatcherHandler::has_handler(const std::string& route) const {
        // 初始化完成后无需加锁
        if (initialized_.load(std::memory_order_acquire)) {
            return handlers_.find(route) != handlers_.end();
        }
        std::lock_guard<std::mutex> lock(mutex_);
        return handlers_.find(route) != handlers_.end();
    }

    void DispatcherHandler::mark_initialized() {
        initialized_.store(true, std::memory_order_release);
        LOG_DEBUG("initialized, lock-free lookup enabled");
    }

    bool DispatcherHandler::is_initialized() const {
        return initialized_.load(std::memory_order_acquire);
    }

    void DispatcherHandler::channel_read(channel::ChannelHandlerContext& ctx, 
                                          channel::InboundMessage&& msg) {
        // 检查是否是 MessagePtr
        if (!std::holds_alternative<channel::MessagePtr>(msg)) {
            ctx.fire_channel_read(std::move(msg));
            return;
        }

        auto& message = std::get<channel::MessagePtr>(msg);
        if (!message) {
            ctx.fire_channel_read(std::move(msg));
            return;
        }

        // 检查授权状态
        if (ctx.is_authorized()) {
            handle_authorized_message(ctx, message);
        } else {
            handle_unauthorized_message(ctx, message);
        }
    }

    void DispatcherHandler::handle_authorized_message(channel::ChannelHandlerContext& ctx,
                                                       const channel::MessagePtr& msg) {
        if (!msg->has_route()) {
            LOG_WARN("message without route from player {}", ctx.player_id());
            send_error_response(ctx, msg, "missing route");
            return;
        }

        // 查找 Handler（初始化完成后无需加锁）
        BusinessHandler handler;
        if (initialized_.load(std::memory_order_acquire)) {
            // 无锁查找
            auto it = handlers_.find(msg->route);
            if (it == handlers_.end()) {
                LOG_WARN("no handler for route: {} from player {}",
                         msg->route, ctx.player_id());
                send_error_response(ctx, msg, "unknown route");
                return;
            }
            handler = it->second;
        } else {
            // 初始化期间加锁查找
            std::lock_guard<std::mutex> lock(mutex_);
            auto it = handlers_.find(msg->route);
            if (it == handlers_.end()) {
                LOG_WARN("no handler for route: {} from player {}",
                         msg->route, ctx.player_id());
                send_error_response(ctx, msg, "unknown route");
                return;
            }
            handler = it->second;
        }

        // 调用 Handler
        try {
            handler(ctx, msg);
        } catch (const std::exception& e) {
            LOG_ERROR("handler exception for route {}: {}", msg->route, e.what());
            send_error_response(ctx, msg, "internal error");
        }
    }

    void DispatcherHandler::handle_unauthorized_message(channel::ChannelHandlerContext& ctx,
                                                         const channel::MessagePtr& msg) {
        // 未授权的连接只能访问 auth.* 路由
        if (!msg->has_route()) {
            LOG_WARN("[Dispatcher] unauthorized message without route");
            ctx.fire_close();
            return;
        }

        // 检查是否是认证相关路由
        if (msg->route.rfind("auth.", 0) == 0) {
            // 认证消息继续传播（可能还有其他 Handler 处理）
            ctx.fire_channel_read(channel::MessagePtr(msg));
            return;
        }

        LOG_WARN("unauthorized access to route: {}", msg->route);
        send_error_response(ctx, msg, "unauthorized");
        ctx.fire_close();
    }

    void DispatcherHandler::send_error_response(channel::ChannelHandlerContext& ctx,
                                                 const channel::MessagePtr& msg,
                                                 const std::string& error) {
        auto resp_msg = std::make_shared<channel::Message>();
        resp_msg->route = msg->route + ".response";
        resp_msg->client_seq = msg->client_seq;
        
        // 简单错误响应（后续可以用 protobuf）
        std::string error_payload = R"({"error":")" + error + R"("})";
        resp_msg->payload.assign(error_payload.begin(), error_payload.end());

        ctx.fire_write(channel::MessagePtr(std::move(resp_msg)));
        ctx.fire_flush();
    }

} // namespace infra::net::dispatcher
