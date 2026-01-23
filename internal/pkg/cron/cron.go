package cron

import (
	"log"
	"time"

	"github.com/qs3c/anal_go_server/internal/service"
)

type Service struct {
	quotaService *service.QuotaService
	stopChan     chan struct{}
}

func NewService(quotaService *service.QuotaService) *Service {
	return &Service{
		quotaService: quotaService,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动定时任务
func (s *Service) Start() {
	go s.runDailyQuotaReset()
	log.Println("Cron service started")
}

// Stop 停止定时任务
func (s *Service) Stop() {
	close(s.stopChan)
	log.Println("Cron service stopped")
}

// runDailyQuotaReset 每日配额重置任务
func (s *Service) runDailyQuotaReset() {
	// 计算到下一个 UTC 00:00 的时间
	now := time.Now().UTC()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	durationUntilMidnight := nextMidnight.Sub(now)

	// 首次等待到午夜
	timer := time.NewTimer(durationUntilMidnight)

	for {
		select {
		case <-s.stopChan:
			timer.Stop()
			return
		case <-timer.C:
			s.resetDailyQuotas()
			// 设置下一次执行时间（24小时后）
			timer.Reset(24 * time.Hour)
		}
	}
}

// resetDailyQuotas 重置所有用户的每日配额
func (s *Service) resetDailyQuotas() {
	log.Println("Starting daily quota reset...")

	if err := s.quotaService.ResetAllQuotas(); err != nil {
		log.Printf("Failed to reset daily quotas: %v", err)
		return
	}

	log.Println("Daily quota reset completed")
}

// RunNow 立即执行配额重置（用于测试或手动触发）
func (s *Service) RunNow() error {
	log.Println("Manual quota reset triggered...")
	return s.quotaService.ResetAllQuotas()
}
