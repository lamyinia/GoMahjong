package mahjong

import (
	"math/rand"
	"time"
)

type Wind int

const (
	WindEast  Wind = iota // 东风
	WindSouth             // 南风
	WindWest              // 西风
	WindNorth             // 北风
)

type TileType int

const (
	// 万子 (0-8)
	Man1 TileType = iota
	Man2
	Man3
	Man4
	Man5
	Man6
	Man7
	Man8
	Man9

	// 筒子 (9-17)
	Pin1
	Pin2
	Pin3
	Pin4
	Pin5
	Pin6
	Pin7
	Pin8
	Pin9

	// 索子 (18-26)
	So1
	So2
	So3
	So4
	So5
	So6
	So7
	So8
	So9

	// 字牌 (27-33)
	East
	South
	West
	North
	White
	Green
	Red
)

const TileLimit = 136

type Tile struct {
	Type TileType
	ID   int // 用于区分相同的牌（0-3）。对于数牌5，ID=0表示赤宝牌，ID=1-3表示普通牌
}

// Wang 王牌结构（固定14张）
type Wang struct {
	// 固定4张岭上牌
	KanTiles [4]Tile
	kanIndex int // 已摸张数 (0-4)

	// 固定5张宝牌指示牌
	DoraIndicators [5]Tile
	doraIndex      int // 已翻开张数 (0-5)

	// 固定5张里宝牌指示牌
	UraDoraIndicators [5]Tile
	uraDoraIndex      int // 已翻开张数 (0-5)
}

type DeckManager struct {
	wall        []Tile
	wallIndex   int
	wang        Wang
	remain34    [34]int
	rng         *rand.Rand
	useRedFives bool
}

func NewDeckManager(useRedFives bool) *DeckManager {
	return &DeckManager{
		wall:      make([]Tile, 0, TileLimit),
		wallIndex: 0,
		wang: Wang{
			KanTiles:          [4]Tile{},
			kanIndex:          0,
			DoraIndicators:    [5]Tile{},
			doraIndex:         0,
			UraDoraIndicators: [5]Tile{},
			uraDoraIndex:      0,
		},
		remain34:    [34]int{},
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		useRedFives: useRedFives,
	}
}

func (dm *DeckManager) InitRound() {
	deck := NewTileDeck(dm.useRedFives)
	dm.rng.Shuffle(len(deck.tiles), func(i, j int) {
		deck.tiles[i], deck.tiles[j] = deck.tiles[j], deck.tiles[i]
	})

	dm.wall = dm.wall[:0]
	dm.wallIndex = 0

	// 重置王牌索引
	dm.wang.kanIndex = 0
	dm.wang.doraIndex = 0
	dm.wang.uraDoraIndex = 0

	for i := 0; i < 34; i++ {
		dm.remain34[i] = 4
	}

	if len(deck.tiles) <= 14 {
		return
	}

	// 取最后14张作为王牌
	deadStart := len(deck.tiles) - 14
	dm.wall = append(dm.wall, deck.tiles[:deadStart]...)

	// 分配王牌：[0-3]岭上牌, [4-8]宝牌指示牌, [9-13]里宝牌指示牌
	wangTiles := deck.tiles[deadStart:]
	copy(dm.wang.KanTiles[:], wangTiles[0:4])
	copy(dm.wang.DoraIndicators[:], wangTiles[4:9])
	copy(dm.wang.UraDoraIndicators[:], wangTiles[9:14])
}

func (dm *DeckManager) Draw() (Tile, bool) {
	if dm.wallIndex >= len(dm.wall) {
		return Tile{}, false
	}
	t := dm.wall[dm.wallIndex]
	dm.wallIndex++
	dm.remain34[int(t.Type)]--
	return t, true
}

func (dm *DeckManager) Deal() (Tile, bool) {
	return dm.Draw()
}

// DrawKanTile 从岭上牌摸一张牌（开杠时使用）
func (dm *DeckManager) DrawKanTile() (Tile, bool) {
	if dm.wang.kanIndex >= 4 {
		return Tile{}, false // 岭上牌已摸完
	}
	tile := dm.wang.KanTiles[dm.wang.kanIndex]
	dm.wang.kanIndex++
	dm.remain34[int(tile.Type)]--
	return tile, true
}

// RemainingKanTiles 返回剩余岭上牌数量
func (dm *DeckManager) RemainingKanTiles() int {
	return 4 - dm.wang.kanIndex
}

// CanKan 检查是否还有岭上牌可以摸（用于开杠）
func (dm *DeckManager) CanKan() bool {
	return dm.wang.kanIndex < 4
}

// RevealDoraIndicator 翻开一张宝牌指示牌
func (dm *DeckManager) RevealDoraIndicator() (Tile, bool) {
	if dm.wang.doraIndex >= 5 {
		return Tile{}, false
	}
	tile := dm.wang.DoraIndicators[dm.wang.doraIndex]
	dm.wang.doraIndex++
	dm.remain34[int(tile.Type)]--
	return tile, true
}

// RevealUraDoraIndicator 翻开一张里宝牌指示牌（立直和牌时使用）
func (dm *DeckManager) RevealUraDoraIndicator() (Tile, bool) {
	if dm.wang.uraDoraIndex >= 5 {
		return Tile{}, false
	}
	tile := dm.wang.UraDoraIndicators[dm.wang.uraDoraIndex]
	dm.wang.uraDoraIndex++
	dm.remain34[int(tile.Type)]--
	return tile, true
}

