#include "domain/game/outbound/out_dispatcher.h"
#include "infrastructure/log/logger.hpp"
#include "infrastructure/net/channel/channel_pipeline.h"

#include <google/protobuf/message.h>

namespace domain::game::outbound {

    void OutDispatcher::setSessionManager(std::shared_ptr<infra::net::session::SessionManager> sessionManager) {
        sessionManager_ = std::move(sessionManager);
    }

    void OutDispatcher::broadcast(const std::vector<std::string>& playerIds,
                                   const std::string& route,
                                   const google::protobuf::Message& dto,
                                   ProtocolPreference preference) {
        for (const auto& playerId : playerIds) {
            send(playerId, route, dto, preference);
        }
    }

    void OutDispatcher::send(const std::string& playerId,
                              const std::string& route,
                              const google::protobuf::Message& dto,
                              ProtocolPreference preference) {
        if (!sessionManager_) {
            LOG_WARN("[OutDispatcher] sessionManager not set, dropping message to {}", playerId);
            return;
        }

        auto session = sessionManager_->get_session(playerId);
        if (!session) {
            LOG_DEBUG("[OutDispatcher] no session for player {}, dropping", playerId);
            return;
        }

        auto channel = selectChannel(*session, preference);
        if (!channel) {
            LOG_DEBUG("[OutDispatcher] no active channel for player {}, dropping", playerId);
            return;
        }

        dispatchToChannel(*channel, route, dto);
    }

    std::shared_ptr<infra::net::channel::IChannel> OutDispatcher::selectChannel(
        const infra::net::session::Session& session,
        ProtocolPreference preference) const {

        namespace ch = infra::net::channel;

        auto findByType = [&](ch::ChannelType type) -> std::shared_ptr<ch::IChannel> {
            auto ch = session.get_channel(type);
            if (ch && ch->is_active()) return ch;
            return nullptr;
        };

        auto findAny = [&]() -> std::shared_ptr<ch::IChannel> {
            for (const auto& [type, ch] : session.channels()) {
                if (ch && ch->is_active()) return ch;
            }
            return nullptr;
        };

        switch (preference) {
            case ProtocolPreference::PreferTcp:
                return findByType(ch::ChannelType::Tcp) ?:
                       findAny();
            case ProtocolPreference::PreferWebsocket:
                return findByType(ch::ChannelType::Websocket) ?:
                       findAny();
            case ProtocolPreference::PreferKcp:
                return findByType(ch::ChannelType::Kcp) ?:
                       findAny();
            case ProtocolPreference::TcpOnly:
                return findByType(ch::ChannelType::Tcp);
            case ProtocolPreference::WebsocketOnly:
                return findByType(ch::ChannelType::Websocket);
            case ProtocolPreference::KcpOnly:
                return findByType(ch::ChannelType::Kcp);
            case ProtocolPreference::Any:
                return findAny();
        }
        return nullptr;
    }

    void OutDispatcher::dispatchToChannel(infra::net::channel::IChannel& channel,
                                            const std::string& route,
                                            const google::protobuf::Message& dto) {
        // 1. 序列化业务 DTO → payload bytes
        std::vector<uint8_t> payload(dto.ByteSizeLong());
        if (!dto.SerializeToArray(payload.data(), static_cast<int>(payload.size()))) {
            LOG_ERROR("[OutDispatcher] failed to serialize DTO for route {}", route);
            return;
        }

        // 2. 构造 MessagePtr，通过 Pipeline 出站链路发送
        //    Pipeline: MessagePtr → ProtobufEncoder(封Envelope+序列化) → LengthFieldEncoder(加长度前缀) → 网络
        auto msg = std::make_shared<infra::net::channel::Message>();
        msg->route = route;
        msg->payload = std::move(payload);

        channel.pipeline().fire_write(infra::net::channel::MessagePtr{msg});
        channel.pipeline().fire_flush();
    }

} // namespace domain::game::outbound
