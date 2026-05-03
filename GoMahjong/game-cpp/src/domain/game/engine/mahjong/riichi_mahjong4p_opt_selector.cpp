#include "domain/game/engine/mahjong/riichi_mahjong4p_engine.h"
#include "domain/game/handler/mahjong_event_handler.h"

#include "infrastructure/log/logger.hpp"
#include "generated/game_mahjong.pb.h"

namespace domain::game::engine {

    // ==================== 操作计算（直接写入 reactions_） ====================

    void RiichiMahjong4PEngine::computeMainActions(int seatIndex) {
        auto& reaction = reactions_[seatIndex];
        reaction.operations.clear();
        reaction.responded = false;

        auto* p = getPlayer(seatIndex);
        if (!p) return;

        auto hand34 = mj::toHand34(p->tiles());
        int fixedMelds = static_cast<int>(p->melds().size());

        // 自摸判断（14张手牌时）
        if (p->tiles().size() % 3 == 2) {  // 14张 = 有摸牌
            if (searcher_.isAgariAll(hand34, fixedMelds)) {
                mj::PlayerOperation op;
                op.type = "HU";
                reaction.operations.push_back(std::move(op));
            }
        }

        // 立直后只能出牌（或自摸），不能暗杠/加杠
        if (p->isRiichi()) {
            // 立直后暗杠：只能杠不影响听牌的牌（严格判断，暂不实现）
            // 立直后不能加杠
            return;
        }

        // 暗杠检测
        for (int i = 0; i < mj::kTileTypeCount; ++i) {
            if (hand34[i] == 4) {
                mj::PlayerOperation op;
                op.type = "GANG";
                auto tileType = mj::fromIndex34(i);
                // 收集该类型的所有实体牌
                for (const auto& t : p->tiles()) {
                    if (t.type == tileType) {
                        op.tiles.push_back(t);
                    }
                }
                reaction.operations.push_back(std::move(op));
            }
        }

        // 加杠检测：已有碰的情况下摸到第4张
        for (const auto& meld : p->melds()) {
            if (meld.type == mj::MeldType::Peng) {
                // 检查手牌中是否有碰的牌类型
                auto meldType34 = mj::toIndex34(meld.tiles[0].type);
                if (hand34[meldType34] >= 1) {
                    mj::PlayerOperation op;
                    op.type = "GANG";
                    for (const auto& t : p->tiles()) {
                        if (t.type == meld.tiles[0].type) {
                            op.tiles.push_back(t);
                            break;  // 加杠只需一张
                        }
                    }
                    reaction.operations.push_back(std::move(op));
                }
            }
        }
    }

    void RiichiMahjong4PEngine::computeReactions(int seatIndex) {
        auto& reaction = reactions_[seatIndex];
        reaction.operations.clear();
        reaction.responded = false;

        auto* p = getPlayer(seatIndex);
        if (!p || !last_discard_.valid) return;

        // 不能对自摸的牌反应
        int discardSeat = last_discard_.seat;
        if (discardSeat == seatIndex) return;

        auto hand34 = mj::toHand34(p->tiles());
        int fixedMelds = static_cast<int>(p->melds().size());
        auto discardType34 = mj::toIndex34(last_discard_.tile.type);

        // 荣和判断
        auto testHand = hand34;
        testHand[discardType34]++;
        if (searcher_.isAgariAll(testHand, fixedMelds)) {
            mj::PlayerOperation op;
            op.type = "HU";
            op.tiles.push_back(last_discard_.tile);
            reaction.operations.push_back(std::move(op));
        }

        // 立直后不能吃碰杠
        if (p->isRiichi()) return;

        // 碰检测
        if (hand34[discardType34] >= 2) {
            mj::PlayerOperation op;
            op.type = "PENG";
            for (const auto& t : p->tiles()) {
                if (t.type == last_discard_.tile.type) {
                    op.tiles.push_back(t);
                    if (op.tiles.size() >= 2) break;
                }
            }
            op.tiles.push_back(last_discard_.tile);
            reaction.operations.push_back(std::move(op));
        }

        // 吃检测（仅下家可吃）
        int relativeSeat = (seatIndex - discardSeat + 4) % 4;
        if (relativeSeat == 1 && mj::isNumbered(last_discard_.tile.type)) {
            int suit = mj::suitOf(last_discard_.tile.type);
            int rank = mj::rankOf(last_discard_.tile.type);

            // 枚举所有可能的吃组合
            // chi pattern: (rank-2,rank-1,rank), (rank-1,rank,rank+1), (rank,rank+1,rank+2)
            struct ChiPattern { int r0, r1, r2; };
            ChiPattern patterns[] = {
                {rank - 2, rank - 1, rank},
                {rank - 1, rank, rank + 1},
                {rank, rank + 1, rank + 2},
            };

            for (const auto& pat : patterns) {
                if (pat.r0 < 1 || pat.r2 > 9) continue;

                auto idx0 = suit * 9 + (pat.r0 - 1);
                auto idx1 = suit * 9 + (pat.r1 - 1);
                // idx2 is discardType34, already in hand34 check

                if (hand34[idx0] >= 1 && hand34[idx1] >= 1) {
                    mj::PlayerOperation op;
                    op.type = "CHI";

                    // 收集手牌中的实体牌
                    for (const auto& t : p->tiles()) {
                        auto t34 = mj::toIndex34(t.type);
                        if (t34 == idx0 || t34 == idx1) {
                            op.tiles.push_back(t);
                        }
                    }
                    op.tiles.push_back(last_discard_.tile);
                    reaction.operations.push_back(std::move(op));
                }
            }
        }

        // 明杠检测
        if (hand34[discardType34] >= 3) {
            mj::PlayerOperation op;
            op.type = "GANG";
            for (const auto& t : p->tiles()) {
                if (t.type == last_discard_.tile.type) {
                    op.tiles.push_back(t);
                    if (op.tiles.size() >= 3) break;
                }
            }
            op.tiles.push_back(last_discard_.tile);
            reaction.operations.push_back(std::move(op));
        }
    }

