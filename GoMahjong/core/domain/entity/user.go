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
	Ranking   int // 段位数值（0-299: 见习, 300-599: 雀士, 600-1199: 豪杰, 1200-1799: 雀圣, 1800+: 魂天）
	CreatedAt time.Time
	UpdatedAt time.Time
	LastLogin time.Time
}

// NewUser 工厂方法：创建新用户
func NewUser(account, password string) (*User, error) {
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
		Ranking:   0, // 默认段位：见习（0-299）
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastLogin: time.Now(),
	}, nil
}

func (u *User) VerifyPassword(plainPassword string) bool {
	return u.Password.Verify(plainPassword)
}

func (u *User) UpdateLastLogin() {
	u.LastLogin = time.Now()
	u.UpdatedAt = time.Now()
}

func (u *User) GetRanking() vo.RankingType {
	return vo.GetRankingByScore(u.Ranking)
}

func (u *User) UpdateRanking(ranking int) {
	if ranking < 0 {
		ranking = 0 // 段位不能为负数
	}
	u.Ranking = ranking
	u.UpdatedAt = time.Now()
}
