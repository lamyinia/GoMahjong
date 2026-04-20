package transfer

import "errors"

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrRouterNotFound  = errors.New("user router not found")

	ErrGameRecordNotFound = errors.New("game record not found")

	ErrMongodb = errors.New("mongodb error happen")
	ErrRedis   = errors.New("redis error happen")
)
