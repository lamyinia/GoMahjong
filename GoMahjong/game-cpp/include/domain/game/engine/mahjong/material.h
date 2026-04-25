#pragma once

#include <array>
#include <cstdint>
#include <random>
#include <utility>
#include <vector>

namespace domain::game::mahjong {

    enum class TileType : std::int8_t {
        // 万子 (0-8)
        Man1 = 0, Man2, Man3, Man4, Man5, Man6, Man7, Man8, Man9,
        // 筒子 (9-17)
        Pin1 = 9, Pin2, Pin3, Pin4, Pin5, Pin6, Pin7, Pin8, Pin9,
        // 索子 (18-26)
        So1 = 18, So2, So3, So4, So5, So6, So7, So8, So9,
        // 字牌-风牌 (27-30)
        East = 27, South, West, North,
        // 字牌-箭牌 (31-33)
        White = 31, Green, Red,

        Invalid = -1,
    };

    // 牌总数
    static constexpr int kTileLimit = 136;
    // 牌类型总数
    static constexpr int kTileTypeCount = 34;
    // 岭上牌数
    static constexpr int kKanTileCount = 4;
    // 宝牌指示牌最大数
    static constexpr int kDoraIndicatorCount = 5;
    // 王牌总数
    static constexpr int kWangTileCount = 14;

    struct Tile {
        TileType type = TileType::Invalid;
        std::int8_t id = 0; // 同类型牌的编号 (0-3)，对于数牌5，id=0 表示赤宝牌

        [[nodiscard]] bool isRedFive() const {
            return id == 0 && (type == TileType::Man5 || type == TileType::Pin5 || type == TileType::So5);
        }

        [[nodiscard]] bool operator==(const Tile& other) const {
            return type == other.type && id == other.id;
        }

        [[nodiscard]] bool operator!=(const Tile& other) const {
            return !(*this == other);
        }
    };

    enum class Wind : std::int8_t {
        East = 0,
        South = 1,
        West = 2,
        North = 3,
    };

    [[nodiscard]] inline bool isNumbered(TileType t) {
        return static_cast<int>(t) >= 0 && static_cast<int>(t) <= 26;
    }

    [[nodiscard]] inline bool isHonor(TileType t) {
        return static_cast<int>(t) >= 27 && static_cast<int>(t) <= 33;
    }

    [[nodiscard]] inline bool isFive(TileType t) {
        return t == TileType::Man5 || t == TileType::Pin5 || t == TileType::So5;
    }

    // 获取花色 (0=万, 1=筒, 2=索, -1=字牌)
    [[nodiscard]] inline int suitOf(TileType t) {
        auto v = static_cast<int>(t);
        if (v <= 8) return 0;
        if (v <= 17) return 1;
        if (v <= 26) return 2;
        return -1;
    }

    // 获取花色内的数值 (1-9)，字牌返回 -1
    [[nodiscard]] inline int rankOf(TileType t) {
        auto v = static_cast<int>(t);
        if (v <= 8) return v + 1;
        if (v >= 9 && v <= 17) return v - 9 + 1;
        if (v >= 18 && v <= 26) return v - 18 + 1;
        return -1;
    }

    // TileType 转数组下标 (0-33)，连续编码下直接强转
    [[nodiscard]] inline int toIndex34(TileType t) {
        return static_cast<int>(t);
    }

    // 数组下标转 TileType
    [[nodiscard]] inline TileType fromIndex34(int idx) {
        return static_cast<TileType>(idx);
    }

    // Hand34: 34 长度数组，按 TileType 索引记录每种牌的张数
    using Hand34 = std::array<std::uint8_t, kTileTypeCount>;

    // 从 Tile 列表构建 Hand34（模板化，兼容 vector/array 等容器）
    template <typename TileRange>
    Hand34 toHand34(const TileRange& tiles) {
        Hand34 h{};
        for (const auto& t : tiles) {
            if (static_cast<int>(t.type) >= 0 && static_cast<int>(t.type) < kTileTypeCount) {
                h[toIndex34(t.type)]++;
            }
        }
        return h;
    }

