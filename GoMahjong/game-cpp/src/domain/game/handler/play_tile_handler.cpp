#include "domain/game/handler/play_tile_handler.h"

#include "domain/game/service/game_service.hpp"
#include "domain/game/room/room_manager.h"
#include "domain/game/event/game_event.h"
#include "infrastructure/net/dispatcher/dispatcher.h"
#include "infrastructure/log/logger.hpp"

// proto 生成的头文件
#include "game_mahjong.pb.h"

namespace domain::game::handler {

    void registerGameHandlers() {
        // 注册出牌处理器
        infra::net::dispatcher::Dispatcher::instance().register_handler(
            "game.playTile", 
            &handlePlayTile
        );
        
        LOG_INFO("[GameHandler] registered game.playTile handler");
    }

    void handlePlayTile(channel::ChannelHandlerContext& ctx, 
                        const channel::MessagePtr& msg) {
        // 解析请求
        gomahjong::game::PlayTileRequest request;
        if (!request.ParseFromArray(msg->payload.data(), static_cast<int>(msg->payload.size()))) {
            LOG_ERROR("[PlayTile] failed to parse request from player {}", ctx.player_id());
            return;
        }

        // 验证牌数据
        if (!request.has_tile()) {
            LOG_WARN("[PlayTile] missing tile from player {}", ctx.player_id());
            return;
        }

        const auto& protoTile = request.tile();
        LOG_DEBUG("[PlayTile] player {} plays tile: type={}, id={}", 
                  ctx.player_id(), protoTile.type(), protoTile.id());

        // 通过 GameService 单例获取 RoomManager
        auto& roomManager = domain::game::service::GameService::instance().room_manager();

        // 获取玩家所在房间
        auto* room = roomManager.get_player_room(ctx.player_id());
        if (!room) {
            LOG_WARN("[PlayTile] player {} not in any room", ctx.player_id());
            return;
        }

        // 转换 proto Tile 到 domain Tile
        event::Tile tile;
        tile.type = static_cast<event::TileType>(protoTile.type());
        tile.id = static_cast<std::int8_t>(protoTile.id());

        // 创建 GameEvent
        auto gameEvent = event::GameEvent::playTile(ctx.player_id(), tile);

        // 提交事件到 Actor 池
        roomManager.submitEvent(room->getId(), gameEvent);
        
        LOG_DEBUG("[PlayTile] event submitted to room {}", room->getId());
    }

} // namespace domain::game::handler
