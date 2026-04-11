#include "infrastructure/net/dispatcher/dispatcher.h"
#include "infrastructure/log/logger.hpp"

namespace infra::net::dispatcher {

    Dispatcher& Dispatcher::instance() {
        static Dispatcher instance;
        return instance;
    }

    void Dispatcher::unregister_handler(const std::string& route) {
        if (dispatcher_) {
            dispatcher_->unregister_handler(route);
        }
    }

    bool Dispatcher::has_handler(const std::string& route) const {
        if (dispatcher_) {
            return dispatcher_->has_handler(route);
        }
        return false;
    }

    std::shared_ptr<DispatcherHandler> Dispatcher::get_dispatcher() {
        ensure_dispatcher();
        return dispatcher_;
    }

    std::shared_ptr<DispatcherHandler> Dispatcher::create_dispatcher() {
        ensure_dispatcher();
        // 返回同一个实例，所有 Channel 共享 Handler 注册表
        return dispatcher_;
    }

    void Dispatcher::ensure_dispatcher() {
        if (!dispatcher_) {
            dispatcher_ = std::make_shared<DispatcherHandler>();
            LOG_DEBUG("global dispatcher 创建");
        }
    }

} // namespace infra::net::dispatcher
