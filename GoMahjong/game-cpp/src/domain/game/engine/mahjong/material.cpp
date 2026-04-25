#include "domain/game/engine/mahjong/material.h"

#include <algorithm>

namespace domain::game::mahjong {

    // === 构造函数：生成136张牌（只做一次）===
    DeckManager::DeckManager(std::uint32_t seed): rng_(seed == 0 ? std::mt19937(std::random_device{}()) : std::mt19937(seed)){
        int idx = 0;

        // 数牌：万(0-8)、筒(9-17)、索(18-26)
        for (int suit = 0; suit < 3; ++suit) {
            for (int rank = 0; rank < 9; ++rank) {
                auto tileType = static_cast<TileType>(suit * 9 + rank);
                for (int id = 0; id < 4; ++id) {
                    wall_[idx++] = Tile{tileType, static_cast<std::int8_t>(id)};
                }
            }
        }
        // 字牌：风(27-30)、箭(31-33)
        for (int honor = 27; honor <= 33; ++honor) {
            auto tileType = static_cast<TileType>(honor);
            for (int id = 0; id < 4; ++id) {
                wall_[idx++] = Tile{tileType, static_cast<std::int8_t>(id)};
            }
        }

        // 34种 × 4张 = 136
        wallSize_ = kTileLimit - kWangTileCount; // 122
    }

    void DeckManager::initRound() {
        shuffle();

        wallIndex_ = 0;
        wang_ = Wang{};

        for (int i = 0; i < kTileTypeCount; ++i) {
            remain34_[i] = 4;
        }

        assignWang();
    }

    void DeckManager::shuffle() {
        // Fisher-Yates 洗牌整个 wall_
        for (int i = kTileLimit - 1; i > 0; --i) {
            auto j = static_cast<int>(rng_() % (i + 1));
            std::swap(wall_[i], wall_[j]);
        }
    }

    void DeckManager::assignWang() {
        // wall_ 末尾14张作为王牌
        // [0-3] 岭上牌, [4-8] 宝牌指示牌, [9-13] 里宝牌指示牌
        int deadStart = wallSize_; // 122
        for (int i = 0; i < kKanTileCount; ++i) {
            wang_.kanTiles[i] = wall_[deadStart + i];
        }
        for (int i = 0; i < kDoraIndicatorCount; ++i) {
            wang_.doraIndicators[i] = wall_[deadStart + 4 + i];
        }
        for (int i = 0; i < kDoraIndicatorCount; ++i) {
            wang_.uraDoraIndicators[i] = wall_[deadStart + 9 + i];
        }
    }

    std::pair<Tile, bool> DeckManager::draw() {
        if (wallIndex_ >= wallSize_) {
            return {Tile{}, false};
        }
        auto tile = wall_[wallIndex_++];
        remain34_[toIndex34(tile.type)]--;
        return {tile, true};
    }

    std::pair<Tile, bool> DeckManager::deal() {
        return draw();
    }

    std::pair<Tile, bool> DeckManager::drawKanTile() {
        if (wang_.kanIndex >= kKanTileCount) {
            return {Tile{}, false};
        }
        auto tile = wang_.kanTiles[wang_.kanIndex++];
        remain34_[toIndex34(tile.type)]--;
        return {tile, true};
    }

    int DeckManager::remainingKanTiles() const {
        return kKanTileCount - wang_.kanIndex;
    }

    bool DeckManager::canKan() const {
        return wang_.kanIndex < kKanTileCount;
    }

    bool DeckManager::revealDoraIndicator() {
        if (wang_.doraIndex >= kDoraIndicatorCount) {
            return false;
        }
        auto tile = wang_.doraIndicators[wang_.doraIndex++];
        remain34_[toIndex34(tile.type)]--;
        return true;
    }

    void DeckManager::revealAllUraDora() {
        if (wang_.uraDoraRevealed) return;
        wang_.uraDoraRevealed = true;
        for (int i = 0; i < wang_.doraIndex; ++i) {
            remain34_[toIndex34(wang_.uraDoraIndicators[i].type)]--;
        }
    }

    std::vector<Tile> DeckManager::getDoraIndicators() const {
        return std::vector<Tile>(wang_.doraIndicators.begin(), wang_.doraIndicators.begin() + wang_.doraIndex);
    }

    std::vector<Tile> DeckManager::getUraDoraIndicators() const {
        if (!wang_.uraDoraRevealed) return {};
        return std::vector<Tile>(wang_.uraDoraIndicators.begin(), wang_.uraDoraIndicators.begin() + wang_.doraIndex);
    }

    Hand34 DeckManager::visible34() const {
        Hand34 result{};
        for (int i = 0; i < kTileTypeCount; ++i) {
            int v = 4 - remain34_[i];
            if (v < 0) v = 0;
            if (v > 4) v = 4;
            result[i] = static_cast<std::uint8_t>(v);
        }
        return result;
    }

    int DeckManager::remainingWallTiles() const {
        return wallSize_ - wallIndex_;
    }

    bool DeckManager::isWallEmpty() const {
        return wallIndex_ >= wallSize_;
    }

    const Wang& DeckManager::wang() const {
        return wang_;
    }

} // namespace domain::game::mahjong