// GetDoraIndicators 获取当前已翻开的宝牌指示牌
func (dm *DeckManager) GetDoraIndicators() []Tile {
	return dm.wang.DoraIndicators[:dm.wang.doraIndex]
}

// GetUraDoraIndicators 获取当前已翻开的里宝牌指示牌
func (dm *DeckManager) GetUraDoraIndicators() []Tile {
	return dm.wang.UraDoraIndicators[:dm.wang.uraDoraIndex]
}

func (dm *DeckManager) Visible34(dst *[34]uint8) {
	for i := 0; i < 34; i++ {
		v := 4 - dm.remain34[i]
		if v < 0 {
			v = 0
		}
		if v > 4 {
			v = 4
		}
		dst[i] = uint8(v)
	}
}

// Wang 返回王牌结构（保留用于兼容性，但建议直接使用 DeckManager 的方法）
func (dm *DeckManager) Wang() *Wang {
	return &dm.wang
}

type Situation struct {
	DealerIndex  int  // 庄家座位(0-3)
	Honba        int  // 本场数
	RoundWind    Wind // 场风
	RoundNumber  int  // 局数(1-4)
	RiichiSticks int  // 立直棒数量
}

type Meld struct {
	Type  string // "Peng", "Gang", "Chi"
	Tiles []Tile
	From  int // 从哪个玩家那里获得
}

type TileDeck struct {
	tiles []Tile
	index int // 当前摸牌位置
}

func NewTileDeck(useRedFives bool) *TileDeck {
	deck := &TileDeck{
		tiles: make([]Tile, 0, TileLimit),
		index: 0,
	}
	deck.initializeTiles(useRedFives)
	return deck
}

func (d *TileDeck) initializeTiles(useRedFives bool) {
	d.tiles = d.tiles[:0] // 清空切片
	// 生成数牌（万、筒、索）
	d.generateSuitTiles(Man1, Man9) // 万子
	d.generateSuitTiles(Pin1, Pin9) // 筒子
	d.generateSuitTiles(So1, So9)   // 索子
	// 生成字牌（风牌和箭牌）
	d.generateHonorTiles(East, Red)
}

// generateSuitTiles 生成一种花色的数牌
func (d *TileDeck) generateSuitTiles(start, end TileType) {
	for tileType := start; tileType <= end; tileType++ {
		for i := 0; i < 4; i++ {
			d.tiles = append(d.tiles, Tile{
				Type: tileType,
				ID:   i,
			})
		}
	}
}

func (d *TileDeck) generateHonorTiles(start, end TileType) {
	for tileType := start; tileType <= end; tileType++ {
		// 每种字牌生成4张
		for i := 0; i < 4; i++ {
			d.tiles = append(d.tiles, Tile{
				Type: tileType,
				ID:   i,
			})
		}
	}
}

func (t TileType) IsNumbered() bool {
	return t >= Man1 && t <= So9
}

func (t TileType) IsHonor() bool {
	return t >= East && t <= Red
}

func (t TileType) IsFive() bool {
	return t == Man5 || t == Pin5 || t == So5
}

func (w Wind) String() string {
	switch w {
	case WindEast:
		return "东"
	case WindSouth:
		return "南"
	case WindWest:
		return "西"
	case WindNorth:
		return "北"
	default:
		return "未知"
	}
}

func (w Wind) Next() Wind {
	return (w + 1) % 4
}

// IsRedFive 判断是否为赤宝牌（ID=0且为数牌5）
func (t Tile) IsRedFive() bool {
	return t.ID == 0 && (t.Type == Man5 || t.Type == Pin5 || t.Type == So5)
}

// IsFive 判断是否为5牌（不区分赤普通）
func (t Tile) IsFive() bool {
	return t.Type == Man5 || t.Type == Pin5 || t.Type == So5
}

// GetTileValue 获取牌的数值（用于和牌算法，赤牌和普通牌视为相同）
func (t Tile) GetTileValue() TileType {
	return t.Type
}

const (
	RoundEndDrawExhaustive = "DRAW_EXHAUSTIVE" // 常规荒牌流局
	RoundEndDraw3Ron       = "DRAW_3RON"       // 三家点铳流局
	RoundEndDraw4Kan       = "DRAW_4KAN"       // 四杠散了流局
	RoundEndTsumo          = "TSUMO"           // 自摸
	RoundEndRon            = "RON"             // 荣和
)

// HuClaim 约定 WinTile 的最后一张牌是 点到的/摸到的 牌
type HuClaim struct {
	WinnerSeat int
	HasLoser   bool
	LoserSeat  int
	WinTile    Tile
}

type PlayerOperation struct {
	Type  string // "HU", "GANG", "PENG", "CHI"
	Tiles []Tile // 操作涉及的牌（对于吃碰杠，包含选择的牌）
}

// PlayerReaction 玩家的反应信息
type PlayerReaction struct {
	Operations []*PlayerOperation // 该玩家可用的所有操作选择
	ChosenOp   *PlayerOperation   // 玩家选择的操作（nil表示未响应）
	Responded  bool               // 是否已响应
}

// ReactionAction 选择的反应操作
type ReactionAction struct {
	Type       string // "HU", "GANG", "PENG", "CHI"
	PlayerSeat int
	Tiles      []Tile
}