    // ==================== 反应收集 ====================

    void RiichiMahjong4PEngine::enterReactionPhase() {
        if (!last_discard_.valid) return;

        reactions_.clear();

        // 计算每个非出牌玩家的可选反应（直接写入 reactions_）
        std::vector<int> eligibleSeats;
        for (int i = 0; i < 4; ++i) {
            if (i == last_discard_.seat) continue;

            computeReactions(i);
            if (!reactions_[i].operations.empty()) {
                eligibleSeats.push_back(i);
            } else {
                reactions_.erase(i);
            }
        }

        if (eligibleSeats.empty()) {
            // 无人可反应，直接进入下家出牌
            int nextSeat = (last_discard_.seat + 1) % 4;
            dropTurn(nextSeat, true);
            return;
        }

        // 推送可选操作给可反应的玩家
        for (int seat : eligibleSeats) {
            pushOperations(seat);
        }

        // 启动反应阶段计时
        if (turnManager_) {
            turnManager_->enterReactionPhase(eligibleSeats);
        }
    }

    void RiichiMahjong4PEngine::recordReaction(int seat, const mj::PlayerOperation& chosenOp) {
        auto it = reactions_.find(seat);
        if (it == reactions_.end()) {
            LOG_WARN("recordReaction: seat {} not in reaction phase", seat);
            return;
        }

        it->second.chosen_op = chosenOp;
        it->second.responded = true;
    }

    bool RiichiMahjong4PEngine::isReactionComplete() const {
        for (const auto& [seat, reaction] : reactions_) {
            if (!reaction.responded) return false;
        }
        return !reactions_.empty();
    }

    void RiichiMahjong4PEngine::resolveReactions() {
        // 优先级：荣和 > 碰/杠 > 吃
        // 检查是否有人荣和
        for (const auto& [seat, reaction] : reactions_) {
            if (reaction.responded && reaction.chosen_op.type == "HU") {
                // 荣和优先，直接处理
                auto* p = getPlayer(seat);
                if (p && context_) {
                    auto* discarder = getPlayer(last_discard_.seat);
                    std::string discarderId = discarder ? discarder->playerId() : "";
                    auto ev = event::GameEvent::ron(p->playerId(), last_discard_.tile, discarderId);
                    context_->submitEvent(context_->roomId(), ev);
                }
                return;
            }
        }

        // 检查碰/杠（优先于吃）
        for (const auto& [seat, reaction] : reactions_) {
            if (reaction.responded &&
                (reaction.chosen_op.type == "GANG" || reaction.chosen_op.type == "PENG")) {
                auto* p = getPlayer(seat);
                if (!p) continue;

                if (reaction.chosen_op.type == "GANG") {
                    auto ev = event::GameEvent::kan(p->playerId(), last_discard_.tile, false, false);
                    context_->submitEvent(context_->roomId(), ev);
                } else {
                    auto ev = event::GameEvent::pon(p->playerId(), last_discard_.tile);
                    context_->submitEvent(context_->roomId(), ev);
                }
                return;
            }
        }

        // 检查吃
        for (const auto& [seat, reaction] : reactions_) {
            if (reaction.responded && reaction.chosen_op.type == "CHI") {
                auto* p = getPlayer(seat);
                if (!p) continue;

                auto ev = event::GameEvent::chi(p->playerId(), last_discard_.tile, reaction.chosen_op.tiles[0].type);
                context_->submitEvent(context_->roomId(), ev);
                return;
            }
        }

        // 全部跳过，进入下家出牌
        int nextSeat = (last_discard_.seat + 1) % 4;
        dropTurn(nextSeat, true);
    }

} // namespace domain::game::engine
