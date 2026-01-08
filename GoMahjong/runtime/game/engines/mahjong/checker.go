package mahjong

// canHu 检查玩家是否可以荣和
func (eg *RiichiMahjong4p) canHu(seatIndex int, tile Tile) bool {
	// fixme: 实现荣和判定逻辑
	// 需要检查玩家是否听牌且能形成和牌
	player := eg.Players[seatIndex]
	if player == nil || !player.IsWaiting {
		return false
	}

	// 暂时返回 false，实际需要实现完整的和牌判定
	return false
}

// canGang 检查玩家是否可以明杠
func (eg *RiichiMahjong4p) canGang(seatIndex int, tile Tile) bool {
	player := eg.Players[seatIndex]
	if player == nil {
		return false
	}
	count := 0
	for _, t := range player.Tiles {
		if t.Type == tile.Type {
			count++
		}
	}
	return count >= 3
}

// canPeng 检查玩家是否可以碰
func (eg *RiichiMahjong4p) canPeng(seatIndex int, tile Tile) bool {
	player := eg.Players[seatIndex]
	if player == nil {
		return false
	}
	count := 0
	for _, t := range player.Tiles {
		if t.Type == tile.Type {
			count++
		}
	}
	return count >= 2
}

// canChi 检查玩家是否可以吃
func (eg *RiichiMahjong4p) canChi(seatIndex int, tile Tile) bool {
	// fixme: 实现吃判定逻辑
	// 需要检查玩家手中是否能形成顺子
	// 吃只能是下家操作
	player := eg.Players[seatIndex]
	if player == nil {
		return false
	}

	// 暂时返回 false，实际需要实现完整的吃牌判定
	return false
}
