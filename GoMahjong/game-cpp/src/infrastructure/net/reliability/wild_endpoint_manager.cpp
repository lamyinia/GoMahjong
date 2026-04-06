#include "infrastructure/net/reliability/wild_endpoint_manager.h"
#include "infrastructure/net/dispatcher/dispatcher.h"
#include "infrastructure/log/logger.hpp"

namespace infra::net::reliability {

    WildEndpointManager::WildEndpointManager(
        boost::asio::any_io_executor executor,
        std::chrono::milliseconds auth_timeout
    )
        : executor_(std::move(executor))
        , auth_timeout_(auth_timeout) {
    }

    WildEndpointManager::~WildEndpointManager() {
        std::lock_guard<std::mutex> lock(mutex_);
        endpoints_.clear();
    }

    void WildEndpointManager::add_channel(std::shared_ptr<channel::IChannel> channel) {
        if (!channel) {
            LOG_ERROR("[WildEndpointManager] channel is null");
            return;
        }

        auto endpoint = std::make_shared<WildEndpoint>(executor_, channel, auth_timeout_);
        auto endpoint_id = endpoint->id();

        LOG_DEBUG("[WildEndpointManager] add channel, endpoint_id={}", endpoint_id);

        auto self = shared_from_this();
        // on_auth_success 回调链路 => auth_handler -> wild_endpoint -> wild_endpoint_mgr
        endpoint->set_on_auth_success([self, endpoint_id](const std::string& player_id) {
            self->on_endpoint_authenticated(endpoint_id, player_id);
        });
        endpoint->set_on_auth_failed([self, endpoint_id]() {
            self->on_endpoint_auth_failed(endpoint_id);
        });

        {
            std::lock_guard<std::mutex> lock(mutex_);
            endpoints_[endpoint_id] = endpoint;
        }

        endpoint->start_wait_auth();
    }

    void WildEndpointManager::remove_endpoint(const std::string& endpoint_id) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = endpoints_.find(endpoint_id);
        if (it != endpoints_.end()) {
            LOG_DEBUG("remove endpoint, endpoint_id={}", endpoint_id);
            endpoints_.erase(it);
        }
    }

    size_t WildEndpointManager::size() const {
        std::lock_guard<std::mutex> lock(mutex_);
        return endpoints_.size();
    }

    void WildEndpointManager::on_endpoint_authenticated(
        const std::string& endpoint_id,
        const std::string& player_id
    ) {
        LOG_DEBUG("endpoint authenticated, endpoint_id={}, player_id={}",
                 endpoint_id, player_id);

        std::shared_ptr<channel::IChannel> channel;
        {
            std::lock_guard<std::mutex> lock(mutex_);
            auto it = endpoints_.find(endpoint_id);
            if (it != endpoints_.end()) {
                channel = it->second->channel();
                endpoints_.erase(it);
            }
        }

        if (channel) {
            auto dispatcher = dispatcher::Dispatcher::instance().create_dispatcher();
            channel->add_inbound(dispatcher);
            LOG_DEBUG("dispatcher added to channel for player {}", player_id);

            if (onAuthenticated_) {
                onAuthenticated_(player_id, channel);
            }
        }
    }

    void WildEndpointManager::on_endpoint_auth_failed(const std::string& endpoint_id) {
        LOG_WARN("endpoint auth failed, endpoint_id={}", endpoint_id);

        std::lock_guard<std::mutex> lock(mutex_);
        auto it = endpoints_.find(endpoint_id);
        if (it != endpoints_.end()) {
            endpoints_.erase(it);
        }
    }

} // namespace infra::net::reliability
