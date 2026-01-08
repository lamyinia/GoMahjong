package mahjong

// callHuPoints 计算和牌点数（统一入口）
// 返回：番数、符数、点数、役列表
func (eg *RiichiMahjong4p) callHuPoints(claim HuClaim, endKind string) (han int, fu int, points int, yakus []Yaku) {
	han, yakumanMult, yakus := eg.evalClaimYakuman(claim, endKind)
	isDealer := claim.WinnerSeat == eg.Situation.DealerIndex
	honba := eg.Situation.Honba

	// 役满：固定点数
	if yakumanMult > 0 {
		base := 8000 * yakumanMult
		if endKind == RoundEndRon {
			if isDealer {
				points = base * 6
			} else {
				points = base * 4
			}
			points += 300 * honba
		} else {
			if isDealer {
				points = base * 2 // 每人支付
			} else {
				points = base // 闲家每人支付
			}
			points += 100 * honba // 自摸本场数每人+100
		}
		return han, 0, points, yakus
	}

	// 满贯以上：固定点数
	if han >= 5 {
		points = eg.getFixedPoints(han, endKind, isDealer)
		// 本场数：荣和+300，自摸+100（每人）
		if endKind == RoundEndRon {
			points += 300 * honba
		} else {
			points += 100 * honba
		}
		return han, 0, points, yakus
	}

	// 普通和牌（<5番）：需要计算符数
	fu = eg.calculateFu(claim, endKind)
	basePoints := eg.calculateBasePoints(han, fu)

	if endKind == RoundEndRon {
		// 荣和
		if isDealer {
			points = basePoints * 6
		} else {
			points = basePoints * 4
		}
	} else {
		// 自摸
		if isDealer {
			points = basePoints * 2 // 每人支付
		} else {
			points = basePoints // 闲家每人支付
		}
	}
	points += 100 * honba

	return han, fu, points, yakus
}

// calculateBasePoints 计算基础点数
func (eg *RiichiMahjong4p) calculateBasePoints(han int, fu int) int {
	// 基础点数 = 符数 × 2^(2+番数)
	base := fu * (1 << (2 + han))

	// 向上取整到100的倍数
	return roundUpTo100(base)
}

// getFixedPoints 获取满贯以上的固定点数
func (eg *RiichiMahjong4p) getFixedPoints(han int, endKind string, isDealer bool) int {
	if endKind == RoundEndRon {
		// 荣和
		switch {
		case han == 5: // 满贯
			if isDealer {
				return 12000
			}
			return 8000
		case han >= 6 && han <= 7: // 跳满
			if isDealer {
				return 18000
			}
			return 12000
		case han >= 8 && han <= 10: // 倍满
			if isDealer {
				return 24000
			}
			return 16000
		case han >= 11 && han <= 12: // 三倍满
			if isDealer {
				return 36000
			}
			return 24000
		default:
			return 0
		}
	} else {
		// 自摸（每人支付）
		switch {
		case han == 5: // 满贯
			if isDealer {
				return 4000 // 每人支付
			}
			return 2000 // 闲家每人支付
		case han >= 6 && han <= 7: // 跳满
			if isDealer {
				return 6000 // 每人支付
			}
			return 3000 // 闲家每人支付
		case han >= 8 && han <= 10: // 倍满
			if isDealer {
				return 8000 // 每人支付
			}
			return 4000 // 闲家每人支付
		case han >= 11 && han <= 12: // 三倍满
			if isDealer {
				return 12000 // 每人支付
			}
			return 6000 // 闲家每人支付
		default:
			return 0
		}
	}
}

