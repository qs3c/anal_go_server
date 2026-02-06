package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/oss"
	"github.com/qs3c/anal_go_server/internal/pkg/queue"
	"github.com/qs3c/anal_go_server/internal/repository"
)

var (
	ErrAnalysisNotFound    = errors.New("分析项目不存在")
	ErrAnalysisPermission  = errors.New("无权操作此分析项目")
	ErrAnalysisNotComplete = errors.New("分析尚未完成，无法分享")
)

type AnalysisService struct {
	analysisRepo  *repository.AnalysisRepository
	jobRepo       *repository.JobRepository
	userRepo      *repository.UserRepository
	quotaService  *QuotaService
	uploadService *UploadService
	ossClient     *oss.Client
	jobQueue      *queue.Queue
	cfg           *config.Config
}

func NewAnalysisService(
	analysisRepo *repository.AnalysisRepository,
	jobRepo *repository.JobRepository,
	userRepo *repository.UserRepository,
	quotaService *QuotaService,
	uploadService *UploadService,
	ossClient *oss.Client,
	jobQueue *queue.Queue,
	cfg *config.Config,
) *AnalysisService {
	return &AnalysisService{
		analysisRepo:  analysisRepo,
		jobRepo:       jobRepo,
		userRepo:      userRepo,
		quotaService:  quotaService,
		uploadService: uploadService,
		ossClient:     ossClient,
		jobQueue:      jobQueue,
		cfg:           cfg,
	}
}

