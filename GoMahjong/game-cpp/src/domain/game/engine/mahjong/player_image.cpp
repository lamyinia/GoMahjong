#include "domain/game/engine/mahjong/player_image.h"

namespace domain::game::mahjong {

PlayerImage::PlayerImage(std::string player_id, int seat_index, int initial_points)
    : player_id_(std::move(player_id))
    , seat_index_(seat_index)
    , points_(initial_points)
{
    tiles_.reserve(14);
    discard_pile_.reserve(18);
    melds_.reserve(4);
}

[[nodiscard]] int PlayerImage::seatIndex() const {
    return seat_index_;
}

[[nodiscard]] const std::string& PlayerImage::playerId() const {
    return player_id_;
}

[[nodiscard]] bool PlayerImage::isRiichi() const {
    return is_riichi_;
}

[[nodiscard]] const std::optional<Tile>& PlayerImage::newestTile() const {
    return newest_tile_;
}

[[nodiscard]] const std::vector<Tile>& PlayerImage::tiles() const {
    return tiles_;
}

[[nodiscard]] const std::vector<Tile>& PlayerImage::discardPile() const {
    return discard_pile_;
}

[[nodiscard]] const std::vector<Meld>& PlayerImage::melds() const {
    return melds_;
}

void PlayerImage::addTile(Tile tile) {
    tiles_.push_back(tile);
}

void PlayerImage::drawTile(Tile tile) {
    tiles_.push_back(tile);
    newest_tile_ = tile;
}

bool PlayerImage::removeTile(Tile tile) {
    for (auto it = tiles_.begin(); it != tiles_.end(); ++it) {
        if (it->type == tile.type && it->id == tile.id) {
            tiles_.erase(it);
            return true;
        }
    }
    return false;
}

bool PlayerImage::discardTile(Tile tile) {
    if (!removeTile(tile)) {
        return false;
    }
    discard_pile_.push_back(tile);
    discarded_types_[toIndex34(tile.type)] = true;
    if (newest_tile_.has_value() && newest_tile_->type == tile.type && newest_tile_->id == tile.id) {
        newest_tile_.reset();
    }
    return true;
}

std::pair<Tile, bool> PlayerImage::discardNewestOrLast() {
    if (tiles_.size() != 14) {
        return {Tile{}, false};
    }
    Tile tile;
    if (newest_tile_.has_value()) {
        tile = *newest_tile_;
    } else {
        tile = tiles_.back();
    }
    if (!discardTile(tile)) {
        return {Tile{}, false};
    }
    return {tile, true};
}

void PlayerImage::addMeld(Meld meld) {
    melds_.push_back(std::move(meld));
}

void PlayerImage::addPoints(int delta) {
    points_ += delta;
}

[[nodiscard]] int PlayerImage::points() const {
    return points_;
}

bool PlayerImage::hasDiscardedType(TileType type) const {
    auto idx = toIndex34(type);
    if (idx < 0 || idx >= kTileTypeCount) return false;
    return discarded_types_[idx];
}

void PlayerImage::setRiichi(bool value) {
    is_riichi_ = value;
}

void PlayerImage::setWaiting(bool value) {
    is_waiting_ = value;
}

void PlayerImage::resetForRound() {
    tiles_.clear();
    discard_pile_.clear();
    melds_.clear();
    is_riichi_ = false;
    is_waiting_ = false;
    discarded_types_.fill(false);
    newest_tile_.reset();
}

} // namespace domain::game::mahjong
