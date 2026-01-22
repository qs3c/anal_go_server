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

var (
	ErrAnalysisNotFound    = errors.New("分析项目不存在")
	ErrAnalysisPermission  = errors.New("无权操作此分析项目")
	ErrAnalysisNotComplete = errors.New("分析尚未完成，无法分享")
)

type AnalysisService struct {
	analysisRepo *repository.AnalysisRepository
	jobRepo      *repository.JobRepository
	userRepo     *repository.UserRepository
	quotaService *QuotaService
	cfg          *config.Config
}

func NewAnalysisService(
	analysisRepo *repository.AnalysisRepository,
	jobRepo *repository.JobRepository,
	userRepo *repository.UserRepository,
	quotaService *QuotaService,
	cfg *config.Config,
) *AnalysisService {
	return &AnalysisService{
		analysisRepo: analysisRepo,
		jobRepo:      jobRepo,
		userRepo:     userRepo,
		quotaService: quotaService,
		cfg:          cfg,
	}
}

// Create 创建分析
func (s *AnalysisService) Create(userID int64, req *dto.CreateAnalysisRequest) (*dto.CreateAnalysisResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	analysis := &model.Analysis{
		UserID:       userID,
		Title:        req.Title,
		CreationType: req.CreationType,
	}

	if req.CreationType == "ai" {
		// AI 分析需要验证配额和权限
		hasQuota, err := s.quotaService.CheckQuota(userID)
		if err != nil {
			return nil, err
		}
		if !hasQuota {
			return nil, ErrQuotaExceeded
		}

		if err := s.quotaService.CheckDepth(user.SubscriptionLevel, req.AnalysisDepth); err != nil {
			return nil, err
		}

		if err := s.quotaService.CheckModelPermission(user.SubscriptionLevel, req.ModelName); err != nil {
			return nil, err
		}

		analysis.RepoURL = req.RepoURL
		analysis.StartStruct = req.StartStruct
		analysis.AnalysisDepth = req.AnalysisDepth
		analysis.ModelName = req.ModelName
		analysis.Status = "pending"
	} else {
		// 手动创建
		analysis.Status = "draft"
		if req.DiagramData != nil {
			// TODO: 上传到 OSS
			analysis.Status = "completed"
		}
	}

	if err := s.analysisRepo.Create(analysis); err != nil {
		return nil, err
	}

	resp := &dto.CreateAnalysisResponse{
		AnalysisID: analysis.ID,
	}

	// 如果是 AI 分析，创建任务
	if req.CreationType == "ai" {
		// 扣除配额
		if err := s.quotaService.UseQuota(userID); err != nil {
			return nil, err
		}

		job := &model.AnalysisJob{
			AnalysisID:  analysis.ID,
			UserID:      userID,
			RepoURL:     req.RepoURL,
			StartStruct: req.StartStruct,
			Depth:       req.AnalysisDepth,
			ModelName:   req.ModelName,
			Status:      "queued",
		}

		if err := s.jobRepo.Create(job); err != nil {
			// 退还配额
			s.quotaService.RefundQuota(userID)
			return nil, err
		}

		resp.JobID = job.ID

		// TODO: 加入 Redis 队列
	}

	return resp, nil
}

// GetByID 获取分析详情
func (s *AnalysisService) GetByID(userID, analysisID int64) (*dto.AnalysisDetail, error) {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	// 验证权限
	if analysis.UserID != userID {
		return nil, ErrAnalysisPermission
	}

	return s.buildAnalysisDetail(analysis), nil
}

// List 获取分析列表
func (s *AnalysisService) List(userID int64, page, pageSize int, search, status string) ([]*dto.AnalysisListItem, int64, error) {
	analyses, total, err := s.analysisRepo.ListByUserID(userID, page, pageSize, search, status)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.AnalysisListItem, len(analyses))
	for i, a := range analyses {
		items[i] = &dto.AnalysisListItem{
			ID:           a.ID,
			Title:        a.Title,
			CreationType: a.CreationType,
			Status:       a.Status,
			IsPublic:     a.IsPublic,
			ViewCount:    a.ViewCount,
			LikeCount:    a.LikeCount,
			CommentCount: a.CommentCount,
			CreatedAt:    a.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    a.UpdatedAt.Format(time.RFC3339),
		}
		if a.Tags != nil {
			items[i].Tags = a.Tags
		}
	}

	return items, total, nil
}

