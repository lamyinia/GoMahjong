#include "domain/game/handler/play_tile_handler.h"

#include "domain/game/service/game_service.hpp"
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

        // TODO: 后续实现游戏引擎处理出牌逻辑
        // 1. 创建 GameEvent
        // 2. 推送到 RoomExecutor 的事件队列
        // 3. RoomExecutor 调用 GameFrame 处理
    }

} // namespace domain::game::handler
