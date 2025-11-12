package entity

import (
	"core/domain/value_object"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// User 是聚合根，代表一个完整的用户业务概念
type User struct {
	ID        primitive.ObjectID
	Account   value_object.Account
	Password  value_object.Password
	Platform  int32
	CreatedAt time.Time
	UpdatedAt time.Time
	LastLogin time.Time
}

// NewUser 工厂方法：创建新用户
func NewUser(account, password string, platform int32) (*User, error) {
	// 验证 Account
	acc, err := value_object.NewAccount(account)
	if err != nil {
		return nil, err
	}

	// 验证 Password
	pwd, err := value_object.NewPassword(password)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        primitive.NewObjectID(),
		Account:   acc,
		Password:  pwd,
		Platform:  platform,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastLogin: time.Now(),
	}, nil
}

// VerifyPassword 业务方法：验证密码
func (u *User) VerifyPassword(plainPassword string) bool {
	return u.Password.Verify(plainPassword)
}

// UpdateLastLogin 业务方法：更新最后登录时间
func (u *User) UpdateLastLogin() {
	u.LastLogin = time.Now()
	u.UpdatedAt = time.Now()
}
