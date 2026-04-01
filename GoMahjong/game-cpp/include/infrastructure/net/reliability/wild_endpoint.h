#pragma once

#include "infrastructure/net/channel/i_channel.h"

#include <boost/asio/any_io_executor.hpp>
#include <boost/asio/steady_timer.hpp>
#include <chrono>
#include <atomic>
#include <functional>
#include <memory>
#include <string>

namespace infra::net::reliability {

    namespace channel = infra::net::channel;

    /**
     * @brief 未认证的端点
     * 
     * 管理一个新连接的认证过程：
     * - 等待客户端发送认证消息
     * - 超时自动关闭连接
     * - 认证成功后创建 Session
     */
    class WildEndpoint : public std::enable_shared_from_this<WildEndpoint> {
    public:
        using OnAuthSuccess = std::function<void(const std::string& player_id)>;
        using OnAuthFailed = std::function<void()>;

        /**
         * @brief 构造函数
         * @param executor io_context executor（用于定时器）
         * @param channel Channel 实例
         * @param timeout 认证超时时间
         */
        WildEndpoint(boost::asio::any_io_executor executor,
                     std::shared_ptr<channel::IChannel> channel,
                     std::chrono::milliseconds timeout = std::chrono::milliseconds(5000));

        ~WildEndpoint();

        // === 回调设置 ===

        void set_on_auth_success(OnAuthSuccess callback) { onAuthSuccess_ = std::move(callback); }
        void set_on_auth_failed(OnAuthFailed callback) { onAuthFailed_ = std::move(callback); }

        // === 生命周期 ===

        /**
         * @brief 开始等待认证
         * 
         * 1. 添加 Codec Handler 到 Pipeline
         * 2. 添加 AuthHandler 到 Pipeline
         * 3. 启动超时定时器
         * 4. 开始读取数据
         */
        void start_wait_auth();

        /**
         * @brief 认证成功
         * @param player_id 玩家 ID
         */
        void on_auth_success(const std::string& player_id);

        /**
         * @brief 认证失败
         */
        void on_auth_failed();

        /**
         * @brief 获取 Channel ID
         */
        const std::string& id() const { return id_; }

        /**
         * @brief 获取 Channel
         */
        std::shared_ptr<channel::IChannel> channel() const { return channel_; }

    private:
        void start_timeout_timer();
        void cancel_timer();
        void handle_timeout(const boost::system::error_code& ec);

        std::string id_;
        boost::asio::any_io_executor executor_;
        std::shared_ptr<channel::IChannel> channel_;
        boost::asio::steady_timer timer_;
        std::chrono::milliseconds timeout_;
        OnAuthSuccess onAuthSuccess_;
        OnAuthFailed onAuthFailed_;
        std::atomic_bool auth_done_{false};
    };

} // namespace infra::net::reliability