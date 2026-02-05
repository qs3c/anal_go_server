package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/repository"
)

var (
	ErrAnalysisNotPublic = errors.New("分析未公开")
	ErrAlreadyLiked      = errors.New("已点赞")
	ErrNotLiked          = errors.New("未点赞")
	ErrAlreadyBookmarked = errors.New("已收藏")
	ErrNotBookmarked     = errors.New("未收藏")
)

type CommunityService struct {
	analysisRepo    *repository.AnalysisRepository
	interactionRepo *repository.InteractionRepository
	cfg             *config.Config
}

func NewCommunityService(
	analysisRepo *repository.AnalysisRepository,
	interactionRepo *repository.InteractionRepository,
	cfg *config.Config,
) *CommunityService {
	return &CommunityService{
		analysisRepo:    analysisRepo,
		interactionRepo: interactionRepo,
		cfg:             cfg,
	}
}

// ListPublicAnalyses 获取公开分析列表
func (s *CommunityService) ListPublicAnalyses(page, pageSize int, sortBy string, tagsStr string) ([]*dto.CommunityAnalysisItem, int64, error) {
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}

	analyses, total, err := s.analysisRepo.ListPublic(page, pageSize, sortBy, tags)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.CommunityAnalysisItem, len(analyses))
	for i, a := range analyses {
		items[i] = s.buildCommunityItem(a)
	}

	return items, total, nil
}

// GetPublicAnalysis 获取公开分析详情
func (s *CommunityService) GetPublicAnalysis(analysisID int64, userID *int64) (*dto.CommunityAnalysisDetail, error) {
	analysis, err := s.analysisRepo.GetByIDWithUser(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	if !analysis.IsPublic {
		return nil, ErrAnalysisNotPublic
	}

	// 增加浏览数
	s.analysisRepo.IncrementViewCount(analysisID)

	detail := s.buildCommunityDetail(analysis)

	// 如果用户已登录，获取互动状态
	if userID != nil {
		interactions, _ := s.interactionRepo.GetByUserAndAnalysis(*userID, analysisID)
		detail.UserInteraction = &dto.UserInteraction{}
		for _, i := range interactions {
			if i.Type == "like" {
				detail.UserInteraction.Liked = true
			}
			if i.Type == "bookmark" {
				detail.UserInteraction.Bookmarked = true
			}
		}
	}

	return detail, nil
}

// Like 点赞
func (s *CommunityService) Like(userID, analysisID int64) (*dto.LikeResponse, error) {
	// 验证分析存在且公开
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	if !analysis.IsPublic {
		return nil, ErrAnalysisNotPublic
	}

	// 检查是否已点赞
	exists, err := s.interactionRepo.Exists(userID, analysisID, "like")
	if err != nil {
		return nil, err
	}

	if exists {
		// 已点赞，返回当前状态（幂等）
		return &dto.LikeResponse{
			Liked:     true,
			LikeCount: analysis.LikeCount,
		}, nil
	}

	// 创建点赞记录
	interaction := &model.Interaction{
		UserID:     userID,
		AnalysisID: analysisID,
		Type:       "like",
	}
	if err := s.interactionRepo.Create(interaction); err != nil {
		return nil, err
	}

	// 增加点赞数
	s.analysisRepo.IncrementLikeCount(analysisID, 1)

	return &dto.LikeResponse{
		Liked:     true,
		LikeCount: analysis.LikeCount + 1,
	}, nil
}

// Unlike 取消点赞
func (s *CommunityService) Unlike(userID, analysisID int64) (*dto.LikeResponse, error) {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	// 检查是否已点赞
	exists, err := s.interactionRepo.Exists(userID, analysisID, "like")
	if err != nil {
		return nil, err
	}

	if !exists {
		// 未点赞，返回当前状态（幂等）
		return &dto.LikeResponse{
			Liked:     false,
			LikeCount: analysis.LikeCount,
		}, nil
	}

	// 删除点赞记录
	if err := s.interactionRepo.Delete(userID, analysisID, "like"); err != nil {
		return nil, err
	}

	// 减少点赞数
	s.analysisRepo.IncrementLikeCount(analysisID, -1)

	newCount := analysis.LikeCount - 1
	if newCount < 0 {
		newCount = 0
	}

	return &dto.LikeResponse{
		Liked:     false,
		LikeCount: newCount,
	}, nil
}

// Bookmark 收藏
func (s *CommunityService) Bookmark(userID, analysisID int64) (*dto.BookmarkResponse, error) {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	if !analysis.IsPublic {
		return nil, ErrAnalysisNotPublic
	}

	exists, err := s.interactionRepo.Exists(userID, analysisID, "bookmark")
	if err != nil {
		return nil, err
	}

	if exists {
		return &dto.BookmarkResponse{
			Bookmarked:    true,
			BookmarkCount: analysis.BookmarkCount,
		}, nil
	}

	interaction := &model.Interaction{
		UserID:     userID,
		AnalysisID: analysisID,
		Type:       "bookmark",
	}
	if err := s.interactionRepo.Create(interaction); err != nil {
		return nil, err
	}

	s.analysisRepo.IncrementBookmarkCount(analysisID, 1)

	return &dto.BookmarkResponse{
		Bookmarked:    true,
		BookmarkCount: analysis.BookmarkCount + 1,
	}, nil
}

// Unbookmark 取消收藏
func (s *CommunityService) Unbookmark(userID, analysisID int64) (*dto.BookmarkResponse, error) {
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}

	exists, err := s.interactionRepo.Exists(userID, analysisID, "bookmark")
	if err != nil {
		return nil, err
	}

	if !exists {
		return &dto.BookmarkResponse{
			Bookmarked:    false,
			BookmarkCount: analysis.BookmarkCount,
		}, nil
	}

	if err := s.interactionRepo.Delete(userID, analysisID, "bookmark"); err != nil {
		return nil, err
	}

	s.analysisRepo.IncrementBookmarkCount(analysisID, -1)

	newCount := analysis.BookmarkCount - 1
	if newCount < 0 {
		newCount = 0
	}

	return &dto.BookmarkResponse{
		Bookmarked:    false,
		BookmarkCount: newCount,
	}, nil
}

