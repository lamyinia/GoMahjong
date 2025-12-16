package mahjong

import (
	"common/log"
	"fmt"
	"framework/game"
	"framework/game/engines"
	"framework/game/share"
	"sync"
	"time"
)

/*
	大概分了几个状态机
	主要有几个模块构成：
	倒计时调度：
		对玩家来说，游戏状态无非就两个
			第一是出牌，玩家对应的操作有(出牌、暗杠、加杠、立直、自摸)
			第二是响应出牌，玩家对应的操作有(吃、碰、明杠、荣和)
		对于整个游戏来说
	和牌设计：

	立直+向听数：
		预估是时间复杂度最高的部分
	番符计算
*/

// PlayerOperation 玩家可用的单个操作
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

type RiichiMahjong4p struct {
	State       engines.GameState
	Worker      *game.Worker               // Game Worker（在 GameContainer 创建原型时注入）
	UserMap     map[string]*share.UserInfo // Room.UserMap 的引用，包含座位索引（Engine 和 Room 共用）
	Situation   *Situation                 // 游戏局面信息
	Players     [4]*PlayerImage            // 座位索引 -> 玩家游戏状态
	Deck        *TileDeck                  // 牌库管理
	Wang        *Wang                      // 王牌
	TurnManager *TurnManager               // 回合管理

	// 反应阶段管理
	Reactions  map[int]*PlayerReaction // 玩家座位 → 反应信息
	ReactMutex sync.RWMutex            // 并发控制
}

// NewRiichiMahjong4p 创建立直麻将 4 人引擎实例
func NewRiichiMahjong4p(worker *game.Worker) *RiichiMahjong4p {
	return &RiichiMahjong4p{
		State:   engines.GameWaiting,
		Worker:  worker,
		UserMap: nil,
		Situation: &Situation{
			DealerIndex:  0,
			Honba:        0,
			RoundWind:    WindEast,
			RoundNumber:  1,
			RiichiSticks: 0,
		},
		Wang: &Wang{
			DeadWall:          make([]Tile, 0),
			DoraIndicators:    make([]Tile, 0),
			UraDoraIndicators: make([]Tile, 0),
		},
		Players:   [4]*PlayerImage{},
		Reactions: make(map[int]*PlayerReaction),
	}
}

const DefaultMaxRoundTime = 30

// InitializeEngine 初始化游戏引擎
func (eg *RiichiMahjong4p) InitializeEngine(userMap map[string]*share.UserInfo) error {
	eg.UserMap = userMap

	// 初始化 PlayerTicker 数组
	tickers := [4]*PlayerTicker{}
	seatIndex := 0
	for _, userInfo := range userMap {
		userInfo.SeatIndex = seatIndex
		ticker := NewPlayerTicker(DefaultMaxRoundTime)
		ticker.SetOnTimeout(eg.makeTimeoutHandler(seatIndex))
		ticker.SetOnStop(eg.makeStopHandler(seatIndex))
		tickers[seatIndex] = ticker

		eg.Players[seatIndex] = &PlayerImage{
			UserID:      userInfo.UserID,
			SeatIndex:   seatIndex,
			Tiles:       make([]Tile, 0),
			DiscardPile: make([]Tile, 0),
			Melds:       make([]Meld, 0),
			IsRiichi:    false,
			IsWaiting:   false,
		}
		seatIndex++
	}
	eg.TurnManager = NewTurnManager(tickers)
	eg.State = engines.GameWaiting
	time.AfterFunc(10*time.Second, eg.roundGameStart)

	return nil
}

