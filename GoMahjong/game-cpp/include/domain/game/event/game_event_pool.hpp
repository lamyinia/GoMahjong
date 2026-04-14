#pragma once

#include "domain/game/event/mahjong_game_event.h"
#include "infrastructure/util/memory/typed_object_pool.hpp"

#include <utility>

namespace domain::game::event {

/**
 * GameEventPool - Specialized object pool for GameEvent
 * 
 * Features:
 * - Type-safe event construction
 * - Automatic memory management
 * - High performance for frequent event allocation
 * 
 * Usage:
 *   GameEventPool pool(4096);
 *   
 *   // Template-based acquisition
 *   auto event = pool.acquire<PlayTileEvent>(playerId, tile);
 *   
 *   // Convenience methods
 *   auto event = pool.playTile(playerId, tile);
 */
class GameEventPool {
public:
    using PooledEvent = infra::util::memory::PooledPtr<GameEvent>;

    /**
     * Constructor
     * @param capacity Initial number of events to preallocate
     */
    explicit GameEventPool(std::size_t capacity = 4096)
        : pool_(capacity)
    {}

    /**
     * Acquire an event from the pool with specific event type
     * @tparam EventT Event data type (e.g., PlayTileEvent)
     * @tparam Args Constructor arguments for EventT
     * @param args Arguments to forward to EventT constructor
     * @return Pooled event pointer
     */
    template <typename EventT, typename... Args>
    PooledEvent acquire(Args&&... args) {
        auto event = pool_.acquire();
        if (!event) {
            return event;
        }
        
        // Set event type based on EventT
        event->type = eventTypeFrom<EventT>();
        
        // Construct event data in-place
        event->data.template emplace<EventT>(std::forward<Args>(args)...);
        
        return event;
    }

    // ==================== Convenience Methods ====================

    /** Create PlayTile event */
    PooledEvent playTile(const std::string& playerId, const Tile& tile) {
        return acquire<PlayTileEvent>(playerId, tile);
    }

    /** Create DrawTile event */
    PooledEvent drawTile(const std::string& playerId, const Tile& tile) {
        return acquire<DrawTileEvent>(playerId, tile);
    }

    /** Create Chi event */
    PooledEvent chi(const std::string& playerId, const Tile& tile, TileType meldType) {
        return acquire<ChiEvent>(playerId, tile, meldType);
    }

    /** Create Pon event */
    PooledEvent pon(const std::string& playerId, const Tile& tile) {
        return acquire<PonEvent>(playerId, tile);
    }

    /** Create Kan event */
    PooledEvent kan(const std::string& playerId, const Tile& tile, bool isAnkan = false, bool isKakan = false) {
        return acquire<KanEvent>(playerId, tile, isAnkan, isKakan);
    }

    /** Create Ron event */
    PooledEvent ron(const std::string& playerId, const Tile& tile, const std::string& targetPlayerId) {
        return acquire<RonEvent>(playerId, tile, targetPlayerId);
    }

    /** Create Tsumo event */
    PooledEvent tsumo(const std::string& playerId, const Tile& tile) {
        return acquire<TsumoEvent>(playerId, tile);
    }

    /** Create Draw (ryuukyoku) event */
    PooledEvent draw(bool isKyuushuKyuukai = false) {
        return acquire<DrawEvent>(isKyuushuKyuukai);
    }

    /** Create PlayerTimeout event */
    PooledEvent playerTimeout(const std::string& playerId, std::int32_t seatIndex) {
        return acquire<PlayerTimeoutEvent>(playerId, seatIndex);
    }

    /** Create TurnStart event */
    PooledEvent turnStart(const std::string& playerId, std::int32_t timeLimit = 30) {
        return acquire<TurnStartEvent>(playerId, timeLimit);
    }

    /** Create TurnEnd event */
    PooledEvent turnEnd(const std::string& playerId) {
        return acquire<TurnEndEvent>(playerId);
    }

    /** Create RoundStart event */
    PooledEvent roundStart(std::int32_t roundNumber, const std::string& oyaPlayerId) {
        return acquire<RoundStartEvent>(roundNumber, oyaPlayerId);
    }

    /** Create RoundEnd event */
    PooledEvent roundEnd(std::int32_t roundNumber) {
        return acquire<RoundEndEvent>(roundNumber);
    }

    /** Create GameStart event */
    PooledEvent gameStart(const std::string& roomId) {
        return acquire<GameStartEvent>(roomId);
    }

    /** Create GameEnd event */
    PooledEvent gameEnd(const std::string& roomId) {
        return acquire<GameEndEvent>(roomId);
    }

    // ==================== Pool Statistics ====================

    [[nodiscard]] std::size_t capacity() const noexcept { return pool_.capacity(); }
    [[nodiscard]] std::size_t used() const noexcept { return pool_.used(); }
    [[nodiscard]] std::size_t available() const noexcept { return pool_.available(); }

    /**
     * Expand pool capacity
     * @param count Number of additional events to preallocate
     */
    bool expand(std::size_t count) {
        return pool_.expand(count);
    }

private:
    infra::util::memory::TypedObjectPool<GameEvent> pool_;

    // ==================== Type-to-EventType Mapping ====================

    template <typename EventT>
    static constexpr EventType eventTypeFrom();

    // Specializations for each event type
    template <> static constexpr EventType eventTypeFrom<PlayTileEvent>() { return EventType::PlayTile; }
    template <> static constexpr EventType eventTypeFrom<DrawTileEvent>() { return EventType::DrawTile; }
    template <> static constexpr EventType eventTypeFrom<ChiEvent>() { return EventType::Chi; }
    template <> static constexpr EventType eventTypeFrom<PonEvent>() { return EventType::Pon; }
    template <> static constexpr EventType eventTypeFrom<KanEvent>() { return EventType::Kan; }
    template <> static constexpr EventType eventTypeFrom<RonEvent>() { return EventType::Ron; }
    template <> static constexpr EventType eventTypeFrom<TsumoEvent>() { return EventType::Tsumo; }
    template <> static constexpr EventType eventTypeFrom<DrawEvent>() { return EventType::Draw; }
    template <> static constexpr EventType eventTypeFrom<PlayerTimeoutEvent>() { return EventType::PlayerTimeout; }
    template <> static constexpr EventType eventTypeFrom<TurnStartEvent>() { return EventType::TurnStart; }
    template <> static constexpr EventType eventTypeFrom<TurnEndEvent>() { return EventType::TurnEnd; }
    template <> static constexpr EventType eventTypeFrom<RoundStartEvent>() { return EventType::RoundStart; }
    template <> static constexpr EventType eventTypeFrom<RoundEndEvent>() { return EventType::RoundEnd; }
    template <> static constexpr EventType eventTypeFrom<GameStartEvent>() { return EventType::GameStart; }
    template <> static constexpr EventType eventTypeFrom<GameEndEvent>() { return EventType::GameEnd; }
};

} // namespace domain::game::event
