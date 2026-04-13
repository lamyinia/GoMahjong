#pragma once

#include "infrastructure/net/channel/i_channel.h"
#include "infrastructure/net/channel/message.h"
#include "infrastructure/net/session/session_manager.h"

#include <memory>
#include <string>
#include <vector>

namespace google::protobuf {
    class Message;
}

namespace domain::game::outbound {

    // 出站协议偏好
    enum class ProtocolPreference {
        PreferTcp,       // 优先 TCP，不可用则 fallback 其他
        PreferWebsocket, // 优先 WebSocket，不可用则 fallback 其他
        PreferKcp,       // 优先 KCP，不可用则 fallback 其他
        TcpOnly,         // 只走 TCP
        WebsocketOnly,   // 只走 WebSocket
        KcpOnly,         // 只走 KCP
        Any              // 任意可用
    };

    // 出站调度器：按 playerId 查找 Session → 选择 Channel → 投递消息
    // 单例，由 ServerHub 创建，线程安全（依赖 SessionManager 的锁）
    class OutDispatcher {
    public:
        OutDispatcher() = default;

        void setSessionManager(std::shared_ptr<infra::net::session::SessionManager> sessionManager);

        // 广播：向多个玩家推送（业务 DTO 自动序列化为 protobuf payload）
        void broadcast(const std::vector<std::string>& playerIds,
                       const std::string& route,
                       const google::protobuf::Message& dto,
                       ProtocolPreference preference = ProtocolPreference::PreferTcp);

        // 单播：向指定玩家推送
        void send(const std::string& playerId,
                  const std::string& route,
                  const google::protobuf::Message& dto,
                  ProtocolPreference preference = ProtocolPreference::PreferTcp);

    private:
        // 按协议偏好选择 Channel
        std::shared_ptr<infra::net::channel::IChannel> selectChannel(
            const infra::net::session::Session& session,
            ProtocolPreference preference) const;

        // 向 Channel 发送消息（DTO 序列化 + Envelope 封装 + send）
        void dispatchToChannel(infra::net::channel::IChannel& channel,
                               const std::string& route,
                               const google::protobuf::Message& dto);

    private:
        std::shared_ptr<infra::net::session::SessionManager> sessionManager_;
    };

} // namespace domain::game::outbound
