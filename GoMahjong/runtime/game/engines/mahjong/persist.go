package mahjong

import (
	"common/log"
	"context"
	"core/domain/entity"
	"core/domain/repository"
	"runtime/game/share"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GamePersister 游戏持久化组件
// 负责在游戏过程中收集事件，游戏结束后异步写入数据库
type GamePersister struct {
	repo         repository.GameRecordRepository
	gameRecord   *entity.GameRecord
	rounds       []*entity.RoundRecord // 所有回合的数组（游戏结束后一次性保存）
	currentRound *entity.RoundRecord   // 当前回合（方便操作）
	eventMu      sync.Mutex            // 保护事件收集的并发安全
	closed       bool
}

// NewGamePersister 创建持久化组件
func NewGamePersister(repo repository.GameRecordRepository, roomID string, userMap map[string]*share.UserInfo) *GamePersister {
	// 构建玩家信息
	players := make([]entity.PlayerInfo, 0, len(userMap))
	for userID, userInfo := range userMap {
		players = append(players, entity.PlayerInfo{
			UserID:    userID,
			SeatIndex: userInfo.SeatIndex,
		})
	}

	// 创建游戏记录
	gameRecord := entity.NewGameRecord(roomID, "riichi_mahjong_4p", players)

	return &GamePersister{
		repo:       repo,
		gameRecord: gameRecord,
		rounds:     make([]*entity.RoundRecord, 0, 8), // 预分配容量（通常一局游戏不超过8个回合）
		closed:     false,
	}
}

// GetGameRecordID 获取游戏记录ID
func (gp *GamePersister) GetGameRecordID() primitive.ObjectID {
	return gp.gameRecord.ID
}

// StartRound 开始新的一局
func (gp *GamePersister) StartRound(roundNumber int, roundWind string, dealerIndex, honba int) {
	if gp.closed {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	// 创建新的局记录
	gp.currentRound = entity.NewRoundRecord(
		gp.gameRecord.ID,
		roundNumber,
		roundWind,
		dealerIndex,
		honba,
	)

	// 添加到回合数组
	gp.rounds = append(gp.rounds, gp.currentRound)

	// 记录回合开始事件
	gp.currentRound.AddEvent(entity.EventTypeRoundStart, -1, map[string]interface{}{
		"dora_indicators": nil, // 客户端会从推送消息中获取
		"current_turn":    dealerIndex,
	})
}

// RecordDrawTile 记录摸牌事件
func (gp *GamePersister) RecordDrawTile(seatIndex int, tile share.Tile) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	data := map[string]interface{}{
		"tile": map[string]interface{}{
			"type": tile.Type,
			"id":   tile.ID,
		},
	}
	gp.currentRound.AddEvent(entity.EventTypeDrawTile, seatIndex, data)
}

// RecordDiscardTile 记录出牌事件
func (gp *GamePersister) RecordDiscardTile(seatIndex int, tile share.Tile) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	data := map[string]interface{}{
		"tile": map[string]interface{}{
			"type": tile.Type,
			"id":   tile.ID,
		},
	}
	gp.currentRound.AddEvent(entity.EventTypeDiscardTile, seatIndex, data)
}

// RecordChi 记录吃牌事件
func (gp *GamePersister) RecordChi(seatIndex, fromSeat int, tiles []share.Tile) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	tileData := make([]map[string]interface{}, len(tiles))
	for i, t := range tiles {
		tileData[i] = map[string]interface{}{
			"type": t.Type,
			"id":   t.ID,
		}
	}
	data := map[string]interface{}{
		"from_seat": fromSeat,
		"tiles":     tileData,
	}
	gp.currentRound.AddEvent(entity.EventTypeChi, seatIndex, data)
}

// RecordPeng 记录碰牌事件
func (gp *GamePersister) RecordPeng(seatIndex, fromSeat int, tiles []share.Tile) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	tileData := make([]map[string]interface{}, len(tiles))
	for i, t := range tiles {
		tileData[i] = map[string]interface{}{
			"type": t.Type,
			"id":   t.ID,
		}
	}
	data := map[string]interface{}{
		"from_seat": fromSeat,
		"tiles":     tileData,
	}
	gp.currentRound.AddEvent(entity.EventTypePeng, seatIndex, data)
}

// RecordGang 记录明杠事件
func (gp *GamePersister) RecordGang(seatIndex, fromSeat int, tiles []share.Tile) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	tileData := make([]map[string]interface{}, len(tiles))
	for i, t := range tiles {
		tileData[i] = map[string]interface{}{
			"type": t.Type,
			"id":   t.ID,
		}
	}
	data := map[string]interface{}{
		"from_seat": fromSeat,
		"tiles":     tileData,
	}
	gp.currentRound.AddEvent(entity.EventTypeGang, seatIndex, data)
}

// RecordAnkan 记录暗杠事件
func (gp *GamePersister) RecordAnkan(seatIndex int, tiles []share.Tile) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	tileData := make([]map[string]interface{}, len(tiles))
	for i, t := range tiles {
		tileData[i] = map[string]interface{}{
			"type": t.Type,
			"id":   t.ID,
		}
	}
	data := map[string]interface{}{
		"tiles": tileData,
	}
	gp.currentRound.AddEvent(entity.EventTypeAnkan, seatIndex, data)
}

