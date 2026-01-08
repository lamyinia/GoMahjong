package persistence

import (
	"common/database"
	"common/log"
	utils "common/utils"
	"context"
	"core/domain/entity"
	"core/domain/repository"
	"core/infrastructure/message/transfer"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type GameRecordRepository struct {
	mongo *database.MongoManager
}

func NewGameRecordRepository(mongo *database.MongoManager) repository.GameRecordRepository {
	return &GameRecordRepository{mongo: mongo}
}

// SaveGameRecord 保存游戏记录（元数据）
func (r *GameRecordRepository) SaveGameRecord(ctx context.Context, record *entity.GameRecord) error {
	collection := r.mongo.Db.Collection("game_records")

	doc := bson.M{
		"_id":          record.ID,
		"room_id":      record.RoomID,
		"game_type":    record.GameType,
		"players":      r.playersToBson(record.Players),
		"start_time":   record.StartTime,
		"end_time":     record.EndTime,
		"duration":     record.Duration,
		"final_result": r.finalResultToBson(record.FinalResult),
		"status":       record.Status,
		"created_at":   record.CreatedAt,
	}

	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Error("保存游戏记录失败: %v", err)
		return transfer.ErrMongodb
	}
	return nil
}

// FindGameRecord 根据ID查找游戏记录
func (r *GameRecordRepository) FindGameRecord(ctx context.Context, recordID primitive.ObjectID) (*entity.GameRecord, error) {
	collection := r.mongo.Db.Collection("game_records")

	var doc bson.M
	err := collection.FindOne(ctx, bson.M{"_id": recordID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, transfer.ErrGameRecordNotFound
		}
		log.Error("查询游戏记录失败: %v", err)
		return nil, err
	}

	return r.docToGameRecord(doc), nil
}

// FindGameRecordsByUser 查找用户参与的游戏记录（分页）
func (r *GameRecordRepository) FindGameRecordsByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.GameRecord, error) {
	collection := r.mongo.Db.Collection("game_records")

	filter := bson.M{"players.user_id": userID}
	opts := options.Find().
		SetSort(bson.M{"start_time": -1}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Error("查询用户游戏记录失败: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []*entity.GameRecord
	if err := cursor.All(ctx, &records); err != nil {
		log.Error("解析游戏记录失败: %v", err)
		return nil, err
	}

	// 需要从 bson.M 转换
	var result []*entity.GameRecord
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result = append(result, r.docToGameRecord(doc))
	}

	return result, nil
}

// FindGameRecordsByRoom 根据房间ID查找游戏记录
func (r *GameRecordRepository) FindGameRecordsByRoom(ctx context.Context, roomID string) (*entity.GameRecord, error) {
	collection := r.mongo.Db.Collection("game_records")

	var doc bson.M
	err := collection.FindOne(ctx, bson.M{"room_id": roomID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, transfer.ErrGameRecordNotFound
		}
		log.Error("查询游戏记录失败: %v", err)
		return nil, err
	}

	return r.docToGameRecord(doc), nil
}

// SaveRoundRecord 保存局记录（每局一个文档）
func (r *GameRecordRepository) SaveRoundRecord(ctx context.Context, round *entity.RoundRecord) error {
	collection := r.mongo.Db.Collection("round_records")

	doc := bson.M{
		"_id":            round.ID,
		"game_record_id": round.GameRecordID,
		"round_number":   round.RoundNumber,
		"round_wind":     round.RoundWind,
		"dealer_index":   round.DealerIndex,
		"honba":          round.Honba,
		"events":         r.eventsToBson(round.Events),
		"round_result":   r.roundResultToBson(round.RoundResult),
		"start_time":     round.StartTime,
		"end_time":       round.EndTime,
		"duration":       round.Duration,
		"created_at":     round.CreatedAt,
	}

	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Error("保存局记录失败: %v", err)
		return transfer.ErrMongodb
	}
	return nil
}

