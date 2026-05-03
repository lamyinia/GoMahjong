#include "domain/game/handler/create_room_handler.h"
#include "domain/game/handler/mahjong_event_handler.h"

#include "domain/game/service/game_service.hpp"
#include "domain/game/room/room_manager.h"
#include "infrastructure/config/config.hpp"
#include "infrastructure/log/logger.hpp"

// proto 生成的头文件
#include "game_mahjong.pb.h"

namespace domain::game::handler {

    void handleDebugCreateRoom(channel::ChannelHandlerContext& ctx,
                               const channel::MessagePtr& msg) {
        gomahjong::game::DebugCreateRoomRequest request;
        if (!request.ParseFromArray(msg->payload.data(), static_cast<int>(msg->payload.size()))) {
            LOG_ERROR("failed to parse request from player {}", ctx.player_id());
            return;
        }

        if (request.player_ids().empty()) {
            LOG_WARN("empty player list from player {}", ctx.player_id());
            return;
        }

        LOG_DEBUG("player {} creating room, engine_type={}, player_count={}",
                  ctx.player_id(), request.engine_type(), request.player_ids().size());

        std::vector<std::string> players(request.player_ids().begin(), request.player_ids().end());

        auto& roomManager = domain::game::service::GameService::instance().room_manager();
        auto roomId = roomManager.create_room(players, request.engine_type());

        if (roomId.empty()) {
            LOG_DEBUG("failed to create room");
            return;
        }

        LOG_DEBUG("room created: {}", roomId);

        // 构造响应
        gomahjong::game::DebugCreateRoomResponse response;
        response.set_room_id(roomId);

        auto respPayload = response.SerializeAsString();

        auto respMsg = std::make_shared<channel::Message>();
        respMsg->route = std::string(route::kDebugCreateRoom) + ".response";
        respMsg->payload.assign(respPayload.begin(), respPayload.end());
        respMsg->client_seq = msg->client_seq;

        ctx.fire_write(channel::MessagePtr(std::move(respMsg)));
        ctx.fire_flush();
    }

} // namespace domain::game::handler
