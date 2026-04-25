#include "domain/game/engine/engine_context.h"
#include "infrastructure/log/logger.hpp"

#include <google/protobuf/message.h>

namespace domain::game::engine {

    void EngineContext::notifyGameOver() {
        LOG_INFO("[EngineContext] room {} game over, notifying", roomId_);
        if (onGameOver_) {
            onGameOver_(roomId_);
        }
    }

    void EngineContext::submitEvent(const std::string& roomId, const event::GameEvent& event) {
        if (submitEvent_) {
            submitEvent_(roomId, event);
        } else {
            LOG_WARN("[EngineContext] submit_event callback not set, dropping event for room {}", roomId);
        }
    }

    void EngineContext::broadcast(const std::string& route,
                                  const google::protobuf::Message& dto,
                                  outbound::ProtocolPreference preference) {
        if (!outDispatcher_) {
            LOG_WARN("[EngineContext] outDispatcher not set, dropping broadcast for room {}", roomId_);
            return;
        }
        outDispatcher_->broadcast(playerIds_, route, dto, preference);
    }

    void EngineContext::send(const std::string& playerId,
                             const std::string& route,
                             const google::protobuf::Message& dto,
                             outbound::ProtocolPreference preference) {
        if (!outDispatcher_) {
            LOG_WARN("[EngineContext] outDispatcher not set, dropping send to {} for room {}", playerId, roomId_);
            return;
        }
        outDispatcher_->send(playerId, route, dto, preference);
    }

} // namespace domain::game::engine