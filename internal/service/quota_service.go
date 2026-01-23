package service

import (
	"errors"
	"time"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/repository"
)

var (
	ErrQuotaExceeded = errors.New("今日配额已用完")
	ErrDepthExceeded = errors.New("分析深度超过限制")
	ErrModelDenied   = errors.New("当前套餐无法使用该模型")
)

type QuotaService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewQuotaService(userRepo *repository.UserRepository, cfg *config.Config) *QuotaService {
	return &QuotaService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// CheckQuota 检查配额
func (s *QuotaService) CheckQuota(userID int64) (bool, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return false, err
	}

	// 检查是否需要重置
	if user.QuotaResetAt != nil && time.Now().After(*user.QuotaResetAt) {
		if err := s.resetUserQuota(userID); err != nil {
			return false, err
		}
		user, _ = s.userRepo.GetByID(userID)
	}

	return user.QuotaUsedToday < user.DailyQuota, nil
}

// UseQuota 使用配额
func (s *QuotaService) UseQuota(userID int64) error {
	return s.userRepo.IncrementQuotaUsed(userID)
}

// RefundQuota 退还配额
func (s *QuotaService) RefundQuota(userID int64) error {
	return s.userRepo.DecrementQuotaUsed(userID)
}

// CheckDepth 检查深度限制
func (s *QuotaService) CheckDepth(subscriptionLevel string, depth int) error {
	level, ok := s.cfg.Subscription.Levels[subscriptionLevel]
	if !ok {
		level = s.cfg.Subscription.Levels["free"]
	}

	if depth > level.MaxDepth {
		return ErrDepthExceeded
	}
	return nil
}

// CheckModelPermission 检查模型权限
func (s *QuotaService) CheckModelPermission(subscriptionLevel, modelName string) error {
	var modelConfig *config.ModelConfig
	for _, m := range s.cfg.Models {
		if m.Name == modelName {
			modelConfig = &m
			break
		}
	}

	if modelConfig == nil {
		return ErrModelDenied
	}

	// 检查权限等级
	switch subscriptionLevel {
	case "free":
		if modelConfig.RequiredLevel != "free" {
			return ErrModelDenied
		}
	case "basic":
		if modelConfig.RequiredLevel == "pro" {
			return ErrModelDenied
		}
	case "pro":
		// pro 可以使用所有模型
	default:
		if modelConfig.RequiredLevel != "free" {
			return ErrModelDenied
		}
	}

	return nil
}

// GetMaxDepth 获取最大深度
func (s *QuotaService) GetMaxDepth(subscriptionLevel string) int {
	level, ok := s.cfg.Subscription.Levels[subscriptionLevel]
	if !ok {
		return s.cfg.Subscription.Levels["free"].MaxDepth
	}
	return level.MaxDepth
}

func (s *QuotaService) resetUserQuota(userID int64) error {
	nextReset := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
	return s.userRepo.ResetQuota(userID, nextReset)
}

// ResetAllQuotas 重置所有用户配额
func (s *QuotaService) ResetAllQuotas() error {
	nextReset := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
	return s.userRepo.ResetAllQuotas(nextReset)
}

// GetQuotaInfo 获取用户配额信息
func (s *QuotaService) GetQuotaInfo(userID int64) (*dto.QuotaInfo, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	// 检查是否需要重置
	if user.QuotaResetAt != nil && time.Now().After(*user.QuotaResetAt) {
		if err := s.resetUserQuota(userID); err != nil {
			return nil, err
		}
		user, _ = s.userRepo.GetByID(userID)
	}

	dailyRemain := user.DailyQuota - user.QuotaUsedToday
	if dailyRemain < 0 {
		dailyRemain = 0
	}

	info := &dto.QuotaInfo{
		Tier:        user.SubscriptionLevel,
		DailyLimit:  user.DailyQuota,
		DailyUsed:   user.QuotaUsedToday,
		DailyRemain: dailyRemain,
		MaxDepth:    s.GetMaxDepth(user.SubscriptionLevel),
	}

	if user.QuotaResetAt != nil {
		info.ResetAt = user.QuotaResetAt.Format(time.RFC3339)
	}

	return info, nil
}
