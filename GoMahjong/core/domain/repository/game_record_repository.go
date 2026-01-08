package repository

import (
	"context"
	"core/domain/entity"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GameRecordRepository 游戏记录仓储接口
type GameRecordRepository interface {
	// SaveGameRecord 保存游戏记录（元数据）
	SaveGameRecord(ctx context.Context, record *entity.GameRecord) error

	// FindGameRecord 根据ID查找游戏记录
	FindGameRecord(ctx context.Context, recordID primitive.ObjectID) (*entity.GameRecord, error)

	// FindGameRecordsByUser 查找用户参与的游戏记录（分页）
	FindGameRecordsByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.GameRecord, error)

	// FindGameRecordsByRoom 根据房间ID查找游戏记录
	FindGameRecordsByRoom(ctx context.Context, roomID string) (*entity.GameRecord, error)

	// SaveRoundRecord 保存局记录（每局一个文档）
	SaveRoundRecord(ctx context.Context, round *entity.RoundRecord) error

	// SaveRoundRecords 批量保存局记录（使用 MongoDB InsertMany）
	SaveRoundRecords(ctx context.Context, rounds []*entity.RoundRecord) error

	// FindRoundRecords 查找游戏的所有局记录（按局数排序）
	FindRoundRecords(ctx context.Context, gameRecordID primitive.ObjectID) ([]*entity.RoundRecord, error)

	// FindRoundRecord 查找指定局数的记录
	FindRoundRecord(ctx context.Context, gameRecordID primitive.ObjectID, roundNumber int) (*entity.RoundRecord, error)
}
