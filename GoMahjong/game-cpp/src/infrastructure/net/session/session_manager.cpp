#include "infrastructure/net/session/session_manager.h"
#include "infrastructure/log/logger.hpp"

namespace infra::net::session {

    SessionManager::~SessionManager() {
        std::lock_guard<std::mutex> lock(mutex_);
        sessions_.clear();
    }

    std::shared_ptr<Session> SessionManager::create_or_get_session(
        const std::string& player_id,
        std::shared_ptr<channel::IChannel> channel
    ) {
        std::lock_guard<std::mutex> lock(mutex_);

        auto it = sessions_.find(player_id);
        std::shared_ptr<Session> session;

        if (it != sessions_.end()) {
            // 玩家已有会话，添加 Channel
            session = it->second;
            if (channel) {
                session->add_channel(channel->type(), std::move(channel));
            }
            LOG_DEBUG("[SessionManager] player {} session updated, channels={}", 
                      player_id, session->channels().size());
        } else {
            // 创建新会话
            session = std::make_shared<Session>(player_id);
            if (channel) {
                session->add_channel(channel->type(), std::move(channel));
            }
            sessions_[player_id] = session;
            LOG_INFO("[SessionManager] player {} session created", player_id);

            // 触发回调
            if (onSessionCreated_) {
                onSessionCreated_(player_id, session);
            }
        }

        return session;
    }

    std::shared_ptr<Session> SessionManager::get_session(const std::string& player_id) const {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = sessions_.find(player_id);
        if (it != sessions_.end()) {
            return it->second;
        }
        return nullptr;
    }

    void SessionManager::remove_session(const std::string& player_id) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = sessions_.find(player_id);
        if (it != sessions_.end()) {
            LOG_INFO("[SessionManager] player {} session removed", player_id);
            
            // 触发回调
            if (onSessionClosed_) {
                onSessionClosed_(player_id);
            }
            
            sessions_.erase(it);
        }
    }

    size_t SessionManager::size() const {
        std::lock_guard<std::mutex> lock(mutex_);
        return sessions_.size();
    }

    bool SessionManager::has_session(const std::string& player_id) const {
        std::lock_guard<std::mutex> lock(mutex_);
        return sessions_.find(player_id) != sessions_.end();
    }

} // namespace infra::net::session