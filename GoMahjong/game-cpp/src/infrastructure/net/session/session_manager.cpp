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
            session = it->second;
            if (channel) {
                auto type = channel->type();
                session->add_channel(type, std::move(channel));
            }
            LOG_DEBUG("[SessionManager] player {} session updated, channels={}",player_id, session->channels().size());
        } else {
            session = std::make_shared<Session>(player_id);
            if (channel) {
                auto type = channel->type();
                session->add_channel(type, std::move(channel));
            }
            sessions_[player_id] = session;
            LOG_DEBUG("[SessionManager] player {} session created", player_id);
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
            LOG_DEBUG("[SessionManager] player {} session removed", player_id);
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