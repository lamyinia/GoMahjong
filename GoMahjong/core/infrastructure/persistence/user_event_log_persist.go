package persistence

import (
	"common/database"
	"common/log"
	utils "common/utils"
	"context"
	"core/domain/entity"
	"core/domain/repository"
	"core/infrastructure/message/transfer"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// TTL 索引过期时间：1年 = 365 * 24 * 3600 秒
	ttlExpireSeconds = 365 * 24 * 3600
	// 异步写入缓冲区大小
	asyncBufferSize = 1000
	// 批量写入批次大小
	batchSize = 100
	// 批量写入超时时间
	batchFlushInterval = 5 * time.Second
)

type UserEventLogRepository struct {
	mongo     *database.MongoManager
	asyncChan chan *entity.UserEventLog
	batchChan chan []*entity.UserEventLog
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	once      sync.Once
	closed    bool
	mu        sync.Mutex
}

// NewUserEventLogRepository 创建用户事件日志仓储
func NewUserEventLogRepository(mongo *database.MongoManager) repository.UserEventLogRepository {
	ctx, cancel := context.WithCancel(context.Background())
	repo := &UserEventLogRepository{
		mongo:     mongo,
		asyncChan: make(chan *entity.UserEventLog, asyncBufferSize),
		batchChan: make(chan []*entity.UserEventLog, 100),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 初始化索引（包括TTL索引）
	repo.initIndexes()

	// 启动异步写入协程
	repo.wg.Add(2)
	go repo.asyncWriteLoop()
	go repo.batchWriteLoop()

	return repo
}

// initIndexes 初始化索引（包括TTL索引）
func (r *UserEventLogRepository) initIndexes() {
	collection := r.mongo.Db.Collection("user_event_logs")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建索引
	indexes := []mongo.IndexModel{
		// TTL 索引：基于 created_at 字段，1年后自动删除
		{
			Keys:    bson.D{{Key: "created_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(ttlExpireSeconds),
		},
		// 用户ID + 时间戳索引（用于按用户查询）
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		// 事件类型 + 时间戳索引（用于按类型查询）
		{
			Keys: bson.D{
				{Key: "event_type", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		// 时间戳索引（用于时间范围查询）
		{
			Keys: bson.D{{Key: "timestamp", Value: -1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		log.Error("创建用户事件日志索引失败: %v", err)
	} else {
		log.Info("用户事件日志索引创建成功")
	}
}

// SaveEventLog 保存单个事件日志（同步）
func (r *UserEventLogRepository) SaveEventLog(ctx context.Context, eventLog *entity.UserEventLog) error {
	collection := r.mongo.Db.Collection("user_event_logs")
	doc := r.logToBson(eventLog)

	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Error("保存用户事件日志失败: %v", err)
		return transfer.ErrMongodb
	}
	return nil
}

// SaveEventLogs 批量保存事件日志（同步）
func (r *UserEventLogRepository) SaveEventLogs(ctx context.Context, logs []*entity.UserEventLog) error {
	if len(logs) == 0 {
		return nil
	}

	collection := r.mongo.Db.Collection("user_event_logs")
	docs := make([]interface{}, 0, len(logs))
	for _, log := range logs {
		if log == nil {
			continue
		}
		docs = append(docs, r.logToBson(log))
	}

	if len(docs) == 0 {
		return nil
	}

	_, err := collection.InsertMany(ctx, docs)
	if err != nil {
		log.Error("批量保存用户事件日志失败: %v", err)
		return transfer.ErrMongodb
	}

	log.Info("批量保存用户事件日志成功: count=%d", len(docs))
	return nil
}

// SaveEventLogAsync 异步保存单个事件日志（非阻塞）
func (r *UserEventLogRepository) SaveEventLogAsync(eventLog *entity.UserEventLog) {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()

	if closed {
		log.Warn("用户事件日志仓储已关闭，无法异步写入")
		return
	}

	select {
	case r.asyncChan <- eventLog:
		// 成功发送到通道
	default:
		// 通道已满，记录警告但不阻塞
		log.Warn("用户事件日志异步通道已满，丢弃日志: userID=%s, eventType=%s", eventLog.UserID, eventLog.EventType)
	}
}

// SaveEventLogsAsync 异步批量保存事件日志（非阻塞）
func (r *UserEventLogRepository) SaveEventLogsAsync(logs []*entity.UserEventLog) {
	if len(logs) == 0 {
		return
	}

	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()

	if closed {
		log.Warn("用户事件日志仓储已关闭，无法异步写入")
		return
	}

	// 过滤掉 nil
	validLogs := make([]*entity.UserEventLog, 0, len(logs))
	for _, log := range logs {
		if log != nil {
			validLogs = append(validLogs, log)
		}
	}

	if len(validLogs) == 0 {
		return
	}

	select {
	case r.batchChan <- validLogs:
		// 成功发送到通道
	default:
		// 通道已满，记录警告但不阻塞
		log.Warn("用户事件日志批量通道已满，丢弃日志: count=%d", len(validLogs))
	}
}

// asyncWriteLoop 异步写入循环（处理单个日志）
func (r *UserEventLogRepository) asyncWriteLoop() {
	defer r.wg.Done()

	batch := make([]*entity.UserEventLog, 0, batchSize)
	ticker := time.NewTicker(batchFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case eventLog, ok := <-r.asyncChan:
			if !ok {
				// 通道已关闭，写入剩余批次
				if len(batch) > 0 {
					r.flushBatch(batch)
				}
				return
			}
			batch = append(batch, eventLog)
			if len(batch) >= batchSize {
				r.flushBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			// 定时刷新批次
			if len(batch) > 0 {
				r.flushBatch(batch)
				batch = batch[:0]
			}
		case <-r.ctx.Done():
			// 上下文取消，写入剩余批次
			if len(batch) > 0 {
				r.flushBatch(batch)
			}
			return
		}
	}
}

// batchWriteLoop 批量写入循环（处理批量日志）
func (r *UserEventLogRepository) batchWriteLoop() {
	defer r.wg.Done()

	for {
		select {
		case logs, ok := <-r.batchChan:
			if !ok {
				return
			}
			r.flushBatch(logs)
		case <-r.ctx.Done():
			return
		}
	}
}

// flushBatch 刷新批次到数据库
func (r *UserEventLogRepository) flushBatch(logs []*entity.UserEventLog) {
	if len(logs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := r.SaveEventLogs(ctx, logs); err != nil {
		log.Error("异步批量写入用户事件日志失败: %v, count=%d", err, len(logs))
	}
}

// FindEventLogsByUser 查找用户的事件日志（分页）
func (r *UserEventLogRepository) FindEventLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.UserEventLog, error) {
	collection := r.mongo.Db.Collection("user_event_logs")

	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Warn("查询用户事件日志失败: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []*entity.UserEventLog
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result = append(result, r.docToLog(doc))
	}

	return result, nil
}

// FindEventLogsByUserAndType 查找用户指定类型的事件日志（分页）
func (r *UserEventLogRepository) FindEventLogsByUserAndType(ctx context.Context, userID, eventType string, limit, offset int) ([]*entity.UserEventLog, error) {
	collection := r.mongo.Db.Collection("user_event_logs")

	filter := bson.M{
		"user_id":    userID,
		"event_type": eventType,
	}
	opts := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Error("查询用户事件日志失败: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []*entity.UserEventLog
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result = append(result, r.docToLog(doc))
	}

	return result, nil
}

// FindEventLogsByTimeRange 按时间范围查找事件日志（分页）
func (r *UserEventLogRepository) FindEventLogsByTimeRange(ctx context.Context, startTime, endTime int64, limit, offset int) ([]*entity.UserEventLog, error) {
	collection := r.mongo.Db.Collection("user_event_logs")

	filter := bson.M{
		"timestamp": bson.M{
			"$gte": time.Unix(startTime, 0),
			"$lte": time.Unix(endTime, 0),
		},
	}
	opts := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Error("查询用户事件日志失败: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []*entity.UserEventLog
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result = append(result, r.docToLog(doc))
	}

	return result, nil
}

// FindEventLogsByUserAndTimeRange 查找用户在指定时间范围内的事件日志（分页）
func (r *UserEventLogRepository) FindEventLogsByUserAndTimeRange(ctx context.Context, userID string, startTime, endTime int64, limit, offset int) ([]*entity.UserEventLog, error) {
	collection := r.mongo.Db.Collection("user_event_logs")

	filter := bson.M{
		"user_id": userID,
		"timestamp": bson.M{
			"$gte": time.Unix(startTime, 0),
			"$lte": time.Unix(endTime, 0),
		},
	}
	opts := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Error("查询用户事件日志失败: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []*entity.UserEventLog
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result = append(result, r.docToLog(doc))
	}

	return result, nil
}

// Close 关闭异步写入通道（优雅关闭）
func (r *UserEventLogRepository) Close() error {
	var err error
	r.once.Do(func() {
		r.mu.Lock()
		r.closed = true
		r.mu.Unlock()

		// 取消上下文
		r.cancel()

		// 关闭通道
		close(r.asyncChan)
		close(r.batchChan)

		// 等待所有协程完成
		r.wg.Wait()

		log.Info("用户事件日志仓储已关闭")
	})
	return err
}

// ==================== 转换辅助方法 ====================

// logToBson 将实体转换为 BSON 文档
func (r *UserEventLogRepository) logToBson(eventLog *entity.UserEventLog) bson.M {
	doc := bson.M{
		"_id":        eventLog.ID,
		"user_id":    eventLog.UserID,
		"event_type": eventLog.EventType,
		"timestamp":  eventLog.Timestamp,
		"created_at": eventLog.CreatedAt,
	}

	if eventLog.IP != "" {
		doc["ip"] = eventLog.IP
	}
	if eventLog.UserAgent != "" {
		doc["user_agent"] = eventLog.UserAgent
	}
	if len(eventLog.Metadata) > 0 {
		doc["metadata"] = eventLog.Metadata
	}

	return doc
}

// docToLog 将 BSON 文档转换为实体
func (r *UserEventLogRepository) docToLog(doc bson.M) *entity.UserEventLog {
	log := &entity.UserEventLog{
		ID:        doc["_id"].(primitive.ObjectID),
		UserID:    doc["user_id"].(string),
		EventType: doc["event_type"].(string),
		Timestamp: utils.ToTime(doc["timestamp"]),
		CreatedAt: utils.ToTime(doc["created_at"]),
	}

	if ip, ok := doc["ip"]; ok && ip != nil {
		log.IP = utils.ToString(ip)
	}
	if userAgent, ok := doc["user_agent"]; ok && userAgent != nil {
		log.UserAgent = utils.ToString(userAgent)
	}
	if metadata, ok := doc["metadata"]; ok && metadata != nil {
		if metaMap, ok := metadata.(bson.M); ok {
			log.Metadata = make(map[string]interface{})
			for k, v := range metaMap {
				log.Metadata[k] = v
			}
		}
	}

	return log
}
