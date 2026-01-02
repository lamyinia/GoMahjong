package mahjong

import (
	"sync"
)

type Hand34 [34]uint8

type Candidate struct {
	DiscardType    TileType
	DiscardOptions []Tile     // 实体牌：红5/普通5供 UI 选择
	Waits          []TileType // 听哪些牌
	Ukeire         int        // 有效张数
}

type Searcher struct {
	mu           sync.RWMutex
	shantenCache map[string]int        // 向听数缓存
	agariCache   map[string]bool       // 和牌缓存
	waitsCache   map[string][]TileType // 听牌缓存
}

func NewSearcher() *Searcher {
	return &Searcher{
		shantenCache: make(map[string]int, 4096),
		agariCache:   make(map[string]bool, 4096),
		waitsCache:   make(map[string][]TileType, 4096),
	}
}

// SeekCandidates 弃牌后,有哪些牌听牌，是否允许立直由引擎层判断
func (s *Searcher) SeekCandidates(hand14 []Tile, fixedMelds int, visible *[34]uint8) []Candidate {

	h14, discardOpts := Hand34FromTiles(hand14)
	var out []Candidate

	for i := 0; i < 34; i++ {
		if h14[i] == 0 {
			continue
		}

		h13 := h14
		h13[i]--

		waits, ukeire := s.WaitsAndUkeire(h13, fixedMelds, visible)
		if len(waits) == 0 {
			continue
		}

		out = append(out, Candidate{
			DiscardType:    TileType(i),
			DiscardOptions: discardOpts[TileType(i)],
			Waits:          waits,
			Ukeire:         ukeire,
		})
	}

	return out
}

// WaitsAndUkeire 枚举听牌 + 计算进张
func (s *Searcher) WaitsAndUkeire(h13 Hand34, fixedMelds int, visible *[34]uint8) ([]TileType, int) {
	key := h13.keyWithFixedMelds(fixedMelds)
	s.mu.RLock()
	if v, ok := s.waitsCache[key]; ok {
		waits := make([]TileType, len(v))
		copy(waits, v)
		s.mu.RUnlock()
		return waits, s.ukeireByWaits(h13, waits, visible)
	}
	s.mu.RUnlock()

	var waits []TileType
	for t := 0; t < 34; t++ {
		if h13[t] >= 4 {
			continue
		}
		work := h13
		work[t]++
		if s.IsAgariAll(work, fixedMelds) {
			waits = append(waits, TileType(t))
		}
	}

	s.mu.Lock()
	s.waitsCache[key] = append([]TileType(nil), waits...)
	s.mu.Unlock()

	return waits, s.ukeireByWaits(h13, waits, visible)
}

// ukeireByWaits 计算听牌的进张数
func (s *Searcher) ukeireByWaits(h13 Hand34, waits []TileType, visible *[34]uint8) int {
	ukeire := 0
	for _, tt := range waits {
		idx := int(tt)
		orig := int(h13[idx])
		add := 4 - orig
		if visible != nil {
			add -= int((*visible)[idx])
			if add < 0 {
				add = 0
			}
		}
		ukeire += add
	}
	return ukeire
}

