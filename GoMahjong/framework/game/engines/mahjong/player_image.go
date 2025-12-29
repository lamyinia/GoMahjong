package mahjong

type PlayerImage struct {
	UserID         string
	SeatIndex      int
	Tiles          []Tile                // 手中的牌
	DiscardPile    []Tile                // 弃牌堆
	Melds          []Meld                // 碰、杠、吃的组合
	IsRiichi       bool                  // 是否立直
	IsWaiting      bool                  // 是否听牌
	DiscardedTiles map[TileType]struct{} // 已弃的牌类型集合（用于振听判断）
	NewestTile     *Tile                 // 最新摸的牌（用于自摸和判断）
	Points         int                   // 当前点数（初始25000或30000）
}

// NewPlayerImage 创建玩家游戏状态实例
func NewPlayerImage(userID string, seatIndex int, initialPoints int) *PlayerImage {
	return &PlayerImage{
		UserID:         userID,
		SeatIndex:      seatIndex,
		Tiles:          make([]Tile, 0, 14),
		DiscardPile:    make([]Tile, 0, 18),
		Melds:          make([]Meld, 0, 4),
		IsRiichi:       false,
		IsWaiting:      false,
		DiscardedTiles: make(map[TileType]struct{}),
		NewestTile:     nil,
		Points:         initialPoints,
	}
}

// AddDiscardedTile 记录已弃的牌（用于振听判断）
func (p *PlayerImage) AddDiscardedTile(tile Tile) {
	p.DiscardedTiles[tile.Type] = struct{}{}
}

// HasDiscardedTile 检查是否弃过某种牌（用于振听判断）
func (p *PlayerImage) HasDiscardedTile(tileType TileType) bool {
	_, exists := p.DiscardedTiles[tileType]
	return exists
}

// SetNewestTile 设置最新摸的牌
func (p *PlayerImage) SetNewestTile(tile *Tile) {
	p.NewestTile = tile
}

// GetNewestTile 获取最新摸的牌
func (p *PlayerImage) GetNewestTile() *Tile {
	return p.NewestTile
}

// AddPoints 增加点数
func (p *PlayerImage) AddPoints(points int) {
	p.Points += points
}

// SubtractPoints 减少点数
func (p *PlayerImage) SubtractPoints(points int) {
	p.Points -= points
}

// GetPoints 获取当前点数
func (p *PlayerImage) GetPoints() int {
	return p.Points
}
