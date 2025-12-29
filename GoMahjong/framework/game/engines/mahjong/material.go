package mahjong

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

type Wang struct {
	DeadWall          []Tile // 岭上牌
	DoraIndicators    []Tile // 宝牌指示牌
	UraDoraIndicators []Tile // 里宝牌指示牌
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
