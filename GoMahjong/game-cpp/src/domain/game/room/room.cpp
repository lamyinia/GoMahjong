#include "domain/game/room/room.h"

#include "domain/game/engine/engine.h"
#include "infrastructure/log/logger.hpp"

#include <algorithm>

namespace domain::game::room {

    Room::Room(std::string id, std::int32_t engineType)
        : id_(std::move(id)), engineType_(engineType) {
    }

    Room::~Room() = default;

    Room::Room(Room&& other) noexcept
        : id_(std::move(other.id_))
        , engineType_(other.engineType_)
        , players_(std::move(other.players_))
        , engine_(std::move(other.engine_))
        , engineContext_(std::move(other.engineContext_)) {
    }

    Room& Room::operator=(Room&& other) noexcept {
        if (this != &other) {
            id_ = std::move(other.id_);
            engineType_ = other.engineType_;
            players_ = std::move(other.players_);
            engine_ = std::move(other.engine_);
            engineContext_ = std::move(other.engineContext_);
        }
        return *this;
    }

    void Room::addPlayer(const std::string& userId) {
        players_.push_back(userId);
    }

    void Room::removePlayer(const std::string& userId) {
        auto it = std::find(players_.begin(), players_.end(), userId);
        if (it != players_.end()) {
            players_.erase(it);
        }
    }

    bool Room::hasPlayer(const std::string& userId) const {
        return std::find(players_.begin(), players_.end(), userId) != players_.end();
    }

    void Room::initGame() {
        // 创建 EngineContext
        engineContext_ = std::make_unique<engine::EngineContext>();
        engineContext_->setRoomId(id_);
        engineContext_->setPlayerIds(players_);

        // 创建游戏状态机
        auto engineType = static_cast<engine::EngineType>(engineType_);
        engine_ = engine::Engine::create(engineType);
        
        if (!engine_) {
            LOG_ERROR("[Room] failed to create engine for room {} with type {}", id_, engineType_);
            return;
        }

        // 注入 EngineContext
        engine_->setContext(engineContext_.get());

        for (const auto& playId : players_) {
            engine_->onPlayerJoin(playId);
        }
        if (engine_->canStart()) {
            engine_->start();
            LOG_DEBUG("game initialized for room {} with {} players", id_, players_.size());
        } else {
            LOG_WARN("cannot start game for room {}, not enough players", id_);
        }
    }

    void Room::handleEvent(const event::GameEvent& event) {
        if (!engine_) {
            LOG_ERROR("[Room] engine not initialized for room {}", id_);
            return;
        }

        engine_->handleEvent(event);
    }

    bool Room::isGameOver() const {
        return engine_ && engine_->isGameOver();
    }

} // namespace domain::game::room
