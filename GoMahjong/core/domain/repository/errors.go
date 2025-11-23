package repository

import "errors"

var (
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrInvalidSMSCode       = errors.New("invalid or expired sms code")

	// 匹配队列相关错误
	ErrPlayerAlreadyInQueue = errors.New("player already in queue")
	ErrPlayerNotInQueue     = errors.New("player not in queue")
	ErrQueueEmpty           = errors.New("queue is empty")
	ErrNotEnoughPlayers     = errors.New("not enough players in queue")

	// 用户路由相关错误
	ErrRouterNotFound = errors.New("user router not found")
)
