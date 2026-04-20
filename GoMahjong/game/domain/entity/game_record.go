package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GameRecord struct {
	ID          primitive.ObjectID `bson:"_id"`
	RoomID      string             `bson:"room_id"`
	GameType    string             `bson:"game_type"`
	Players     []PlayerInfo       `bson:"players"`
	StartTime   time.Time          `bson:"start_time"`
	EndTime     time.Time          `bson:"end_time"`
	Duration    int                `bson:"duration"`
	FinalResult *GameFinalResult   `bson:"final_result"`
	Status      string             `bson:"status"`
	CreatedAt   time.Time          `bson:"created_at"`
}

type PlayerInfo struct {
	UserID    string `bson:"user_id"`
	SeatIndex int    `bson:"seat_index"`
	Nickname  string `bson:"nickname,omitempty"`
}

type GameFinalResult struct {
	Rankings []PlayerRanking `bson:"rankings"`
	Points   [4]int          `bson:"points"`
}

type PlayerRanking struct {
	SeatIndex int    `bson:"seat_index"`
	UserID    string `bson:"user_id"`
	Points    int    `bson:"points"`
	Rank      int    `bson:"rank"`
}

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

func (gr *GameRecord) CompleteGame(finalResult *GameFinalResult) {
	gr.EndTime = time.Now()
	gr.Duration = int(gr.EndTime.Sub(gr.StartTime).Seconds())
	gr.FinalResult = finalResult
	gr.Status = "completed"
}

func (gr *GameRecord) AbortGame() {
	gr.EndTime = time.Now()
	gr.Duration = int(gr.EndTime.Sub(gr.StartTime).Seconds())
	gr.Status = "aborted"
}
