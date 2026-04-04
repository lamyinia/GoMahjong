#pragma once

#include "domain/game/event/game_event.h"

#include <memory>
#include <string>
#include <vector>

namespace domain::game::engine {

    // === 引擎类型 ===
    enum class EngineType : std::int32_t {
        Unknown = 0,
        // 麻将
        RiichiMahjong4P = 1,    // 日本麻将 4 人
        RiichiMahjong3P = 2,    // 日本麻将 3 人
        RiichiMahjong2P = 3,    // 日本麻将 2 人
        // 扑克
        TexasHoldem = 10,       // 德州扑克
        Omaha = 11,             // 奥马哈
        // 其他
        SanZhang = 20,          // 三张牌
        DouDiZhu = 21,          // 斗地主
    };

    // === 游戏阶段 ===
    enum class GamePhase {
        Waiting,    // 等待玩家
        Ready,      // 准备阶段
        Playing,    // 游戏进行中
        Finished,   // 一局结束
        GameOver    // 整个游戏结束
    };

    // === 引擎接口 ===
    class Engine {
    public:
        virtual ~Engine() = default;

        // === 核心事件处理 ===
        // 统一入口，所有游戏事件都通过此方法处理
        virtual void handleEvent(const event::GameEvent& event) = 0;

        // === 玩家管理 ===
        virtual void onPlayerJoin(const std::string& userId) = 0;
        virtual void onPlayerLeave(const std::string& userId) = 0;
        virtual bool hasPlayer(const std::string& userId) const = 0;
        virtual std::size_t playerCount() const = 0;

        // === 游戏状态 ===
        [[nodiscard]] virtual GamePhase getPhase() const = 0;
        [[nodiscard]] virtual bool isGameOver() const = 0;
        [[nodiscard]] virtual bool canStart() const = 0;

        // === 状态序列化 ===
        // 返回 JSON 格式的游戏状态，用于推送给客户端
        [[nodiscard]] virtual std::string getGameState() const = 0;
        
        // 获取指定玩家的私有状态（如手牌）
        [[nodiscard]] virtual std::string getPlayerState(const std::string& userId) const = 0;

        // === 游戏控制 ===
        virtual void start() = 0;       // 开始游戏
        virtual void reset() = 0;       // 重置游戏（准备下一局）
        virtual void destroy() = 0;     // 销毁游戏

        // === 工厂方法 ===
        static std::unique_ptr<Engine> create(EngineType type);
    };

} // namespace domain::game::engine