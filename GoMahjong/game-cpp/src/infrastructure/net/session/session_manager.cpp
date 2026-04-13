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
        } else {
            session = std::make_shared<Session>(player_id);
            sessions_[player_id] = session;
            LOG_DEBUG("player {} session created", player_id);
        }

        if (channel) {
            auto type = channel->type();
            set_on_inactive_callback(channel, player_id, type);
            session->add_channel(type, std::move(channel));
        }

        LOG_DEBUG("player {} session updated, channels={}", player_id, session->channels().size());
        return session;
    }

    void SessionManager::set_on_inactive_callback(
        std::shared_ptr<channel::IChannel> channel,
        const std::string& player_id,
        channel::ChannelType type
    ) {
        auto self = shared_from_this();
        auto pid = player_id;
        channel->set_on_inactive([self, pid, type]() {
            auto s = self->get_session(pid);
            if (s) {
                s->remove_channel(type);
                if (!s->has_active_channel()) {
                    self->remove_session(pid);
                    LOG_DEBUG("session auto-removed (no active channels): {}", pid);
                }
            }
        });
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