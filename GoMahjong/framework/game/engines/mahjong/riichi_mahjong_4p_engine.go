package mahjong

import (
	"common/log"
	"fmt"
	"framework/game"
	"framework/game/engines"
	"framework/game/share"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultMaxRoundTime      = 30   // 每回合的最多分配时间
	UseRedFive               = true // 是否使用赤牌
	DefaultRoundCompensation = 5    // 默认回合补偿
)

func toMahjongTile(t share.Tile) Tile {
	return Tile{Type: TileType(t.Type), ID: t.ID}
}

/*
	注意：
		1.有自摸，一定不能立直
		2.最后一巡，不能立直
		3.二杯口和4刻字同时出现，只算二杯口；清一色，吃不了二杯口
		4.立直后，不可以吃、碰、明杠，但可以暗杠(有限制，听牌必须没有改变)
		5.立直后如果进行了暗杠，则“一发”的役会立即失效
		6.河底牌打出后，任然可以吃碰杠，用于改变听牌的形状，吃罚符

	大概分了几个状态机
		玩家操作前，需要有牌库(牌墙、王牌、手牌)初始化和发牌的逻辑，这是 1 个状态
		玩家开始操作，游戏状态无非就两个:
			第一是出牌，玩家对应的操作有(出牌、暗杠、加杠、立直、自摸)
			第二是响应出牌，玩家对应的操作有(吃、碰、明杠、荣和)
		所以这里有 8 个状态，玩家 1 出牌、反应玩家 1 出牌、玩家 2 出牌 ...
		(这里可以继续逻辑细化为，收集玩家 1 的可选操作，收集非玩家 1 的可选操作... 这里不进行细化)
		游戏结束计算，可能是和牌，也可能荒牌流局了，需要进行点数增减，然后继续判断需不需要进行下一轮游戏，这是 1 个状态
		泛化为 10 个状态
	主要有几个模块构成：
	倒计时调度：

		对于整个游戏来说
	和牌设计：
		枚举雀头，递归回溯面子，时间性能在 us，不是瓶颈
	听牌：
		预估是时间复杂度最高的部分，大概思路就是枚举打出的牌，然后在这个基础上找哪些牌可以可以胡
		就是枚举打出的牌，然后跑和牌算法，给 ai 评测了一下，跑一次，平均情况 1ms 以内是可以跑完的，最坏 3 ms，自己测了一点数据大概只需要 0.2 ms
	番符计算

	todo 通知、广播：
		具体包括：通知每个人的手牌、广播 dora 牌，广播玩家打牌、吃牌、碰牌、杠牌、立直、荣胡、自摸，广播回合结果，广播游戏结果
	自己手牌（完整）, 其他三家手牌（不可见，只给副露/牌河/立直状态）
	场况：庄家、场风、局数、本场、供托,宝牌指示牌（公开）
	当前轮到谁、当前状态（dropping/selecting/reacting）
	若在 reacting：该玩家可选操作列表（仅对该玩家）

	todo 算法收集和响应，包括出牌者和非出牌者的收集和响应

	todo 立直，立直后只能暗杠，进入类似一种托管的状态，需要加额外逻辑处理
	立直资格，具体来说，包括是否门清（len(Melds)==0 且无副露），当前不是最后巡/海底，点数 >= 1000，是否已经立直，立直宣言后扣棒、供托处理
*/

// RiichiMahjong4p 日麻四人游戏引擎
type RiichiMahjong4p struct {
	State           engines.GameState
	Worker          *game.Worker               // Game Worker（在 GameContainer 创建原型时注入）
	RoomID          string                     // 房间 ID（用于请求销毁房间）
	UserMap         map[string]*share.UserInfo // Room.UserMap 的引用，包含座位索引（Engine 和 Room 共用）
	Situation       *Situation                 // 游戏局面信息
	Players         [4]*PlayerImage            // 座位索引 -> 玩家游戏状态
	DeckManager     *DeckManager               // 牌库管理（含王牌、宝牌指示牌、remain34）
	TurnManager     *TurnManager               // 回合管理
	roundStartTimer *time.Timer                // 开局延迟计时器（用于 Close 时停止）
	lastDiscard     LastDiscard

	gameEvents chan share.GameEvent
	gameDone   chan struct{}
	actorExit  chan struct{}
	closed     atomic.Bool // 接收游戏事件的关闭开关

	// 反应阶段管理
	Reactions map[int]*PlayerReaction // 玩家座位 → 反应信息
	closeOnce sync.Once
}

