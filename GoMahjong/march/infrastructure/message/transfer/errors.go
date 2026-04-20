package transfer

import (
	"errors"
)

var (
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrInvalidSMSCode       = errors.New("invalid or expired sms code")

	ErrPlayerAlreadyInQueue = errors.New("user already in queue")
	ErrPlayerNotInQueue     = errors.New("user not in queue")
	ErrQueueEmpty           = errors.New("queue is empty")
	ErrNotEnoughPlayers     = errors.New("not enough players in queue")

	ErrRouterNotFound = errors.New("user router not found")

	ErrMongodb = errors.New("mongodb error happen")
	ErrRedis   = errors.New("redis error happen")

	ErrArgument = errors.New("argument error")
	ErrService  = errors.New("service error")
)
