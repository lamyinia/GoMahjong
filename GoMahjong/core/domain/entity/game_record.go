package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GameRecord 游戏记录元数据（聚合根）
// 存储游戏基本信息、玩家信息、最终结果
type GameRecord struct {
	ID          primitive.ObjectID `bson:"_id"`
	RoomID      string             `bson:"room_id"`      // 房间ID
	GameType    string             `bson:"game_type"`    // "riichi_mahjong_4p"
	Players     []PlayerInfo       `bson:"players"`      // 玩家信息（座位、用户ID）
	StartTime   time.Time          `bson:"start_time"`   // 游戏开始时间
	EndTime     time.Time          `bson:"end_time"`     // 游戏结束时间（游戏结束时设置）
	Duration    int                `bson:"duration"`     // 游戏时长（秒）
	FinalResult *GameFinalResult   `bson:"final_result"` // 最终结果（游戏结束时设置）
	Status      string             `bson:"status"`       // "completed", "aborted"
	CreatedAt   time.Time          `bson:"created_at"`
}

// PlayerInfo 玩家信息
type PlayerInfo struct {
	UserID    string `bson:"user_id"`
	SeatIndex int    `bson:"seat_index"`
	Nickname  string `bson:"nickname,omitempty"` // 可选
}

// GameFinalResult 游戏最终结果
type GameFinalResult struct {
	Rankings []PlayerRanking `bson:"rankings"` // 最终排名（按排名排序）
	Points   [4]int          `bson:"points"`   // 最终点数（按座位索引）
}

// PlayerRanking 玩家排名
type PlayerRanking struct {
	SeatIndex int    `bson:"seat_index"` // 座位索引
	UserID    string `bson:"user_id"`    // 用户ID
	Points    int    `bson:"points"`     // 最终点数
	Rank      int    `bson:"rank"`       // 排名 (1-4)
}

// NewGameRecord 创建游戏记录
func NewGameRecord(roomID, gameType string, players []PlayerInfo) *GameRecord {
	return &GameRecord{
		ID:        primitive.NewObjectID(),
		RoomID:    roomID,
		GameType:  gameType,
		Players:   players,
		StartTime: time.Now(),
		Status:    "in_progress",
		CreatedAt: time.Now(),
	}
}

// CompleteGame 完成游戏（设置最终结果）
func (gr *GameRecord) CompleteGame(finalResult *GameFinalResult) {
	gr.EndTime = time.Now()
	gr.Duration = int(gr.EndTime.Sub(gr.StartTime).Seconds())
	gr.FinalResult = finalResult
	gr.Status = "completed"
}

// AbortGame 中止游戏
func (gr *GameRecord) AbortGame() {
	gr.EndTime = time.Now()
	gr.Duration = int(gr.EndTime.Sub(gr.StartTime).Seconds())
	gr.Status = "aborted"
}
