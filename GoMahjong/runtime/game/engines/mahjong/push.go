package mahjong

import (
	"common/log"
	"core/infrastructure/message/protocol"
	"core/infrastructure/message/transfer"
	"encoding/json"
	"fmt"
	"runtime/game/share"
)

// 目前有 16 个推送场景，分别是
// 1. 匹配成功
// 2. 回合开始
// 3. 可选操作
// 4. 摸牌
// 5. 吃牌
// 6. 碰牌
// 7. 明杠
// 8. 暗杠
// 9. 立直
// 10. 荣和
// 11. 自摸
// 12. 荒牌流局
// 13. 回合结束
// 14. 游戏结束
// 15. 超时
// 16. 断线重连

// pushMatchSuccessMessage 推送匹配成功消息
func (eg *RiichiMahjong4p) pushMatchSuccessMessage(userMap map[string]*share.UserInfo) {
	// 构建匹配成功消息
	matchSuccessMsg := &transfer.MatchSuccessDTO{
		GameNodeID: eg.Worker.NodeID,
		Players:    make(map[string]string), // userID -> connectorNodeID
	}
	// 收集所有用户ID和connector信息
	userIDs := make([]string, 0, len(userMap))
	for userID, userInfo := range userMap {
		matchSuccessMsg.Players[userID] = userInfo.ConnectorNodeID
		userIDs = append(userIDs, userID)
	}
	msgData, err := json.Marshal(matchSuccessMsg)
	if err != nil {
		log.Error("pushMatchSuccessMessage: 序列化消息失败: %v", err)
		return
	}
	eg.dispatchPush(userIDs, transfer.MatchingSuccess, transfer.MatchingSuccess, msgData)
	log.Info("pushMatchSuccessMessage: 推送匹配成功消息给 %d 个玩家", len(userIDs))
}

// broadcastOperations 下发操作给客户端
func (eg *RiichiMahjong4p) broadcastOperations(reactions map[int]*PlayerReaction) {
	for seatIndex, reaction := range reactions {
		if len(reaction.Operations) == 0 {
			continue
		}
		userID := eg.Players[seatIndex].UserID
		if userID == "" {
			log.Warn("玩家 %d 没有 userID", seatIndex)
			continue
		}
		data, err := json.Marshal(reaction.Operations)
		if err != nil {
			log.Warn("JSON序列化失败: %v", err)
			continue
		}

		eg.dispatchPush([]string{userID}, transfer.GamePush, transfer.DispatchWaitReaction, data)
	}
}

// broadcastRoundStart 推送回合开始（每个玩家收到不同的手牌）
func (eg *RiichiMahjong4p) broadcastRoundStart() {
	if eg.DeckManager == nil {
		log.Warn("broadcastRoundStart: DeckManager 为空")
		return
	}
	// 获取宝牌指示牌（只返回已翻开的）
	doraIndicators := eg.DeckManager.GetDoraIndicators()
	// 构建场况信息
	situationDTO := SituationDTO{
		DealerIndex:  eg.Situation.DealerIndex,
		RoundWind:    eg.Situation.RoundWind.String(),
		RoundNumber:  eg.Situation.RoundNumber,
		Honba:        eg.Situation.Honba,
		RiichiSticks: eg.Situation.RiichiSticks,
	}

	// 为每个玩家推送（手牌内容不同）
	for _, player := range eg.Players {
		if player == nil || player.UserID == "" {
			continue
		}

		// 构建该玩家的回合开始信息（包含自己的手牌）
		roundStart := RoundStartDTO{
			DoraIndicators: doraIndicators,
			Situation:      situationDTO,
			HandTiles:      make([]Tile, len(player.Tiles)),
			CurrentTurn:    eg.TurnManager.GetCurrentPlayer(),
		}
		copy(roundStart.HandTiles, player.Tiles)

		data, err := json.Marshal(roundStart)
		if err != nil {
			log.Error("broadcastRoundStart: 序列化失败: %v", err)
			continue
		}

		eg.dispatchPush([]string{player.UserID}, transfer.GamePush, transfer.GameplayRoundStart, data)
	}

	log.Info("broadcastRoundStart: 推送回合开始给所有玩家")
}

