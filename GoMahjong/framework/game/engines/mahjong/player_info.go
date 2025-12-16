package mahjong

type PlayerImage struct {
	UserID      string
	SeatIndex   int
	Tiles       []Tile // 手中的牌
	DiscardPile []Tile // 弃牌堆
	Melds       []Meld // 碰、杠、吃的组合
	IsRiichi    bool   // 是否立直
	IsWaiting   bool   // 是否听牌
}
