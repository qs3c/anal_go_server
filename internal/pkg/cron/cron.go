package cron

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/service"
)

type Service struct {
	quotaService   *service.QuotaService
	analysisRepo   *repository.AnalysisRepository
	uploadTempDir  string
	expireHours    int
	stopChan       chan struct{}
}

func NewService(
	quotaService *service.QuotaService,
	analysisRepo *repository.AnalysisRepository,
	uploadTempDir string,
	expireHours int,
) *Service {
	return &Service{
		quotaService:  quotaService,
		analysisRepo:  analysisRepo,
		uploadTempDir: uploadTempDir,
		expireHours:   expireHours,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动定时任务
func (s *Service) Start() {
	go s.runDailyQuotaReset()
	go s.runCleanup()
	log.Println("Cron service started (quota reset + temp cleanup)")
}

// Stop 停止定时任务
func (s *Service) Stop() {
	close(s.stopChan)
	log.Println("Cron service stopped")
}

// runDailyQuotaReset 每日配额重置任务
func (s *Service) runDailyQuotaReset() {
	now := time.Now().UTC()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	timer := time.NewTimer(nextMidnight.Sub(now))

	for {
		select {
		case <-s.stopChan:
			timer.Stop()
			return
		case <-timer.C:
			s.resetDailyQuotas()
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

// runCleanup 每小时执行一次全量清理
func (s *Service) runCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.cleanupAll()
		}
	}
}

// cleanupAll 执行所有清理任务
func (s *Service) cleanupAll() {
	expireHours := s.expireHours
	if expireHours <= 0 {
		expireHours = 1
	}
	expireDuration := time.Duration(expireHours) * time.Hour

	c1 := s.cleanupUploadDirs(expireDuration)
	c2 := s.cleanupCloneDirs(expireDuration)
	c3 := s.cleanupMigratedDiagrams()

	total := c1 + c2 + c3
	if total > 0 {
		log.Printf("Cleanup summary: uploads=%d, clones=%d, diagrams=%d", c1, c2, c3)
	}
}

// cleanupUploadDirs 清理过期的用户上传临时目录（/tmp/uploads/<upload_id>/）
func (s *Service) cleanupUploadDirs(expireDuration time.Duration) int {
	if s.uploadTempDir == "" {
		return 0
	}

	entries, err := os.ReadDir(s.uploadTempDir)
	if err != nil {
		log.Printf("Cleanup uploads: failed to read dir %s: %v", s.uploadTempDir, err)
		return 0
	}

	cleaned := 0
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "diagrams" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > expireDuration {
			dirPath := filepath.Join(s.uploadTempDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				log.Printf("Cleanup uploads: failed to remove %s: %v", dirPath, err)
			} else {
				cleaned++
			}
		}
	}
	return cleaned
}

// cleanupCloneDirs 清理过期的 git clone 临时目录（/tmp/analysis_*）
func (s *Service) cleanupCloneDirs(expireDuration time.Duration) int {
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		log.Printf("Cleanup clones: failed to read dir %s: %v", tmpDir, err)
		return 0
	}

	cleaned := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "analysis_") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > expireDuration {
			dirPath := filepath.Join(tmpDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				log.Printf("Cleanup clones: failed to remove %s: %v", dirPath, err)
			} else {
				cleaned++
			}
		}
	}
	return cleaned
}

// cleanupMigratedDiagrams 清理已迁移到 OSS 的本地 diagram 文件
func (s *Service) cleanupMigratedDiagrams() int {
	if s.uploadTempDir == "" || s.analysisRepo == nil {
		return 0
	}

	diagramDir := filepath.Join(s.uploadTempDir, "diagrams")
	if _, err := os.Stat(diagramDir); os.IsNotExist(err) {
		return 0
	}

	// 查询已迁移到 OSS 的分析 ID
	migratedIDs, err := s.analysisRepo.ListOSSMigratedDiagramIDs()
	if err != nil {
		log.Printf("Cleanup diagrams: failed to query migrated IDs: %v", err)
		return 0
	}

	if len(migratedIDs) == 0 {
		return 0
	}

	cleaned := 0
	for _, id := range migratedIDs {
		localPath := filepath.Join(diagramDir, fmt.Sprintf("%d.json", id))
		if err := os.Remove(localPath); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("Cleanup diagrams: failed to remove %s: %v", localPath, err)
			}
		} else {
			cleaned++
		}
	}
	return cleaned
}

// RunNow 立即执行配额重置（用于测试或手动触发）
func (s *Service) RunNow() error {
	log.Println("Manual quota reset triggered...")
	return s.quotaService.ResetAllQuotas()
}
