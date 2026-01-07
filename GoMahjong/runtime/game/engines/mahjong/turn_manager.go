package mahjong

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type TickerState int

const (
	StateIdle    TickerState = iota // 空闲
	StateRunning                    // 计时中
	StateStopped                    // 已停止
	StateTimeout                    // 已超时
)

type TurnState int // 空闲、收集、等待出牌者反应、等待非出牌者反应

const (
	TurnStateIdle           TurnState = iota // 等待开始
	TurnStateWaitMain                        // 等待出牌、立直
	TurnStateSelecting                       // 吃、碰、杠、立直收集
	TurnStateWaitReactions                   // 等待反应（吃碰杠和）
	TurnStateApplyOperation                  // 棋牌格局归属改变，如洗牌、打牌、鸣牌
)

type TurnManager struct {
	TurnPointer int       // 当前出牌玩家座位
	State       TurnState // 当前回合状态
	Tickers     [4]*PlayerTicker
}

// NewTurnManager 创建新的回合管理器
func NewTurnManager(tickers [4]*PlayerTicker) *TurnManager {
	return &TurnManager{
		TurnPointer: 0,
		State:       TurnStateIdle,
		Tickers:     tickers,
	}
}

// NextTurn 下一个玩家出牌
func (tm *TurnManager) NextTurn() int {
	tm.TurnPointer = (tm.TurnPointer + 1) % 4
	return tm.TurnPointer
}

// GetCurrentPlayer 获取当前出牌玩家座位
func (tm *TurnManager) GetCurrentPlayer() int {
	return tm.TurnPointer
}

// GetState 获取当前回合状态
func (tm *TurnManager) GetState() TurnState {
	return tm.State
}

func (tm *TurnManager) stopAllTickers() {
	for i := 0; i < 4; i++ {
		if tm.Tickers[i].GetState() == StateRunning {
			tm.Tickers[i].Stop()
		}
	}
}

// EnterDropPhase 进入出牌阶段
// roundCompensation: 本回合补偿时间（秒），默认 5 秒
func (tm *TurnManager) EnterDropPhase(seatIndex int, roundCompensation int) error {
	if seatIndex < 0 || seatIndex >= 4 {
		return fmt.Errorf("无效的座位索引: %d", seatIndex)
	}

	tm.stopAllTickers()
	tm.TurnPointer = seatIndex
	tm.State = TurnStateWaitMain

	// 启动出牌玩家的计时
	// 分配时间 = 玩家总剩余时间 + 本回合补偿
	ticker := tm.Tickers[seatIndex]
	allocatedTime := ticker.Available + roundCompensation
	if allocatedTime > DefaultMaxRoundTime {
		allocatedTime = DefaultMaxRoundTime
	}
	ticker.SetAvailable(allocatedTime)
	if err := ticker.Start(allocatedTime); err != nil {
		return fmt.Errorf("启动出牌计时失败: %v", err)
	}

	return nil
}

// EnterSelectingPhase 进入收集阶段（吃碰杠）
// 此阶段不需要计时
func (tm *TurnManager) EnterSelectingPhase() {
	tm.stopAllTickers()
	tm.State = TurnStateSelecting
}

// EnterReactingPhase 进入等待反应阶段（吃碰杠）
// 此阶段不需要计时
func (tm *TurnManager) EnterReactingPhase() {
	tm.stopAllTickers()
	tm.State = TurnStateWaitReactions
}

// EnterChoosingPhase 进入选择阶段（吃碰杠）
// 此阶段不需要计时
func (tm *TurnManager) EnterChoosingPhase() {
	tm.stopAllTickers()
	tm.State = TurnStateApplyOperation
}

// GetPlayerTicker 获取玩家的计时器
func (tm *TurnManager) GetPlayerTicker(seatIndex int) *PlayerTicker {
	return tm.Tickers[seatIndex]
}

// GetAllPlayerTimerStates 获取所有玩家的计时器状态
func (tm *TurnManager) GetAllPlayerTimerStates() [4]TickerState {
	var states [4]TickerState
	for i := 0; i < 4; i++ {
		states[i] = tm.Tickers[i].GetState()
	}
	return states
}

type PlayerTicker struct {
	// 时间管理（单位：秒）
	Available      int       // 总剩余时间（跨回合累计）
	RoundStartTime time.Time // 本回合开始时间

	// 状态管理
	State     TickerState
	isRunning bool // 防止重复启动
	ctx       context.Context
	cancel    context.CancelFunc

	// 回调函数
	onTimeout     func()
	onStop        func()
	onStateChange func(oldState, newState TickerState)

	// 并发控制
	sync.RWMutex
}

