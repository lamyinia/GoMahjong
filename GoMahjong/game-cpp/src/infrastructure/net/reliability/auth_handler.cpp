#include "infrastructure/net/reliability/auth_handler.h"
#include "infrastructure/net/reliability/wild_endpoint.h"
#include "infrastructure/log/logger.hpp"

#include "auth.pb.h"

namespace infra::net::reliability {

    AuthHandler::AuthHandler(std::shared_ptr<WildEndpoint> endpoint)
        : endpoint_(std::move(endpoint)) {
    }

    void AuthHandler::channel_read(channel::ChannelHandlerContext& ctx, channel::InboundMessage&& msg) {
        if (!std::holds_alternative<channel::MessagePtr>(msg)) {
            ctx.fire_channel_read(std::move(msg));
            return;
        }

        auto& message = std::get<channel::MessagePtr>(msg);
        if (!message) {
            ctx.fire_channel_read(std::move(msg));
            return;
        }

        if (!ctx.is_authorized() && message->route == "auth.login") {
            handle_auth_request(ctx, message);
            return;
        }

        ctx.fire_channel_read(std::move(msg));
    }

    void AuthHandler::handle_auth_request(channel::ChannelHandlerContext& ctx, 
                                          const channel::MessagePtr& msg) {
        // 解析 AuthRequest
        gomahjong::net::AuthRequest auth_req;
        if (!auth_req.ParseFromArray(msg->payload.data(), static_cast<int>(msg->payload.size()))) {
            LOG_WARN("[AuthHandler] failed to parse AuthRequest");
            
            // 发送失败响应
            gomahjong::net::AuthResponse resp;
            resp.set_success(false);
            resp.set_message("invalid request format");

            auto resp_msg = std::make_shared<channel::Message>();
            resp_msg->route = "auth.login.response";
            auto payload = resp.SerializeAsString();
            resp_msg->payload.assign(payload.begin(), payload.end());
            resp_msg->client_seq = msg->client_seq;

            ctx.fire_write(channel::MessagePtr(std::move(resp_msg)));
            ctx.fire_flush();

            if (auto endpoint = endpoint_.lock()) {
                endpoint->on_auth_failed();
            }
            return;
        }

        LOG_DEBUG("auth request, token={}", auth_req.token().substr(0, 16));

        std::string player_id;
        bool verified = false;

        // 简化验证逻辑
        auto pos = auth_req.token().find(':');
        if (pos != std::string::npos && auth_req.token().length() > pos + 1) {
            player_id = auth_req.token().substr(0, pos);
            verified = !player_id.empty();
            if (verified) {
                size_t digit_start = player_id.find_last_not_of("0123456789");
                if (digit_start != std::string::npos) {
                    player_id = player_id.substr(digit_start + 1);
                }
                verified = !player_id.empty();
            }
        }

        // 发送响应
        gomahjong::net::AuthResponse resp;
        resp.set_success(verified);

        if (verified) {
            resp.set_pid(std::stoull(player_id));
            resp.set_message("ok");
            ctx.set_authorized(player_id);
            
            LOG_INFO("[AuthHandler] auth success, player_id={}", player_id);
        } else {
            resp.set_message("invalid token");
            LOG_WARN("[AuthHandler] auth failed, invalid token");
        }

        auto resp_msg = std::make_shared<channel::Message>();
        resp_msg->route = "auth.login.response";
        auto payload = resp.SerializeAsString();
        resp_msg->payload.assign(payload.begin(), payload.end());
        resp_msg->client_seq = msg->client_seq;

        ctx.fire_write(channel::MessagePtr(std::move(resp_msg)));
        ctx.fire_flush();

        if (auto endpoint = endpoint_.lock()) {
            if (verified) {
                endpoint->on_auth_success(player_id);
            } else {
                endpoint->on_auth_failed();
            }
        }
    }
} // namespace infra::net::reliability