    [[nodiscard]] inline Wind nextWind(Wind w) {
        return static_cast<Wind>((static_cast<int>(w) + 1) % 4);
    }

    enum class MeldType : std::int8_t {
        Chi,    // 吃
        Peng,   // 碰
        Gang,   // 明杠
        Ankan,  // 暗杠
        Kakan,  // 加杠
    };

    struct Meld {
        MeldType type = MeldType::Chi;
        std::vector<Tile> tiles;
        int from_seat = -1;  // 来源玩家座位（暗杠=-1）
    };

    // 场况
    struct Situation {
        int dealer_index = 0;   // 庄家座位 (0-3)
        Wind round_wind = Wind::East; // 场风
        int round_number = 1;   // 局数 (1-4)
        int honba = 0;         // 本场数
        int riichi_sticks = 0;  // 供托（立直棒）
    };

    // 玩家操作
    struct PlayerOperation {
        std::string type;          // "HU", "GANG", "PENG", "CHI", "SKIP"
        std::vector<Tile> tiles;   // 操作涉及的牌
    };

    // 玩家反应信息
    struct PlayerReaction {
        std::vector<PlayerOperation> operations; // 该玩家可用的所有操作选择
        PlayerOperation chosen_op;               // 玩家选择的操作
        bool responded = false;                  // 是否已响应
    };

    // 最后出牌记录
    struct LastDiscard {
        int seat = -1;
        Tile tile;
        bool valid = false;
    };

    struct Wang {
        std::array<Tile, kKanTileCount> kanTiles{};           // 岭上牌 [0-3]
        int kanIndex = 0;                                      // 已摸岭上牌数

        std::array<Tile, kDoraIndicatorCount> doraIndicators{};  // 宝牌指示牌 [0-4]
        int doraIndex = 0;                                     // 已翻开宝牌数

        std::array<Tile, kDoraIndicatorCount> uraDoraIndicators{}; // 里宝牌指示牌 [0-4]
        bool uraDoraRevealed = false;                              // 里宝牌是否已全部翻开
    };

    class DeckManager {
    public:
        explicit DeckManager(std::uint32_t seed = 0);

        // 初始化一局（洗牌 + 重置索引 + 分配王牌，不重新生成牌）
        void initRound();

        // 从牌墙摸一张牌
        [[nodiscard]] std::pair<Tile, bool> draw();

        // 发牌（同 draw，语义别名）
        [[nodiscard]] std::pair<Tile, bool> deal();

        // 从岭上摸一张牌（开杠时使用）
        [[nodiscard]] std::pair<Tile, bool> drawKanTile();

        // 剩余岭上牌数
        [[nodiscard]] int remainingKanTiles() const;

        // 是否还能开杠（岭上牌未摸完）
        [[nodiscard]] bool canKan() const;

        // 翻开一张宝牌指示牌
        bool revealDoraIndicator();

        // 立直和牌时一次性翻开所有里宝牌指示牌
        void revealAllUraDora();

        // 获取已翻开的宝牌指示牌
        [[nodiscard]] std::vector<Tile> getDoraIndicators() const;

        // 获取里宝牌指示牌（未翻开返回空，已翻开返回全部）
        [[nodiscard]] std::vector<Tile> getUraDoraIndicators() const;

        // 获取可见牌快照（4 - remain34_[i]）
        [[nodiscard]] Hand34 visible34() const;

        // 牌墙剩余张数
        [[nodiscard]] int remainingWallTiles() const;

        // 牌墙是否已空
        [[nodiscard]] bool isWallEmpty() const;

        // 获取王牌结构（只读）
        [[nodiscard]] const Wang& wang() const;

    private:
        std::array<Tile, kTileLimit> wall_{};  // 136张牌（构造时生成，initRound只洗牌）
        int wallSize_ = 0;                     // 牌墙有效长度（= kTileLimit - kWangTileCount = 122）
        int wallIndex_ = 0;                    // 当前摸牌位置
        Wang wang_;
        Hand34 remain34_{};                    // 每种牌的剩余张数
        std::mt19937 rng_;

        // 洗牌
        void shuffle();
        // 分配王牌（从 wall_ 末尾取14张）
        void assignWang();
    };

} // namespace domain::game::mahjong