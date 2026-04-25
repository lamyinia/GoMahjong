#pragma once

#include <string>
#include <optional>
#include "material.h"

namespace domain::game::mahjong {
    class PlayerImage {
    public:
        explicit PlayerImage(std::string player_id, int seat_index, int initial_points);

        // 状态查询
        [[nodiscard]] int seatIndex() const;
        [[nodiscard]] const std::string& playerId() const;
        [[nodiscard]] bool isRiichi() const;
        [[nodiscard]] const std::optional<Tile>& newestTile() const;

        // 手牌/弃牌/副露只读访问
        [[nodiscard]] const std::vector<Tile>& tiles() const;
        [[nodiscard]] const std::vector<Tile>& discardPile() const;
        [[nodiscard]] const std::vector<Meld>& melds() const;

        // 手牌操作
        void addTile(Tile tile);
        void drawTile(Tile tile);                    // 摸牌（同时设置 newest_tile_）
        bool removeTile(Tile tile);
        bool discardTile(Tile tile);                 // 出牌（从手牌移到弃牌堆）
        std::pair<Tile, bool> discardNewestOrLast(); // 立直后代打/超时自动出牌

        // 副露
        void addMeld(Meld meld);

        // 点数
        void addPoints(int delta);
        [[nodiscard]] int points() const;

        // 振听判断
        [[nodiscard]] bool hasDiscardedType(TileType type) const;

        // 立直/听牌设置
        void setRiichi(bool value);
        void setWaiting(bool value);

        // 重置（新一局，保留 points/player_id/seat_index）
        void resetForRound();

    private:
        std::string player_id_;
        int seat_index_{0};
        int points_{0};
        std::vector<Tile> tiles_;                              // 手牌 (max 14)
        std::vector<Tile> discard_pile_;                       // 弃牌堆 (max ~18)
        std::vector<Meld> melds_;                              // 副露 (max 4)
        bool is_riichi_{false};
        bool is_waiting_{false};
        std::array<bool, kTileTypeCount> discarded_types_{};   // 已弃牌类型（振听判断）
        std::optional<Tile> newest_tile_;                      // 最新摸的牌
    };
}
