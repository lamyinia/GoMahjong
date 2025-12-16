package mahjong

// calculateAvailableOperations 计算可用操作
func (eg *RiichiMahjong4p) calculateAvailableOperations(excludeSeat int) map[int]*PlayerReaction {
	reactions := make(map[int]*PlayerReaction)

	// 获取出牌玩家打出的最后一张牌
	droppingPlayer := eg.TurnManager.GetCurrentPlayer()
	droppingPlayerObj := eg.Players[droppingPlayer]
	if droppingPlayerObj == nil || len(droppingPlayerObj.DiscardPile) == 0 {
		return reactions
	}

	droppedTile := droppingPlayerObj.DiscardPile[len(droppingPlayerObj.DiscardPile)-1]

	// 检查每个反应玩家的操作
	for i := 0; i < 4; i++ {
		if i == excludeSeat {
			continue
		}

		playerOps := []*PlayerOperation{}

		// 检查是否可以荣和
		if eg.canHu(i, droppedTile) {
			playerOps = append(playerOps, &PlayerOperation{
				Type:  "HU",
				Tiles: []Tile{droppedTile},
			})
		}

		// 检查是否可以明杠
		if eg.canGang(i, droppedTile) {
			playerOps = append(playerOps, &PlayerOperation{
				Type:  "GANG",
				Tiles: []Tile{droppedTile},
			})
		}

		// 检查是否可以碰
		pengOps := eg.getPengOptions(i, droppedTile)
		playerOps = append(playerOps, pengOps...)

		// 检查是否可以吃（只有下家可以吃）
		if (droppingPlayer+1)%4 == i {
			chiOps := eg.getChiOptions(i, droppedTile)
			playerOps = append(playerOps, chiOps...)
		}

		if len(playerOps) > 0 {
			reactions[i] = &PlayerReaction{
				Operations: playerOps,
				ChosenOp:   nil,
				Responded:  false,
			}
		}
	}

	return reactions
}

// getPengOptions 获取碰牌的所有选择（考虑红5p等特殊情况）
func (eg *RiichiMahjong4p) getPengOptions(seatIndex int, droppedTile Tile) []*PlayerOperation {
	var ops []*PlayerOperation

	if !eg.canPeng(seatIndex, droppedTile) {
		return ops
	}

	// 获取玩家手中与出牌相同的牌
	player := eg.Players[seatIndex]
	matchingTiles := []Tile{}
	for _, tile := range player.Tiles {
		if eg.isSameTile(tile, droppedTile) {
			matchingTiles = append(matchingTiles, tile)
		}
	}

	// 如果有多个相同的牌（如红5p和普通5p），提供选择
	if len(matchingTiles) > 1 {
		// 优先选择不暴露红5p的组合
		for _, tile := range matchingTiles {
			ops = append(ops, &PlayerOperation{
				Type:  "PENG",
				Tiles: []Tile{tile},
			})
		}
	} else if len(matchingTiles) == 1 {
		ops = append(ops, &PlayerOperation{
			Type:  "PENG",
			Tiles: matchingTiles,
		})
	}

	return ops
}

// getChiOptions 获取吃牌的所有选择
func (eg *RiichiMahjong4p) getChiOptions(seatIndex int, droppedTile Tile) []*PlayerOperation {
	var ops []*PlayerOperation

	if !eg.canChi(seatIndex, droppedTile) {
		return ops
	}

	// 获取所有可能的吃牌组合
	player := eg.Players[seatIndex]
	chiCombos := eg.findChiCombinations(player.Tiles, droppedTile)

	for _, combo := range chiCombos {
		ops = append(ops, &PlayerOperation{
			Type:  "CHI",
			Tiles: combo,
		})
	}

	return ops
}

// findChiCombinations 查找所有可能的吃牌组合
func (eg *RiichiMahjong4p) findChiCombinations(hand []Tile, droppedTile Tile) [][]Tile {
	var combos [][]Tile
	// TODO: 实现吃牌组合查找逻辑
	return combos
}

// isSameTile 判断两张牌是否相同（考虑红5p）
func (eg *RiichiMahjong4p) isSameTile(tile1, tile2 Tile) bool {
	// TODO: 实现牌比较逻辑
	return tile1 == tile2
}
