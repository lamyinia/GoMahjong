package service

import (
	"common/database"
	"common/log"
	"context"
	"core/domain/entity"
	"core/domain/repository"
	"time"
	"user/application/dto"
)

// AuthService 应用服务（usecase）
type AuthService struct {
	userRepo repository.UserRepository
	redis    *database.RedisManager
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo repository.UserRepository, redis *database.RedisManager) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		redis:    redis,
	}
}

// Register 注册用户
func (s *AuthService) Register(ctx context.Context, cmd *dto.RegisterCommand) (*dto.RegisterResponse, error) {
	// 1. 验证 SMS 码
	if err := s.verifySMSCode(ctx, cmd.Account, cmd.SMSCode); err != nil {
		log.Warn("SMS 验证失败: %v", err)
		return nil, err
	}

	// 2. 创建用户聚合根（包含验证）
	user, err := entity.NewUser(cmd.Account, cmd.Password, cmd.Platform)
	if err != nil {
		log.Warn("创建用户失败: %v", err)
		return nil, err
	}

	// 3. 保存用户
	if err := s.userRepo.Save(ctx, user); err != nil {
		log.Error("保存用户失败: %v", err)
		return nil, err
	}

	log.Info("用户注册成功: %s", user.ID.Hex())

	// 4. 删除 SMS 码
	s.deleteSMSCode(ctx, cmd.Account)

	return &dto.RegisterResponse{UID: user.ID.Hex()}, nil
}

// Login 登录
func (s *AuthService) Login(ctx context.Context, cmd *dto.LoginCommand) (*dto.LoginResponse, error) {
	// 1. 查询用户
	user, err := s.userRepo.FindByAccount(ctx, cmd.Account)
	if err != nil {
		log.Warn("用户不存在: %v", err)
		return nil, err
	}

	// 2. 验证密码
	if !user.VerifyPassword(cmd.Password) {
		log.Warn("密码错误: %s", cmd.Account)
		return nil, repository.ErrInvalidPassword
	}

	// 3. 更新最后登录时间
	user.UpdateLastLogin()
	if err := s.userRepo.Save(ctx, user); err != nil {
		log.Error("更新用户失败: %v", err)
		return nil, err
	}

	log.Info("用户登录成功: %s", user.ID.Hex())

	return &dto.LoginResponse{UID: user.ID.Hex()}, nil
}

// GetSMSCode 获取 SMS 码
func (s *AuthService) GetSMSCode(ctx context.Context, account string) error {
	// 生成 SMS 码（6 位随机数）
	smsCode := generateSMSCode()

	// 存储到 Redis，5 分钟过期
	key := "sms:" + account
	if err := s.redis.Set(ctx, key, smsCode, 5*time.Minute); err != nil {
		log.Error("存储 SMS 码失败: %v", err)
		return err
	}

	log.Info("SMS 码已发送: %s", account)
	return nil
}

// verifySMSCode 验证 SMS 码
func (s *AuthService) verifySMSCode(ctx context.Context, account, smsCode string) error {
	key := "sms:" + account
	cmd := s.redis.Get(ctx, key)
	code, err := cmd.Result()
	if err != nil {
		log.Warn("获取 SMS 码失败: %v", err)
		return repository.ErrInvalidSMSCode
	}
	if code != smsCode {
		log.Warn("SMS 码错误: %s", account)
		return repository.ErrInvalidSMSCode
	}
	return nil
}

// deleteSMSCode 删除 SMS 码
func (s *AuthService) deleteSMSCode(ctx context.Context, account string) {
	key := "sms:" + account
	s.redis.Del(ctx, key)
}

// generateSMSCode 生成 SMS 码（6 位随机数）
func generateSMSCode() string {
	return "000000" // TODO: 实现真实的随机数生成
}
