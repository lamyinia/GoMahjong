package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RoundRecord 局记录（每局一个文档）
// 存储该局的所有事件流和回合结果
type RoundRecord struct {
	ID           primitive.ObjectID `bson:"_id"`
	GameRecordID primitive.ObjectID `bson:"game_record_id"` // 关联游戏记录
	RoundNumber  int                `bson:"round_number"`   // 局数 (1-4)
	RoundWind    string             `bson:"round_wind"`     // 场风 "East", "South", "West", "North"
	DealerIndex  int                `bson:"dealer_index"`   // 庄家座位
	Honba        int                `bson:"honba"`          // 本场数
	Events       []RoundEvent       `bson:"events"`         // 事件流（按时间顺序）
	RoundResult  *RoundResult       `bson:"round_result"`   // 回合结果（回合结束时设置）
	StartTime    time.Time          `bson:"start_time"`     // 回合开始时间
	EndTime      time.Time          `bson:"end_time"`       // 回合结束时间
	Duration     int                `bson:"duration"`       // 回合时长（秒）
	CreatedAt    time.Time          `bson:"created_at"`
}

// RoundEvent 回合事件（只存事件，不存快照）
type RoundEvent struct {
	Sequence  int                    `bson:"sequence"`   // 事件序号（从0开始，该局内递增）
	EventType string                 `bson:"event_type"` // 事件类型
	Timestamp time.Time              `bson:"timestamp"`  // 事件发生时间
	SeatIndex int                    `bson:"seat_index"` // 操作玩家座位（-1表示系统事件）
	Data      map[string]interface{} `bson:"data"`       // 事件数据（JSON格式，灵活存储）
}

// RoundResult 回合结果
type RoundResult struct {
	EndType    string    `bson:"end_type"`    // "RON", "TSUMO", "DRAW_EXHAUSTIVE", "DRAW_3RON", "DRAW_4KAN"
	Claims     []HuClaim `bson:"claims"`      // 和牌信息（如果有）
	Delta      [4]int    `bson:"delta"`       // 点数变化（按座位索引）
	Points     [4]int    `bson:"points"`      // 回合结束后的点数（按座位索引）
	Reason     string    `bson:"reason"`      // 流局原因（如果有）
	NextDealer int       `bson:"next_dealer"` // 下一局庄家（-1表示游戏结束）
}

// HuClaim 和牌信息
type HuClaim struct {
	WinnerSeat int      `bson:"winner_seat"` // 和牌玩家座位
	LoserSeat  int      `bson:"loser_seat"`  // 放铳玩家座位（荣和时有值，自摸时为-1）
	WinTile    Tile     `bson:"win_tile"`    // 和牌
	Han        int      `bson:"han"`         // 番数
	Fu         int      `bson:"fu"`          // 符数
	Yaku       []string `bson:"yaku"`        // 役列表
	Points     int      `bson:"points"`      // 点数
}

// Tile 牌（用于存储）
type Tile struct {
	Type int `bson:"type"` // TileType
	ID   int `bson:"id"`   // 牌的ID
}

// NewRoundRecord 创建局记录
func NewRoundRecord(gameRecordID primitive.ObjectID, roundNumber int, roundWind string, dealerIndex, honba int) *RoundRecord {
	return &RoundRecord{
		ID:           primitive.NewObjectID(),
		GameRecordID: gameRecordID,
		RoundNumber:  roundNumber,
		RoundWind:    roundWind,
		DealerIndex:  dealerIndex,
		Honba:        honba,
		Events:       make([]RoundEvent, 0, 100), // 预分配容量
		StartTime:    time.Now(),
		CreatedAt:    time.Now(),
	}
}

// AddEvent 添加事件
func (rr *RoundRecord) AddEvent(eventType string, seatIndex int, data map[string]interface{}) {
	event := RoundEvent{
		Sequence:  len(rr.Events),
		EventType: eventType,
		Timestamp: time.Now(),
		SeatIndex: seatIndex,
		Data:      data,
	}
	rr.Events = append(rr.Events, event)
}

// CompleteRound 完成回合（设置回合结果）
func (rr *RoundRecord) CompleteRound(result *RoundResult) {
	rr.EndTime = time.Now()
	rr.Duration = int(rr.EndTime.Sub(rr.StartTime).Seconds())
	rr.RoundResult = result
}

// 事件类型常量
const (
	EventTypeRoundStart  = "round_start"  // 回合开始
	EventTypeDrawTile    = "draw_tile"    // 摸牌
	EventTypeDiscardTile = "discard_tile" // 出牌
	EventTypeChi         = "chi"          // 吃
	EventTypePeng        = "peng"         // 碰
	EventTypeGang        = "gang"         // 明杠
	EventTypeAnkan       = "ankan"        // 暗杠
	EventTypeKakan       = "kakan"        // 加杠
	EventTypeRiichi      = "riichi"       // 立直
	EventTypeRon         = "ron"          // 荣和
	EventTypeTsumo       = "tsumo"        // 自摸
	EventTypeRoundEnd    = "round_end"    // 回合结束
)
