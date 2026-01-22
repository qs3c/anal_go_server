# Phase 3: 社区功能 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现社区功能，包括 OSS 客户端、互动（点赞/收藏）、评论、广场 API。

**Architecture:** 延续前两阶段的分层架构，新增 OSS 客户端、Interaction/Comment 模型和服务。

**Tech Stack:** Go 1.22+, Gin, GORM, MySQL, Redis, aliyun-oss-go-sdk

---

## Task 1: Interaction Model

**Files:**
- Create: `internal/model/interaction.go`

**Step 1: 创建 internal/model/interaction.go**

```go
package model

import "time"

// Interaction 互动记录（点赞、收藏）
type Interaction struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"not null;index:idx_user_id" json:"user_id"`
	AnalysisID int64     `gorm:"not null;index:idx_analysis_type" json:"analysis_id"`
	Type       string    `gorm:"type:varchar(20);not null;index:idx_analysis_type" json:"type"` // like, bookmark
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relations
	User     *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Analysis *Analysis `gorm:"foreignKey:AnalysisID" json:"analysis,omitempty"`
}

func (Interaction) TableName() string {
	return "interactions"
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 2: Interaction Repository

**Files:**
- Create: `internal/repository/interaction_repo.go`

**Step 1: 创建 internal/repository/interaction_repo.go**

```go
package repository

import (
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type InteractionRepository struct {
	db *gorm.DB
}

func NewInteractionRepository(db *gorm.DB) *InteractionRepository {
	return &InteractionRepository{db: db}
}

// Create 创建互动记录
func (r *InteractionRepository) Create(interaction *model.Interaction) error {
	return r.db.Create(interaction).Error
}

// Delete 删除互动记录
func (r *InteractionRepository) Delete(userID, analysisID int64, interactionType string) error {
	return r.db.Where("user_id = ? AND analysis_id = ? AND type = ?", userID, analysisID, interactionType).
		Delete(&model.Interaction{}).Error
}

// Exists 检查互动是否存在
func (r *InteractionRepository) Exists(userID, analysisID int64, interactionType string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Interaction{}).
		Where("user_id = ? AND analysis_id = ? AND type = ?", userID, analysisID, interactionType).
		Count(&count).Error
	return count > 0, err
}

// GetByUserAndAnalysis 获取用户对某分析的互动状态
func (r *InteractionRepository) GetByUserAndAnalysis(userID, analysisID int64) ([]*model.Interaction, error) {
	var interactions []*model.Interaction
	err := r.db.Where("user_id = ? AND analysis_id = ?", userID, analysisID).Find(&interactions).Error
	return interactions, err
}

// GetUserLikedAnalyses 获取用户点赞的分析列表
func (r *InteractionRepository) GetUserLikedAnalyses(userID int64, page, pageSize int) ([]int64, int64, error) {
	var total int64
	var ids []int64

	query := r.db.Model(&model.Interaction{}).Where("user_id = ? AND type = ?", userID, "like")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Pluck("analysis_id", &ids).Error
	return ids, total, err
}

// GetUserBookmarkedAnalyses 获取用户收藏的分析列表
func (r *InteractionRepository) GetUserBookmarkedAnalyses(userID int64, page, pageSize int) ([]int64, int64, error) {
	var total int64
	var ids []int64

	query := r.db.Model(&model.Interaction{}).Where("user_id = ? AND type = ?", userID, "bookmark")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Pluck("analysis_id", &ids).Error
	return ids, total, err
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 3: Comment Model

**Files:**
- Create: `internal/model/comment.go`

**Step 1: 创建 internal/model/comment.go**

```go
package model

import "time"

// Comment 评论
type Comment struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"not null;index:idx_user_id" json:"user_id"`
	AnalysisID int64     `gorm:"not null;index:idx_analysis_id" json:"analysis_id"`
	ParentID   *int64    `gorm:"index:idx_parent_id" json:"parent_id"` // NULL 表示一级评论
	Content    string    `gorm:"type:text;not null" json:"content"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	User     *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Analysis *Analysis  `gorm:"foreignKey:AnalysisID" json:"analysis,omitempty"`
	Parent   *Comment   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Replies  []*Comment `gorm:"-" json:"replies,omitempty"` // 子回复（不通过 GORM 关联）
}

