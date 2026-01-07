package persistence

import (
	"common/database"
	"common/log"
	"common/utils"
	"context"
	"core/domain/entity"
	"core/domain/repository"
	"core/domain/vo"
	"core/infrastructure/message/transfer"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	mongo *database.MongoManager
	redis *database.RedisManager
}

const userSeqKey = "user:seq:id"

func NewUserRepository(mongo *database.MongoManager, redis *database.RedisManager) repository.UserRepository {
	return &UserRepository{mongo: mongo, redis: redis}
}

func (r *UserRepository) Save(ctx context.Context, user *entity.User) error {
	collection := r.mongo.Db.Collection("users")

	seqID, err := r.redis.Incr(ctx, userSeqKey)
	if err != nil {
		return err
	}
	doc := bson.M{
		"_id":           user.ID,
		"user_no":       seqID,
		"account":       user.Account.String(),
		"password_hash": user.Password.Hash(),
		"ranking":       user.Ranking,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"last_login":    user.LastLogin,
	}

	_, err = collection.InsertOne(ctx, doc)
	if mongo.IsDuplicateKeyError(err) {
		return transfer.ErrAccountAlreadyExists
	}
	if err != nil {
		return transfer.ErrMongodb
	}
	return nil
}

func (r *UserRepository) FindByAccount(ctx context.Context, account string) (*entity.User, error) {
	collection := r.mongo.Db.Collection("users")

	var doc bson.M
	err := collection.FindOne(ctx, bson.M{"account": account}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, transfer.ErrUserNotFound
		}
		log.Error("查询用户失败: %v", err)
		return nil, err
	}

	return r.docToEntity(doc), nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	collection := r.mongo.Db.Collection("users")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, transfer.ErrUserNotFound
	}

	var doc bson.M
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, transfer.ErrUserNotFound
		}
		return nil, err
	}
	return r.docToEntity(doc), nil
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, user *entity.User) error {
	collection := r.mongo.Db.Collection("users")
	filter := bson.M{"_id": user.ID}
	user.UpdateLastLogin()
	update := bson.M{
		"$set": bson.M{
			"last_login": user.LastLogin,
			"updated_at": user.UpdatedAt,
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil || result.MatchedCount == 0 {
		return transfer.ErrMongodb
	}
	return nil
}

func (r *UserRepository) docToEntity(doc bson.M) *entity.User {
	return &entity.User{
		ID:        doc["_id"].(primitive.ObjectID),
		Account:   vo.NewAccountFromString(doc["account"].(string)),
		Password:  vo.NewPasswordFromHash(doc["password_hash"].(string)),
		Ranking:   utils.ToInt(doc["ranking"]),
		CreatedAt: utils.ToTime(doc["created_at"]),
		UpdatedAt: utils.ToTime(doc["updated_at"]),
		LastLogin: utils.ToTime(doc["last_login"]),
	}
}