// DriveEngine 驱动游戏逻辑
func (eg *RiichiMahjong4p) DriveEngine(event share.GameEvent) {
	if event == nil {
		log.Warn("事件为空")
		return
	}

	eventType := event.GetEventType()
	log.Info("处理游戏事件: %s", eventType)

	switch eventType {
	case "DropTile":
		if dropEvent, ok := event.(*share.DropTileEvent); ok {
			eg.handleDropTileEvent(dropEvent)
		}
	case "Peng":
		if pengEvent, ok := event.(*share.PengTileEvent); ok {
			eg.handlePengEvent(pengEvent)
		}
	case "Gang":
		if gangEvent, ok := event.(*share.GangEvent); ok {
			eg.handleGangEvent(gangEvent)
		}
	case "Chi":
		if chiEvent, ok := event.(*share.ChiEvent); ok {
			eg.handleChiEvent(chiEvent)
		}
	case "Hu":
		if huEvent, ok := event.(*share.HuEvent); ok {
			eg.handleHuEvent(huEvent)
		}
	case "Riichi":
		if riichiEvent, ok := event.(*share.RiichiEvent); ok {
			eg.handleRiichiEvent(riichiEvent)
		}
	default:
		log.Warn("不支持的事件类型: %s", eventType)
	}
}

// handleDropTileEvent 处理出牌事件
func (eg *RiichiMahjong4p) handleDropTileEvent(event *share.DropTileEvent) {
	log.Info("处理出牌事件")
	if eg.TurnManager.GetState() != StateDropping {
		log.Warn("当前不在出牌阶段，状态: %v", eg.TurnManager.GetState())
		return
	}
	seatIndex, err := eg.getSeatIndex(event.GetUserID())
	if err != nil {
		log.Warn("获取玩家座位失败: %v", err)
		return
	}
	if seatIndex != eg.TurnManager.GetCurrentPlayer() {
		log.Warn("不是当前玩家的回合，当前玩家: %d, 事件玩家: %d", eg.TurnManager.GetCurrentPlayer(), seatIndex)
		return
	}
	ticker := eg.TurnManager.GetPlayerTicker(seatIndex)

	ticker.Stop()

	// 处理出牌逻辑
	player := eg.Players[seatIndex]
	if player == nil {
		log.Warn("玩家 %d 不存在", seatIndex)
		return
	}

	// TODO 从手牌中移除出牌
	tile := event.GetTile()
	removed := false
	if !removed {
		log.Warn("玩家 %d 手中没有该牌: %v", seatIndex, tile)
		return
	}

	// 添加到弃牌堆
	//player.DiscardPile = append(player.DiscardPile, tile)

	log.Info("玩家 %d 出牌: %v", seatIndex, tile)

	eg.waitReaction(seatIndex)
}

func (eg *RiichiMahjong4p) waitReaction(excludeSeat int) {
	eg.ReactMutex.Lock()
	defer eg.ReactMutex.Unlock()

	// 计算可用操作
	reactions := eg.calculateAvailableOperations(excludeSeat)
	eg.Reactions = reactions

	if len(eg.Reactions) == 0 {
		// 没有玩家可以反应，直接进入下一个出牌阶段
		nextPlayer := eg.TurnManager.NextTurn()
		if err := eg.TurnManager.EnterDropPhase(nextPlayer, 5); err != nil {
			log.Error("进入出牌阶段失败: %v", err)
		}
		return
	}

	// 下发操作给客户端
	eg.broadcastOperations(eg.Reactions)

	// 启动反应阶段计时
	eg.TurnManager.State = StateReacting

	for seatIndex := range eg.Reactions {
		ticker := eg.TurnManager.GetPlayerTicker(seatIndex)
		allocatedTime := ticker.Available + 3
		ticker.SetAvailable(allocatedTime)
		if err := ticker.Start(allocatedTime); err != nil {
			log.Error("启动反应计时失败 (座位 %d): %v", seatIndex, err)
		}
	}
}

// broadcastOperations 下发操作给客户端
func (eg *RiichiMahjong4p) broadcastOperations(reactions map[int]*PlayerReaction) {
	for seatIndex, reaction := range reactions {
		log.Info("玩家 %d 可用操作数: %d", seatIndex, len(reaction.Operations))
		for _, op := range reaction.Operations {
			log.Info("  - %s: %v", op.Type, op.Tiles)
		}
		// TODO: 通过 Worker 下发消息给客户端
	}
}

