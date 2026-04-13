#pragma once

#include <cstdint>
#include <string>
#include <variant>

namespace domain::game::event {

    // === 牌的定义 ===
    enum class TileType : std::int8_t {
        Invalid = 0,
        // 万子
        Wan1 = 1, Wan2, Wan3, Wan4, Wan5, Wan6, Wan7, Wan8, Wan9,
        // 条子
        Tiao1 = 11, Tiao2, Tiao3, Tiao4, Tiao5, Tiao6, Tiao7, Tiao8, Tiao9,
        // 筒子
        Tong1 = 21, Tong2, Tong3, Tong4, Tong5, Tong6, Tong7, Tong8, Tong9,
        // 风牌
        FengDong = 31, FengNan, FengXi, FengBei,
        // 三元牌
        SanYuanBai = 41, SanYuanFa, SanYuanZhong
    };

    struct Tile {
        TileType type = TileType::Invalid;
        std::int8_t id = 0;  // 同类型牌的编号 (0-3)
    };

    // === 事件类型 ===
    enum class EventType {
        // 出牌相关
        PlayTile,       // 出牌
        DrawTile,       // 摸牌
        
        // 副露
        Chi,            // 吃
        Pon,            // 碰
        Kan,            // 杠（大明杠、小明杠、暗杠）
        
        // 胡牌
        Ron,            // 荣胡
        Tsumo,          // 自摸
        Draw,           // 流局
        
        // 超时
        PlayerTimeout,  // 玩家超时

        // 游戏流程
        TurnStart,      // 回合开始
        TurnEnd,        // 回合结束
        RoundStart,     // 一局开始
        RoundEnd,       // 一局结束
        GameStart,      // 游戏开始
        GameEnd         // 游戏结束
    };

    // === 具体事件 ===
    
    // 出牌事件
    struct PlayTileEvent {
        std::string playerId;
        Tile tile;
    };

    // 摸牌事件
    struct DrawTileEvent {
        std::string playerId;
        Tile tile;
    };

    // 吃事件
    struct ChiEvent {
        std::string playerId;
        Tile tile;           // 吃的牌
        TileType meldType;   // 吃的牌型
    };

    // 碰事件
    struct PonEvent {
        std::string playerId;
        Tile tile;
    };

    // 杠事件
    struct KanEvent {
        std::string playerId;
        Tile tile;
        bool isAnkan = false;  // 是否暗杠
        bool isKakan = false;  // 是否加杠
    };

    // 荣胡事件
    struct RonEvent {
        std::string playerId;
        Tile tile;
        std::string targetPlayerId;  // 被胡的玩家
    };

    // 自摸事件
    struct TsumoEvent {
        std::string playerId;
        Tile tile;
    };

    // 流局事件
    struct DrawEvent {
        bool isKyuushuKyuukai = false;  // 是否九种九牌流局
    };

    // 玩家超时事件
    struct PlayerTimeoutEvent {
        std::string playerId;
        std::int32_t seatIndex = 0;
    };

    // 回合开始事件
    struct TurnStartEvent {
        std::string playerId;
        std::int32_t timeLimit = 30;  // 时间限制（秒）
    };

    // 回合结束事件
    struct TurnEndEvent {
        std::string playerId;
    };

    // 一局开始事件
    struct RoundStartEvent {
        std::int32_t roundNumber = 1;
        std::string oyaPlayerId;  // 庄家
    };

    // 一局结束事件
    struct RoundEndEvent {
        std::int32_t roundNumber = 1;
    };

    // 游戏开始事件
    struct GameStartEvent {
        std::string roomId;
    };

    // 游戏结束事件
    struct GameEndEvent {
        std::string roomId;
    };

    // === GameEvent（事件包装器）===
    struct GameEvent {
        EventType type;
        
        // 使用 variant 存储不同类型的事件数据
        std::variant<
            PlayTileEvent,
            DrawTileEvent,
            ChiEvent,
            PonEvent,
            KanEvent,
            RonEvent,
            TsumoEvent,
            DrawEvent,
            PlayerTimeoutEvent,
            TurnStartEvent,
            TurnEndEvent,
            RoundStartEvent,
            RoundEndEvent,
            GameStartEvent,
            GameEndEvent
        > data;

        // === 便捷构造函数 ===
        static GameEvent playTile(const std::string& playerId, const Tile& tile) {
            GameEvent e;
            e.type = EventType::PlayTile;
            e.data = PlayTileEvent{playerId, tile};
            return e;
        }

        static GameEvent drawTile(const std::string& playerId, const Tile& tile) {
            GameEvent e;
            e.type = EventType::DrawTile;
            e.data = DrawTileEvent{playerId, tile};
            return e;
        }

        static GameEvent chi(const std::string& playerId, const Tile& tile, TileType meldType) {
            GameEvent e;
            e.type = EventType::Chi;
            e.data = ChiEvent{playerId, tile, meldType};
            return e;
        }

        static GameEvent pon(const std::string& playerId, const Tile& tile) {
            GameEvent e;
            e.type = EventType::Pon;
            e.data = PonEvent{playerId, tile};
            return e;
        }

        static GameEvent kan(const std::string& playerId, const Tile& tile, bool isAnkan = false, bool isKakan = false) {
            GameEvent e;
            e.type = EventType::Kan;
            e.data = KanEvent{playerId, tile, isAnkan, isKakan};
            return e;
        }

        static GameEvent ron(const std::string& playerId, const Tile& tile, const std::string& targetPlayerId) {
            GameEvent e;
            e.type = EventType::Ron;
            e.data = RonEvent{playerId, tile, targetPlayerId};
            return e;
        }

        static GameEvent tsumo(const std::string& playerId, const Tile& tile) {
            GameEvent e;
            e.type = EventType::Tsumo;
            e.data = TsumoEvent{playerId, tile};
            return e;
        }

        static GameEvent draw(bool isKyuushuKyuukai = false) {
            GameEvent e;
            e.type = EventType::Draw;
            e.data = DrawEvent{isKyuushuKyuukai};
            return e;
        }

        static GameEvent playerTimeout(const std::string& playerId, std::int32_t seatIndex) {
            GameEvent e;
            e.type = EventType::PlayerTimeout;
            e.data = PlayerTimeoutEvent{playerId, seatIndex};
            return e;
        }

        static GameEvent turnStart(const std::string& playerId, std::int32_t timeLimit = 30) {
            GameEvent e;
            e.type = EventType::TurnStart;
            e.data = TurnStartEvent{playerId, timeLimit};
            return e;
        }

        static GameEvent turnEnd(const std::string& playerId) {
            GameEvent e;
            e.type = EventType::TurnEnd;
            e.data = TurnEndEvent{playerId};
            return e;
        }

        static GameEvent roundStart(std::int32_t roundNumber, const std::string& oyaPlayerId) {
            GameEvent e;
            e.type = EventType::RoundStart;
            e.data = RoundStartEvent{roundNumber, oyaPlayerId};
            return e;
        }

        static GameEvent roundEnd(std::int32_t roundNumber) {
            GameEvent e;
            e.type = EventType::RoundEnd;
            e.data = RoundEndEvent{roundNumber};
            return e;
        }

        static GameEvent gameStart(const std::string& roomId) {
            GameEvent e;
            e.type = EventType::GameStart;
            e.data = GameStartEvent{roomId};
            return e;
        }

        static GameEvent gameEnd(const std::string& roomId) {
            GameEvent e;
            e.type = EventType::GameEnd;
            e.data = GameEndEvent{roomId};
            return e;
        }
    };

} // namespace domain::game::event