func (Comment) TableName() string {
	return "comments"
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 4: Comment Repository

**Files:**
- Create: `internal/repository/comment_repo.go`

**Step 1: 创建 internal/repository/comment_repo.go**

```go
package repository

import (
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create 创建评论
func (r *CommentRepository) Create(comment *model.Comment) error {
	return r.db.Create(comment).Error
}

// GetByID 根据 ID 获取评论
func (r *CommentRepository) GetByID(id int64) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetByIDWithUser 获取评论及用户信息
func (r *CommentRepository) GetByIDWithUser(id int64) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.Preload("User").Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// Delete 删除评论
func (r *CommentRepository) Delete(id int64) error {
	return r.db.Delete(&model.Comment{}, id).Error
}

// DeleteByParentID 删除子评论
func (r *CommentRepository) DeleteByParentID(parentID int64) (int64, error) {
	result := r.db.Where("parent_id = ?", parentID).Delete(&model.Comment{})
	return result.RowsAffected, result.Error
}

// ListByAnalysisID 获取分析的一级评论列表
func (r *CommentRepository) ListByAnalysisID(analysisID int64, page, pageSize int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := r.db.Model(&model.Comment{}).
		Preload("User").
		Where("analysis_id = ? AND parent_id IS NULL", analysisID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&comments).Error
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// GetRepliesByParentIDs 批量获取回复
func (r *CommentRepository) GetRepliesByParentIDs(parentIDs []int64) ([]*model.Comment, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}

	var replies []*model.Comment
	err := r.db.Preload("User").
		Where("parent_id IN ?", parentIDs).
		Order("created_at ASC").
		Find(&replies).Error
	return replies, err
}

// CountByAnalysisID 获取分析的评论数
func (r *CommentRepository) CountByAnalysisID(analysisID int64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Comment{}).Where("analysis_id = ?", analysisID).Count(&count).Error
	return count, err
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 5: Comment DTOs

**Files:**
- Create: `internal/model/dto/comment_dto.go`

**Step 1: 创建 internal/model/dto/comment_dto.go**

```go
package dto

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required,min=1,max=500"`
	ParentID *int64 `json:"parent_id,omitempty"`
}

// CommentItem 评论项
type CommentItem struct {
	ID        int64          `json:"id"`
	User      *CommentUser   `json:"user"`
	Content   string         `json:"content"`
	ParentID  *int64         `json:"parent_id"`
	Replies   []*CommentItem `json:"replies,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// CommentUser 评论用户信息
type CommentUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 6: Community DTOs

**Files:**
- Create: `internal/model/dto/community_dto.go`

**Step 1: 创建 internal/model/dto/community_dto.go**

```go
package dto

// CommunityListRequest 社区列表请求参数
type CommunityListRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Sort     string `form:"sort,default=latest"` // latest, hot
	Tags     string `form:"tags"`                // 逗号分隔
}

// LikeResponse 点赞响应
type LikeResponse struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"like_count"`
}

// BookmarkResponse 收藏响应
type BookmarkResponse struct {
	Bookmarked    bool `json:"bookmarked"`
	BookmarkCount int  `json:"bookmark_count"`
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 7: Community Service

**Files:**
- Create: `internal/service/community_service.go`

**Step 1: 创建 internal/service/community_service.go**

```go
package service