func (s *CommunityService) buildCommunityItem(a *model.Analysis) *dto.CommunityAnalysisItem {
	item := &dto.CommunityAnalysisItem{
		ID:               a.ID,
		ShareTitle:       a.ShareTitle,
		ShareDescription: a.ShareDescription,
		ViewCount:        a.ViewCount,
		LikeCount:        a.LikeCount,
		CommentCount:     a.CommentCount,
		BookmarkCount:    a.BookmarkCount,
	}

	if a.Tags != nil {
		item.Tags = a.Tags
	} else {
		item.Tags = []string{}
	}

	if a.SharedAt != nil {
		item.SharedAt = a.SharedAt.Format(time.RFC3339)
	}

	if a.User != nil {
		item.Author = &dto.AuthorInfo{
			ID:        a.User.ID,
			Username:  a.User.Username,
			AvatarURL: a.User.AvatarURL,
		}
	}

	return item
}

func (s *CommunityService) buildCommunityDetail(a *model.Analysis) *dto.CommunityAnalysisDetail {
	// 处理本地存储的 URL，转换为 API 端点
	diagramURL := a.DiagramOSSURL
	if strings.HasPrefix(diagramURL, "local://") {
		diagramURL = fmt.Sprintf("/api/v1/analyses/%d/diagram", a.ID)
	}

	detail := &dto.CommunityAnalysisDetail{
		ID:               a.ID,
		ShareTitle:       a.ShareTitle,
		ShareDescription: a.ShareDescription,
		DiagramOSSURL:    diagramURL,
		CreationType:     a.CreationType,
		RepoURL:          a.RepoURL,
		ViewCount:        a.ViewCount,
		LikeCount:        a.LikeCount,
		CommentCount:     a.CommentCount,
		BookmarkCount:    a.BookmarkCount,
	}

	if a.Tags != nil {
		detail.Tags = a.Tags
	} else {
		detail.Tags = []string{}
	}

	if a.SharedAt != nil {
		detail.SharedAt = a.SharedAt.Format(time.RFC3339)
	}

	if a.User != nil {
		detail.Author = &dto.AuthorInfo{
			ID:        a.User.ID,
			Username:  a.User.Username,
			AvatarURL: a.User.AvatarURL,
			Bio:       a.User.Bio,
		}
	}

	return detail
}
