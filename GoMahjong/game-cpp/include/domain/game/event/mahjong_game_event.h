#pragma once

#include "domain/game/engine/mahjong/material.h"
#include <cstdint>
#include <string>
#include <variant>

namespace domain::game::event {
    using TileType = mahjong::TileType;
    using Tile = mahjong::Tile;

    enum class EventType {
        // 出牌相关
        PlayTile,       // 出牌
        Riichi,         // 立直

        // 副露
        Chi,            // 吃
        Pon,            // 碰
        Kan,            // 杠（大明杠、小明杠、暗杠）
        
        // 胡牌
        Ron,            // 荣胡
        Tsumo,          // 自摸
        Draw,           // 流局
        
        // 反应
        Skip,           // 跳过（不吃不碰不和）
        KyuushuKyuukai, // 九种九牌流局宣告
        Snapshoot,      // 请求快照（重连）

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

    // 立直事件
    struct RiichiEvent {
        std::string playerId;
        Tile tile;           // 立直时打出的牌
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

    // 跳过事件（反应阶段放弃操作）
    struct SkipEvent {
        std::string playerId;
    };

    // 九种九牌流局宣告事件
    struct KyuushuKyuukaiEvent {
        std::string playerId;
    };

    // 快照请求事件（重连用）
    struct SnapshootEvent {
        std::string playerId;
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
    struct RoundStartEvent {};

    // 一局结束事件
    struct RoundEndEvent {};

    // 游戏开始事件
    struct GameStartEvent {
        std::string roomId;
    };

    // 游戏结束事件
    struct GameEndEvent {
        std::string roomId;
    };

    struct GameEvent {
        EventType type;
        
        // 使用 variant 存储不同类型的事件数据
        std::variant<
            PlayTileEvent,
            RiichiEvent,
            ChiEvent,
            PonEvent,
            KanEvent,
            RonEvent,
            TsumoEvent,
            DrawEvent,
            SkipEvent,
            KyuushuKyuukaiEvent,
            SnapshootEvent,
            PlayerTimeoutEvent,
            TurnStartEvent,
            TurnEndEvent,
            RoundStartEvent,
            RoundEndEvent,
            GameStartEvent,
            GameEndEvent
        > data;

        static GameEvent playTile(const std::string& playerId, const Tile& tile) {
            GameEvent e;
            e.type = EventType::PlayTile;
            e.data = PlayTileEvent{playerId, tile};
            return e;
        }

        static GameEvent riichi(const std::string& playerId, const Tile& tile) {
            GameEvent e;
            e.type = EventType::Riichi;
            e.data = RiichiEvent{playerId, tile};
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

        static GameEvent skip(const std::string& playerId) {
            GameEvent e;
            e.type = EventType::Skip;
            e.data = SkipEvent{playerId};
            return e;
        }

        static GameEvent kyuushuKyuukai(const std::string& playerId) {
            GameEvent e;
            e.type = EventType::KyuushuKyuukai;
            e.data = KyuushuKyuukaiEvent{playerId};
            return e;
        }

        static GameEvent snapshoot(const std::string& playerId) {
            GameEvent e;
            e.type = EventType::Snapshoot;
            e.data = SnapshootEvent{playerId};
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

        static GameEvent roundStart() {
            GameEvent e;
            e.type = EventType::RoundStart;
            e.data = RoundStartEvent{};
            return e;
        }

        static GameEvent roundEnd() {
            GameEvent e;
            e.type = EventType::RoundEnd;
            e.data = RoundEndEvent{};
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