// IsAgariAll 是否和牌
func (s *Searcher) IsAgariAll(h Hand34, fixedMelds int) bool {
	key := h.keyWithFixedMelds(fixedMelds)
	s.mu.RLock()
	if v, ok := s.agariCache[key]; ok {
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	var ok bool
	if fixedMelds > 0 {
		ok = IsAgariNormal(h, fixedMelds)
	} else {
		ok = IsAgariNormal(h, 0) || IsAgariChiitoi(h) || IsAgariKokushi(h)
	}

	s.mu.Lock()
	s.agariCache[key] = ok
	s.mu.Unlock()
	return ok
}

// IsAgariNormal 普通牌型是否和牌，核心思想，找雀头、组面子
func IsAgariNormal(h Hand34, fixedMelds int) bool {
	need := 4 - fixedMelds // 需要组成的面子数
	if need < 0 {
		return false
	}

	for j := 0; j < 34; j++ {
		if h[j] < 2 {
			continue
		}
		work := h
		work[j] -= 2
		if canFormMelds(&work, need) {
			return true
		}
	}
	return false
}

// IsAgariChiitoi 七对子是否和牌
func IsAgariChiitoi(h Hand34) bool {
	pairs := 0
	for i := 0; i < 34; i++ {
		pairs += int(h[i] / 2)
	}
	return pairs >= 7
}

// IsAgariKokushi 国士无双是否和牌
func IsAgariKokushi(h Hand34) bool {
	unique := 0
	pair := false
	for _, idx := range kokushiTiles {
		if h[idx] > 0 {
			unique++
			if h[idx] >= 2 {
				pair = true
			}
		}
	}
	return unique == 13 && pair
}

func canFormMelds(h *Hand34, need int) bool {
	if need == 0 {
		for i := 0; i < 34; i++ {
			if (*h)[i] != 0 {
				return false
			}
		}
		return true
	}

	// 找第一个非 0
	i := -1
	for k := 0; k < 34; k++ {
		if (*h)[k] > 0 {
			i = k
			break
		}
	}
	if i == -1 {
		return false
	}
	// 刻子
	if (*h)[i] >= 3 {
		(*h)[i] -= 3
		if canFormMelds(h, need-1) {
			(*h)[i] += 3
			return true
		}
		(*h)[i] += 3
	}
	// 顺子（仅数牌）
	if isNumberTile(i) && i+2 < 34 && suitOf(i) == suitOf(i+1) && suitOf(i) == suitOf(i+2) {
		if (*h)[i] > 0 && (*h)[i+1] > 0 && (*h)[i+2] > 0 {
			(*h)[i]--
			(*h)[i+1]--
			(*h)[i+2]--
			if canFormMelds(h, need-1) {
				(*h)[i]++
				(*h)[i+1]++
				(*h)[i+2]++
				return true
			}
			(*h)[i]++
			(*h)[i+1]++
			(*h)[i+2]++
		}
	}

	return false
}

// -------------- 基础工具：转换与 key --------------

func Hand34FromTiles(tiles []Tile) (Hand34, map[TileType][]Tile) {
	var h Hand34
	opts := make(map[TileType][]Tile, 34)
	for _, t := range tiles {
		h[int(t.Type)]++
		opts[t.Type] = append(opts[t.Type], t)
	}
	return h, opts
}

func (h Hand34) keyWithFixedMelds(fixedMelds int) string {
	var b [35]byte
	for i := 0; i < 34; i++ {
		b[i] = byte(h[i])
	}
	b[34] = byte(fixedMelds)
	return string(b[:])
}

func isNumberTile(i int) bool { return i >= int(Man1) && i <= int(So9) }

func suitOf(i int) int {
	switch {
	case i >= int(Man1) && i <= int(Man9):
		return 0
	case i >= int(Pin1) && i <= int(Pin9):
		return 1
	case i >= int(So1) && i <= int(So9):
		return 2
	default:
		return -1
	}
}

var kokushiTiles = [13]int{
	int(Man1), int(Man9),
	int(Pin1), int(Pin9),
	int(So1), int(So9),
	int(East), int(South), int(West), int(North),
	int(White), int(Green), int(Red),
}

// ShantenAll 向听数，带副露
func (s *Searcher) ShantenAll(h Hand34, fixedMelds int) int {
	key := h.keyWithFixedMelds(fixedMelds)
	s.mu.RLock()
	if v, ok := s.shantenCache[key]; ok {
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	best := s.ShantenNormal(h, fixedMelds)
	if fixedMelds == 0 {
		if v := ShantenChiitoi(h); v < best {
			best = v
		}
		if v := ShantenKokushi(h); v < best {
			best = v
		}
	}

	s.mu.Lock()
	s.shantenCache[key] = best
	s.mu.Unlock()
	return best
}

// ShantenKokushi 国士无双向听数
func ShantenKokushi(h Hand34) int {
	unique := 0
	pair := false
	for _, idx := range kokushiTiles {
		if h[idx] > 0 {
			unique++
			if h[idx] >= 2 {
				pair = true
			}
		}
	}
	sh := 13 - unique
	if pair {
		sh--
	}
	return sh
}

// ShantenChiitoi 七对子向听数
func ShantenChiitoi(h Hand34) int {
	pairs := 0
	unique := 0
	for i := 0; i < 34; i++ {
		if h[i] > 0 {
			unique++
		}
		pairs += int(h[i] / 2)
	}
	sh := 6 - pairs
	if unique < 7 {
		sh += 7 - unique
	}
	return sh
}

func (s *Searcher) ShantenNormal(h Hand34, fixedMelds int) int {
	best := 8 // 一般型最差上界
	work := h
	dfsNormalShanten(&work, fixedMelds, 0, 0, &best)
	return best
}

// dfsNormalShanten 普通牌型向听数搜索 m：当前已经形成的面子数(包含 fixedMelds)、p：雀头数（0/1）、t：搭子数（taatsu）、best：全局最小向听
func dfsNormalShanten(h *Hand34, m int, p int, t int, best *int) {
	if m > 4 {
		return
	}

	t2 := t
	if limit := 4 - m; t2 > limit {
		t2 = limit
	}

	sh := 8 - 2*m - t2 - p
	if sh < *best {
		*best = sh
	}

	i := -1
	for k := 0; k < 34; k++ {
		if (*h)[k] > 0 {
			i = k
			break
		}
	}
	if i == -1 {
		return
	}

	if !isNumberTile(i) {
		if (*h)[i] >= 3 {
			(*h)[i] -= 3
			dfsNormalShanten(h, m+1, p, t, best)
			(*h)[i] += 3
		}

		if p == 0 && (*h)[i] >= 2 {
			(*h)[i] -= 2
			dfsNormalShanten(h, m, 1, t, best)
			(*h)[i] += 2
		}

		(*h)[i]--
		dfsNormalShanten(h, m, p, t, best)
		(*h)[i]++
		return
	}

	if (*h)[i] >= 3 {
		(*h)[i] -= 3
		dfsNormalShanten(h, m+1, p, t, best)
		(*h)[i] += 3
	}

	if i+2 < 34 && suitOf(i) == suitOf(i+1) && suitOf(i) == suitOf(i+2) {
		if (*h)[i] > 0 && (*h)[i+1] > 0 && (*h)[i+2] > 0 {
			(*h)[i]--
			(*h)[i+1]--
			(*h)[i+2]--
			dfsNormalShanten(h, m+1, p, t, best)
			(*h)[i]++
			(*h)[i+1]++
			(*h)[i+2]++
		}
	}

	if p == 0 && (*h)[i] >= 2 {
		(*h)[i] -= 2
		dfsNormalShanten(h, m, 1, t, best)
		(*h)[i] += 2
	}

	if i+1 < 34 && suitOf(i) == suitOf(i+1) {
		if (*h)[i] > 0 && (*h)[i+1] > 0 {
			(*h)[i]--
			(*h)[i+1]--
			dfsNormalShanten(h, m, p, t+1, best)
			(*h)[i]++
			(*h)[i+1]++
		}
	}

	if i+2 < 34 && suitOf(i) == suitOf(i+2) {
		if (*h)[i] > 0 && (*h)[i+2] > 0 {
			(*h)[i]--
			(*h)[i+2]--
			dfsNormalShanten(h, m, p, t+1, best)
			(*h)[i]++
			(*h)[i+2]++
		}
	}

	(*h)[i]--
	dfsNormalShanten(h, m, p, t, best)
	(*h)[i]++
}
