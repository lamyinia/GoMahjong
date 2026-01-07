package grpc

import (
	"context"
	"core/infrastructure/message/transfer"
	"errors"
	"strings"
	"user/application/dto"
	"user/application/service"
	pb "user/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthProvider struct {
	pb.UnimplementedUserServiceServer
	authService *service.AuthService
}

func NewAuthProvider(authService *service.AuthService) *AuthProvider {
	return &AuthProvider{
		authService: authService,
	}
}

func (h *AuthProvider) Register(ctx context.Context, in *pb.RegisterParams) (*pb.RegisterResponse, error) {
	if err := validateRegisterParams(in); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	cmd := &dto.RegisterCommand{
		Account:  in.Account,
		Password: in.Password,
		Platform: in.LoginPlatform,
		SMSCode:  in.SmsCode,
	}
	resp, err := h.authService.Register(ctx, cmd)
	if err != nil {
		return nil, transfer.MapError(err)
	}
	return &pb.RegisterResponse{Uid: resp.UID}, nil
}

func (h *AuthProvider) Login(ctx context.Context, in *pb.LoginParams) (*pb.LoginResponse, error) {
	if err := validateLoginParams(in); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	cmd := &dto.LoginCommand{
		Account:  in.Account,
		Password: in.Password,
		Platform: in.LoginPlatform,
	}
	resp, err := h.authService.Login(ctx, cmd)
	if err != nil {
		return nil, transfer.MapError(err)
	}
	return &pb.LoginResponse{Uid: resp.UID}, nil
}

func (h *AuthProvider) GetSMSCode(ctx context.Context, in *pb.GetSMSCodeParams) (*pb.Empty, error) {
	if err := validateGetSMSCodeParams(in); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := h.authService.GetSMSCode(ctx, in.PhoneNumber); err != nil {
		return nil, transfer.MapError(err)
	}
	return &pb.Empty{}, nil
}

func validateRegisterParams(in *pb.RegisterParams) error {
	if strings.TrimSpace(in.Account) == "" {
		return errors.New("account is required")
	}
	if strings.TrimSpace(in.Password) == "" {
		return errors.New("password is required")
	}
	if len(in.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if strings.TrimSpace(in.SmsCode) == "" {
		return errors.New("sms code is required")
	}
	return nil
}

func validateLoginParams(in *pb.LoginParams) error {
	if strings.TrimSpace(in.Account) == "" {
		return errors.New("account is required")
	}
	if strings.TrimSpace(in.Password) == "" {
		return errors.New("password is required")
	}
	return nil
}

func validateGetSMSCodeParams(in *pb.GetSMSCodeParams) error {
	if strings.TrimSpace(in.PhoneNumber) == "" {
		return errors.New("phone number is required")
	}
	return nil
}
