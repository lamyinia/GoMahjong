package mahjong

import "math"

// Yaku 役种（和牌方式）
type Yaku int

// 役种常量定义
const (
	// 基本役
	YakuRiichi Yaku = iota // 立直：门清状态下宣布立直，并放置1000点棒
	YakuTsumo              // 门前清自摸和：门清状态下自摸和牌

	// 平和系
	YakuPinfu     // 平和：4顺子+非役牌雀头，两面听牌
	YakuIppeiko   // 一杯口：同种花色、同种顺子有两组
	YakuRyanpeiko // 二杯口：手牌中有两个不同的一杯口

	// 役牌系
	YakuYakuhai // 役牌：场风、自风、三元牌的刻子/杠子

	// 断幺系
	YakuTanyao // 断幺九：手牌全部由数牌2-8组成

	// 顺子系
	YakuSanshoku // 三色同顺：相同顺子在三种花色中都出现
	YakuIttsu    // 一气通贯：同种花色有123、456、789三个顺子

	// 带幺系
	YakuChanta  // 混全带幺九：所有面子都包含幺九牌
	YakuJunchan // 纯全带幺九：所有面子都包含数牌幺九(1、9)

	// 老头系
	YakuHonroto  // 混老头：全部由幺九牌(1、9、字牌)组成的对对和
	YakuChinroto // 清老头：全部由数牌幺九(1、9)组成的对对和

	// 清一色系
	YakuHonitsu  // 混一色：一种花色+字牌
	YakuChinitsu // 清一色：同一种花色(无字牌)

	// 刻子系
	YakuToitoi    // 对对和：4个刻子(杠子)+1个对子
	YakuSananko   // 三暗刻：手牌中有3个暗刻
	YakuSankantsu // 三杠子：手牌中有3个杠子

	// 特殊型
	YakuChiitoi // 七对子：7个不同的对子
	YakuKokushi // 国士无双(十三幺)：13种幺九牌各1张+其中任意1张

	// 役满役种
	YakuSuuankou      // 四暗刻：手牌中有四个暗刻
	YakuSuuankouTanki // 四暗刻单骑：四暗刻且听牌形式为单骑
	YakuDaisushi      // 大四喜（双倍）
	YakuKokushi13     // 国士十三面（双倍）
	YakuChuuren       // 九莲宝灯：同一种花色的1112345678999，加上任意一张同花色的牌
	YakuJunseiChuuren // 纯正九莲宝灯：九莲宝灯听所有的9种牌
	YakuKazoeYakuman  // 累计役满：手牌的番数累计达到或超过13番
)

type RoundScoreDetail struct {
	Yakus        []Yaku
	YakumanValue int
}

type YakuContext struct {
	Claim     HuClaim
	Winner    *PlayerImage
	Situation *Situation
	EndKind   string
}

type YakuChecker interface {
	ID() Yaku
	Check(ctx *YakuContext) (int, int)
}

type yakuCheckerFunc struct {
	id    Yaku
	check func(ctx *YakuContext) (int, int)
}

func (f yakuCheckerFunc) ID() Yaku { return f.id }

func (f yakuCheckerFunc) Check(ctx *YakuContext) (int, int) { return f.check(ctx) }

func (eg *RiichiMahjong4p) GetFanfuAndYakus(claim HuClaim) (int, int, []Yaku) {
	var winner *PlayerImage
	if claim.WinnerSeat >= 0 && claim.WinnerSeat < 4 {
		winner = eg.Players[claim.WinnerSeat]
	}

	ctx := &YakuContext{Claim: claim, Winner: winner, Situation: eg.Situation}
	results := make([]Yaku, 0, 8)
	for _, checker := range RiichiMahjong4pYakuRegistry {
		han, yakumanMult := checker.Check(ctx)
		if han > 0 || yakumanMult > 0 {
			results = append(results, checker.ID())
		}
	}
	return 0, 0, results
}

func roundUpTo100(x int) int {
	return int(math.Ceil(float64(x)/100.0)) * 100
}

