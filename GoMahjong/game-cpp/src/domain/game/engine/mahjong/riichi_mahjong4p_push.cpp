#include "domain/game/engine/mahjong/riichi_mahjong4p_engine.h"
#include "domain/game/handler/mahjong_event_handler.h"

#include "infrastructure/log/logger.hpp"
#include "generated/game_mahjong.pb.h"

namespace domain::game::engine {

    void RiichiMahjong4PEngine::broadcastRoundStart() {
        if (!context_) return;

        auto doraIndicators = deck_manager_.getDoraIndicators();

        // 构建 Situation protobuf
        gomahjong::game::Situation sitProto;
        sitProto.set_dealer_index(situation_.dealer_index);
        sitProto.set_round_wind(
            situation_.round_wind == mj::Wind::East  ? "East"  :
            situation_.round_wind == mj::Wind::South ? "South" :
            situation_.round_wind == mj::Wind::West  ? "West"  : "North");
        sitProto.set_round_number(situation_.round_number);
        sitProto.set_honba(situation_.honba);
        sitProto.set_riichi_sticks(situation_.riichi_sticks);

        // 构建座位映射
        std::vector<gomahjong::game::PlayerSeat> seats;
        seats.reserve(4);
        for (int i = 0; i < 4; ++i) {
            auto* p = getPlayer(i);
            if (!p) continue;
            gomahjong::game::PlayerSeat seat;
            seat.set_seat_index(i);
            seat.set_player_id(p->playerId());
            seats.push_back(std::move(seat));
        }

        int currentTurn = situation_.dealer_index;

        // 逐人推送（手牌不同）
        for (int i = 0; i < 4; ++i) {
            auto* p = getPlayer(i);
            if (!p) continue;

            gomahjong::game::RoundStartPush push;

            // 座位映射
            for (const auto& seat : seats) {
                *push.add_seats() = seat;
            }

            // 宝牌指示牌
            for (const auto& dora : doraIndicators) {
                auto* t = push.add_dora_indicators();
                t->set_type(static_cast<int>(dora.type));
                t->set_id(dora.id);
            }

            // 场况
            *push.mutable_situation() = sitProto;

            // 自己的手牌（13张）
            for (const auto& tile : p->tiles()) {
                auto* t = push.add_hand_tiles();
                t->set_type(static_cast<int>(tile.type));
                t->set_id(tile.id);
            }

            push.set_current_turn(currentTurn);

            context_->send(p->playerId(), handler::route::kRoundStart, push);
        }

        LOG_DEBUG("broadcastRoundStart done, dora count={}", doraIndicators.size());
    }

    void RiichiMahjong4PEngine::pushDrawTile(int seatIndex, const mj::Tile& tile, bool isKanDraw) {
        if (!context_) return;

        auto* p = getPlayer(seatIndex);
        if (!p) return;

        gomahjong::game::DrawTilePush push;
        auto* t = push.mutable_tile();
        t->set_type(static_cast<int>(tile.type));
        t->set_id(tile.id);
        push.set_is_kan_draw(isKanDraw);

        context_->send(p->playerId(), handler::route::kDrawTile, push);
    }

    void RiichiMahjong4PEngine::broadcastDiscardTile(int seatIndex, const mj::Tile& tile) {
        if (!context_) return;

        gomahjong::game::DiscardTilePush push;
        push.set_seat_index(seatIndex);
        auto* t = push.mutable_tile();
        t->set_type(static_cast<int>(tile.type));
        t->set_id(tile.id);

        // 广播给所有玩家
        for (int i = 0; i < 4; ++i) {
            auto* p = getPlayer(i);
            if (p) {
                context_->send(p->playerId(), handler::route::kDiscardTile, push);
            }
        }
    }

    void RiichiMahjong4PEngine::pushOperations(int seatIndex) {
        if (!context_) return;

        auto* p = getPlayer(seatIndex);
        if (!p) return;

        auto it = reactions_.find(seatIndex);
        if (it == reactions_.end() || it->second.operations.empty()) return;

        gomahjong::game::OperationsPush push;

        for (const auto& op : it->second.operations) {
            auto* protoOp = push.add_operations();
            protoOp->set_type(op.type);
            for (const auto& tile : op.tiles) {
                auto* t = protoOp->add_tiles();
                t->set_type(static_cast<int>(tile.type));
                t->set_id(tile.id);
            }
        }

        // 获取可用时间
        int availableSecs = 0;
        if (turnManager_) {
            auto* ticker = turnManager_->getPlayerTicker(seatIndex);
            if (ticker) {
                availableSecs = ticker->getAvailable();
            }
        }
        push.set_available_secs(availableSecs);

        context_->send(p->playerId(), handler::route::kOperations, push);
    }

} // namespace domain::game::engine