// Update 更新分析
func (s *AnalysisService) Update(userID, analysisID int64, req *dto.UpdateAnalysisRequest) (*dto.AnalysisDetail, error) {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	if analysis.UserID != userID {
		return nil, ErrAnalysisPermission
	}

	if req.Title != nil {
		analysis.Title = *req.Title
	}
	if req.Description != nil {
		analysis.Description = *req.Description
	}
	if req.DiagramData != nil {
		// TODO: 上传到 OSS
	}

	if err := s.analysisRepo.Update(analysis); err != nil {
		return nil, err
	}

	return s.buildAnalysisDetail(analysis), nil
}

// Delete 删除分析
func (s *AnalysisService) Delete(userID, analysisID int64) error {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAnalysisNotFound
		}
		return err
	}

	if analysis.UserID != userID {
		return ErrAnalysisPermission
	}

	// 取消进行中的任务
	s.jobRepo.CancelByAnalysisID(analysisID)

	// TODO: 删除 OSS 文件

	return s.analysisRepo.Delete(analysisID)
}

// Share 分享到广场
func (s *AnalysisService) Share(userID, analysisID int64, req *dto.ShareAnalysisRequest) error {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAnalysisNotFound
		}
		return err
	}

	if analysis.UserID != userID {
		return ErrAnalysisPermission
	}

	if analysis.Status != "completed" {
		return ErrAnalysisNotComplete
	}

	now := time.Now()
	analysis.IsPublic = true
	analysis.SharedAt = &now
	analysis.ShareTitle = req.ShareTitle
	analysis.ShareDescription = req.ShareDescription
	analysis.Tags = req.Tags

	return s.analysisRepo.Update(analysis)
}

// Unshare 取消分享
func (s *AnalysisService) Unshare(userID, analysisID int64) error {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAnalysisNotFound
		}
		return err
	}

	if analysis.UserID != userID {
		return ErrAnalysisPermission
	}

	analysis.IsPublic = false
	analysis.SharedAt = nil

	return s.analysisRepo.Update(analysis)
}

// GetJobStatus 获取任务状态
func (s *AnalysisService) GetJobStatus(userID, analysisID int64) (*dto.JobStatusResponse, error) {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	if analysis.UserID != userID {
		return nil, ErrAnalysisPermission
	}

	job, err := s.jobRepo.GetByAnalysisID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("任务不存在")
		}
		return nil, err
	}

	resp := &dto.JobStatusResponse{
		JobID:        job.ID,
		AnalysisID:   job.AnalysisID,
		Status:       job.Status,
		CurrentStep:  job.CurrentStep,
		ErrorMessage: job.ErrorMessage,
	}

	if job.StartedAt != nil {
		resp.StartedAt = job.StartedAt.Format(time.RFC3339)
		resp.ElapsedSeconds = int(time.Since(*job.StartedAt).Seconds())
	}

	return resp, nil
}

func (s *AnalysisService) buildAnalysisDetail(a *model.Analysis) *dto.AnalysisDetail {
	detail := &dto.AnalysisDetail{
		ID:               a.ID,
		Title:            a.Title,
		Description:      a.Description,
		CreationType:     a.CreationType,
		RepoURL:          a.RepoURL,
		StartStruct:      a.StartStruct,
		AnalysisDepth:    a.AnalysisDepth,
		ModelName:        a.ModelName,
		DiagramOSSURL:    a.DiagramOSSURL,
		DiagramSize:      a.DiagramSize,
		Status:           a.Status,
		ErrorMessage:     a.ErrorMessage,
		IsPublic:         a.IsPublic,
		ShareTitle:       a.ShareTitle,
		ShareDescription: a.ShareDescription,
		ViewCount:        a.ViewCount,
		LikeCount:        a.LikeCount,
		CommentCount:     a.CommentCount,
		BookmarkCount:    a.BookmarkCount,
		CreatedAt:        a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        a.UpdatedAt.Format(time.RFC3339),
	}

	if a.Tags != nil {
		detail.Tags = a.Tags
	}
	if a.StartedAt != nil {
		detail.StartedAt = a.StartedAt.Format(time.RFC3339)
	}
	if a.CompletedAt != nil {
		detail.CompletedAt = a.CompletedAt.Format(time.RFC3339)
	}

	return detail
}