// NewPlayerTicker 创建新的玩家计时器
func NewPlayerTicker(totalTime int) *PlayerTicker {
	return &PlayerTicker{
		Available: totalTime,
		State:     StateIdle,
		isRunning: false,
	}
}

// Start 启动计时
// duration: 本次分配的时间（秒）
// 返回 error 如果时间不足或已在运行
func (pt *PlayerTicker) Start(duration int) error {
	pt.Lock()
	defer pt.Unlock()

	if pt.isRunning {
		return fmt.Errorf("计时已在运行，无法重复启动")
	}
	if pt.Available < duration {
		return fmt.Errorf("剩余时间 %d 秒不足 %d 秒", pt.Available, duration)
	}

	pt.isRunning = true
	oldState := pt.State
	pt.State = StateRunning
	pt.RoundStartTime = time.Now()

	// 触发状态变化回调
	if pt.onStateChange != nil {
		pt.onStateChange(oldState, StateRunning)
	}
	go pt.timerLoop(duration)

	return nil
}

// timerLoop 计时循环（在 goroutine 中运行）
func (pt *PlayerTicker) timerLoop(duration int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(duration)*time.Second)
	defer cancel()
	pt.Lock()
	pt.ctx = ctx
	pt.cancel = cancel
	pt.Unlock()
	<-ctx.Done()

	pt.Lock()
	defer pt.Unlock()

	// 检查是否是超时还是被取消
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		oldState := pt.State
		pt.State = StateTimeout
		pt.isRunning = false
		pt.Available = 0

		if pt.onStateChange != nil {
			pt.onStateChange(oldState, StateTimeout)
		}
		if pt.onTimeout != nil {
			pt.onTimeout()
		}
	} else if errors.Is(ctx.Err(), context.Canceled) {
		// 被取消处理（玩家操作）
		usedTime := int(time.Since(pt.RoundStartTime).Seconds())
		pt.Available = max(0, pt.Available-usedTime)
		oldState := pt.State
		pt.State = StateStopped
		pt.isRunning = false

		if pt.onStateChange != nil {
			pt.onStateChange(oldState, StateStopped)
		}
		if pt.onStop != nil {
			pt.onStop()
		}
	}
}

// Stop 停止计时
// 返回已用时间（秒）
func (pt *PlayerTicker) Stop() bool {
	pt.Lock()
	defer pt.Unlock()
	if !pt.isRunning || pt.cancel == nil {
		return false
	}
	pt.cancel()
	return true
}

func (pt *PlayerTicker) SetAvailable(Available int) int {
	pt.Lock()
	defer pt.Unlock()
	pt.Available = Available
	return pt.Available
}

// GetState 获取当前状态
func (pt *PlayerTicker) GetState() TickerState {
	pt.RLock()
	defer pt.RUnlock()
	return pt.State
}

// SetOnTimeout 设置超时回调
func (pt *PlayerTicker) SetOnTimeout(callback func()) {
	pt.Lock()
	defer pt.Unlock()
	pt.onTimeout = callback
}

// SetOnStop 设置停止回调
func (pt *PlayerTicker) SetOnStop(callback func()) {
	pt.Lock()
	defer pt.Unlock()
	pt.onStop = callback
}

// SetOnStateChange 设置状态变化回调
func (pt *PlayerTicker) SetOnStateChange(callback func(oldState, newState TickerState)) {
	pt.Lock()
	defer pt.Unlock()
	pt.onStateChange = callback
}

/*
	废弃设计，利用 EventOrder 的时间戳来实现时间的合并和超时处理，不易管理

type PlayerTicker struct {
	Available    time.Duration
	TickerChan   chan int
	EventOrder   atomic.Int32
	lastTickTime int64
}

func NewPlayerTicker() *PlayerTicker {
	return &PlayerTicker{
		Available:  30 * time.Second,
		TickerChan: make(chan int, 5),
	}
}

func (pt *PlayerTicker) StartTick() {
	order := pt.EventOrder.Load()
	time.AfterFunc(pt.Available, func() {
		pt.putChan(int(order))
	})
	pt.lastTickTime = time.Now().UnixNano()
ticktag:
	select {
	case i := <-pt.TickerChan:
		if i < int(pt.EventOrder.Load()) {
			goto ticktag
		}
		// pt.PlayerTicker = 计算差值
	}
}

func (pt *PlayerTicker) putChan(i int) {
	pt.TickerChan <- i
}

func (pt *PlayerTicker) putChanInstant() {
	i := pt.EventOrder.Load()
	pt.TickerChan <- int(i)
}


*/
