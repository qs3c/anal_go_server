package service

import (
	"errors"
	"io"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/oss"
	"github.com/qs3c/anal_go_server/internal/repository"
)

type UserService struct {
	userRepo  *repository.UserRepository
	ossClient *oss.Client
	cfg       *config.Config
}

func NewUserService(userRepo *repository.UserRepository, ossClient *oss.Client, cfg *config.Config) *UserService {
	return &UserService{
		userRepo:  userRepo,
		ossClient: ossClient,
		cfg:       cfg,
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

// UpdateAvatar 更新用户头像 URL
func (s *UserService) UpdateAvatar(userID int64, avatarURL string) error {
	return s.userRepo.UpdateFields(userID, map[string]interface{}{
		"avatar_url": avatarURL,
	})
}

// UploadAvatar 上传用户头像到 OSS
func (s *UserService) UploadAvatar(userID int64, file io.Reader, filename string) (string, error) {
	if s.ossClient == nil {
		return "", errors.New("OSS 客户端未配置")
	}

	// 读取文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// 获取文件扩展名
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg"
	}

	// 上传到 OSS
	avatarURL, err := s.ossClient.UploadAvatar(userID, data, ext)
	if err != nil {
		return "", err
	}

	// 更新用户头像 URL
	if err := s.UpdateAvatar(userID, avatarURL); err != nil {
		return "", err
	}

	return avatarURL, nil
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
