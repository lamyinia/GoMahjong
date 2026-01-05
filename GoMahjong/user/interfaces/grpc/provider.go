package grpc

import (
	"common/log"
	"context"
	"core/domain/repository"
	"errors"
	"strings"
	"user/application/dto"
	"user/application/service"
	pb "user/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthProvider gRPC 认证处理器
type AuthProvider struct {
	pb.UnimplementedUserServiceServer
	authService *service.AuthService
}

// NewAuthProvider 创建认证处理器
func NewAuthProvider(authService *service.AuthService) *AuthProvider {
	return &AuthProvider{
		authService: authService,
	}
}

// Register 注册用户
func (h *AuthProvider) Register(ctx context.Context, in *pb.RegisterParams) (*pb.RegisterResponse, error) {
	// 1. 参数验证
	if err := validateRegisterParams(in); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 2. 转换为 DTO
	cmd := &dto.RegisterCommand{
		Account:  in.Account,
		Password: in.Password,
		Platform: in.LoginPlatform,
		SMSCode:  in.SmsCode,
	}

	// 3. 调用应用服务
	resp, err := h.authService.Register(ctx, cmd)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.RegisterResponse{Uid: resp.UID}, nil
}

// Login 登录
func (h *AuthProvider) Login(ctx context.Context, in *pb.LoginParams) (*pb.LoginResponse, error) {
	// 1. 参数验证
	if err := validateLoginParams(in); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 2. 转换为 DTO
	cmd := &dto.LoginCommand{
		Account:  in.Account,
		Password: in.Password,
		Platform: in.LoginPlatform,
	}

	// 3. 调用应用服务
	resp, err := h.authService.Login(ctx, cmd)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.LoginResponse{Uid: resp.UID}, nil
}

// GetSMSCode 获取 SMS 码
func (h *AuthProvider) GetSMSCode(ctx context.Context, in *pb.GetSMSCodeParams) (*pb.Empty, error) {
	// 1. 参数验证
	if err := validateGetSMSCodeParams(in); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 2. 调用应用服务
	if err := h.authService.GetSMSCode(ctx, in.PhoneNumber); err != nil {
		return nil, mapError(err)
	}

	return &pb.Empty{}, nil
}

// validateRegisterParams 验证注册参数
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

// validateLoginParams 验证登录参数
func validateLoginParams(in *pb.LoginParams) error {
	if strings.TrimSpace(in.Account) == "" {
		return errors.New("account is required")
	}
	if strings.TrimSpace(in.Password) == "" {
		return errors.New("password is required")
	}
	return nil
}

// validateGetSMSCodeParams 验证获取 SMS 码参数
func validateGetSMSCodeParams(in *pb.GetSMSCodeParams) error {
	if strings.TrimSpace(in.PhoneNumber) == "" {
		return errors.New("phone number is required")
	}
	return nil
}

// mapError 将应用层错误映射为 gRPC 错误
func mapError(err error) error {
	switch {
	case errors.Is(err, repository.ErrAccountAlreadyExists):
		return status.Error(codes.AlreadyExists, "account already exists")
	case errors.Is(err, repository.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, repository.ErrInvalidPassword):
		return status.Error(codes.Unauthenticated, "invalid password")
	case errors.Is(err, repository.ErrInvalidSMSCode):
		return status.Error(codes.InvalidArgument, "invalid or expired sms code")
	default:
		log.Error("未知错误: %v", err)
		return status.Error(codes.Internal, "internal error")
	}
}
