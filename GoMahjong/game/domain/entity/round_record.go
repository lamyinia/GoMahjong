package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoundRecord struct {
	ID           primitive.ObjectID `bson:"_id"`
	GameRecordID primitive.ObjectID `bson:"game_record_id"`
	RoundNumber  int                `bson:"round_number"`
	RoundWind    string             `bson:"round_wind"`
	DealerIndex  int                `bson:"dealer_index"`
	Honba        int                `bson:"honba"`
	Events       []RoundEvent       `bson:"events"`
	RoundResult  *RoundResult       `bson:"round_result"`
	StartTime    time.Time          `bson:"start_time"`
	EndTime      time.Time          `bson:"end_time"`
	Duration     int                `bson:"duration"`
	CreatedAt    time.Time          `bson:"created_at"`
}

type RoundEvent struct {
	Sequence  int                    `bson:"sequence"`
	EventType string                 `bson:"event_type"`
	Timestamp time.Time              `bson:"timestamp"`
	SeatIndex int                    `bson:"seat_index"`
	Data      map[string]interface{} `bson:"data"`
}

type RoundResult struct {
	EndType    string    `bson:"end_type"`
	Claims     []HuClaim `bson:"claims"`
	Delta      [4]int    `bson:"delta"`
	Points     [4]int    `bson:"points"`
	Reason     string    `bson:"reason"`
	NextDealer int       `bson:"next_dealer"`
}

type HuClaim struct {
	WinnerSeat int      `bson:"winner_seat"`
	LoserSeat  int      `bson:"loser_seat"`
	WinTile    Tile     `bson:"win_tile"`
	Han        int      `bson:"han"`
	Fu         int      `bson:"fu"`
	Yaku       []string `bson:"yaku"`
	Points     int      `bson:"points"`
}

type Tile struct {
	Type int `bson:"type"`
	ID   int `bson:"id"`
}

func NewRoundRecord(gameRecordID primitive.ObjectID, roundNumber int, roundWind string, dealerIndex, honba int) *RoundRecord {
	return &RoundRecord{
		ID:           primitive.NewObjectID(),
		GameRecordID: gameRecordID,
		RoundNumber:  roundNumber,
		RoundWind:    roundWind,
		DealerIndex:  dealerIndex,
		Honba:        honba,
		Events:       make([]RoundEvent, 0, 100),
		StartTime:    time.Now(),
		CreatedAt:    time.Now(),
	}
}

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

func (rr *RoundRecord) CompleteRound(result *RoundResult) {
	rr.EndTime = time.Now()
	rr.Duration = int(rr.EndTime.Sub(rr.StartTime).Seconds())
	rr.RoundResult = result
}

const (
	EventTypeRoundStart  = "round_start"
	EventTypeDrawTile    = "draw_tile"
	EventTypeDiscardTile = "discard_tile"
	EventTypeChi         = "chi"
	EventTypePeng        = "peng"
	EventTypeGang        = "gang"
	EventTypeAnkan       = "ankan"
	EventTypeKakan       = "kakan"
	EventTypeRiichi      = "riichi"
	EventTypeRon         = "ron"
	EventTypeTsumo       = "tsumo"
	EventTypeRoundEnd    = "round_end"
)