// SaveRoundRecords 批量保存局记录（使用 MongoDB InsertMany）
func (r *GameRecordRepository) SaveRoundRecords(ctx context.Context, rounds []*entity.RoundRecord) error {
	if len(rounds) == 0 {
		return nil
	}

	collection := r.mongo.Db.Collection("round_records")

	docs := make([]any, 0, len(rounds))
	for _, round := range rounds {
		if round == nil {
			continue
		}
		doc := bson.M{
			"_id":            round.ID,
			"game_record_id": round.GameRecordID,
			"round_number":   round.RoundNumber,
			"round_wind":     round.RoundWind,
			"dealer_index":   round.DealerIndex,
			"honba":          round.Honba,
			"events":         r.eventsToBson(round.Events),
			"round_result":   r.roundResultToBson(round.RoundResult),
			"start_time":     round.StartTime,
			"end_time":       round.EndTime,
			"duration":       round.Duration,
			"created_at":     round.CreatedAt,
		}
		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		return nil
	}

	_, err := collection.InsertMany(ctx, docs)
	if err != nil {
		log.Error("批量保存局记录失败: %v", err)
		return transfer.ErrMongodb
	}

	log.Info("批量保存局记录成功: count=%d", len(docs))
	return nil
}

// FindRoundRecords 查找游戏的所有局记录（按局数排序）
func (r *GameRecordRepository) FindRoundRecords(ctx context.Context, gameRecordID primitive.ObjectID) ([]*entity.RoundRecord, error) {
	collection := r.mongo.Db.Collection("round_records")

	filter := bson.M{"game_record_id": gameRecordID}
	opts := options.Find().SetSort(bson.M{"round_number": 1})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Error("查询局记录失败: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []*entity.RoundRecord
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result = append(result, r.docToRoundRecord(doc))
	}

	return result, nil
}

// FindRoundRecord 查找指定局数的记录
func (r *GameRecordRepository) FindRoundRecord(ctx context.Context, gameRecordID primitive.ObjectID, roundNumber int) (*entity.RoundRecord, error) {
	collection := r.mongo.Db.Collection("round_records")

	filter := bson.M{
		"game_record_id": gameRecordID,
		"round_number":   roundNumber,
	}

	var doc bson.M
	err := collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, transfer.ErrGameRecordNotFound
		}
		log.Error("查询局记录失败: %v", err)
		return nil, err
	}

	return r.docToRoundRecord(doc), nil
}

// ==================== 转换辅助方法 ====================

func (r *GameRecordRepository) playersToBson(players []entity.PlayerInfo) []bson.M {
	result := make([]bson.M, len(players))
	for i, p := range players {
		result[i] = bson.M{
			"user_id":    p.UserID,
			"seat_index": p.SeatIndex,
			"nickname":   p.Nickname,
		}
	}
	return result
}

func (r *GameRecordRepository) finalResultToBson(result *entity.GameFinalResult) bson.M {
	if result == nil {
		return nil
	}
	rankings := make([]bson.M, len(result.Rankings))
	for i, r := range result.Rankings {
		rankings[i] = bson.M{
			"seat_index": r.SeatIndex,
			"user_id":    r.UserID,
			"points":     r.Points,
			"rank":       r.Rank,
		}
	}
	return bson.M{
		"rankings": rankings,
		"points":   result.Points,
	}
}

func (r *GameRecordRepository) eventsToBson(events []entity.RoundEvent) []bson.M {
	result := make([]bson.M, len(events))
	for i, e := range events {
		result[i] = bson.M{
			"sequence":   e.Sequence,
			"event_type": e.EventType,
			"timestamp":  e.Timestamp,
			"seat_index": e.SeatIndex,
			"data":       e.Data,
		}
	}
	return result
}

func (r *GameRecordRepository) roundResultToBson(result *entity.RoundResult) bson.M {
	if result == nil {
		return nil
	}
	claims := make([]bson.M, len(result.Claims))
	for i, c := range result.Claims {
		claims[i] = bson.M{
			"winner_seat": c.WinnerSeat,
			"loser_seat":  c.LoserSeat,
			"win_tile": bson.M{
				"type": c.WinTile.Type,
				"id":   c.WinTile.ID,
			},
			"han":    c.Han,
			"fu":     c.Fu,
			"yaku":   c.Yaku,
			"points": c.Points,
		}
	}
	return bson.M{
		"end_type":    result.EndType,
		"claims":      claims,
		"delta":       result.Delta,
		"points":      result.Points,
		"reason":      result.Reason,
		"next_dealer": result.NextDealer,
	}
}