var RiichiMahjong4pYakuRegistry = []YakuChecker{
	yakuCheckerFunc{id: YakuSuuankouTanki, check: func(ctx *YakuContext) (int, int) {
		if checkSuuankouTanki(ctx) {
			return 0, 2
		}
		return 0, 0
	}},
	yakuCheckerFunc{id: YakuDaisushi, check: func(ctx *YakuContext) (int, int) {
		if checkDaisushi(ctx) {
			return 0, 2
		}
		return 0, 0
	}},
	yakuCheckerFunc{id: YakuKokushi13, check: func(ctx *YakuContext) (int, int) {
		if checkKokushi13(ctx) {
			return 0, 2
		}
		return 0, 0
	}},
	yakuCheckerFunc{id: YakuJunseiChuuren, check: func(ctx *YakuContext) (int, int) {
		if checkJunseiChuuren(ctx) {
			return 0, 2
		}
		return 0, 0
	}},

	// 基本役
	yakuCheckerFunc{id: YakuRiichi, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuTsumo, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},

	// 平和系
	yakuCheckerFunc{id: YakuPinfu, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuIppeiko, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuRyanpeiko, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},

	// 役牌系
	yakuCheckerFunc{id: YakuYakuhai, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},

	// 断幺系
	yakuCheckerFunc{id: YakuTanyao, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},

	// 顺子系
	yakuCheckerFunc{id: YakuSanshoku, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuIttsu, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},

	// 带幺系
	yakuCheckerFunc{id: YakuChanta, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuJunchan, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},

	// 老头系
	yakuCheckerFunc{id: YakuHonroto, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuChinroto, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},

	// 清一色系
	yakuCheckerFunc{id: YakuHonitsu, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuChinitsu, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},

	// 刻子系
	yakuCheckerFunc{id: YakuToitoi, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuSananko, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuSankantsu, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},

	// 特殊型
	yakuCheckerFunc{id: YakuChiitoi, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
	yakuCheckerFunc{id: YakuKokushi, check: func(ctx *YakuContext) (int, int) { return 0, 0 }},
}

func isHonor(tt TileType) bool { return tt >= East }

func suitOfTileType(tt TileType) int {
	switch {
	case tt >= Man1 && tt <= Man9:
		return 0
	case tt >= Pin1 && tt <= Pin9:
		return 1
	case tt >= So1 && tt <= So9:
		return 2
	default:
		return -1
	}
}

func numberIndex(tt TileType) int {
	switch {
	case tt >= Man1 && tt <= Man9:
		return int(tt - Man1)
	case tt >= Pin1 && tt <= Pin9:
		return int(tt - Pin1)
	case tt >= So1 && tt <= So9:
		return int(tt - So1)
	default:
		return -1
	}
}

func kokushiTileTypes() []TileType {
	return []TileType{Man1, Man9, Pin1, Pin9, So1, So9, East, South, West, North, White, Green, Red}
}

func isKokushiTileType(tt TileType) bool {
	switch tt {
	case Man1, Man9, Pin1, Pin9, So1, So9, East, South, West, North, White, Green, Red:
		return true
	default:
		return false
	}
}

// checkSuuankouTanki 四暗刻单骑
func checkSuuankouTanki(ctx *YakuContext) bool {
	if ctx == nil || ctx.Winner == nil {
		return false
	}
	if len(ctx.Winner.Melds) != 0 {
		return false
	}
	counts, total := buildTileTypeCountsForClaim(ctx)
	if total != 14 {
		return false
	}
	trip := 0
	pair := TileType(-1)
	for _, c := range counts {
		if c == 0 {
			continue
		}
		switch c {
		case 2:
			pair = 0
		case 3, 4:
			trip++
		default:
			return false
		}
	}
	// need exactly one pair and four triplets/quads
	pairCount := 0
	for tt, c := range counts {
		if c == 2 {
			pair = tt
			pairCount++
		}
	}
	if trip != 4 || pairCount != 1 {
		return false
	}
	return pair == ctx.Claim.WinTile.Type
}

// checkDaisushi check 大四喜
func checkDaisushi(ctx *YakuContext) bool {
	counts, _ := buildTileTypeCountsForClaim(ctx)
	return counts[East] >= 3 && counts[South] >= 3 && counts[West] >= 3 && counts[North] >= 3
}

// checkKokushi13 check 国士无双
func checkKokushi13(ctx *YakuContext) bool {
	if ctx == nil || ctx.Winner == nil {
		return false
	}
	if len(ctx.Winner.Melds) != 0 {
		return false
	}
	counts, total := buildTileTypeCountsForClaim(ctx)
	if total != 14 {
		return false
	}
	winTT := ctx.Claim.WinTile.Type
	for tt, c := range counts {
		if c == 0 {
			continue
		}
		if !isKokushiTileType(tt) {
			return false
		}
		if c > 2 {
			return false
		}
	}
	for _, tt := range kokushiTileTypes() {
		c := counts[tt]
		if tt == winTT {
			if c != 2 {
				return false
			}
		} else {
			if c != 1 {
				return false
			}
		}
	}
	return true
}

// checkJunseiChuuren check 纯正九莲宝灯
func checkJunseiChuuren(ctx *YakuContext) bool {
	if ctx == nil || ctx.Winner == nil {
		return false
	}
	if len(ctx.Winner.Melds) != 0 {
		return false
	}
	counts, total := buildTileTypeCountsForClaim(ctx)
	if total != 14 {
		return false
	}
	if isHonor(ctx.Claim.WinTile.Type) {
		return false
	}

	suit := -1
	var c9 [9]int
	for tt, c := range counts {
		if c == 0 {
			continue
		}
		if isHonor(tt) {
			return false
		}
		s := suitOfTileType(tt)
		if s < 0 {
			return false
		}
		if suit == -1 {
			suit = s
		} else if suit != s {
			return false
		}
		n := numberIndex(tt)
		if n < 0 {
			return false
		}
		c9[n] = c
	}
	if suit == -1 {
		return false
	}
	if suitOfTileType(ctx.Claim.WinTile.Type) != suit {
		return false
	}

	base := [9]int{3, 1, 1, 1, 1, 1, 1, 1, 3}
	idx := numberIndex(ctx.Claim.WinTile.Type)
	if idx < 0 {
		return false
	}
	work := c9
	work[idx]--
	for i := 0; i < 9; i++ {
		if work[i] != base[i] {
			return false
		}
	}
	return true
}

func buildTileTypeCountsForClaim(ctx *YakuContext) (map[TileType]int, int) {
	counts := make(map[TileType]int, 34)
	total := 0
	if ctx == nil || ctx.Winner == nil {
		return counts, 0
	}
	for _, t := range ctx.Winner.Tiles {
		counts[t.Type]++
		total++
	}
	for _, m := range ctx.Winner.Melds {
		for _, t := range m.Tiles {
			counts[t.Type]++
			total++
		}
	}
	if ctx.Claim.HasLoser {
		counts[ctx.Claim.WinTile.Type]++
		total++
	}
	return counts, total
}
