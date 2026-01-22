package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewUserService(userRepo *repository.UserRepository, cfg *config.Config) *UserService {
	return &UserService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// GetProfile 获取用户详情
func (s *UserService) GetProfile(userID int64) (*dto.UserInfo, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return s.buildUserInfoWithQuota(user), nil
}

// UpdateProfile 更新用户信息
func (s *UserService) UpdateProfile(userID int64, req *dto.UpdateProfileRequest) (*dto.UserInfo, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 检查用户名是否已被占用
	if req.Username != nil && *req.Username != user.Username {
		exists, err := s.userRepo.ExistsByUsername(*req.Username)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUsernameExists
		}
		user.Username = *req.Username
	}

	if req.Bio != nil {
		user.Bio = *req.Bio
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return s.buildUserInfoWithQuota(user), nil
}

// UpdateAvatar 更新用户头像
func (s *UserService) UpdateAvatar(userID int64, avatarURL string) error {
	return s.userRepo.UpdateFields(userID, map[string]interface{}{
		"avatar_url": avatarURL,
	})
}

func (s *UserService) buildUserInfoWithQuota(user *model.User) *dto.UserInfo {
	info := &dto.UserInfo{
		ID:                user.ID,
		Username:          user.Username,
		AvatarURL:         user.AvatarURL,
		Bio:               user.Bio,
		SubscriptionLevel: user.SubscriptionLevel,
		EmailVerified:     user.EmailVerified,
		CreatedAt:         user.CreatedAt.Format(time.RFC3339),
	}

	if user.Email != nil {
		info.Email = *user.Email
	}

	// 添加配额信息
	quotaRemaining := user.DailyQuota - user.QuotaUsedToday
	if quotaRemaining < 0 {
		quotaRemaining = 0
	}

	info.QuotaInfo = &dto.QuotaInfo{
		DailyQuota:     user.DailyQuota,
		QuotaUsedToday: user.QuotaUsedToday,
		QuotaRemaining: quotaRemaining,
	}

	if user.QuotaResetAt != nil {
		info.QuotaInfo.QuotaResetAt = user.QuotaResetAt.Format(time.RFC3339)
	}

	return info
}
