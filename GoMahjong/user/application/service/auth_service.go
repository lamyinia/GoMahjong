package service

import (
	"common/database"
	"common/log"
	"context"
	"core/domain/entity"
	"core/domain/repository"
	"core/infrastructure/message/transfer"
	"time"
	"user/application/dto"
)

type AuthService struct {
	userRepo repository.UserRepository
	redis    *database.RedisManager
}

func NewAuthService(userRepo repository.UserRepository, redis *database.RedisManager) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		redis:    redis,
	}
}

func (s *AuthService) Register(ctx context.Context, cmd *dto.RegisterCommand) (*dto.RegisterResponse, error) {
	/*	if err := s.verifySMSCode(ctx, cmd.Account, cmd.SMSCode); err != nil {
		log.Warn("SMS 验证失败: %v", err)
		return nil, err
	}*/
	user, err := entity.NewUser(cmd.Account, cmd.Password)
	if err != nil {
		return nil, err
	}
	t1 := time.Now()
	if err := s.userRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	log.Info("用户注册成功: %s，插入时间 %d ms", user.ID.Hex(), time.Since(t1).Milliseconds())
	//s.deleteSMSCode(ctx, cmd.Account)

	return &dto.RegisterResponse{UID: user.ID.Hex()}, nil
}

func (s *AuthService) Login(ctx context.Context, cmd *dto.LoginCommand) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByAccount(ctx, cmd.Account)
	if err != nil {
		return nil, err
	}
	if !user.VerifyPassword(cmd.Password) {
		return nil, transfer.ErrInvalidPassword
	}
	if err := s.userRepo.UpdateLastLogin(ctx, user); err != nil {
		log.Warn("更新用户失败: %v", err)
	}
	return &dto.LoginResponse{UID: user.ID.Hex()}, nil
}

func (s *AuthService) GetSMSCode(ctx context.Context, account string) error {
	smsCode := generateSMSCode()
	key := "sms:" + account
	if err := s.redis.Set(ctx, key, smsCode, 5*time.Minute); err != nil {
		log.Error("存储 SMS 码失败: %v", err)
		return err
	}
	log.Info("SMS 码已发送: %s", account)
	return nil
}

func (s *AuthService) verifySMSCode(ctx context.Context, account, smsCode string) error {
	key := "sms:" + account
	cmd := s.redis.Get(ctx, key)
	code, err := cmd.Result()
	if err != nil {
		return transfer.ErrInvalidSMSCode
	}
	if code != smsCode {
		return transfer.ErrInvalidSMSCode
	}
	return nil
}

func (s *AuthService) deleteSMSCode(ctx context.Context, account string) {
	key := "sms:" + account
	s.redis.Del(ctx, key)
}

func generateSMSCode() string {
	return "000000" // fixme: 实现真实的随机数生成
}
