package repository

import (
	"context"
	"core/domain/entity"
)

// UserRepository 定义用户仓储接口
type UserRepository interface {
	Save(ctx context.Context, user *entity.User) error
	FindByAccount(ctx context.Context, account string) (*entity.User, error)
	FindByID(ctx context.Context, id string) (*entity.User, error)
	UpdateLastLogin(ctx context.Context, user *entity.User) error
}
