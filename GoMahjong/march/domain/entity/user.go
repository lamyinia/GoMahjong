package entity

import (
	"march/domain/vo"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID
	Account   vo.Account
	Password  vo.Password
	Ranking   int
	CreatedAt time.Time
	UpdatedAt time.Time
	LastLogin time.Time
}

func NewUser(account, password string) (*User, error) {
	acc, err := vo.NewAccount(account)
	if err != nil {
		return nil, err
	}

	pwd, err := vo.NewPassword(password)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        primitive.NewObjectID(),
		Account:   acc,
		Password:  pwd,
		Ranking:   0,
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
		ranking = 0
	}
	u.Ranking = ranking
	u.UpdatedAt = time.Now()
}