func (r *GameRecordRepository) docToGameRecord(doc bson.M) *entity.GameRecord {
	playersDoc := doc["players"].(bson.A)
	players := make([]entity.PlayerInfo, len(playersDoc))
	for i, p := range playersDoc {
		pMap := p.(bson.M)
		players[i] = entity.PlayerInfo{
			UserID:    pMap["user_id"].(string),
			SeatIndex: utils.ToInt(pMap["seat_index"]),
			Nickname:  utils.ToString(pMap["nickname"]),
		}
	}

	var finalResult *entity.GameFinalResult
	if doc["final_result"] != nil {
		frDoc := doc["final_result"].(bson.M)
		rankingsDoc := frDoc["rankings"].(bson.A)
		rankings := make([]entity.PlayerRanking, len(rankingsDoc))
		for i, r := range rankingsDoc {
			rMap := r.(bson.M)
			rankings[i] = entity.PlayerRanking{
				SeatIndex: utils.ToInt(rMap["seat_index"]),
				UserID:    rMap["user_id"].(string),
				Points:    utils.ToInt(rMap["points"]),
				Rank:      utils.ToInt(rMap["rank"]),
			}
		}
		finalResult = &entity.GameFinalResult{
			Rankings: rankings,
			Points:   utils.ToIntArray(frDoc["points"]),
		}
	}

	return &entity.GameRecord{
		ID:          doc["_id"].(primitive.ObjectID),
		RoomID:      doc["room_id"].(string),
		GameType:    doc["game_type"].(string),
		Players:     players,
		StartTime:   utils.ToTime(doc["start_time"]),
		EndTime:     utils.ToTime(doc["end_time"]),
		Duration:    utils.ToInt(doc["duration"]),
		FinalResult: finalResult,
		Status:      doc["status"].(string),
		CreatedAt:   utils.ToTime(doc["created_at"]),
	}
}

func (r *GameRecordRepository) docToRoundRecord(doc bson.M) *entity.RoundRecord {
	eventsDoc := doc["events"].(bson.A)
	events := make([]entity.RoundEvent, len(eventsDoc))
	for i, e := range eventsDoc {
		eMap := e.(bson.M)
		events[i] = entity.RoundEvent{
			Sequence:  utils.ToInt(eMap["sequence"]),
			EventType: eMap["event_type"].(string),
			Timestamp: utils.ToTime(eMap["timestamp"]),
			SeatIndex: utils.ToInt(eMap["seat_index"]),
			Data:      eMap["data"].(map[string]interface{}),
		}
	}

	var roundResult *entity.RoundResult
	if doc["round_result"] != nil {
		rrDoc := doc["round_result"].(bson.M)
		claimsDoc := rrDoc["claims"].(bson.A)
		claims := make([]entity.HuClaim, len(claimsDoc))
		for i, c := range claimsDoc {
			cMap := c.(bson.M)
			winTileMap := cMap["win_tile"].(bson.M)
			claims[i] = entity.HuClaim{
				WinnerSeat: utils.ToInt(cMap["winner_seat"]),
				LoserSeat:  utils.ToInt(cMap["loser_seat"]),
				WinTile: entity.Tile{
					Type: utils.ToInt(winTileMap["type"]),
					ID:   utils.ToInt(winTileMap["id"]),
				},
				Han:    utils.ToInt(cMap["han"]),
				Fu:     utils.ToInt(cMap["fu"]),
				Yaku:   utils.ToStringArray(cMap["yaku"]),
				Points: utils.ToInt(cMap["points"]),
			}
		}
		roundResult = &entity.RoundResult{
			EndType:    rrDoc["end_type"].(string),
			Claims:     claims,
			Delta:      utils.ToIntArray(rrDoc["delta"]),
			Points:     utils.ToIntArray(rrDoc["points"]),
			Reason:     utils.ToString(rrDoc["reason"]),
			NextDealer: utils.ToInt(rrDoc["next_dealer"]),
		}
	}

	return &entity.RoundRecord{
		ID:           doc["_id"].(primitive.ObjectID),
		GameRecordID: doc["game_record_id"].(primitive.ObjectID),
		RoundNumber:  utils.ToInt(doc["round_number"]),
		RoundWind:    doc["round_wind"].(string),
		DealerIndex:  utils.ToInt(doc["dealer_index"]),
		Honba:        utils.ToInt(doc["honba"]),
		Events:       events,
		RoundResult:  roundResult,
		StartTime:    utils.ToTime(doc["start_time"]),
		EndTime:      utils.ToTime(doc["end_time"]),
		Duration:     utils.ToInt(doc["duration"]),
		CreatedAt:    utils.ToTime(doc["created_at"]),
	}
}
