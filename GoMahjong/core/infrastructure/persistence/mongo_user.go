package persistence

import (
	"common/database"
	"common/log"
	"context"
	"core/domain/entity"
	"core/domain/repository"
	"core/domain/vo"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoUserRepository MongoDB 用户仓储实现
type MongoUserRepository struct {
	mongo *database.MongoManager
}

// NewMongoUserRepository 创建 MongoDB 用户仓储
func NewMongoUserRepository(mongo *database.MongoManager) repository.UserRepository {
	return &MongoUserRepository{mongo: mongo}
}

// Save 保存用户，依赖于 mongodb 的 findAndModify 生成 UserID, IdentifyID
func (r *MongoUserRepository) Save(ctx context.Context, user *entity.User) error {
	collection := r.mongo.Db.Collection("users")

	doc := bson.M{
		"_id":           user.ID,
		"account":       user.Account.String(),
		"password_hash": user.Password.Hash(),
		"platform":      user.Platform,
		"ranking":       user.Ranking,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"last_login":    user.LastLogin,
	}

	_, err := collection.InsertOne(ctx, doc)
	if mongo.IsDuplicateKeyError(err) {
		log.Error("账号已存在: %v", err)
		return repository.ErrAccountAlreadyExists
	}
	if err != nil {
		log.Error("插入用户失败: %v", err)
		return err
	}
	return nil
}

// FindByAccount 根据账号查询用户
func (r *MongoUserRepository) FindByAccount(ctx context.Context, account string) (*entity.User, error) {
	collection := r.mongo.Db.Collection("users")

	var doc bson.M
	err := collection.FindOne(ctx, bson.M{"account": account}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, repository.ErrUserNotFound
		}
		log.Error("查询用户失败: %v", err)
		return nil, err
	}

	return r.docToEntity(doc), nil
}

// FindByID 根据 ID 查询用户
func (r *MongoUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	collection := r.mongo.Db.Collection("users")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, repository.ErrUserNotFound
	}

	var doc bson.M
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, repository.ErrUserNotFound
		}
		log.Error("查询用户失败: %v", err)
		return nil, err
	}

	return r.docToEntity(doc), nil
}

// docToEntity 将 MongoDB 文档转换为聚合根
func (r *MongoUserRepository) docToEntity(doc bson.M) *entity.User {
	// 处理 ranking 字段（兼容旧数据，默认 0）
	ranking := 0
	if rankingVal, ok := doc["ranking"]; ok {
		if rankingInt, ok := rankingVal.(int32); ok {
			ranking = int(rankingInt)
		} else if rankingInt, ok := rankingVal.(int64); ok {
			ranking = int(rankingInt)
		} else if rankingInt, ok := rankingVal.(int); ok {
			ranking = rankingInt
		}
	}

	return &entity.User{
		ID:        doc["_id"].(primitive.ObjectID),
		Account:   vo.NewAccountFromString(doc["account"].(string)),
		Password:  vo.NewPasswordFromHash(doc["password_hash"].(string)),
		Platform:  doc["platform"].(int32),
		Ranking:   ranking,
		CreatedAt: doc["created_at"].(time.Time),
		UpdatedAt: doc["updated_at"].(time.Time),
		LastLogin: doc["last_login"].(time.Time),
	}
}