// recordPlayerResponse 记录玩家响应
func (eg *RiichiMahjong4p) recordPlayerResponse(seatIndex int, chosenOp *PlayerOperation) {
	eg.ReactMutex.Lock()
	defer eg.ReactMutex.Unlock()

	reaction, exists := eg.Reactions[seatIndex]
	if !exists {
		log.Warn("玩家 %d 不在反应列表中", seatIndex)
		return
	}

	reaction.ChosenOp = chosenOp
	reaction.Responded = true

	// 停止该玩家的计时
	ticker := eg.TurnManager.GetPlayerTicker(seatIndex)
	if ticker.GetState() == StateRunning {
		ticker.Stop()
	}

	log.Info("玩家 %d 响应: %s", seatIndex, chosenOp.Type)

	// 检查是否所有响应已收集
	if eg.isReactionComplete() {
		eg.handleReactionComplete()
	}
}

// isReactionComplete 检查是否所有响应已收集
func (eg *RiichiMahjong4p) isReactionComplete() bool {
	for _, reaction := range eg.Reactions {
		if !reaction.Responded {
			return false
		}
	}
	return true
}

// handlePengEvent 处理碰牌事件
func (eg *RiichiMahjong4p) handlePengEvent(event *share.PengTileEvent) {
	log.Info("处理碰牌事件")

	if eg.TurnManager.GetState() != StateReacting {
		log.Warn("当前不在反应阶段")
		return
	}

	seatIndex, err := eg.getSeatIndex(event.GetUserID())
	if err != nil {
		log.Warn("获取玩家座位失败: %v", err)
		return
	}

	eg.ReactMutex.RLock()
	reaction, exists := eg.Reactions[seatIndex]
	eg.ReactMutex.RUnlock()

	if !exists {
		log.Warn("玩家 %d 不在反应列表中", seatIndex)
		return
	}

	// 查找碰牌操作
	var pengOp *PlayerOperation
	for _, op := range reaction.Operations {
		if op.Type == "PENG" {
			pengOp = op
			break
		}
	}

	if pengOp == nil {
		log.Warn("玩家 %d 没有碰牌操作", seatIndex)
		return
	}

	eg.recordPlayerResponse(seatIndex, pengOp)
}

// handleGangEvent 处理杠牌事件
func (eg *RiichiMahjong4p) handleGangEvent(event *share.GangEvent) {
	log.Info("处理杠牌事件")

	if eg.TurnManager.GetState() != StateReacting {
		log.Warn("当前不在反应阶段")
		return
	}

	seatIndex, err := eg.getSeatIndex(event.GetUserID())
	if err != nil {
		log.Warn("获取玩家座位失败: %v", err)
		return
	}

	eg.ReactMutex.RLock()
	reaction, exists := eg.Reactions[seatIndex]
	eg.ReactMutex.RUnlock()

	if !exists {
		log.Warn("玩家 %d 不在反应列表中", seatIndex)
		return
	}

	// 查找杠牌操作
	var gangOp *PlayerOperation
	for _, op := range reaction.Operations {
		if op.Type == "GANG" {
			gangOp = op
			break
		}
	}

	if gangOp == nil {
		log.Warn("玩家 %d 没有杠牌操作", seatIndex)
		return
	}

	eg.recordPlayerResponse(seatIndex, gangOp)
}

