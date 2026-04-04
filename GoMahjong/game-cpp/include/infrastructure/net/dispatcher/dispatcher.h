#pragma once

#include "infrastructure/net/dispatcher/dispatcher_handler.h"
#include <memory>
#include <string>

namespace infra::net::dispatcher {

    /**
     * @brief 全局 Dispatcher 管理器
     * 
     * 单例模式，管理全局的 Handler 注册
     * 所有 Channel 共享同一个 Dispatcher 实例
     */
    class Dispatcher {
    public:
        static Dispatcher& instance();

        // === Handler 注册 ===

        template<typename Func>
        void register_handler(const std::string& route, Func&& handler) {
            ensure_dispatcher();
            dispatcher_->register_handler(route, std::forward<Func>(handler));
        }

        void unregister_handler(const std::string& route);

        bool has_handler(const std::string& route) const;

        /**
         * @brief 获取 DispatcherHandler 实例
         * 用于添加到 Channel Pipeline
         */
        std::shared_ptr<DispatcherHandler> get_dispatcher();

        /**
         * @brief 创建新的 DispatcherHandler 实例
         * 每个 Channel 使用独立的实例，共享同一份 Handler 注册表
         */
        std::shared_ptr<DispatcherHandler> create_dispatcher();

    private:
        Dispatcher() = default;
        
        void ensure_dispatcher();

        std::shared_ptr<DispatcherHandler> dispatcher_;
    };

    // 便捷宏：注册 Handler
    #define REGISTER_HANDLER(route, handler) \
        infra::net::dispatcher::Dispatcher::instance().register_handler(route, handler)

} // namespace infra::net::dispatcher
