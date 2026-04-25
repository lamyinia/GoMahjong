#include "domain/game/engine/mahjong/hu_searcher.h"

#include <algorithm>

namespace domain::game::mahjong {

    namespace {

        [[nodiscard]] int suitOfIndex(int i) {
            if (i <= 8) return 0;
            if (i <= 17) return 1;
            if (i <= 26) return 2;
            return -1;
        }

        [[nodiscard]] bool isNumberIndex(int i) {
            return i >= 0 && i <= 26;
        }

    }


[[nodiscard]] std::pair<Hand34, std::unordered_map<TileType, std::vector<Tile>>>
hand34FromTiles(const std::vector<Tile>& tiles) {
    Hand34 h{};
    std::unordered_map<TileType, std::vector<Tile>> opts;
    for (const auto& t : tiles) {
        auto idx = toIndex34(t.type);
        if (idx >= 0 && idx < kTileTypeCount) {
            h[idx]++;
            opts[t.type].push_back(t);
        }
    }
    return {h, opts};
}

Searcher::Searcher() = default;

[[nodiscard]] std::vector<Candidate> Searcher::seekCandidates(const std::vector<Tile>& hand14, int fixed_melds) {
    auto [h14, discardOpts] = hand34FromTiles(hand14);
    std::vector<Candidate> out;
    for (int i = 0; i < kTileTypeCount; ++i) {
        if (h14[i] == 0) continue;
        Hand34 h13 = h14;
        h13[i]--;
        auto w = waits(h13, fixed_melds);
        if (w.empty()) continue;
        auto tt = fromIndex34(i);
        out.push_back(Candidate{
            tt,
            discardOpts.contains(tt) ? discardOpts[tt] : std::vector<Tile>{},
            std::move(w),
        });
    }
    return out;
}

[[nodiscard]] std::vector<TileType> Searcher::waits(Hand34 h13, int fixed_melds) {
    auto key = makeKey(h13, fixed_melds);
    {
        std::shared_lock lock(mu_);
        auto it = waits_cache_.find(key);
        if (it != waits_cache_.end()) {
            return it->second;
        }
    }

    std::vector<TileType> result;
    for (int t = 0; t < kTileTypeCount; ++t) {
        if (h13[t] >= 4) continue;
        Hand34 work = h13;
        work[t]++;
        if (isAgariAll(work, fixed_melds)) {
            result.push_back(fromIndex34(t));
        }
    }

    {
        std::unique_lock lock(mu_);
        waits_cache_[key] = result;
    }
    return result;
}

[[nodiscard]] bool Searcher::isAgariAll(Hand34 h, int fixed_melds) {
    auto key = makeKey(h, fixed_melds);
    {
        std::shared_lock lock(mu_);
        auto it = agari_cache_.find(key);
        if (it != agari_cache_.end()) {
            return it->second;
        }
    }

    bool ok = false;
    if (fixed_melds > 0) {
        ok = isAgariNormal(h, fixed_melds);
    } else {
        ok = isAgariNormal(h, 0) || isAgariChiitoi(h) || isAgariKokushi(h);
    }

    {
        std::unique_lock lock(mu_);
        agari_cache_[key] = ok;
    }
    return ok;
}

[[nodiscard]] bool Searcher::isAgariNormal(Hand34 h, int fixed_melds) {
    int need = 4 - fixed_melds;
    if (need < 0) return false;

    for (int j = 0; j < kTileTypeCount; ++j) {
        if (h[j] < 2) continue;
        Hand34 work = h;
        work[j] -= 2;
        if (canFormMelds(work, need)) {
            return true;
        }
    }
    return false;
}

[[nodiscard]] bool Searcher::isAgariChiitoi(Hand34 h) {
    int pairs = 0;
    for (int i = 0; i < kTileTypeCount; ++i) {
        pairs += h[i] / 2;
    }
    return pairs >= 7;
}

[[nodiscard]] bool Searcher::isAgariKokushi(Hand34 h) {
    int unique = 0;
    bool pair = false;
    for (int idx : kKokushiTiles) {
        if (h[idx] > 0) {
            unique++;
            if (h[idx] >= 2) {
                pair = true;
            }
        }
    }
    return unique == 13 && pair;
}

[[nodiscard]] bool Searcher::canFormMelds(Hand34& h, int need) {
    if (need == 0) {
        for (int i = 0; i < kTileTypeCount; ++i) {
            if (h[i] != 0) return false;
        }
        return true;
    }

    // 找第一个非 0
    int i = -1;
    for (int k = 0; k < kTileTypeCount; ++k) {
        if (h[k] > 0) {
            i = k;
            break;
        }
    }
    if (i == -1) return false;

    // 刻子
    if (h[i] >= 3) {
        h[i] -= 3;
        if (canFormMelds(h, need - 1)) {
            h[i] += 3;
            return true;
        }
        h[i] += 3;
    }

    // 顺子（仅数牌，同花色三连）
    if (isNumberIndex(i) && i + 2 < kTileTypeCount
        && suitOfIndex(i) == suitOfIndex(i + 1) && suitOfIndex(i) == suitOfIndex(i + 2)) {
        if (h[i] > 0 && h[i + 1] > 0 && h[i + 2] > 0) {
            h[i]--;
            h[i + 1]--;
            h[i + 2]--;
            if (canFormMelds(h, need - 1)) {
                h[i]++;
                h[i + 1]++;
                h[i + 2]++;
                return true;
            }
            h[i]++;
            h[i + 1]++;
            h[i + 2]++;
        }
    }

    return false;
}

[[nodiscard]] std::string Searcher::makeKey(Hand34 h, int fixed_melds) {
    std::string key(35, '\0');
    for (int i = 0; i < kTileTypeCount; ++i) {
        key[static_cast<size_t>(i)] = static_cast<char>(h[i]);
    }
    key[34] = static_cast<char>(fixed_melds);
    return key;
}

} // namespace domain::game::mahjong
