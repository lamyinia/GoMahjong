#include "domain/game/handler/play_tile_handler.h"
#include "domain/game/handler/create_room_handler.h"

#include "domain/game/service/game_service.hpp"
#include "domain/game/room/room_manager.h"
#include "domain/game/event/mahjong_game_event.h"
#include "infrastructure/config/config.hpp"
#include "infrastructure/net/dispatcher/dispatcher.h"
#include "infrastructure/log/logger.hpp"

#include <sstream>
#include <iomanip>

// proto 生成的头文件
#include "game_mahjong.pb.h"

namespace domain::game::handler {

    static std::string fmt_payload(const std::vector<uint8_t>& data) {
        std::ostringstream oss;
        size_t limit = std::min(data.size(), size_t(64));
        for (size_t i = 0; i < limit; ++i) {
            oss << std::hex << std::setw(2) << std::setfill('0') << static_cast<int>(data[i]) << ' ';
        }
        if (data.size() > limit) oss << "...";
        return oss.str();
    }

    void registerGameHandlers() {
        // 注册出牌处理器
        infra::net::dispatcher::Dispatcher::instance().register_handler(
            "game.playTile", 
            &handlePlayTile
        );

        // Debug 模式：注册创建房间处理器
        if (infra::config::Config::instance().server().debug.enabled) {
            infra::net::dispatcher::Dispatcher::instance().register_handler(
                "game.createRoom",
                &handleDebugCreateRoom
            );
            LOG_INFO("debug mode: registered game.createRoom handler");
        }
    }

    void handlePlayTile(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg) {
        gomahjong::game::PlayTileRequest request;
        if (!request.ParseFromArray(msg->payload.data(), static_cast<int>(msg->payload.size()))) {
            LOG_ERROR("failed to parse PlayTileRequest from player {}, payload_size={}, hex={}",
                ctx.player_id(), msg->payload.size(),
                fmt_payload(msg->payload));
            ctx.send_error_response("game.playTile", msg->client_seq, "playTile 解析错误");
            return;
        }
        if (!request.has_tile()) {
            ctx.send_error_response("game.playTile", msg->client_seq, "playTile 牌是空的");
            LOG_WARN("missing tile from player {}", ctx.player_id());
            return;
        }

        const auto& protoTile = request.tile();
        LOG_DEBUG("player {} plays tile: type={}, id={}", ctx.player_id(), protoTile.type(), protoTile.id());

        auto& roomManager = domain::game::service::GameService::instance().room_manager();
        auto roomId = roomManager.get_player_room_id(ctx.player_id());
        if (!roomId) {
            ctx.send_error_response("game.playTile", msg->client_seq, "没有找到对应的 roomId");
            LOG_WARN("player {} not in any room", ctx.player_id());
            return;
        }
        event::Tile tile;
        tile.type = static_cast<event::TileType>(protoTile.type());
        tile.id = static_cast<std::int8_t>(protoTile.id());
        auto gameEvent = event::GameEvent::playTile(ctx.player_id(), tile);
        roomManager.submit_event(*roomId, gameEvent);
    }

} // namespace domain::game::handler