// RecordKakan 记录加杠事件
func (gp *GamePersister) RecordKakan(seatIndex, fromSeat int, tiles []share.Tile) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	tileData := make([]map[string]interface{}, len(tiles))
	for i, t := range tiles {
		tileData[i] = map[string]interface{}{
			"type": t.Type,
			"id":   t.ID,
		}
	}
	data := map[string]interface{}{
		"from_seat": fromSeat,
		"tiles":     tileData,
	}
	gp.currentRound.AddEvent(entity.EventTypeKakan, seatIndex, data)
}

// RecordRiichi 记录立直事件
func (gp *GamePersister) RecordRiichi(seatIndex int) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	gp.currentRound.AddEvent(entity.EventTypeRiichi, seatIndex, map[string]interface{}{})
}

// RecordRon 记录荣和事件
func (gp *GamePersister) RecordRon(winnerSeat, loserSeat int, winTile share.Tile) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	data := map[string]interface{}{
		"winner_seat": winnerSeat,
		"loser_seat":  loserSeat,
		"win_tile": map[string]interface{}{
			"type": winTile.Type,
			"id":   winTile.ID,
		},
	}
	gp.currentRound.AddEvent(entity.EventTypeRon, winnerSeat, data)
}

// RecordTsumo 记录自摸事件
func (gp *GamePersister) RecordTsumo(winnerSeat int, winTile share.Tile) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	data := map[string]interface{}{
		"winner_seat": winnerSeat,
		"win_tile": map[string]interface{}{
			"type": winTile.Type,
			"id":   winTile.ID,
		},
	}
	gp.currentRound.AddEvent(entity.EventTypeTsumo, winnerSeat, data)
}

// CompleteRound 完成当前局（设置回合结果）
func (gp *GamePersister) CompleteRound(endType string, claims []HuClaimDTO, delta [4]int, points [4]int, reason string, nextDealer int) {
	if gp.closed || gp.currentRound == nil {
		return
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	// 转换 HuClaimDTO 为 entity.HuClaim
	huClaims := make([]entity.HuClaim, 0, len(claims))
	for _, c := range claims {
		huClaims = append(huClaims, entity.HuClaim{
			WinnerSeat: c.WinnerSeat,
			LoserSeat:  c.LoserSeat,
			WinTile: entity.Tile{
				Type: int(c.WinTile.Type),
				ID:   c.WinTile.ID,
			},
			Han:    c.Han,
			Fu:     c.Fu,
			Yaku:   c.Yaku,
			Points: c.Points,
		})
	}

	result := &entity.RoundResult{
		EndType:    endType,
		Claims:     huClaims,
		Delta:      delta,
		Points:     points,
		Reason:     reason,
		NextDealer: nextDealer,
	}

	gp.currentRound.CompleteRound(result)

	// 记录回合结束事件
	gp.currentRound.AddEvent(entity.EventTypeRoundEnd, -1, map[string]interface{}{})
}

// FinalizeGame 完成游戏（异步写入数据库）
// 在游戏结束时调用，会保存所有局记录和游戏记录
func (gp *GamePersister) FinalizeGame(finalRankings []PlayerRankingDTO, finalPoints [4]int) {
	if gp.closed {
		return
	}

	gp.eventMu.Lock()
	gp.closed = true
	rounds := make([]*entity.RoundRecord, len(gp.rounds))
	copy(rounds, gp.rounds) // 复制数组，避免在异步中访问时数据被修改
	gp.eventMu.Unlock()

	// 异步写入数据库
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 转换最终排名
		rankings := make([]entity.PlayerRanking, 0, len(finalRankings))
		for _, r := range finalRankings {
			rankings = append(rankings, entity.PlayerRanking{
				SeatIndex: r.SeatIndex,
				UserID:    r.UserID,
				Points:    r.Points,
				Rank:      r.Rank,
			})
		}

		// 设置游戏最终结果
		finalResult := &entity.GameFinalResult{
			Rankings: rankings,
			Points:   finalPoints,
		}
		gp.gameRecord.CompleteGame(finalResult)

		// 保存游戏记录（元数据）
		if err := gp.repo.SaveGameRecord(ctx, gp.gameRecord); err != nil {
			log.Error("保存游戏记录失败: %v", err)
			return
		}

		// 批量保存所有局记录（每个小场一个文档）
		if err := gp.repo.SaveRoundRecords(ctx, rounds); err != nil {
			log.Error("批量保存局记录失败: %v", err)
			return
		}

		log.Info("游戏记录保存成功: gameRecordID=%s, rounds=%d", gp.gameRecord.ID.Hex(), len(rounds))
	}()
}

// SaveCurrentRound 保存当前局记录（用于中途保存，可选）
// 注意：正常情况下不需要调用，游戏结束后会一次性保存所有回合
func (gp *GamePersister) SaveCurrentRound() error {
	if gp.closed || gp.currentRound == nil {
		return nil
	}

	gp.eventMu.Lock()
	defer gp.eventMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return gp.repo.SaveRoundRecord(ctx, gp.currentRound)
}