// handleChiEvent 处理吃牌事件
func (eg *RiichiMahjong4p) handleChiEvent(event *share.ChiEvent) {
	log.Info("处理吃牌事件")

	if eg.TurnManager.GetState() != StateReacting {
		log.Warn("当前不在反应阶段")
		return
	}

	seatIndex, err := eg.getSeatIndex(event.GetUserID())
	if err != nil {
		log.Warn("获取玩家座位失败: %v", err)
		return
	}

	eg.ReactMutex.RLock()
	reaction, exists := eg.Reactions[seatIndex]
	eg.ReactMutex.RUnlock()

	if !exists {
		log.Warn("玩家 %d 不在反应列表中", seatIndex)
		return
	}

	// 查找吃牌操作
	var chiOp *PlayerOperation
	for _, op := range reaction.Operations {
		if op.Type == "CHI" {
			chiOp = op
			break
		}
	}

	if chiOp == nil {
		log.Warn("玩家 %d 没有吃牌操作", seatIndex)
		return
	}

	eg.recordPlayerResponse(seatIndex, chiOp)
}

// handleHuEvent 处理和牌事件
func (eg *RiichiMahjong4p) handleHuEvent(event *share.HuEvent) {
	log.Info("处理和牌事件")

	if eg.TurnManager.GetState() != StateReacting {
		log.Warn("当前不在反应阶段")
		return
	}

	seatIndex, err := eg.getSeatIndex(event.GetUserID())
	if err != nil {
		log.Warn("获取玩家座位失败: %v", err)
		return
	}

	eg.ReactMutex.RLock()
	reaction, exists := eg.Reactions[seatIndex]
	eg.ReactMutex.RUnlock()

	if !exists {
		log.Warn("玩家 %d 不在反应列表中", seatIndex)
		return
	}

	// 查找和牌操作
	var huOp *PlayerOperation
	for _, op := range reaction.Operations {
		if op.Type == "HU" {
			huOp = op
			break
		}
	}

	if huOp == nil {
		log.Warn("玩家 %d 没有和牌操作", seatIndex)
		return
	}

	eg.recordPlayerResponse(seatIndex, huOp)
}

// handleRiichiEvent 处理立直事件
func (eg *RiichiMahjong4p) handleRiichiEvent(event *share.RiichiEvent) {
	log.Info("处理立直事件")

	seatIndex, err := eg.getSeatIndex(event.GetUserID())
	if err != nil {
		log.Warn("获取玩家座位失败: %v", err)
		return
	}
	player := eg.Players[seatIndex]
	if player == nil {
		log.Error("玩家 %d 不存在", seatIndex)
		return
	}

	// 标记玩家为听牌状态
	player.IsWaiting = true
	log.Info("玩家 %d 立直", seatIndex)
}

// makeTimeoutHandler 创建超时处理回调
func (eg *RiichiMahjong4p) makeTimeoutHandler(seatIndex int) func() {
	return func() {
		log.Info("玩家 %d 超时", seatIndex)

		state := eg.TurnManager.GetState()
		switch state {
		case StateDropping:
			eg.handleDropTimeout(seatIndex)
		case StateReacting:
			eg.handleReactionTimeout(seatIndex)
		}
	}
}

// makeStopHandler 创建停止处理回调
func (eg *RiichiMahjong4p) makeStopHandler(seatIndex int) func() {
	return func() {
		log.Info("玩家 %d 停止计时", seatIndex)

		state := eg.TurnManager.GetState()
		switch state {
		case StateDropping:
			// 玩家主动出牌，已经处理
		case StateReacting:
			// 玩家反应，需要收集所有反应，才开始处理
			eg.handlePlayerReacted(seatIndex)
		}
	}
}

// handleDropTimeout 处理出牌超时
func (eg *RiichiMahjong4p) handleDropTimeout(seatIndex int) {
	log.Info("玩家 %d 出牌超时，自动打出摸到的手牌", seatIndex)

	player := eg.Players[seatIndex]
	if player == nil || len(player.Tiles) == 0 {
		log.Error("玩家 %d 手牌为空，无法出牌", seatIndex)
		return
	}

	// 自动打出最后一张摸到的手牌（通常是最后一张）
	tileToDiscard := player.Tiles[len(player.Tiles)-1]
	player.Tiles = player.Tiles[:len(player.Tiles)-1]
	player.DiscardPile = append(player.DiscardPile, tileToDiscard)

	log.Info("玩家 %d 自动打出牌: %v", seatIndex, tileToDiscard)

	// 进入反应阶段
	eg.waitReaction(seatIndex)
}