// pushDrawTile 推送摸牌（仅自己可见）
func (eg *RiichiMahjong4p) pushDrawTile(seatIndex int, tile Tile) {
	player := eg.Players[seatIndex]
	if player == nil {
		return
	}

	userID := player.UserID
	if userID == "" {
		log.Warn("pushDrawTile: 玩家 %d 没有 userID", seatIndex)
		return
	}

	// 记录摸牌事件
	if eg.Persister != nil {
		eg.Persister.RecordDrawTile(seatIndex, share.Tile{Type: int(tile.Type), ID: tile.ID})
	}

	drawTile := DrawTileDTO{
		Tile: tile,
	}

	data, err := json.Marshal(drawTile)
	if err != nil {
		log.Error("pushDrawTile: 序列化失败: %v", err)
		return
	}

	eg.dispatchPush([]string{userID}, transfer.GamePush, transfer.GameplayDraw, data)
	log.Info("pushDrawTile: 推送摸牌给玩家 %d, tile: %v", seatIndex, tile)
}

// broadcastDiscard 广播出牌（所有玩家可见）
func (eg *RiichiMahjong4p) broadcastDiscard(seatIndex int, tile Tile) {
	// 记录出牌事件
	if eg.Persister != nil {
		eg.Persister.RecordDiscardTile(seatIndex, share.Tile{Type: int(tile.Type), ID: tile.ID})
	}

	discardTile := DiscardTileDTO{
		SeatIndex: seatIndex,
		Tile:      tile,
	}

	data, err := json.Marshal(discardTile)
	if err != nil {
		log.Error("broadcastDiscard: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	eg.dispatchPush(userIDs, transfer.GamePush, transfer.GameplayDiscard, data)
	log.Info("broadcastDiscard: 广播出牌，玩家 %d 打出 %v", seatIndex, tile)
}

// broadcastRiichi 广播立直（所有玩家可见）
func (eg *RiichiMahjong4p) broadcastRiichi(seatIndex int) {
	// 记录立直事件
	if eg.Persister != nil {
		eg.Persister.RecordRiichi(seatIndex)
	}

	riichi := RiichiDTO{
		SeatIndex: seatIndex,
	}

	data, err := json.Marshal(riichi)
	if err != nil {
		log.Error("broadcastRiichi: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	eg.dispatchPush(userIDs, transfer.GamePush, transfer.GameplayRiichi, data)
	log.Info("broadcastRiichi: 广播立直，玩家 %d 立直", seatIndex)
}

// broadcastMeldAction 广播鸣牌（吃、碰、明杠）
func (eg *RiichiMahjong4p) broadcastMeldAction(actionType string, seatIndex, fromSeat int, tiles []Tile) {
	// 记录鸣牌事件
	if eg.Persister != nil {
		shareTiles := make([]share.Tile, len(tiles))
		for i, t := range tiles {
			shareTiles[i] = share.Tile{Type: int(t.Type), ID: t.ID}
		}
		switch actionType {
		case "CHI":
			eg.Persister.RecordChi(seatIndex, fromSeat, shareTiles)
		case "PENG":
			eg.Persister.RecordPeng(seatIndex, fromSeat, shareTiles)
		case "GANG":
			eg.Persister.RecordGang(seatIndex, fromSeat, shareTiles)
		}
	}

	meldAction := MeldActionDTO{
		ActionType: actionType,
		SeatIndex:  seatIndex,
		FromSeat:   fromSeat,
		Tiles:      tiles,
	}

	data, err := json.Marshal(meldAction)
	if err != nil {
		log.Error("broadcastMeldAction: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	route := transfer.GameplayChi
	switch actionType {
	case "CHI":
		route = transfer.GameplayChi
	case "PENG":
		route = transfer.GameplayPeng
	case "GANG":
		route = transfer.GameplayGang
	}

	eg.dispatchPush(userIDs, transfer.GamePush, route, data)
	log.Info("broadcastMeldAction: 广播鸣牌，玩家 %d %s，来自玩家 %d", seatIndex, actionType, fromSeat)
}

// broadcastAnkan 广播暗杠（所有玩家可见）
func (eg *RiichiMahjong4p) broadcastAnkan(seatIndex int, tiles []Tile) {
	// 记录暗杠事件
	if eg.Persister != nil {
		shareTiles := make([]share.Tile, len(tiles))
		for i, t := range tiles {
			shareTiles[i] = share.Tile{Type: int(t.Type), ID: t.ID}
		}
		eg.Persister.RecordAnkan(seatIndex, shareTiles)
	}

	ankanAction := MeldActionDTO{
		ActionType: "ANKAN",
		SeatIndex:  seatIndex,
		FromSeat:   -1, // -1 表示暗杠
		Tiles:      tiles,
	}

	data, err := json.Marshal(ankanAction)
	if err != nil {
		log.Error("broadcastAnkan: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	eg.dispatchPush(userIDs, transfer.GamePush, transfer.GameplayAnkan, data)
	log.Info("broadcastAnkan: 广播暗杠，玩家 %d 暗杠", seatIndex)
}

// broadcastKakan 广播加杠（所有玩家可见）
func (eg *RiichiMahjong4p) broadcastKakan(seatIndex, fromSeat int, tiles []Tile) {
	// 记录加杠事件
	if eg.Persister != nil {
		shareTiles := make([]share.Tile, len(tiles))
		for i, t := range tiles {
			shareTiles[i] = share.Tile{Type: int(t.Type), ID: t.ID}
		}
		eg.Persister.RecordKakan(seatIndex, fromSeat, shareTiles)
	}

	kakanAction := MeldActionDTO{
		ActionType: "KAKAN",
		SeatIndex:  seatIndex,
		FromSeat:   fromSeat, // 原碰的 From（表示来自哪个玩家）
		Tiles:      tiles,
	}

	data, err := json.Marshal(kakanAction)
	if err != nil {
		log.Error("broadcastKakan: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	eg.dispatchPush(userIDs, transfer.GamePush, transfer.GameplayKakan, data)
	log.Info("broadcastKakan: 广播加杠，玩家 %d 加杠，原碰来自玩家 %d", seatIndex, fromSeat)
}

// broadcastRon 广播荣和
func (eg *RiichiMahjong4p) broadcastRon(winnerSeat, loserSeat int, winTile Tile) {
	// 记录荣和事件
	if eg.Persister != nil {
		eg.Persister.RecordRon(winnerSeat, loserSeat, share.Tile{Type: int(winTile.Type), ID: winTile.ID})
	}

	ron := RonDTO{
		WinnerSeat: winnerSeat,
		LoserSeat:  loserSeat,
		WinTile:    winTile,
	}

	data, err := json.Marshal(ron)
	if err != nil {
		log.Error("broadcastRon: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	eg.dispatchPush(userIDs, transfer.GamePush, transfer.GameplayRon, data)
	log.Info("broadcastRon: 广播荣和，玩家 %d 荣和，放铳玩家 %d", winnerSeat, loserSeat)
}

// broadcastTsumo 广播自摸
func (eg *RiichiMahjong4p) broadcastTsumo(winnerSeat int, winTile Tile) {
	// 记录自摸事件
	if eg.Persister != nil {
		eg.Persister.RecordTsumo(winnerSeat, share.Tile{Type: int(winTile.Type), ID: winTile.ID})
	}

	tsumo := TsumoDTO{
		WinnerSeat: winnerSeat,
		WinTile:    winTile,
	}

	data, err := json.Marshal(tsumo)
	if err != nil {
		log.Error("broadcastTsumo: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	eg.dispatchPush(userIDs, transfer.GamePush, transfer.GameplayTsumo, data)
	log.Info("broadcastTsumo: 广播自摸，玩家 %d 自摸", winnerSeat)
}

// broadcastRoundEnd 广播回合结束
func (eg *RiichiMahjong4p) broadcastRoundEnd(endType string, claims []HuClaimDTO, delta [4]int, reason string, nextDealer int) {
	// 获取当前点数
	points := [4]int{}
	for i := 0; i < 4; i++ {
		if eg.Players[i] != nil {
			points[i] = eg.Players[i].Points
		}
	}

	// 记录回合结束
	if eg.Persister != nil {
		eg.Persister.CompleteRound(endType, claims, delta, points, reason, nextDealer)
	}

	roundEnd := RoundEndDTO{
		EndType:    endType,
		Claims:     claims,
		Delta:      delta,
		Points:     points,
		Reason:     reason,
		NextDealer: nextDealer,
	}

	data, err := json.Marshal(roundEnd)
	if err != nil {
		log.Error("broadcastRoundEnd: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	eg.dispatchPush(userIDs, transfer.GamePush, transfer.GameplayRoundEnd, data)
	log.Info("broadcastRoundEnd: 广播回合结束，类型: %s", endType)
}

// broadcastGameEnd 广播游戏结束
func (eg *RiichiMahjong4p) broadcastGameEnd() {
	// 计算排名
	rankings := [4]*PlayerRankingDTO{}
	playerList := make([]struct {
		seatIndex int
		points    int
		userID    string
	}, 0, 4)

	finalPoints := [4]int{}
	for i := 0; i < 4; i++ {
		if eg.Players[i] != nil {
			finalPoints[i] = eg.Players[i].Points
			playerList = append(playerList, struct {
				seatIndex int
				points    int
				userID    string
			}{
				seatIndex: i,
				points:    eg.Players[i].Points,
				userID:    eg.Players[i].UserID,
			})
		}
	}

	// 按点数排序（降序）
	for i := 0; i < len(playerList)-1; i++ {
		for j := i + 1; j < len(playerList); j++ {
			if playerList[i].points < playerList[j].points {
				playerList[i], playerList[j] = playerList[j], playerList[i]
			}
		}
	}

	// 分配排名
	finalRankings := make([]PlayerRankingDTO, 0, 4)
	for rank, p := range playerList {
		ranking := PlayerRankingDTO{
			SeatIndex: p.seatIndex,
			UserID:    p.userID,
			Points:    p.points,
			Rank:      rank + 1,
		}
		rankings[p.seatIndex] = &ranking
		finalRankings = append(finalRankings, ranking)
	}

	// 异步保存游戏记录
	if eg.Persister != nil {
		eg.Persister.FinalizeGame(finalRankings, finalPoints)
	}

	gameEnd := GameEndDTO{
		FinalRanking: rankings,
	}

	data, err := json.Marshal(gameEnd)
	if err != nil {
		log.Error("broadcastGameEnd: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	eg.dispatchPush(userIDs, transfer.GamePush, transfer.GameplayGameEnd, data)
	log.Info("broadcastGameEnd: 广播游戏结束")
}

// broadcastStateUpdate 广播游戏状态更新
func (eg *RiichiMahjong4p) broadcastStateUpdate() {
	// 获取当前点数
	points := [4]int{}
	for i := 0; i < 4; i++ {
		if eg.Players[i] != nil {
			points[i] = eg.Players[i].Points
		}
	}

	// 构建场况信息
	situationDTO := SituationDTO{
		DealerIndex:  eg.Situation.DealerIndex,
		RoundWind:    eg.Situation.RoundWind.String(),
		RoundNumber:  eg.Situation.RoundNumber,
		Honba:        eg.Situation.Honba,
		RiichiSticks: eg.Situation.RiichiSticks,
	}

	// 获取回合状态字符串
	turnStateStr := "idle"
	switch eg.TurnManager.GetState() {
	case TurnStateWaitMain:
		turnStateStr = "waitMain"
	case TurnStateSelecting:
		turnStateStr = "selecting"
	case TurnStateWaitReactions:
		turnStateStr = "waitReactions"
	case TurnStateApplyOperation:
		turnStateStr = "applyOperation"
	}

	stateUpdate := GameStateUpdateDTO{
		Situation:   situationDTO,
		CurrentTurn: eg.TurnManager.GetCurrentPlayer(),
		TurnState:   turnStateStr,
		Points:      points,
	}

	data, err := json.Marshal(stateUpdate)
	if err != nil {
		log.Error("broadcastStateUpdate: 序列化失败: %v", err)
		return
	}

	// 收集所有玩家ID
	userIDs := make([]string, 0, 4)
	for _, player := range eg.Players {
		if player != nil && player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}

	eg.dispatchPush(userIDs, transfer.GamePush, transfer.GameplayStateUpdate, data)
	log.Info("broadcastStateUpdate: 广播状态更新")
}

// convertHuClaimToDTOWithFanFu 将 HuClaim 转换为 HuClaimDTO（使用已计算的番符和役列表）
func (eg *RiichiMahjong4p) convertHuClaimToDTOWithFanFu(claim HuClaim, endKind string, han int, fu int, points int, yakus []Yaku) HuClaimDTO {
	// 将 Yaku 转换为字符串（简化版，使用数字表示）
	yakuStrs := make([]string, 0, len(yakus))
	for _, yaku := range yakus {
		yakuStrs = append(yakuStrs, fmt.Sprintf("Yaku%d", int(yaku)))
	}

	return HuClaimDTO{
		WinnerSeat: claim.WinnerSeat,
		LoserSeat:  claim.LoserSeat,
		WinTile:    claim.WinTile,
		Han:        han,
		Fu:         fu,
		Yaku:       yakuStrs,
		Points:     points,
	}
}

// dispatchPush 聚合推送消息（按 connector 分组）
func (eg *RiichiMahjong4p) dispatchPush(users []string, connectorRoute, clientRoute string, data []byte) {
	if len(users) == 0 {
		log.Warn("dispatchPush: 用户列表为空")
		return
	}

	connectorGroups := make(map[string][]string) // connectorNodeID -> []userID
	for _, userID := range users {
		if userID == "" {
			continue
		}
		// 从 UserMap 获取 connector 信息（无需加锁，因为 UserMap 在 actor 线程中）
		userInfo, exists := eg.UserMap[userID]
		if !exists {
			log.Warn("dispatchPush: 用户 %s 不在 UserMap 中", userID)
			continue
		}
		connectorNodeID := userInfo.ConnectorNodeID
		if connectorNodeID == "" {
			log.Warn("dispatchPush: 用户 %s 没有 connector 信息", userID)
			continue
		}
		connectorGroups[connectorNodeID] = append(connectorGroups[connectorNodeID], userID)
	}

	for connectorNodeID, userIDs := range connectorGroups {
		packet := &transfer.ServicePacket{
			Source:      eg.Worker.NodeID,
			Destination: connectorNodeID,
			Route:       connectorRoute, // 服务间路由
			PushUser:    userIDs,        // 该 connector 下的所有用户
			Body: &protocol.Message{
				Type:  protocol.Push,
				Route: clientRoute, // 客户端路由
				Data:  data,
			},
		}
		err := eg.Worker.PushMessage(packet)
		if err != nil {
			log.Warn("dispatchPush: 推送给 connector %s 失败: %v, users: %v", connectorNodeID, err, userIDs)
			continue
		}
		log.Info("dispatchPush: 推送给 connector %s, users: %v, route: %s", connectorNodeID, userIDs, clientRoute)
	}
}

// ==================== 推送数据结构 ====================

// RoundStartDTO 回合开始信息
type RoundStartDTO struct {
	DoraIndicators []Tile       `json:"doraIndicators"` // 宝牌指示牌
	Situation      SituationDTO `json:"situation"`      // 场况信息
	HandTiles      []Tile       `json:"handTiles"`      // 自己的手牌（仅自己可见）
	CurrentTurn    int          `json:"currentTurn"`    // 当前出牌玩家座位
}

// SituationDTO 场况信息
type SituationDTO struct {
	DealerIndex  int    `json:"dealerIndex"`  // 庄家座位
	RoundWind    string `json:"roundWind"`    // "East", "South", "West", "North"
	RoundNumber  int    `json:"roundNumber"`  // 局数 (1-4)
	Honba        int    `json:"honba"`        // 本场
	RiichiSticks int    `json:"riichiSticks"` // 供托
}

// DrawTileDTO 摸牌信息
type DrawTileDTO struct {
	Tile Tile `json:"tile"` // 摸到的牌
}

// DiscardTileDTO 出牌信息
type DiscardTileDTO struct {
	SeatIndex int  `json:"seatIndex"` // 出牌玩家座位
	Tile      Tile `json:"tile"`      // 打出的牌
}

// RiichiDTO 立直信息
type RiichiDTO struct {
	SeatIndex int `json:"seatIndex"` // 立直玩家座位
}

// MeldActionDTO 鸣牌信息（吃、碰、明杠）
type MeldActionDTO struct {
	ActionType string `json:"actionType"` // "CHI", "PENG", "GANG"
	SeatIndex  int    `json:"seatIndex"`  // 鸣牌玩家座位
	FromSeat   int    `json:"fromSeat"`   // 来自哪个玩家
	Tiles      []Tile `json:"tiles"`      // 副露的牌
}

// RonDTO 荣和信息
type RonDTO struct {
	WinnerSeat int  `json:"winnerSeat"` // 和牌玩家座位
	LoserSeat  int  `json:"loserSeat"`  // 放铳玩家座位
	WinTile    Tile `json:"winTile"`    // 和牌
}

// TsumoDTO 自摸信息
type TsumoDTO struct {
	WinnerSeat int  `json:"winnerSeat"` // 和牌玩家座位
	WinTile    Tile `json:"winTile"`    // 和牌
}

// RoundEndDTO 回合结束信息
type RoundEndDTO struct {
	EndType    string       `json:"endType"`    // "RON", "TSUMO", "DRAW_EXHAUSTIVE", "DRAW_3RON", "DRAW_OTHER"
	Claims     []HuClaimDTO `json:"claims"`     // 和牌信息（如果有）
	Delta      [4]int       `json:"delta"`      // 点数变化
	Points     [4]int       `json:"points"`     // 当前点数
	Reason     string       `json:"reason"`     // 流局原因（如果有）
	NextDealer int          `json:"nextDealer"` // 下一局庄家（-1表示游戏结束）
}

// HuClaimDTO 和牌信息
type HuClaimDTO struct {
	WinnerSeat int      `json:"winnerSeat"` // 和牌玩家座位
	LoserSeat  int      `json:"loserSeat"`  // 放铳玩家座位（荣和时有值）
	WinTile    Tile     `json:"winTile"`    // 和牌
	Han        int      `json:"han"`        // 番数
	Fu         int      `json:"fu"`         // 符数
	Yaku       []string `json:"yaku"`       // 役列表
	Points     int      `json:"points"`     // 点数
}

// GameEndDTO 游戏结束信息
type GameEndDTO struct {
	FinalRanking [4]*PlayerRankingDTO `json:"finalRanking"` // 最终排名
}

// PlayerRankingDTO 玩家排名
type PlayerRankingDTO struct {
	SeatIndex int    `json:"seatIndex"` // 座位索引
	UserID    string `json:"userId"`    // 用户ID
	Points    int    `json:"points"`    // 最终点数
	Rank      int    `json:"rank"`      // 排名 (1-4)
}

// GameStateUpdateDTO 游戏状态更新
type GameStateUpdateDTO struct {
	Situation   SituationDTO `json:"situation"`   // 场况信息
	CurrentTurn int          `json:"currentTurn"` // 当前出牌玩家座位
	TurnState   string       `json:"turnState"`   // 回合状态
	Points      [4]int       `json:"points"`      // 当前点数
}
