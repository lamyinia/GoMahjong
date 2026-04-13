#pragma once

#include "domain/game/event/mahjong_game_event.h"
#include "domain/game/outbound/out_dispatcher.h"

#include <functional>
#include <string>
#include <vector>

namespace google::protobuf {
    class Message;
}

namespace domain::game::engine {

    // Engine 与外界的桥梁：Engine 通过 Context 通知生命周期事件和广播
    // 由 Room 创建并持有，注入 Engine；由 RoomActor 配置回调和 OutDispatcher
    class EngineContext {
    public:
        using GameOverCallback = std::function<void(const std::string& roomId)>;
        // Engine 内部组件（如 TurnManager）通过此回调投递游戏事件到 RoomActor 队列
        using SubmitEventCallback = std::function<void(const std::string& roomId, const event::GameEvent& event)>;

        void setRoomId(std::string roomId) { roomId_ = std::move(roomId); }
        void setPlayerIds(std::vector<std::string> playerIds) { playerIds_ = std::move(playerIds); }
        void setGameOverCallback(GameOverCallback cb) { onGameOver_ = std::move(cb); }
        void setSubmitEventCallback(SubmitEventCallback cb) { submitEvent_ = std::move(cb); }
        void setOutDispatcher(outbound::OutDispatcher* dispatcher) { outDispatcher_ = dispatcher; }

        [[nodiscard]] const std::string& roomId() const { return roomId_; }
        [[nodiscard]] const std::vector<std::string>& playerIds() const { return playerIds_; }

        // Engine 调用：游戏结束时主动通知
        void notifyGameOver();

        // Engine 内部组件调用：投递游戏事件到 RoomActor 队列（跨线程安全）
        void submitEvent(const std::string& roomId, const event::GameEvent& event);

        // Engine 调用：向房间所有玩家广播（业务 DTO 自动序列化为 protobuf）
        void broadcast(const std::string& route,
                       const google::protobuf::Message& dto,
                       outbound::ProtocolPreference preference = outbound::ProtocolPreference::PreferTcp);

        // Engine 调用：向指定玩家推送
        void send(const std::string& playerId,
                  const std::string& route,
                  const google::protobuf::Message& dto,
                  outbound::ProtocolPreference preference = outbound::ProtocolPreference::PreferTcp);

    private:
        std::string roomId_;
        std::vector<std::string> playerIds_;
        GameOverCallback onGameOver_;
        SubmitEventCallback submitEvent_;
        outbound::OutDispatcher* outDispatcher_{nullptr};
    };

} // namespace domain::game::engine