type LastDiscard struct {
	Seat  int
	Tile  Tile
	Valid bool
}

func (eg *RiichiMahjong4p) setLastDiscard(seat int, tile Tile) {
	eg.lastDiscard = LastDiscard{Seat: seat, Tile: tile, Valid: true}
}

func (eg *RiichiMahjong4p) clearLastDiscard() {
	eg.lastDiscard.Valid = false
}

type TimeoutEvent struct {
	share.GameMessageEvent
	SeatIndex int
}

func (e *TimeoutEvent) GetEventType() string {
	return "Timeout"
}

type StartRoundEvent struct {
	share.GameMessageEvent
}

func (e *StartRoundEvent) GetEventType() string {
	return "StartRound"
}

const (
	RoundEndDrawExhaustive = "DRAW_EXHAUSTIVE"
	RoundEndDraw3Ron       = "DRAW_3RON"
	RoundEndTsumo          = "TSUMO"
	RoundEndRon            = "RON"
)

// HuClaim 约定 WinTile 的最后一张牌是 点到的/摸到的 牌
type HuClaim struct {
	WinnerSeat int
	HasLoser   bool
	LoserSeat  int
	WinTile    Tile
}

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

// ReactionAction 选择的反应操作
type ReactionAction struct {
	Type       string // "HU", "GANG", "PENG", "CHI"
	PlayerSeat int
	Tiles      []Tile
}

// NewRiichiMahjong4p 创建立直麻将 4 人引擎实例
func NewRiichiMahjong4p(worker *game.Worker) *RiichiMahjong4p {
	return &RiichiMahjong4p{
		State:   engines.GameWaiting,
		Worker:  worker,
		RoomID:  "",
		UserMap: nil,
		Situation: &Situation{
			DealerIndex:  0,
			Honba:        0,
			RoundWind:    WindEast,
			RoundNumber:  1,
			RiichiSticks: 0,
		},
		Players:   [4]*PlayerImage{},
		Reactions: make(map[int]*PlayerReaction),
	}
}

// InitializeEngine 初始化游戏引擎
func (eg *RiichiMahjong4p) InitializeEngine(roomID string, userMap map[string]*share.UserInfo) error {
	eg.RoomID = roomID
	eg.UserMap = userMap

	eg.closed.Store(false)
	eg.gameEvents = make(chan share.GameEvent, 256)
	eg.gameDone = make(chan struct{})
	eg.actorExit = make(chan struct{})
	if eg.DeckManager == nil {
		eg.DeckManager = NewDeckManager(UseRedFive)
	}
	// 初始化 PlayerTicker 数组
	tickers := [4]*PlayerTicker{}
	seatIndex := 0
	for _, userInfo := range userMap {
		userInfo.SeatIndex = seatIndex
		ticker := NewPlayerTicker(DefaultMaxRoundTime)
		ticker.SetOnTimeout(eg.makeTimeoutHandler(seatIndex))
		ticker.SetOnStop(eg.makeStopHandler(seatIndex))
		tickers[seatIndex] = ticker

		eg.Players[seatIndex] = NewPlayerImage(userInfo.UserID, seatIndex, 25000)
		seatIndex++
	}
	eg.TurnManager = NewTurnManager(tickers)
	eg.State = engines.GameWaiting
	eg.roundStartTimer = time.AfterFunc(8*time.Second, func() {
		eg.State = engines.GameInProgress
		eg.NotifyEvent(&StartRoundEvent{})
	})
	go eg.actorLoop()

	return nil
}

func (eg *RiichiMahjong4p) NotifyEvent(event share.GameEvent) {
	if event == nil {
		return
	}
	if eg.closed.Load() {
		return
	}
	ch := eg.gameEvents
	done := eg.gameDone
	select {
	case <-done:
		return
	case ch <- event:
		return
	default:
		log.Warn("gameEvents 队列已满, eventType=%s", event.GetEventType())
		return
	}
}

func (eg *RiichiMahjong4p) actorLoop() {
	defer func() {
		if eg.actorExit != nil {
			close(eg.actorExit)
		}
	}()
	for {
		select {
		case <-eg.gameDone:
			return
		case event := <-eg.gameEvents:
			eg.processEvent(event)
		}
	}
}