// Create 创建分析
func (s *AnalysisService) Create(userID int64, req *dto.CreateAnalysisRequest) (*dto.CreateAnalysisResponse, error) {
	// 临时禁用 user 查询 - 因为配额检查已禁用
	// user, err := s.userRepo.GetByID(userID)
	// if err != nil {
	// 	return nil, err
	// }

	analysis := &model.Analysis{
		UserID:       userID,
		Title:        req.Title,
		CreationType: req.CreationType,
	}

	if req.CreationType == "ai" {
		// 临时禁用配额检查 - 用于测试
		// hasQuota, err := s.quotaService.CheckQuota(userID)
		// if err != nil {
		// 	return nil, err
		// }
		// if !hasQuota {
		// 	return nil, ErrQuotaExceeded
		// }

		// if err := s.quotaService.CheckDepth(user.SubscriptionLevel, req.AnalysisDepth); err != nil {
		// 	return nil, err
		// }

		// if err := s.quotaService.CheckModelPermission(user.SubscriptionLevel, req.ModelName); err != nil {
		// 	return nil, err
		// }

		// Handle source type
		sourceType := req.SourceType
		if sourceType == "" {
			sourceType = "github" // default
		}

		if sourceType == "upload" {
			if req.UploadID == "" {
				return nil, errors.New("upload_id 不能为空")
			}
			if s.uploadService != nil {
				if _, err := s.uploadService.GetUploadPath(req.UploadID); err != nil {
					return nil, err
				}
			}
			analysis.SourceType = "upload"
			analysis.UploadID = req.UploadID
			analysis.StartFile = req.StartFile
		} else {
			analysis.SourceType = "github"
			analysis.RepoURL = req.RepoURL
		}

		analysis.StartStruct = req.StartStruct
		analysis.AnalysisDepth = req.AnalysisDepth
		analysis.ModelName = req.ModelName
		analysis.Status = "pending"
	} else {
		// 手动创建
		analysis.Status = "draft"
		if req.DiagramData != nil {
			// 序列化并上传到 OSS
			data, err := json.Marshal(req.DiagramData)
			if err != nil {
				return nil, err
			}
			if s.ossClient != nil {
				// 先创建分析获取 ID
				if err := s.analysisRepo.Create(analysis); err != nil {
					return nil, err
				}
				ossURL, err := s.ossClient.UploadDiagram(analysis.ID, data)
				if err != nil {
					return nil, err
				}
				analysis.DiagramOSSURL = ossURL
				analysis.DiagramSize = len(data)
				analysis.Status = "completed"
				now := time.Now()
				analysis.CompletedAt = &now
				if err := s.analysisRepo.Update(analysis); err != nil {
					return nil, err
				}
				return &dto.CreateAnalysisResponse{AnalysisID: analysis.ID}, nil
			}
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
		// 临时禁用配额扣除 - 用于测试
		// if err := s.quotaService.UseQuota(userID); err != nil {
		// 	return nil, err
		// }

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
			// 临时禁用配额退还 - 因为没有扣除配额
			// s.quotaService.RefundQuota(userID)
			return nil, err
		}

		resp.JobID = job.ID

		// 加入 Redis 队列
		if s.jobQueue != nil {
			jobMsg := &queue.JobMessage{
				JobID:       job.ID,
				AnalysisID:  analysis.ID,
				UserID:      userID,
				SourceType:  analysis.SourceType,
				RepoURL:     req.RepoURL,
				UploadID:    req.UploadID,
				StartFile:   req.StartFile,
				StartStruct: req.StartStruct,
				Depth:       req.AnalysisDepth,
				ModelName:   req.ModelName,
			}
			if err := s.jobQueue.Push(context.Background(), jobMsg); err != nil {
				// 临时禁用配额退还 - 因为没有扣除配额
				// s.quotaService.RefundQuota(userID)
				return nil, err
			}
		}
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
		// 序列化并上传到 OSS
		data, err := json.Marshal(req.DiagramData)
		if err != nil {
			return nil, err
		}
		if s.ossClient != nil {
			ossURL, err := s.ossClient.UploadDiagram(analysis.ID, data)
			if err != nil {
				return nil, err
			}
			analysis.DiagramOSSURL = ossURL
			analysis.DiagramSize = len(data)
			if analysis.Status == "draft" {
				analysis.Status = "completed"
				now := time.Now()
				analysis.CompletedAt = &now
			}
		}
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

	// 删除 OSS 文件
	if analysis.DiagramOSSURL != "" && s.ossClient != nil {
		objectKey := s.ossClient.ExtractObjectKey(analysis.DiagramOSSURL)
		s.ossClient.Delete(objectKey)
	}

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
	// 处理 diagram URL
	diagramURL := a.DiagramOSSURL

	if strings.HasPrefix(diagramURL, "local://") {
		// 本地存储：走 API 代理
		diagramURL = fmt.Sprintf("/api/v1/analyses/%d/diagram", a.ID)
	} else if strings.HasPrefix(diagramURL, "https://") && s.ossClient != nil {
		// OSS 存储：生成签名 URL，前端直接从 OSS 下载
		objectKey := s.ossClient.ExtractObjectKey(diagramURL)
		if signedURL, err := s.ossClient.GetSignedURL(objectKey); err == nil {
			diagramURL = signedURL
		}
	}

	detail := &dto.AnalysisDetail{
		ID:               a.ID,
		Title:            a.Title,
		Description:      a.Description,
		CreationType:     a.CreationType,
		RepoURL:          a.RepoURL,
		StartStruct:      a.StartStruct,
		AnalysisDepth:    a.AnalysisDepth,
		ModelName:        a.ModelName,
		DiagramOSSURL:    diagramURL,
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

	// Check if diagram is stored locally (OSS upload failed)
	if strings.HasPrefix(a.DiagramOSSURL, "local://") {
		detail.Warnings = append(detail.Warnings, "图表数据暂存本地，稍后将自动同步到云端")
	}

	// Check if model had valid LLM config
	if a.CreationType == "ai" && a.ModelName != "" {
		hasKey := false
		for _, m := range s.cfg.Models {
			if m.Name == a.ModelName && m.APIKey != "" {
				hasKey = true
				break
			}
		}
		if !hasKey {
			detail.Warnings = append(detail.Warnings, "该模型未配置 API Key，分析结果不包含 AI 生成的描述信息")
		}
	}

	return detail
}

// GetDiagramData 获取图表数据
// 支持本地存储和 OSS 存储
func (s *AnalysisService) GetDiagramData(userID, analysisID int64) ([]byte, error) {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	// 验证权限（私有分析只能自己访问）
	if !analysis.IsPublic && analysis.UserID != userID {
		return nil, ErrAnalysisPermission
	}

	if analysis.DiagramOSSURL == "" {
		return nil, errors.New("diagram not available")
	}

	// 检查是否是本地存储
	if strings.HasPrefix(analysis.DiagramOSSURL, "local://") {
		// 本地存储：从文件读取
		localPath := filepath.Join(s.cfg.Upload.TempDir, "diagrams", fmt.Sprintf("%d.json", analysisID))
		data, err := os.ReadFile(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read local diagram: %w", err)
		}
		return data, nil
	}

	// OSS 存储：从 OSS 下载并返回
	if s.ossClient != nil {
		objectKey := s.ossClient.ExtractObjectKey(analysis.DiagramOSSURL)
		data, err := s.ossClient.Download(objectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to download diagram from OSS: %w", err)
		}
		return data, nil
	}

	return nil, errors.New("OSS client not initialized")
}
