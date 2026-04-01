#pragma once

#include "infrastructure/net/session/session.h"
#include "infrastructure/net/reliability/wild_endpoint.h"
#include "infrastructure/net/channel/i_channel.h"

#include <boost/asio/any_io_executor.hpp>
#include <chrono>
#include <functional>
#include <memory>
#include <mutex>
#include <unordered_map>

namespace infra::net::reliability {

    namespace channel = infra::net::channel;
    namespace session = infra::net::session;

    /**
     * @brief 认证成功回调
     * @param player_id 玩家 ID
     * @param channel 已认证的 Channel
     */
    using OnAuthenticated = std::function<void(const std::string& player_id, 
                                                std::shared_ptr<channel::IChannel> channel)>;

    /**
     * @brief 未认证端点管理器
     * 
     * 管理所有新建立的但尚未认证的连接：
     * - 接收新 Channel
     * - 创建 WildEndpoint 等待认证
     * - 认证成功后创建 Session 并通知上层
     * - 认证失败或超时自动清理
     */
    class WildEndpointManager : public std::enable_shared_from_this<WildEndpointManager> {
    public:
        /**
         * @brief 构造函数
         * @param executor io_context executor
         * @param auth_timeout 认证超时时间
         */
        explicit WildEndpointManager(
            boost::asio::any_io_executor executor,
            std::chrono::milliseconds auth_timeout = std::chrono::milliseconds(5000)
        );

        ~WildEndpointManager();

        // === 回调设置 ===

        void set_on_authenticated(OnAuthenticated callback) { 
            onAuthenticated_ = std::move(callback); 
        }

        // === 端点管理 ===

        /**
         * @brief 添加新 Channel
         * 
         * 创建 WildEndpoint 并开始等待认证
         * 
         * @param channel 新建立的 Channel
         */
        void add_channel(std::shared_ptr<channel::IChannel> channel);

        /**
         * @brief 移除端点
         * @param endpoint_id WildEndpoint ID
         */
        void remove_endpoint(const std::string& endpoint_id);

        /**
         * @brief 获取端点数量
         */
        size_t size() const;

        // === 认证结果处理 ===

        /**
         * @brief 认证成功
         * 
         * 由 WildEndpoint 调用，通知 Manager 认证成功
         * 
         * @param endpoint_id WildEndpoint ID
         * @param player_id 玩家 ID
         */
        void on_endpoint_authenticated(const std::string& endpoint_id, 
                                        const std::string& player_id);

        /**
         * @brief 认证失败
         * 
         * 由 WildEndpoint 调用，通知 Manager 认证失败
         * 
         * @param endpoint_id WildEndpoint ID
         */
        void on_endpoint_auth_failed(const std::string& endpoint_id);

    private:
        boost::asio::any_io_executor executor_;
        std::chrono::milliseconds auth_timeout_;
        
        mutable std::mutex mutex_;
        std::unordered_map<std::string, std::shared_ptr<WildEndpoint>> endpoints_;
        
        OnAuthenticated onAuthenticated_;
    };

} // namespace infra::net::reliability