// todo 细化荣胡和自摸事件
func (eg *RiichiMahjong4p) processEvent(event share.GameEvent) {
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
	case "RongHu":
		if rongHuEvent, ok := event.(*share.RongHuEvent); ok {
			eg.handleRongHuEvent(rongHuEvent)
		}
	case "TouchHu":
		if touchHuEvent, ok := event.(*share.TouchHuEvent); ok {
			eg.handleTouchHuEvent(touchHuEvent)
		}
	case "Riichi":
		if riichiEvent, ok := event.(*share.RiichiEvent); ok {
			eg.handleRiichiEvent(riichiEvent)
		}
	case "Reconnect":
		if reconnectEvent, ok := event.(*share.ReconnectEvent); ok {
			eg.handleReconnectEvent(reconnectEvent)
		}
	case "Timeout":
		if t, ok := event.(*TimeoutEvent); ok {
			eg.handleTimeoutEvent(t)
		}
	case "StartRound":
		if _, ok := event.(*StartRoundEvent); ok {
			eg.handleStartRoundEvent()
		}
	default:
		log.Warn("不支持的事件类型: %s", eventType)
	}
}

func (eg *RiichiMahjong4p) handleRongHuEvent(event *share.RongHuEvent) {
	if event == nil {
		return
	}
	eg.handleReactionHuEvent(&share.HuEvent{GameMessageEvent: event.GameMessageEvent})
}

func (eg *RiichiMahjong4p) handleTouchHuEvent(event *share.TouchHuEvent) {
	if event == nil {
		return
	}
	log.Info("处理自摸事件")
	seatIndex, err := eg.getSeatIndex(event.GetUserID())
	if err != nil {
		log.Warn("获取玩家座位失败: %v", err)
		return
	}
	p := eg.Players[seatIndex]
	if p == nil || p.NewestTile == nil {
		log.Warn("自摸结算失败: 玩家或 NewestTile 为空: seat=%d", seatIndex)
		return
	}
	eg.handleRoundOverEvent([]HuClaim{{WinnerSeat: seatIndex, WinTile: *p.NewestTile}}, RoundEndTsumo)
}

func (eg *RiichiMahjong4p) handleReconnectEvent(event *share.ReconnectEvent) {
	if event == nil {
		return
	}
	log.Info("处理断线重连: user=%s", event.GetUserID())
	// todo 下发该玩家可见的状态快照
}

// todo TurnManager 需要重新初始化，TurnManager 提供开放重新初始化的方法
func (eg *RiichiMahjong4p) handleStartRoundEvent() {
	log.Info("新的一局游戏开始：%v", eg.Situation)
	if eg.DeckManager == nil {
		eg.DeckManager = NewDeckManager(UseRedFive)
	}

	eg.DeckManager.InitRound()
	eg.DeckManager.RevealDoraIndicator()
	eg.distributeCard()
	eg.DropTurn(eg.Situation.DealerIndex, true)
}

// distributeCard 发牌
func (eg *RiichiMahjong4p) distributeCard() {
	for i := 0; i < 4; i++ {
		p := eg.Players[i]
		if p == nil {
			continue
		}
		// 这里后续可以加 cap 的垃圾回收切片重建保护
		p.Tiles = p.Tiles[:0]
		p.DiscardPile = p.DiscardPile[:0]
		p.Melds = p.Melds[:0]
		p.IsRiichi = false
		p.IsWaiting = false
		p.NewestTile = nil
		p.DiscardedTiles = make(map[TileType]struct{})
		p.TenpaiWaits = make(map[TileType]TenpaiWaitState)
		p.TenpaiValid = false
	}

	for r := 0; r < 13; r++ {
		for i := 0; i < 4; i++ {
			t, ok := eg.DeckManager.Deal()
			if !ok {
				log.Warn("发牌失败: 牌山不足")
				return
			}
			p := eg.Players[i]
			if p == nil {
				continue
			}
			p.AddTile(t)
		}
	}

	dealer := eg.Situation.DealerIndex
	if dealer >= 0 && dealer < 4 {
		t, ok := eg.DeckManager.Deal()
		if !ok {
			log.Warn("庄家补牌失败: 牌山不足")
			return
		}
		p := eg.Players[dealer]
		if p != nil {
			p.DrawTile(t)
		}
	}
}

