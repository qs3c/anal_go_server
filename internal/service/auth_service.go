package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/jwt"
	"github.com/qs3c/anal_go_server/internal/repository"
)

var (
	ErrEmailExists        = errors.New("邮箱已被注册")
	ErrUsernameExists     = errors.New("用户名已被使用")
	ErrInvalidCredentials = errors.New("邮箱或密码错误")
	ErrEmailNotVerified   = errors.New("邮箱尚未验证")
	ErrInvalidVerifyCode  = errors.New("验证码无效或已过期")
	ErrUserNotFound       = errors.New("用户不存在")
)

type AuthService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Register 用户注册
func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// 检查邮箱是否存在
	exists, err := s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}

	// 检查用户名是否存在
	exists, err = s.userRepo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 生成验证码
	verifyCode, err := generateRandomCode(32)
	if err != nil {
		return nil, err
	}

	passwordStr := string(hashedPassword)
	expiresAt := time.Now().Add(24 * time.Hour)
	resetAt := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)

	user := &model.User{
		Username:              req.Username,
		Email:                 &req.Email,
		PasswordHash:          &passwordStr,
		SubscriptionLevel:     "free",
		DailyQuota:            s.cfg.Subscription.Levels["free"].DailyQuota,
		QuotaResetAt:          &resetAt,
		VerificationCode:      &verifyCode,
		VerificationExpiresAt: &expiresAt,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// TODO: 发送验证邮件

	return &dto.RegisterResponse{
		UserID: user.ID,
	}, nil
}

// Login 用户登录
func (s *AuthService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// 检查邮箱是否验证
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	// 验证密码
	if user.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 生成 Token
	token, err := jwt.GenerateToken(user.ID, s.cfg.JWT.Secret, s.cfg.JWT.ExpireHours)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: token,
		User:  s.buildUserInfo(user),
	}, nil
}

// VerifyEmail 验证邮箱
func (s *AuthService) VerifyEmail(code string) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByVerificationCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidVerifyCode
		}
		return nil, err
	}

	// 检查验证码是否过期
	if user.VerificationExpiresAt == nil || time.Now().After(*user.VerificationExpiresAt) {
		return nil, ErrInvalidVerifyCode
	}

	// 更新用户状态
	user.EmailVerified = true
	user.VerificationCode = nil
	user.VerificationExpiresAt = nil
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	// 生成 Token
	token, err := jwt.GenerateToken(user.ID, s.cfg.JWT.Secret, s.cfg.JWT.ExpireHours)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: token,
		User:  s.buildUserInfo(user),
	}, nil
}

// GetUserByID 根据 ID 获取用户
func (s *AuthService) GetUserByID(id int64) (*model.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *AuthService) buildUserInfo(user *model.User) *dto.UserInfo {
	info := &dto.UserInfo{
		ID:                user.ID,
		Username:          user.Username,
		AvatarURL:         user.AvatarURL,
		Bio:               user.Bio,
		SubscriptionLevel: user.SubscriptionLevel,
		EmailVerified:     user.EmailVerified,
	}

	if user.Email != nil {
		info.Email = *user.Email
	}

	return info
}

func generateRandomCode(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