import (
	"errors"
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
	detail := &dto.CommunityAnalysisDetail{
		ID:               a.ID,
		ShareTitle:       a.ShareTitle,
		ShareDescription: a.ShareDescription,
		DiagramOSSURL:    a.DiagramOSSURL,
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
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 8: Comment Service

**Files:**
- Create: `internal/service/comment_service.go`

**Step 1: 创建 internal/service/comment_service.go**

```go
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
	ErrCommentNotFound    = errors.New("评论不存在")
	ErrCommentPermission  = errors.New("无权操作此评论")
	ErrParentNotFound     = errors.New("父评论不存在")
	ErrParentNotInAnalysis = errors.New("父评论不属于该分析")
)

type CommentService struct {
	commentRepo  *repository.CommentRepository
	analysisRepo *repository.AnalysisRepository
	userRepo     *repository.UserRepository
	cfg          *config.Config
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	analysisRepo *repository.AnalysisRepository,
	userRepo *repository.UserRepository,
	cfg *config.Config,
) *CommentService {
	return &CommentService{
		commentRepo:  commentRepo,
		analysisRepo: analysisRepo,
		userRepo:     userRepo,
		cfg:          cfg,
	}
}

// Create 创建评论
func (s *CommentService) Create(userID, analysisID int64, req *dto.CreateCommentRequest) (*dto.CommentItem, error) {
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

	// 如果是回复，验证父评论
	if req.ParentID != nil {
		parent, err := s.commentRepo.GetByID(*req.ParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrParentNotFound
			}
			return nil, err
		}

		// 验证父评论属于同一分析
		if parent.AnalysisID != analysisID {
			return nil, ErrParentNotInAnalysis
		}

		// 只支持一级回复
		if parent.ParentID != nil {
			req.ParentID = parent.ParentID
		}
	}

	// 获取用户信息
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	// 创建评论
	comment := &model.Comment{
		UserID:     userID,
		AnalysisID: analysisID,
		ParentID:   req.ParentID,
		Content:    req.Content,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	// 增加评论数
	s.analysisRepo.IncrementCommentCount(analysisID, 1)

	return &dto.CommentItem{
		ID:       comment.ID,
		ParentID: comment.ParentID,
		Content:  comment.Content,
		User: &dto.CommentUser{
			ID:        user.ID,
			Username:  user.Username,
			AvatarURL: user.AvatarURL,
		},
		CreatedAt: comment.CreatedAt.Format(time.RFC3339),
	}, nil
}

// Delete 删除评论
func (s *CommentService) Delete(userID, commentID int64) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCommentNotFound
		}
		return err
	}

	// 验证权限
	if comment.UserID != userID {
		return ErrCommentPermission
	}

	// 删除子回复并计算删除数量
	deletedReplies, _ := s.commentRepo.DeleteByParentID(commentID)

	// 删除评论
	if err := s.commentRepo.Delete(commentID); err != nil {
		return err
	}

	// 减少评论数（包括子回复）
	totalDeleted := 1 + int(deletedReplies)
	s.analysisRepo.IncrementCommentCount(comment.AnalysisID, -totalDeleted)

	return nil
}

// ListByAnalysisID 获取分析的评论列表
func (s *CommentService) ListByAnalysisID(analysisID int64, page, pageSize int) ([]*dto.CommentItem, int64, error) {
	// 验证分析存在且公开
	analysis, err := s.analysisRepo.GetByID(analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, ErrAnalysisNotFound
		}
		return nil, 0, err
	}

	if !analysis.IsPublic {
		return nil, 0, ErrAnalysisNotPublic
	}

	// 获取一级评论
	comments, total, err := s.commentRepo.ListByAnalysisID(analysisID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	if len(comments) == 0 {
		return []*dto.CommentItem{}, 0, nil
	}

	// 收集一级评论ID
	parentIDs := make([]int64, len(comments))
	for i, c := range comments {
		parentIDs[i] = c.ID
	}

	// 批量获取回复
	replies, _ := s.commentRepo.GetRepliesByParentIDs(parentIDs)

	// 构建回复映射
	repliesMap := make(map[int64][]*model.Comment)
	for _, r := range replies {
		if r.ParentID != nil {
			repliesMap[*r.ParentID] = append(repliesMap[*r.ParentID], r)
		}
	}

	// 组装结果
	items := make([]*dto.CommentItem, len(comments))
	for i, c := range comments {
		items[i] = s.buildCommentItem(c)

		// 添加回复
		if childReplies, ok := repliesMap[c.ID]; ok {
			items[i].Replies = make([]*dto.CommentItem, len(childReplies))
			for j, r := range childReplies {
				items[i].Replies[j] = s.buildCommentItem(r)
			}
		}
	}

	return items, total, nil
}

func (s *CommentService) buildCommentItem(c *model.Comment) *dto.CommentItem {
	item := &dto.CommentItem{
		ID:        c.ID,
		ParentID:  c.ParentID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}

	if c.User != nil {
		item.User = &dto.CommentUser{
			ID:        c.User.ID,
			Username:  c.User.Username,
			AvatarURL: c.User.AvatarURL,
		}
	}

	return item
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 9: Community Handler

**Files:**
- Create: `internal/api/handler/community.go`

**Step 1: 创建 internal/api/handler/community.go**

```go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type CommunityHandler struct {
	communityService *service.CommunityService
}

func NewCommunityHandler(communityService *service.CommunityService) *CommunityHandler {
	return &CommunityHandler{
		communityService: communityService,
	}
}

// List 获取广场分析列表
// GET /api/v1/community/analyses
func (h *CommunityHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sort := c.DefaultQuery("sort", "latest")
	tags := c.Query("tags")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	items, total, err := h.communityService.ListPublicAnalyses(page, pageSize, sort, tags)
	if err != nil {
		response.ServerError(c, "")
		return
	}

	response.SuccessPage(c, total, page, pageSize, items)
}

// Get 获取广场分析详情
// GET /api/v1/community/analyses/:id
func (h *CommunityHandler) Get(c *gin.Context) {
	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	// 获取用户ID（可选）
	var userID *int64
	if id, ok := middleware.GetUserID(c); ok {
		userID = &id
	}

	detail, err := h.communityService.GetPublicAnalysis(analysisID, userID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisNotPublic:
			response.NotFoundError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.Success(c, detail)
}

// Like 点赞
// POST /api/v1/community/analyses/:id/like
func (h *CommunityHandler) Like(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	resp, err := h.communityService.Like(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisNotPublic:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "点赞成功", resp)
}

// Unlike 取消点赞
// DELETE /api/v1/community/analyses/:id/like
func (h *CommunityHandler) Unlike(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	resp, err := h.communityService.Unlike(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "已取消点赞", resp)
}

// Bookmark 收藏
// POST /api/v1/community/analyses/:id/bookmark
func (h *CommunityHandler) Bookmark(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	resp, err := h.communityService.Bookmark(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisNotPublic:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "收藏成功", resp)
}

// Unbookmark 取消收藏
// DELETE /api/v1/community/analyses/:id/bookmark
func (h *CommunityHandler) Unbookmark(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	resp, err := h.communityService.Unbookmark(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "已取消收藏", resp)
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 10: Comment Handler

**Files:**
- Create: `internal/api/handler/comment.go`

**Step 1: 创建 internal/api/handler/comment.go**

```go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

// List 获取评论列表
// GET /api/v1/analyses/:id/comments
func (h *CommentHandler) List(c *gin.Context) {
	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	items, total, err := h.commentService.ListByAnalysisID(analysisID, page, pageSize)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisNotPublic:
			response.NotFoundError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessPage(c, total, page, pageSize, items)
}

// Create 发表评论
// POST /api/v1/analyses/:id/comments
func (h *CommentHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	comment, err := h.commentService.Create(userID, analysisID, &req)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisNotPublic:
			response.PermissionError(c, err.Error())
		case service.ErrParentNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrParentNotInAnalysis:
			response.ParamError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "评论成功", comment)
}

// Delete 删除评论
// DELETE /api/v1/comments/:id
func (h *CommentHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的评论ID")
		return
	}

	if err := h.commentService.Delete(userID, commentID); err != nil {
		switch err {
		case service.ErrCommentNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrCommentPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 11: Update Router with Phase 3 Routes

**Files:**
- Modify: `internal/api/router.go`

**Step 1: 更新 internal/api/router.go**

添加 CommunityHandler 和 CommentHandler 到 Router 结构体，并注册新路由。

需要修改的内容：
1. 添加 communityHandler 和 commentHandler 字段
2. 更新 NewRouter 函数签名
3. 在 Setup 中添加社区和评论路由

```go
// Router 结构体添加字段
type Router struct {
	authHandler      *handler.AuthHandler
	userHandler      *handler.UserHandler
	analysisHandler  *handler.AnalysisHandler
	modelsHandler    *handler.ModelsHandler
	websocketHandler *handler.WebSocketHandler
	communityHandler *handler.CommunityHandler
	commentHandler   *handler.CommentHandler
	cfg              *config.Config
}

// NewRouter 添加参数
func NewRouter(
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	analysisHandler *handler.AnalysisHandler,
	modelsHandler *handler.ModelsHandler,
	websocketHandler *handler.WebSocketHandler,
	communityHandler *handler.CommunityHandler,
	commentHandler *handler.CommentHandler,
	cfg *config.Config,
) *Router

// Setup 中的社区路由组
community := api.Group("/community")
community.Use(middleware.OptionalAuth(r.cfg.JWT.Secret))
{
	community.GET("/analyses", r.communityHandler.List)
	community.GET("/analyses/:id", r.communityHandler.Get)
}

// 需要认证的社区操作
communityAuth := api.Group("/community")
communityAuth.Use(middleware.Auth(r.cfg.JWT.Secret))
{
	communityAuth.POST("/analyses/:id/like", r.communityHandler.Like)
	communityAuth.DELETE("/analyses/:id/like", r.communityHandler.Unlike)
	communityAuth.POST("/analyses/:id/bookmark", r.communityHandler.Bookmark)
	communityAuth.DELETE("/analyses/:id/bookmark", r.communityHandler.Unbookmark)
}

// 评论路由（公开读取）
api.GET("/analyses/:id/comments", r.commentHandler.List)

// 评论路由（需认证）
authenticated.POST("/analyses/:id/comments", r.commentHandler.Create)
authenticated.DELETE("/comments/:id", r.commentHandler.Delete)
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 12: Update Server Main with Phase 3 Dependencies

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: 更新 cmd/server/main.go**

添加新的 repository、service 和 handler 初始化。

```go
// 新增 Repository 初始化
interactionRepo := repository.NewInteractionRepository(db)
commentRepo := repository.NewCommentRepository(db)

// 新增 Service 初始化
communityService := service.NewCommunityService(analysisRepo, interactionRepo, cfg)
commentService := service.NewCommentService(commentRepo, analysisRepo, userRepo, cfg)

// 新增 Handler 初始化
communityHandler := handler.NewCommunityHandler(communityService)
commentHandler := handler.NewCommentHandler(commentService)

// 更新 Router 初始化
router := api.NewRouter(
	authHandler,
	userHandler,
	analysisHandler,
	modelsHandler,
	websocketHandler,
	communityHandler,
	commentHandler,
	cfg,
)
```

**Step 2: 验证编译**

```bash
go build ./...
```

---

## Task 13: Tidy Dependencies and Final Verification

**Step 1: 整理 go.mod**

```bash
go mod tidy
```

**Step 2: 验证项目可以编译**

```bash
go build ./...
```

---

## Summary

Phase 3 完成后的功能：

| 功能 | 状态 |
|------|------|
| Interaction Model | ✅ |
| Interaction Repository | ✅ |
| Comment Model | ✅ |
| Comment Repository | ✅ |
| Comment DTOs | ✅ |
| Community DTOs | ✅ |
| Community Service | ✅ |
| Comment Service | ✅ |
| Community Handler | ✅ |
| Comment Handler | ✅ |
| Phase 3 Routes | ✅ |
| Server Main Update | ✅ |

**下一步（Phase 4）：**
- OSS Client 实现
- Worker 完整实现（调用 anal_go_agent）
- 邮件服务实现
- 微信 OAuth
