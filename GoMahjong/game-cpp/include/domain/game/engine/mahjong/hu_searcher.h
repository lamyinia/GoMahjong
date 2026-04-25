#pragma once

#include <array>
#include <cstdint>
#include <mutex>
#include <shared_mutex>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>
#include "material.h"

namespace domain::game::mahjong {

    struct Candidate {
        TileType discard_type = TileType::Invalid;
        std::vector<Tile> discard_options;  // 该类型可打出的实体牌
        std::vector<TileType> waits;         // 听哪些牌
    };

    class Searcher {
    public:
        Searcher();

        // 弃牌后有哪些牌听牌（立直宣言验证用）
        [[nodiscard]] std::vector<Candidate> seekCandidates(const std::vector<Tile> &hand14, int fixed_melds);

        // 枚举听牌（振听/立直判断用）
        [[nodiscard]] std::vector<TileType> waits(Hand34 h13, int fixed_melds);

        // 是否和牌（普通+七对+国士）
        [[nodiscard]] bool isAgariAll(Hand34 h, int fixed_melds);

        // ---- 纯函数 ----

        // 普通牌型和牌判断
        [[nodiscard]] static bool isAgariNormal(Hand34 h, int fixed_melds);

        // 七对子和牌判断
        [[nodiscard]] static bool isAgariChiitoi(Hand34 h);

        // 国士无双和牌判断
        [[nodiscard]] static bool isAgariKokushi(Hand34 h);

    private:
        // 递归：能否组成 need 个面子
        [[nodiscard]] static bool canFormMelds(Hand34 &h, int need);

        // 缓存 key
        [[nodiscard]] static std::string makeKey(Hand34 h, int fixed_melds);

        std::shared_mutex mu_;
        std::unordered_map<std::string, bool> agari_cache_;
        std::unordered_map<std::string, std::vector<TileType>> waits_cache_;
    };

// 国士无双相关牌索引
    inline constexpr std::array<int, 13> kKokushiTiles = {
            0, 8,   // Man1, Man9
            9, 17,  // Pin1, Pin9
            18, 26, // So1, So9
            27, 28, 29, 30, // East, South, West, North
            31, 32, 33,     // White, Green, Red
    };

    // Hand34 从 Tile 列表构建（含 discard_options 映射）
    [[nodiscard]] std::pair<Hand34, std::unordered_map<TileType, std::vector<Tile>>>
    hand34FromTiles(const std::vector<Tile> &tiles);

} // namespace domain::game::mahjong