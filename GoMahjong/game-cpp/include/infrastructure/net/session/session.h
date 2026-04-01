#pragma once

#include <string>
#include <unordered_map>
#include <memory>

#include "infrastructure/net/channel/i_channel.h"

namespace infra::net::session {

    namespace channel = infra::net::channel;

    /**
     * @brief 玩家会话
     * 
     * 管理玩家的多个 Channel 连接（支持多协议）
     */
    class Session {
    public:
        Session() = default;
        explicit Session(std::string playerId);

        // === 玩家信息 ===

        const std::string& player_id() const { return player_id_; }
        void set_player_id(const std::string& id) { player_id_ = id; }

        // === Channel 管理 ===

        /**
         * @brief 添加 Channel
         * @param type Channel 类型
         * @param ch Channel 实例
         */
        void add_channel(channel::ChannelType type, std::shared_ptr<channel::IChannel> ch);

        /**
         * @brief 获取指定类型的 Channel
         * @param type Channel 类型
         * @return Channel 实例，不存在返回 nullptr
         */
        std::shared_ptr<channel::IChannel> get_channel(channel::ChannelType type) const;

        /**
         * @brief 移除指定类型的 Channel
         */
        void remove_channel(channel::ChannelType type);

        /**
         * @brief 获取所有 Channel
         */
        const std::unordered_map<channel::ChannelType, std::shared_ptr<channel::IChannel>>& channels() const {
            return channels_;
        }

        /**
         * @brief 检查是否有活跃连接
         */
        bool has_active_channel() const;

        /**
         * @brief 关闭所有 Channel
         */
        void close_all();

    private:
        std::string player_id_;
        std::unordered_map<channel::ChannelType, std::shared_ptr<channel::IChannel>> channels_;
    };

} // namespace infra::net::session