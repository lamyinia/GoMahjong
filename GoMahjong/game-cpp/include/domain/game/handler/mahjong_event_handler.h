#pragma once

#include "infrastructure/net/channel/channel_handler_context.h"
#include "infrastructure/net/channel/message.h"

#include <functional>
#include <memory>
#include <string_view>

namespace domain::game::handler {

    namespace channel = infra::net::channel;

    namespace route {
        // C→S 请求
        constexpr std::string_view kSnapshoot    = "rmj4p.snapshoot";
        constexpr std::string_view kPlayTile    = "rmj4p.playTile";
        constexpr std::string_view kMeld        = "rmj4p.meld";
        constexpr std::string_view kAnkan       = "rmj4p.ankan";
        constexpr std::string_view kKakan       = "rmj4p.kakan";
        constexpr std::string_view kRiichi      = "rmj4p.riichi";
        constexpr std::string_view kSkip             = "rmj4p.skip";
        constexpr std::string_view kKyuushuKyuukai  = "rmj4p.kyuushuKyuukai";

        // S→C 推送
        constexpr std::string_view kRoundStart       = "rmj4p.roundStart";
        constexpr std::string_view kDrawTile         = "rmj4p.drawTile";
        constexpr std::string_view kDiscardTile      = "rmj4p.discardTile";
        constexpr std::string_view kRiichiPush       = "rmj4p.riichi";
        constexpr std::string_view kMeldAction       = "rmj4p.meldAction";
        constexpr std::string_view kAnkanPush        = "rmj4p.ankan";
        constexpr std::string_view kKakanPush        = "rmj4p.kakan";
        constexpr std::string_view kRon              = "rmj4p.ron";
        constexpr std::string_view kTsumo            = "rmj4p.tsumo";
        constexpr std::string_view kRoundEnd         = "rmj4p.roundEnd";
        constexpr std::string_view kGameEnd          = "rmj4p.gameEnd";
        constexpr std::string_view kPlayerDisconnect = "rmj4p.playerDisconnect";
        constexpr std::string_view kPlayerReconnect  = "rmj4p.playerReconnect";
        constexpr std::string_view kOperations       = "rmj4p.operations";
        constexpr std::string_view kGameState        = "rmj4p.gameState";

        // Debug
        constexpr std::string_view kDebugCreateRoom = "rmj4p.debug.createRoom";
    }

    /**
     * @brief 注册游戏相关的 Handler
     */
    void registerGameHandlers();

    // C→S 请求处理
    void handlePlayTile(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg);
    void handleMeld(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg);
    void handleAnkan(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg);
    void handleKakan(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg);
    void handleRiichi(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg);
    void handleSkip(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg);
    void handleKyuushuKyuukai(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg);
    void handleSnapshoot(channel::ChannelHandlerContext& ctx, const channel::MessagePtr& msg);

} // namespace domain::game::handler
