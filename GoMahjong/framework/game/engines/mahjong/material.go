package mahjong

type TileType int

const (
	// 万子 (0-8)
	Man1 TileType = iota
	Man2
	Man3
	Man4
	Man5
	Man5Red // 赤五万
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
	Pin5Red // 赤五筒
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
	So5Red // 赤五索
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

type Tile struct {
	Type TileType
	ID   int // 用于区分相同的牌（0-3）
}

type Wall struct {
	LiveWall          []Tile // 王牌
	DeadWall          []Tile // 岭上牌
	DoraIndicators    []Tile // 宝牌指示牌
	UraDoraIndicators []Tile // 里宝牌指示牌
}