// DropTurn 进入打牌回合，todo 嵌入是否摸牌以及算法搜集的逻辑，如果无牌可摸，荒牌流局
func (eg *RiichiMahjong4p) DropTurn(seatIndex int, needTile bool) {
	if needTile {
		if eg.DeckManager == nil {
			log.Warn("DeckManager 为空")
			return
		}
		t, ok := eg.DeckManager.Draw()
		if !ok {
			eg.handleRoundOverEvent(nil, RoundEndDrawExhaustive)
			return
		}
		p := eg.Players[seatIndex]
		if p != nil {
			p.DrawTile(t)
		}
	}
	if err := eg.TurnManager.EnterDropPhase(seatIndex, DefaultRoundCompensation); err != nil {
		eg.HappenDamageError("DropTurn 异常")
	}

}

// todo 回合结束，根据是否流局，进行番符计算，番符计算的逻辑较为复杂，必须由 RiichiMahjong4p 调用，尽量不能独立出组件
func (eg *RiichiMahjong4p) handleRoundOverEvent(claims []HuClaim, endKind string) {
	if eg.TurnManager != nil {
		eg.TurnManager.stopAllTickers()
	}
	if eg.Situation == nil {
		log.Warn("Situation 为空")
		return
	}

	switch endKind {
	case RoundEndDrawExhaustive:
		eg.LeadNormalDrawEnding()
	case RoundEndDraw3Ron:
		eg.LeadHalfwayDrawEnding("三家点铳")
	case RoundEndTsumo:
		if len(claims) == 0 {
			log.Warn("自摸结算 claims 为空")
			return
		}
		eg.LeadTsumoEnding(claims[0])
	case RoundEndRon:
		if len(claims) == 0 {
			log.Warn("荣和结算 claims 为空")
			return
		}
		eg.LeadRonEnding(claims)
	default:
		log.Warn("未知回合结束类型: %s", endKind)
		return
	}
}

// LeadNormalDrawEnding 常规荒牌流局，需要罚符
func (eg *RiichiMahjong4p) LeadNormalDrawEnding() {
	var delta [4]int
	tenpaiSeats := make([]int, 0, 4)
	notenSeats := make([]int, 0, 4)
	dealerTenpai := false
	dealer := eg.Situation.DealerIndex

	for i := 0; i < 4; i++ {
		p := eg.Players[i]
		if p == nil {
			notenSeats = append(notenSeats, i)
			continue
		}
		isTenpai := p.TenpaiValid && len(p.TenpaiWaits) > 0
		if isTenpai {
			tenpaiSeats = append(tenpaiSeats, i)
			if i == dealer {
				dealerTenpai = true
			}
		} else {
			notenSeats = append(notenSeats, i)
		}
	}

	if len(tenpaiSeats) > 0 && len(tenpaiSeats) < 4 {
		winEach := 3000 / len(tenpaiSeats)
		loseEach := 3000 / len(notenSeats)
		for _, s := range tenpaiSeats {
			delta[s] += winEach
		}
		for _, s := range notenSeats {
			delta[s] -= loseEach
		}
	}

	if dealerTenpai {
		eg.Situation.Honba++
	} else {
		eg.Situation.Honba = 0
		eg.Situation.DealerIndex = (eg.Situation.DealerIndex + 1) % 4
		eg.Situation.RoundNumber++
	}

	eg.finalizeRound(delta, -1)
}

// LeadHalfwayDrawEnding 中途流局，不需要罚符
func (eg *RiichiMahjong4p) LeadHalfwayDrawEnding(reason string) {
	var delta [4]int
	eg.Situation.Honba++
	eg.finalizeRound(delta, -1)
}

// LeadRonEnding 荣和
func (eg *RiichiMahjong4p) LeadRonEnding(claims []HuClaim) {
	if eg.Situation == nil {
		return
	}
	var delta [4]int
	stickWinner := selectStickWinnerRonA(claims)
	honba := eg.Situation.Honba
	dealer := eg.Situation.DealerIndex
	dealerWin := false

	for _, c := range claims {
		if c.WinnerSeat == dealer {
			dealerWin = true
		}
		_, yakumanMult, _ := eg.evalClaimYakuman(c, RoundEndRon)
		if yakumanMult <= 0 {
			continue
		}
		base := 8000 * yakumanMult
		pay := base * 4
		if c.WinnerSeat == dealer {
			pay = base * 6
		}
		pay += 300 * honba
		delta[c.WinnerSeat] += pay
		if c.HasLoser {
			delta[c.LoserSeat] -= pay
		}
	}

	if dealerWin {
		eg.Situation.Honba++
	} else {
		eg.Situation.Honba = 0
		eg.Situation.DealerIndex = (eg.Situation.DealerIndex + 1) % 4
		eg.Situation.RoundNumber++
	}

	eg.finalizeRound(delta, stickWinner)
}

