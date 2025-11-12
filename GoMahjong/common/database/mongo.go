package database

import (
	"common/config"
	"common/log"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongoManager struct {
	Cli *mongo.Client
	Db  *mongo.Database
}

func NewMongo() *MongoManager {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoConf := config.Conf.DatabaseConf.MongoConf
	clientOptions := options.Client().ApplyURI(mongoConf.Url)
	clientOptions.SetMinPoolSize(uint64(mongoConf.MinPoolSize))
	clientOptions.SetMaxPoolSize(uint64(mongoConf.MaxPoolSize))

	if mongoConf.Username != "" && mongoConf.Password != "" {
		clientOptions.SetAuth(options.Credential{
			Username: mongoConf.Username,
			Password: mongoConf.Password,
		})
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("mongodb 连接错误: %v", err)
		return nil
	}
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("mongodb Ping 错误: %v", err)
		return nil
	}
	m := &MongoManager{
		Cli: client,
	}
	m.Db = m.Cli.Database(config.Conf.DatabaseConf.MongoConf.Db)

	return m
}

func (m *MongoManager) Close() error {
	if m == nil {
		return nil
	}
	return m.Cli.Disconnect(context.TODO())
}
