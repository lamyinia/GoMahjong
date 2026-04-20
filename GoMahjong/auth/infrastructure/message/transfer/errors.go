package transfer

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrInvalidSMSCode       = errors.New("invalid or expired sms code")

	ErrMongodb = errors.New("mongodb error happen")
	ErrRedis   = errors.New("redis error happen")
)

func MapError(err error) error {
	switch {
	case errors.Is(err, ErrAccountAlreadyExists):
		return status.Error(codes.AlreadyExists, "account already exists")
	case errors.Is(err, ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, ErrInvalidPassword):
		return status.Error(codes.Unauthenticated, "invalid password")
	case errors.Is(err, ErrInvalidSMSCode):
		return status.Error(codes.InvalidArgument, "invalid or expired sms code")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