// LeadTsumoEnding 自摸
func (eg *RiichiMahjong4p) LeadTsumoEnding(claim HuClaim) {
	if eg.Situation == nil {
		return
	}
	var delta [4]int
	winner := claim.WinnerSeat
	dealer := eg.Situation.DealerIndex
	honba := eg.Situation.Honba
	_, yakumanMult, _ := eg.evalClaimYakuman(claim, RoundEndTsumo)
	if yakumanMult > 0 {
		base := 8000 * yakumanMult
		if winner == dealer {
			payEach := base*2 + 100*honba
			for i := 0; i < 4; i++ {
				if i == winner {
					continue
				}
				delta[i] -= payEach
				delta[winner] += payEach
			}
		} else {
			for i := 0; i < 4; i++ {
				if i == winner {
					continue
				}
				pay := base + 100*honba
				if i == dealer {
					pay = base*2 + 100*honba
				}
				delta[i] -= pay
				delta[winner] += pay
			}
		}
	}

	if winner == dealer {
		eg.Situation.Honba++
	} else {
		eg.Situation.Honba = 0
		eg.Situation.DealerIndex = (eg.Situation.DealerIndex + 1) % 4
		eg.Situation.RoundNumber++
	}

	eg.finalizeRound(delta, winner)
}

func (eg *RiichiMahjong4p) finalizeRound(delta [4]int, stickWinner int) {
	if eg.Situation == nil {
		return
	}
	if stickWinner >= 0 && stickWinner < 4 && eg.Situation.RiichiSticks > 0 {
		delta[stickWinner] += eg.Situation.RiichiSticks * 1000
		eg.Situation.RiichiSticks = 0
	}

	for i := 0; i < 4; i++ {
		p := eg.Players[i]
		if p == nil {
			continue
		}
		if delta[i] != 0 {
			p.AddPoints(delta[i])
		}
	}

	for i := 0; i < 4; i++ {
		p := eg.Players[i]
		if p != nil && p.Points < 0 {
			eg.handlerGameOverEvent()
			return
		}
	}

	if eg.Situation.RoundNumber > 4 {
		maxPoints := -1
		for i := 0; i < 4; i++ {
			p := eg.Players[i]
			if p == nil {
				continue
			}
			if p.Points > maxPoints {
				maxPoints = p.Points
			}
		}
		if maxPoints >= 30000 {
			eg.handlerGameOverEvent()
			return
		}
		eg.Situation.RoundNumber = 1
		//eg.Situation.RoundWind = nextWind(eg.Situation.RoundWind)
	}

	eg.Reactions = make(map[int]*PlayerReaction)
	eg.clearLastDiscard()
	eg.NotifyEvent(&StartRoundEvent{})
}

func (eg *RiichiMahjong4p) evalClaimYakuman(claim HuClaim, endKind string) (int, int, []Yaku) {
	var winner *PlayerImage
	if claim.WinnerSeat >= 0 && claim.WinnerSeat < 4 {
		winner = eg.Players[claim.WinnerSeat]
	}
	ctx := &YakuContext{Claim: claim, Winner: winner, Situation: eg.Situation, EndKind: endKind}

	results := make([]Yaku, 0, 8)
	hanSum := 0
	yakumanMultSum := 0
	for _, checker := range RiichiMahjong4pYakuRegistry {
		han, ym := checker.Check(ctx)
		if han > 0 || ym > 0 {
			results = append(results, checker.ID())
			hanSum += han
			yakumanMultSum += ym
		}
	}
	return hanSum, yakumanMultSum, results
}

func selectStickWinnerRonA(claims []HuClaim) int {
	if len(claims) == 0 {
		return -1
	}
	loser := claims[0].LoserSeat
	best := -1
	bestDist := 5
	for _, c := range claims {
		w := c.WinnerSeat
		d := (w - loser + 4) % 4
		if d == 0 {
			continue
		}
		if d < bestDist {
			bestDist = d
			best = w
		}
	}
	return best
}

