package entity

import (
	"core/domain/vo"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User 是聚合根，代表一个完整的用户业务概念
type User struct {
	ID        primitive.ObjectID
	Account   vo.Account
	Password  vo.Password
	Platform  int32
	Ranking   int // 段位数值（0-299: 见习, 300-599: 雀士, 600-1199: 豪杰, 1200-1799: 雀圣, 1800+: 魂天）
	CreatedAt time.Time
	UpdatedAt time.Time
	LastLogin time.Time
}

// NewUser 工厂方法：创建新用户
func NewUser(account, password string, platform int32) (*User, error) {
	// 验证 Account
	acc, err := vo.NewAccount(account)
	if err != nil {
		return nil, err
	}

	// 验证 Password
	pwd, err := vo.NewPassword(password)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        primitive.NewObjectID(),
		Account:   acc,
		Password:  pwd,
		Platform:  platform,
		Ranking:   0, // 默认段位：见习（0-299）
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

// GetRanking 获取段位枚举（根据数值计算）
func (u *User) GetRanking() vo.RankingType {
	return vo.GetRankingByScore(u.Ranking)
}

// UpdateRanking 更新段位数值（游戏结束后调用，立即生效）
// ranking: 新的段位数值
func (u *User) UpdateRanking(ranking int) {
	if ranking < 0 {
		ranking = 0 // 段位不能为负数
	}
	u.Ranking = ranking
	u.UpdatedAt = time.Now()
}