// handleReactionTimeout 处理反应超时
func (eg *RiichiMahjong4p) handleReactionTimeout(seatIndex int) {
	log.Info("玩家 %d 反应超时，自动跳过", seatIndex)

	// 超时时记录为跳过（选择第一个可用操作或跳过）
	skipOp := &PlayerOperation{
		Type:  "SKIP",
		Tiles: []Tile{},
	}
	eg.recordPlayerResponse(seatIndex, skipOp)
}

// handlePlayerReacted 处理玩家反应
func (eg *RiichiMahjong4p) handlePlayerReacted(seatIndex int) {
	log.Info("玩家 %d 反应", seatIndex)
	// 注意：此方法在反应阶段不应被调用，因为反应已通过 DriveEngine 处理
}

// handleReactionComplete 处理玩家
func (eg *RiichiMahjong4p) handleReactionComplete() {
	log.Info("所有玩家反应完成")

	// 进入选择阶段
	if err := eg.TurnManager.EnterSelectingPhase(); err != nil {
		log.Error("进入选择阶段失败: %v", err)
		return
	}

	// 执行吃碰杠选择算法
	// 优先级：荣和 > 明杠 > 碰 > 吃
	selectedAction := eg.selectBestReaction()

	if selectedAction == nil {
		// 没有有效的反应，进入下一个出牌阶段
		nextPlayer := eg.TurnManager.NextTurn()
		if err := eg.TurnManager.EnterDropPhase(nextPlayer, 5); err != nil {
			log.Error("进入出牌阶段失败: %v", err)
		}
		return
	}

	// 执行选中的操作
	eg.executeReaction(selectedAction)
}

// ReactionAction 反应操作
type ReactionAction struct {
	Type       string // "HU", "GANG", "PENG", "CHI"
	PlayerSeat int
	Tiles      []Tile
}

// selectBestReaction 选择最优的反应操作
// 优先级：荣和 > 明杠 > 碰 > 吃
func (eg *RiichiMahjong4p) selectBestReaction() *ReactionAction {
	eg.ReactMutex.RLock()
	defer eg.ReactMutex.RUnlock()

	// 从收集的响应中选择最优操作（按优先级）
	// 优先级 1：荣和（最高）
	for seatIndex, reaction := range eg.Reactions {
		if reaction.ChosenOp != nil && reaction.ChosenOp.Type == "HU" {
			log.Info("玩家 %d 荣和", seatIndex)
			return &ReactionAction{
				Type:       "HU",
				PlayerSeat: seatIndex,
				Tiles:      reaction.ChosenOp.Tiles,
			}
		}
	}

	// 优先级 2：明杠
	for seatIndex, reaction := range eg.Reactions {
		if reaction.ChosenOp != nil && reaction.ChosenOp.Type == "GANG" {
			log.Info("玩家 %d 明杠", seatIndex)
			return &ReactionAction{
				Type:       "GANG",
				PlayerSeat: seatIndex,
				Tiles:      reaction.ChosenOp.Tiles,
			}
		}
	}

	// 优先级 3：碰
	for seatIndex, reaction := range eg.Reactions {
		if reaction.ChosenOp != nil && reaction.ChosenOp.Type == "PENG" {
			log.Info("玩家 %d 碰", seatIndex)
			return &ReactionAction{
				Type:       "PENG",
				PlayerSeat: seatIndex,
				Tiles:      reaction.ChosenOp.Tiles,
			}
		}
	}

	// 优先级 4：吃
	for seatIndex, reaction := range eg.Reactions {
		if reaction.ChosenOp != nil && reaction.ChosenOp.Type == "CHI" {
			log.Info("玩家 %d 吃", seatIndex)
			return &ReactionAction{
				Type:       "CHI",
				PlayerSeat: seatIndex,
				Tiles:      reaction.ChosenOp.Tiles,
			}
		}
	}

	return nil
}

