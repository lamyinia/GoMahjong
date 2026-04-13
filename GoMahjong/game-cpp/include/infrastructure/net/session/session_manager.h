#pragma once

#include "infrastructure/net/session/session.h"
#include "infrastructure/net/channel/i_channel.h"

#include <memory>
#include <mutex>
#include <unordered_map>
#include <functional>

namespace infra::net::session {

    namespace channel = infra::net::channel;

    /**
     * @brief 会话管理器
     * 
     * 管理所有已认证玩家的会话
     */
    class SessionManager : public std::enable_shared_from_this<SessionManager> {
    public:

        SessionManager() = default;
        ~SessionManager();

        // === 会话管理 ===

        /**
         * @brief 创建或获取会话
         * 
         * 如果玩家已有会话，返回现有会话并添加 Channel
         * 如果玩家没有会话，创建新会话
         * 
         * @param player_id 玩家 ID
         * @param channel Channel 实例
         * @return Session 实例
         */
        std::shared_ptr<Session> create_or_get_session(
            const std::string& player_id,
            std::shared_ptr<channel::IChannel> channel
        );

        /**
         * @brief 获取会话
         * @param player_id 玩家 ID
         * @return Session 实例，不存在返回 nullptr
         */
        std::shared_ptr<Session> get_session(const std::string& player_id) const;

        /**
         * @brief 移除会话
         * @param player_id 玩家 ID
         */
        void remove_session(const std::string& player_id);

        /**
         * @brief 获取会话数量
         */
        size_t size() const;

        /**
         * @brief 检查是否有会话
         */
        bool has_session(const std::string& player_id) const;

    private:
        void set_on_inactive_callback(
            std::shared_ptr<channel::IChannel> channel,
            const std::string& player_id,
            channel::ChannelType type
        );

        mutable std::mutex mutex_;
        std::unordered_map<std::string, std::shared_ptr<Session>> sessions_;
    };

} // namespace infra::net::session
