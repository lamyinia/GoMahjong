#include "infrastructure/net/session/session.h"

namespace infra::net::session {

    Session::Session(std::string playerId)
        : player_id_(std::move(playerId)) {}

    void Session::add_channel(channel::ChannelType type, std::shared_ptr<channel::IChannel> ch) {
        channels_[type] = std::move(ch);
    }

    std::shared_ptr<channel::IChannel> Session::get_channel(channel::ChannelType type) const {
        auto it = channels_.find(type);
        if (it != channels_.end()) {
            return it->second;
        }
        return nullptr;
    }

    void Session::remove_channel(channel::ChannelType type) {
        channels_.erase(type);
    }

    bool Session::has_active_channel() const {
        for (const auto& [type, ch] : channels_) {
            if (ch && ch->is_active()) {
                return true;
            }
        }
        return false;
    }

    void Session::close_all() {
        for (auto& [type, ch] : channels_) {
            if (ch) {
                ch->close();
            }
        }
        channels_.clear();
    }

} // namespace infra::net::session