// calculateFu 计算符数
func (eg *RiichiMahjong4p) calculateFu(claim HuClaim, endKind string) int {
	winner := eg.Players[claim.WinnerSeat]
	if winner == nil {
		return 0
	}

	fu := 20 // 副底

	// 和牌方式
	if endKind == RoundEndTsumo {
		fu += 2 // 自摸+2符
	}

	// 检查是否有平和（平和固定30符荣和，20符自摸）
	hasPinfu := eg.checkPinfu(claim, winner)
	if hasPinfu {
		if endKind == RoundEndRon {
			return 30 // 平和荣和固定30符
		}
		return 20 // 平和自摸固定20符
	}

	// 雀头符数
	fu += eg.calculatePairFu(claim, winner)

	// 面子符数
	fu += eg.calculateMeldFu(winner)

	// 听牌形式符数（边张/嵌张/单骑）
	fu += eg.calculateWaitFu(claim, winner)

	// 向上取整到10的倍数
	return ((fu + 9) / 10) * 10
}

// checkPinfu 检查是否是平和
func (eg *RiichiMahjong4p) checkPinfu(claim HuClaim, winner *PlayerImage) bool {
	// 平和条件：
	// 1. 门清（无副露）
	// 2. 4个顺子 + 非役牌雀头
	// 3. 两面听牌
	// 4. 荣和时30符，自摸时20符

	if len(winner.Melds) > 0 {
		return false // 有副露，不是平和
	}

	// TODO: 需要根据实际和牌结构判断
	// 这里简化处理，如果门清且没有刻子/杠子，可能是平和
	// 实际应该检查是否真的是4顺子+非役牌雀头+两面听牌
	return false // 暂时返回false，需要实现完整的平和判断
}

// calculatePairFu 计算雀头符数
func (eg *RiichiMahjong4p) calculatePairFu(claim HuClaim, winner *PlayerImage) int {
	// 雀头是自风/场风/三元牌时+2符
	// 需要知道雀头是什么牌，这里简化处理

	// TODO: 需要根据实际和牌结构判断雀头
	// 暂时返回0，需要实现完整的雀头判断
	return 0
}

// calculateMeldFu 计算面子符数
func (eg *RiichiMahjong4p) calculateMeldFu(winner *PlayerImage) int {
	fu := 0

	for _, meld := range winner.Melds {
		isYaochu := eg.isYaochu(meld.Tiles[0].Type)
		isAnkan := meld.Type == "Ankan"
		isKakan := meld.Type == "Kakan"
		isGang := meld.Type == "Gang"
		isPeng := meld.Type == "Peng"

		if isAnkan {
			// 暗杠
			if isYaochu {
				fu += 32 // 幺九暗杠+32符
			} else {
				fu += 16 // 中张暗杠+16符
			}
		} else if isKakan || isGang {
			// 明杠
			if isYaochu {
				fu += 16 // 幺九明杠+16符
			} else {
				fu += 8 // 中张明杠+8符
			}
		} else if isPeng {
			// 明刻
			if isYaochu {
				fu += 4 // 幺九明刻+4符
			} else {
				fu += 2 // 中张明刻+2符
			}
		}
	}

	// 手牌中的暗刻（需要统计手牌中的刻子）
	// TODO: 需要根据实际和牌结构判断手牌中的暗刻
	// 暂时简化处理

	return fu
}

// calculateWaitFu 计算听牌形式符数
func (eg *RiichiMahjong4p) calculateWaitFu(claim HuClaim, winner *PlayerImage) int {
	// 边张/嵌张/单骑+2符
	// 两面/双碰+0符

	// TODO: 需要根据实际听牌形式判断
	// 暂时返回0，需要实现完整的听牌形式判断
	return 0
}

// isYaochu 判断是否是幺九牌（1、9、字牌）
func (eg *RiichiMahjong4p) isYaochu(tileType TileType) bool {
	if tileType >= East && tileType <= Red {
		return true // 字牌
	}
	// 数牌1、9
	if tileType == Man1 || tileType == Man9 ||
		tileType == Pin1 || tileType == Pin9 ||
		tileType == So1 || tileType == So9 {
		return true
	}
	return false
}
