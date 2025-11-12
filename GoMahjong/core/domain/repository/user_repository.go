package repository

import (
	"context"
	"core/domain/entity"
)

// UserRepository 定义用户仓储接口
type UserRepository interface {
	// Save 保存用户
	Save(ctx context.Context, user *entity.User) error

	// FindByAccount 根据账号查询用户
	FindByAccount(ctx context.Context, account string) (*entity.User, error)

	// FindByID 根据 ID 查询用户
	FindByID(ctx context.Context, id string) (*entity.User, error)
}