// todo 游戏结束，生命周期结束，通知结果，自毁回调
func (eg *RiichiMahjong4p) handlerGameOverEvent() {
	log.Info("游戏结束")
	eg.Terminate()
}

func (eg *RiichiMahjong4p) handleTimeoutEvent(event *TimeoutEvent) {
	seatIndex := event.SeatIndex
	log.Info("玩家 %d 超时", seatIndex)

	state := eg.TurnManager.GetState()
	switch state {
	case StateDropping:
		eg.handleDropTimeout(seatIndex)
	case StateReacting:
		eg.handleReactionTimeout(seatIndex)
	}
}

func (eg *RiichiMahjong4p) handleDropTileEvent(event *share.DropTileEvent) {
	log.Info("处理出牌事件")
	if eg.TurnManager.GetState() != StateDropping {
		log.Warn("当前状态不是 StateDropping，而是: %v", eg.TurnManager.GetState())
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
	ok := ticker.Stop()
	if !ok {
		log.Warn("handleDropTileEvent 已经超时处理, %v", event)
		return
	}

	// 处理出牌逻辑
	player := eg.Players[seatIndex]
	if player == nil {
		log.Warn("玩家 %d 不存在", seatIndex)
		return
	}

	tile := toMahjongTile(event.GetTile())
	if !player.DiscardTile(tile) {
		log.Warn("玩家 %d 手中没有该牌: %v", seatIndex, tile)
		return
	}
	eg.setLastDiscard(seatIndex, tile)

	log.Info("玩家 %d 出牌: %v", seatIndex, tile)

	eg.waitReaction(seatIndex)
}

func (eg *RiichiMahjong4p) waitReaction(excludeSeat int) {
	if eg.TurnManager.GetState() != StateDropping {
		log.Warn("当前状态不是 StateDropping，而是: %v", eg.TurnManager.GetState())
		return
	}

	// 搜索可用操作
	eg.TurnManager.EnterSelectingPhase()
	reactions := eg.calculateAvailableOperations(excludeSeat)
	eg.Reactions = reactions

	if len(eg.Reactions) == 0 {
		nextPlayer := eg.TurnManager.NextTurn()
		eg.DropTurn(nextPlayer, true)
		return
	}

	// 下发操作给客户端
	eg.broadcastOperations(eg.Reactions)

	if eg.TurnManager.GetState() != StateSelecting {
		log.Warn("当前状态不是 StateSelecting，而是: %v", eg.TurnManager.GetState())
		return
	}
	eg.TurnManager.EnterReactingPhase()

	for seatIndex := range eg.Reactions {
		ticker := eg.TurnManager.GetPlayerTicker(seatIndex)
		allocatedTime := ticker.Available + 3
		ticker.SetAvailable(allocatedTime)
		if err := ticker.Start(allocatedTime); err != nil {
			log.Error("启动反应计时失败 (座位 %d): %v", seatIndex, err)
		}
	}
}

// recordPlayerResponse 记录玩家响应
func (eg *RiichiMahjong4p) recordPlayerResponse(seatIndex int, chosenOp *PlayerOperation) {
	ticker := eg.TurnManager.GetPlayerTicker(seatIndex)
	ok := ticker.Stop()
	if !ok {
		log.Warn("recordPlayerResponse 响应已经超时处理, %v", chosenOp)
		return
	}

	reaction, exists := eg.Reactions[seatIndex]
	if !exists {
		log.Warn("玩家 %d 不在反应列表中", seatIndex)
		return
	}
	reaction.ChosenOp = chosenOp
	reaction.Responded = true
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

	reaction, exists := eg.Reactions[seatIndex]

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

	reaction, exists := eg.Reactions[seatIndex]

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

	reaction, exists := eg.Reactions[seatIndex]

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

func (eg *RiichiMahjong4p) handleReactionHuEvent(event *share.HuEvent) {
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
	reaction, exists := eg.Reactions[seatIndex]
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

func (eg *RiichiMahjong4p) handleRiichiEvent(event *share.RiichiEvent) {
	log.Info("处理立直事件")
	seatIndex, err := eg.getSeatIndex(event.GetUserID())
	if err != nil {
		log.Warn("获取玩家座位失败: %v", err)
		return
	}
	player := eg.Players[seatIndex]
	if player == nil {
		log.Warn("玩家 %d 不存在", seatIndex)
		return
	}

	// 标记玩家为听牌状态
	player.IsWaiting = true
	log.Info("玩家 %d 立直", seatIndex)
}

// makeTimeoutHandler 创建超时处理回调
func (eg *RiichiMahjong4p) makeTimeoutHandler(seatIndex int) func() {
	return func() {
		eg.NotifyEvent(&TimeoutEvent{SeatIndex: seatIndex})
	}
}

// makeStopHandler 玩家倒计时主动停止回调，这里的逻辑已经交给 NotifyEvent/actorLoop 了，占位编程
func (eg *RiichiMahjong4p) makeStopHandler(seatIndex int) func() {
	return func() {
		state := eg.TurnManager.GetState()
		switch state {
		case StateDropping:
		case StateReacting:
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
	tileToDiscard, ok := player.DiscardNewestOrLast()
	if !ok {
		log.Warn("玩家 %d 自动出牌失败", seatIndex)
		return
	}
	log.Info("玩家 %d 自动打出牌: %v", seatIndex, tileToDiscard)
	eg.setLastDiscard(seatIndex, tileToDiscard)
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

// handleReactionComplete 处理玩家
func (eg *RiichiMahjong4p) handleReactionComplete() {
	log.Info("所有玩家反应完成")

	if eg.TurnManager.GetState() != StateReacting {
		eg.HappenDamageError("状态出错，应该是 StateReacting")
		return
	}
	eg.TurnManager.EnterChoosingPhase()

	ronSeats := make([]int, 0, 3)
	for seatIndex, reaction := range eg.Reactions {
		if reaction.ChosenOp != nil && reaction.ChosenOp.Type == "HU" {
			ronSeats = append(ronSeats, seatIndex)
		}
	}
	if len(ronSeats) > 0 {
		if len(ronSeats) >= 3 {
			log.Info("一炮三响，荒牌流局")
			eg.handleRoundOverEvent(nil, RoundEndDraw3Ron)
			return
		}
		claims := make([]HuClaim, 0, len(ronSeats))
		for _, w := range ronSeats {
			claims = append(claims, HuClaim{WinnerSeat: w, HasLoser: true, LoserSeat: eg.lastDiscard.Seat, WinTile: eg.lastDiscard.Tile})
		}
		if len(ronSeats) == 2 {
			log.Info("一炮两响，累计计算: winners=%v, loser=%d, tile=%v", ronSeats, eg.lastDiscard.Seat, eg.lastDiscard.Tile)
			eg.handleRoundOverEvent(claims, RoundEndRon)
			return
		}
		log.Info("荣和: winner=%d, loser=%d, tile=%v", ronSeats[0], eg.lastDiscard.Seat, eg.lastDiscard.Tile)
		eg.handleRoundOverEvent(claims, RoundEndRon)
		return
	}

	// 执行吃碰杠选择算法
	// 优先级：荣和 > 明杠 > 碰 > 吃
	selectedAction := eg.selectBestReaction()

	if selectedAction == nil {
		// 没有有效的反应，进入下一个出牌阶段
		nextPlayer := eg.TurnManager.NextTurn()
		if err := eg.TurnManager.EnterDropPhase(nextPlayer, DefaultRoundCompensation); err != nil {
			log.Error("进入出牌阶段失败: %v", err)
		}
		return
	}

	// 执行选中的操作
	eg.executeReaction(selectedAction)
}

// selectBestReaction 选择最优的反应操作 todo 无法处理一炮多响
// 优先级：荣和 > 明杠 > 碰 > 吃
func (eg *RiichiMahjong4p) selectBestReaction() *ReactionAction {
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
	if !eg.lastDiscard.Valid {
		log.Warn("没有 lastDiscard，无法执行反应")
		return
	}
	discarder := eg.lastDiscard.Seat
	discarderPlayer := eg.Players[discarder]
	if discarderPlayer == nil || len(discarderPlayer.DiscardPile) == 0 {
		log.Warn("放铳者弃牌堆为空")
		return
	}
	called := discarderPlayer.DiscardPile[len(discarderPlayer.DiscardPile)-1]
	if called.Type != eg.lastDiscard.Tile.Type || called.ID != eg.lastDiscard.Tile.ID {
		log.Warn("lastDiscard 与弃牌堆最后一张不一致: last=%v, pile=%v", eg.lastDiscard.Tile, called)
		return
	}

	caller := eg.Players[action.PlayerSeat]
	if caller == nil {
		log.Warn("鸣牌玩家不存在: %d", action.PlayerSeat)
		return
	}

	switch action.Type {
	case "PENG":
		if len(action.Tiles) != 2 {
			log.Warn("PENG 参数异常")
			return
		}
		t1 := action.Tiles[0]
		t2 := action.Tiles[1]
		if !caller.RemoveTile(t1) || !caller.RemoveTile(t2) {
			log.Warn("PENG 找不到手牌: %v %v", t1, t2)
			return
		}
		discarderPlayer.DiscardPile = discarderPlayer.DiscardPile[:len(discarderPlayer.DiscardPile)-1]
		caller.Melds = append(caller.Melds, Meld{Type: "Peng", Tiles: []Tile{called, t1, t2}, From: discarder})
		eg.clearLastDiscard()
		eg.DropTurn(action.PlayerSeat, false)
		return

	case "CHI":
		if len(action.Tiles) != 2 {
			log.Warn("CHI 参数异常")
			return
		}
		t1 := action.Tiles[0]
		t2 := action.Tiles[1]
		if !caller.RemoveTile(t1) || !caller.RemoveTile(t2) {
			log.Warn("CHI 找不到手牌: %v %v", t1, t2)
			return
		}
		discarderPlayer.DiscardPile = discarderPlayer.DiscardPile[:len(discarderPlayer.DiscardPile)-1]
		caller.Melds = append(caller.Melds, Meld{Type: "Chi", Tiles: []Tile{called, t1, t2}, From: discarder})
		eg.clearLastDiscard()
		eg.DropTurn(action.PlayerSeat, false)
		return

	case "GANG":
		if len(action.Tiles) != 3 {
			log.Warn("GANG 参数异常")
			return
		}
		t1 := action.Tiles[0]
		t2 := action.Tiles[1]
		t3 := action.Tiles[2]
		if !caller.RemoveTile(t1) || !caller.RemoveTile(t2) || !caller.RemoveTile(t3) {
			log.Warn("GANG 找不到手牌: %v %v %v", t1, t2, t3)
			return
		}
		discarderPlayer.DiscardPile = discarderPlayer.DiscardPile[:len(discarderPlayer.DiscardPile)-1]
		caller.Melds = append(caller.Melds, Meld{Type: "Gang", Tiles: []Tile{called, t1, t2, t3}, From: discarder})
		eg.clearLastDiscard()
		eg.DropTurn(action.PlayerSeat, true)
		return
	default:
		log.Warn("不支持的反应类型: %s", action.Type)
		return
	}
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

	clonedPlayers := [4]*PlayerImage{}

	return &RiichiMahjong4p{
		State:       engines.GameWaiting,
		Worker:      eg.Worker,
		UserMap:     nil,
		Situation:   clonedSituation,
		DeckManager: NewDeckManager(UseRedFive),
		Players:     clonedPlayers,
		TurnManager: nil,
	}
}

// HappenDamageError 发生引擎崩坏的重大事件
func (eg *RiichiMahjong4p) HappenDamageError(err string) {
	log.Warn("引擎崩坏: %s", err)
	eg.Terminate()
}

// Terminate 自毁程序
func (eg *RiichiMahjong4p) Terminate() {
	eg.requestDestroyRoom()
	return
}

func (eg *RiichiMahjong4p) requestDestroyRoom() {
	if eg.Worker == nil {
		return
	}
	if eg.RoomID == "" {
		return
	}
	eg.Worker.RequestDestroyRoom(eg.RoomID)
}

func (eg *RiichiMahjong4p) Close() {
	eg.closeOnce.Do(func() {
		eg.closed.Store(true)
		if eg.gameDone != nil {
			close(eg.gameDone)
		}
		if eg.actorExit != nil {
			<-eg.actorExit
		}

		eg.Worker = nil
		eg.State = engines.GameFinished

		if eg.roundStartTimer != nil {
			eg.roundStartTimer.Stop()
			eg.roundStartTimer = nil
		}

		if eg.TurnManager != nil {
			eg.TurnManager.stopAllTickers()
			eg.TurnManager = nil
		}
		eg.Reactions = nil

		eg.UserMap = nil
		eg.Players = [4]*PlayerImage{}
		eg.DeckManager = nil
	})
}
