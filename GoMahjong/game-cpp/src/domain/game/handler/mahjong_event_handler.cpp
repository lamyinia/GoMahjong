#include "domain/game/handler/mahjong_event_handler.h"
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
        using namespace route;
        auto& disp = infra::net::dispatcher::Dispatcher::instance();

        // C→S 请求处理器
        disp.register_handler(std::string(kPlayTile),  &handlePlayTile);
        disp.register_handler(std::string(kMeld),      &handleMeld);
        disp.register_handler(std::string(kAnkan),     &handleAnkan);
        disp.register_handler(std::string(kKakan),     &handleKakan);
        disp.register_handler(std::string(kRiichi),    &handleRiichi);
        disp.register_handler(std::string(kSkip),             &handleSkip);
        disp.register_handler(std::string(kKyuushuKyuukai),  &handleKyuushuKyuukai);
        disp.register_handler(std::string(kSnapshoot), &handleSnapshoot);

        // Debug 模式：注册创建房间处理器
        if (infra::config::Config::instance().server().debug.enabled) {
            disp.register_handler(std::string(kDebugCreateRoom), &handleDebugCreateRoom);
            LOG_INFO("debug mode: registered {} handler", kDebugCreateRoom);
        }
    }

    // 辅助：获取玩家所在房间并提交事件
    static bool submitToRoom(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg, const event::GameEvent& gameEvent) {
        auto& roomManager = domain::game::service::GameService::instance().room_manager();
        auto roomId = roomManager.get_player_room_id(ctx.player_id());
        if (!roomId) {
            ctx.send_error_response(msg->route, msg->client_seq, "没有找到对应的 roomId");
            LOG_WARN("player {} not in any room", ctx.player_id());
            return false;
        }
        roomManager.submit_event(*roomId, gameEvent);
        return true;
    }

    void handlePlayTile(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg) {
        gomahjong::game::PlayTileRequest request;
        if (!request.ParseFromArray(msg->payload.data(), static_cast<int>(msg->payload.size()))) {
            LOG_ERROR("failed to parse PlayTileRequest from player {}, payload_size={}, hex={}",
                ctx.player_id(), msg->payload.size(),
                fmt_payload(msg->payload));
            ctx.send_error_response(msg->route, msg->client_seq, "playTile 解析错误");
            return;
        }
        if (!request.has_tile()) {
            ctx.send_error_response(msg->route, msg->client_seq, "playTile 牌是空的");
            LOG_WARN("missing tile from player {}", ctx.player_id());
            return;
        }

        const auto& protoTile = request.tile();
        LOG_DEBUG("player {} plays tile: type={}, id={}", ctx.player_id(), protoTile.type(), protoTile.id());

        event::Tile tile;
        tile.type = static_cast<event::TileType>(protoTile.type());
        tile.id = static_cast<std::int8_t>(protoTile.id());
        submitToRoom(ctx, msg, event::GameEvent::playTile(ctx.player_id(), tile));
    }

    void handleMeld(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg) {
        gomahjong::game::MeldRequest request;
        if (!request.ParseFromArray(msg->payload.data(), static_cast<int>(msg->payload.size()))) {
            ctx.send_error_response(msg->route, msg->client_seq, "meld 解析错误");
            return;
        }
        const auto& actionType = request.action_type();
        if (request.tiles_size() == 0) {
            ctx.send_error_response(msg->route, msg->client_seq, "meld 牌是空的");
            return;
        }

        LOG_DEBUG("player {} meld: actionType={}, tiles_count={}", ctx.player_id(), actionType, request.tiles_size());

        // 取第一张牌作为代表牌
        event::Tile tile;
        tile.type = static_cast<event::TileType>(request.tiles(0).type());
        tile.id = static_cast<std::int8_t>(request.tiles(0).id());

        if (actionType == "CHI") {
            // 吃：meldType 取第一张牌的花色
            event::TileType meldType = tile.type;
            submitToRoom(ctx, msg, event::GameEvent::chi(ctx.player_id(), tile, meldType));
        } else if (actionType == "PENG") {
            submitToRoom(ctx, msg, event::GameEvent::pon(ctx.player_id(), tile));
        } else if (actionType == "GANG") {
            submitToRoom(ctx, msg, event::GameEvent::kan(ctx.player_id(), tile, false, false));
        } else {
            ctx.send_error_response(msg->route, msg->client_seq, "meld 未知 actionType");
            LOG_WARN("unknown meld actionType: {} from player {}", actionType, ctx.player_id());
        }
    }

    void handleAnkan(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg) {
        gomahjong::game::AnkanRequest request;
        if (!request.ParseFromArray(msg->payload.data(), static_cast<int>(msg->payload.size()))) {
            ctx.send_error_response(msg->route, msg->client_seq, "ankan 解析错误");
            return;
        }
        if (request.tiles_size() == 0) {
            ctx.send_error_response(msg->route, msg->client_seq, "ankan 牌是空的");
            return;
        }

        LOG_DEBUG("player {} ankan: tiles_count={}", ctx.player_id(), request.tiles_size());

        // 取第一张牌作为代表牌，标记 isAnkan
        event::Tile tile;
        tile.type = static_cast<event::TileType>(request.tiles(0).type());
        tile.id = static_cast<std::int8_t>(request.tiles(0).id());
        submitToRoom(ctx, msg, event::GameEvent::kan(ctx.player_id(), tile, true, false));
    }

    void handleKakan(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg) {
        gomahjong::game::KakanRequest request;
        if (!request.ParseFromArray(msg->payload.data(), static_cast<int>(msg->payload.size()))) {
            ctx.send_error_response(msg->route, msg->client_seq, "kakan 解析错误");
            return;
        }
        if (!request.has_tile()) {
            ctx.send_error_response(msg->route, msg->client_seq, "kakan 牌是空的");
            return;
        }

        LOG_DEBUG("player {} kakan: type={}, id={}", ctx.player_id(), request.tile().type(), request.tile().id());

        event::Tile tile;
        tile.type = static_cast<event::TileType>(request.tile().type());
        tile.id = static_cast<std::int8_t>(request.tile().id());
        submitToRoom(ctx, msg, event::GameEvent::kan(ctx.player_id(), tile, false, true));
    }

    void handleRiichi(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg) {
        gomahjong::game::RiichiRequest request;
        if (!request.ParseFromArray(msg->payload.data(), static_cast<int>(msg->payload.size()))) {
            ctx.send_error_response(msg->route, msg->client_seq, "riichi 解析错误");
            return;
        }
        if (!request.has_tile()) {
            ctx.send_error_response(msg->route, msg->client_seq, "riichi 牌是空的");
            return;
        }

        LOG_DEBUG("player {} riichi: type={}, id={}", ctx.player_id(), request.tile().type(), request.tile().id());

        event::Tile tile;
        tile.type = static_cast<event::TileType>(request.tile().type());
        tile.id = static_cast<std::int8_t>(request.tile().id());
        submitToRoom(ctx, msg, event::GameEvent::riichi(ctx.player_id(), tile));
    }

    void handleSkip(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg) {
        LOG_DEBUG("player {} skip", ctx.player_id());
        submitToRoom(ctx, msg, event::GameEvent::skip(ctx.player_id()));
    }

    void handleKyuushuKyuukai(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg) {
        LOG_DEBUG("player {} declares kyuushu kyuukai", ctx.player_id());
        submitToRoom(ctx, msg, event::GameEvent::kyuushuKyuukai(ctx.player_id()));
    }

    void handleSnapshoot(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg) {
        LOG_DEBUG("player {} snapshoot", ctx.player_id());
        submitToRoom(ctx, msg, event::GameEvent::snapshoot(ctx.player_id()));
    }

} // namespace domain::game::handler