// executeReaction 执行反应操作
func (eg *RiichiMahjong4p) executeReaction(action *ReactionAction) {
	log.Info("执行反应操作: 类型=%s, 玩家=%d", action.Type, action.PlayerSeat)

}

func (eg *RiichiMahjong4p) roundGameStart() {
	eg.State = engines.GameInProgress
	if err := eg.TurnManager.EnterDropPhase(eg.Situation.DealerIndex, 5); err != nil {
		log.Warn("游戏失败: %v", err)
		return
	}
	log.Info("游戏开始，庄家出牌")
}

// CalculateScore 计算分数
func (eg *RiichiMahjong4p) CalculateScore() map[string]int {
	// TODO: 实现分数计算逻辑
	return nil
}

// getSeatIndex 从 UserMap 中查找玩家座位
func (eg *RiichiMahjong4p) getSeatIndex(userID string) (int, error) {
	if eg.UserMap == nil {
		return -1, fmt.Errorf("UserMap 未初始化")
	}

	userInfo, exists := eg.UserMap[userID]
	if !exists {
		return -1, fmt.Errorf("玩家 %s 不在房间中", userID)
	}

	return userInfo.SeatIndex, nil
}

// Clone 克隆引擎实例（用于原型模式）
func (eg *RiichiMahjong4p) Clone() engines.Engine {
	// 深拷贝 Situation
	clonedSituation := &Situation{
		DealerIndex:  eg.Situation.DealerIndex,
		Honba:        eg.Situation.Honba,
		RoundWind:    eg.Situation.RoundWind,
		RoundNumber:  eg.Situation.RoundNumber,
		RiichiSticks: eg.Situation.RiichiSticks,
	}

	// 深拷贝 Wang
	clonedWall := &Wang{
		DeadWall:          make([]Tile, len(eg.Wang.DeadWall)),
		DoraIndicators:    make([]Tile, len(eg.Wang.DoraIndicators)),
		UraDoraIndicators: make([]Tile, len(eg.Wang.UraDoraIndicators)),
	}
	copy(clonedWall.DeadWall, eg.Wang.DeadWall)
	copy(clonedWall.DoraIndicators, eg.Wang.DoraIndicators)
	copy(clonedWall.UraDoraIndicators, eg.Wang.UraDoraIndicators)

	// 防御性编程，深拷贝 Players 数组，由于克隆时 Players 本来就是空, 这里暂时没有意义
	clonedPlayers := [4]*PlayerImage{}
	for i, player := range eg.Players {
		if player != nil {
			clonedPlayers[i] = &PlayerImage{
				UserID:      player.UserID,
				SeatIndex:   player.SeatIndex,
				Tiles:       make([]Tile, len(player.Tiles)),
				DiscardPile: make([]Tile, len(player.DiscardPile)),
				Melds:       make([]Meld, len(player.Melds)),
				IsRiichi:    player.IsRiichi,
				IsWaiting:   player.IsWaiting,
			}
			copy(clonedPlayers[i].Tiles, player.Tiles)
			copy(clonedPlayers[i].DiscardPile, player.DiscardPile)
			copy(clonedPlayers[i].Melds, player.Melds)
		}
	}

	// 克隆 TurnManager（注意：克隆时 TurnManager 为 nil，因为克隆是为了创建新的原型）
	// 实际的 TurnManager 会在 InitializeEngine 中创建
	var clonedTurnManager *TurnManager
	if eg.TurnManager != nil {
		// 如果原引擎有 TurnManager，创建新的空 TurnManager
		clonedTurnManager = NewTurnManager([4]*PlayerTicker{})
	}

	return &RiichiMahjong4p{
		State:       engines.GameWaiting,
		Worker:      eg.Worker,
		UserMap:     nil,
		Situation:   clonedSituation,
		Wang:        clonedWall,
		Players:     clonedPlayers,
		TurnManager: clonedTurnManager,
	}
}
