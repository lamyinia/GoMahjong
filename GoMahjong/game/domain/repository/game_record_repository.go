package repository

import (
	"context"
	"game/domain/entity"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GameRecordRepository interface {
	SaveGameRecord(ctx context.Context, record *entity.GameRecord) error
	FindGameRecord(ctx context.Context, recordID primitive.ObjectID) (*entity.GameRecord, error)
	FindGameRecordsByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.GameRecord, error)
	FindGameRecordsByRoom(ctx context.Context, roomID string) (*entity.GameRecord, error)
	SaveRoundRecord(ctx context.Context, round *entity.RoundRecord) error
	SaveRoundRecords(ctx context.Context, rounds []*entity.RoundRecord) error
	FindRoundRecords(ctx context.Context, gameRecordID primitive.ObjectID) ([]*entity.RoundRecord, error)
	FindRoundRecord(ctx context.Context, gameRecordID primitive.ObjectID, roundNumber int) (*entity.RoundRecord, error)
}
