# Phase 2: 核心功能 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现核心功能，包括用户管理、GitHub OAuth、分析项目 CRUD、OSS 存储、Redis 队列、WebSocket 实时推送和 Worker 进程。

**Architecture:** 延续 Phase 1 的分层架构（Handler → Service → Repository → Model），新增 OSS 客户端、Redis 队列、WebSocket Hub 等基础设施组件。

**Tech Stack:** Go 1.22+, Gin, GORM, MySQL, Redis, JWT, gorilla/websocket, aliyun-oss-go-sdk, golang.org/x/oauth2

---

## Task 1: GitHub OAuth 客户端

**Files:**
- Create: `internal/pkg/oauth/github.go`

**Step 1: 安装 OAuth 依赖**

```bash
go get golang.org/x/oauth2
```

**Step 2: 创建 internal/pkg/oauth/github.go**

```go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GithubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
}

type GithubOAuth struct {
	config *oauth2.Config
}

func NewGithubOAuth(clientID, clientSecret, redirectURI string) *GithubOAuth {
	return &GithubOAuth{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
	}
}

// GetAuthURL 获取 GitHub 授权 URL
func (g *GithubOAuth) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state)
}

// Exchange 用授权码换取 access token
func (g *GithubOAuth) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.config.Exchange(ctx, code)
}

// GetUser 获取 GitHub 用户信息
func (g *GithubOAuth) GetUser(ctx context.Context, token *oauth2.Token) (*GithubUser, error) {
	client := g.config.Client(ctx, token)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api error: %s", string(body))
	}

	var user GithubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// 如果邮箱为空，尝试获取主邮箱
	if user.Email == "" {
		email, err := g.getPrimaryEmail(ctx, client)
		if err == nil {
			user.Email = email
		}
	}

	return &user, nil
}

func (g *GithubOAuth) getPrimaryEmail(ctx context.Context, client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", nil
}
```

**Step 3: 验证编译**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add .
git commit -m "feat: add GitHub OAuth client

- Add GithubOAuth with auth URL generation
- Add token exchange functionality
- Add user info fetching with email fallback"
```

---

## Task 2: Auth Service 添加 GitHub OAuth

**Files:**
- Modify: `internal/service/auth_service.go`

**Step 1: 更新 internal/service/auth_service.go 添加 GitHub OAuth 方法**

在 AuthService 结构体中添加 GithubOAuth 字段并实现相关方法。

添加以下代码到 auth_service.go:

```go
// 在导入中添加
import (
	// ... 现有导入
	"github.com/qs3c/anal_go_server/internal/pkg/oauth"
)

// 修改 AuthService 结构体
type AuthService struct {
	userRepo    *repository.UserRepository
	cfg         *config.Config
	githubOAuth *oauth.GithubOAuth
}

// 修改 NewAuthService 构造函数
func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		cfg:      cfg,
		githubOAuth: oauth.NewGithubOAuth(
			cfg.OAuth.Github.ClientID,
			cfg.OAuth.Github.ClientSecret,
			cfg.OAuth.Github.RedirectURI,
		),
	}
}

// 添加 GitHub OAuth 方法

// GetGithubAuthURL 获取 GitHub 授权 URL
func (s *AuthService) GetGithubAuthURL(state string) string {
	return s.githubOAuth.GetAuthURL(state)
}

// GithubCallback 处理 GitHub OAuth 回调
func (s *AuthService) GithubCallback(ctx context.Context, code string) (*dto.LoginResponse, error) {
	// 用 code 换取 token
	token, err := s.githubOAuth.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// 获取 GitHub 用户信息
	githubUser, err := s.githubOAuth.GetUser(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get github user: %w", err)
	}

	githubIDStr := fmt.Sprintf("%d", githubUser.ID)

	// 检查用户是否已存在
	user, err := s.userRepo.GetByGithubID(githubIDStr)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if user == nil {
		// 创建新用户
		resetAt := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
		user = &model.User{
			Username:          githubUser.Login,
			GithubID:          &githubIDStr,
			AvatarURL:         githubUser.AvatarURL,
			SubscriptionLevel: "free",
			DailyQuota:        s.cfg.Subscription.Levels["free"].DailyQuota,
			QuotaResetAt:      &resetAt,
			EmailVerified:     true, // OAuth 用户默认已验证
		}

		// 如果有邮箱，设置邮箱
		if githubUser.Email != "" {
			user.Email = &githubUser.Email
		}

		// 确保用户名唯一
		exists, _ := s.userRepo.ExistsByUsername(user.Username)
		if exists {
			user.Username = fmt.Sprintf("%s_%d", githubUser.Login, githubUser.ID)
		}

		if err := s.userRepo.Create(user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// 生成 JWT Token
	jwtToken, err := jwt.GenerateToken(user.ID, s.cfg.JWT.Secret, s.cfg.JWT.ExpireHours)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: jwtToken,
		User:  s.buildUserInfo(user),
	}, nil
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add GitHub OAuth to auth service

- Add GetGithubAuthURL method
- Add GithubCallback for OAuth flow
- Handle new user creation from GitHub"
```

---

## Task 3: Auth Handler 添加 GitHub OAuth 路由

**Files:**
- Modify: `internal/api/handler/auth.go`
- Modify: `internal/api/router.go`

**Step 1: 更新 internal/api/handler/auth.go 添加 GitHub OAuth handler**

```go
// 添加到 AuthHandler

// GithubAuth GitHub OAuth 登录
// GET /api/v1/auth/github
func (h *AuthHandler) GithubAuth(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		state = "random_state" // 生产环境应该使用随机值并存储验证
	}
	authURL := h.authService.GetGithubAuthURL(state)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// GithubCallback GitHub OAuth 回调
// GET /api/v1/auth/github/callback
func (h *AuthHandler) GithubCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.ParamError(c, "missing code parameter")
		return
	}

	resp, err := h.authService.GithubCallback(c.Request.Context(), code)
	if err != nil {
		response.ServerError(c, "GitHub 登录失败")
		return
	}

	// 重定向到前端，携带 token
	frontendURL := c.Query("redirect_uri")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, resp.Token)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}
```

在文件顶部添加必要的导入:
```go
import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)
```

**Step 2: 更新 internal/api/router.go 添加 GitHub OAuth 路由**

在 auth 路由组中添加:
```go
auth.GET("/github", r.authHandler.GithubAuth)
auth.GET("/github/callback", r.authHandler.GithubCallback)
```

**Step 3: 验证编译**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add .
git commit -m "feat: add GitHub OAuth routes

- Add /auth/github redirect endpoint
- Add /auth/github/callback handler
- Redirect to frontend with token"
```

---

## Task 4: User Service 和 Handler

**Files:**
- Create: `internal/service/user_service.go`
- Create: `internal/api/handler/user.go`

**Step 1: 创建 internal/service/user_service.go**

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
```

**Step 2: 创建 internal/api/handler/user.go**

```go
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetProfile 获取当前用户信息
// GET /api/v1/user/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		response.ServerError(c, "")
		return
	}

	response.Success(c, profile)
}

// UpdateProfile 更新用户信息
// PUT /api/v1/user/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	profile, err := h.userService.UpdateProfile(userID, &req)
	if err != nil {
		if err == service.ErrUsernameExists {
			response.ParamError(c, err.Error())
			return
		}
		response.ServerError(c, "")
		return
	}

	response.SuccessWithMessage(c, "更新成功", profile)
}

// UploadAvatar 上传头像
// POST /api/v1/user/avatar
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.ParamError(c, "请选择文件")
		return
	}

	// 验证文件大小 (5MB)
	if file.Size > 5*1024*1024 {
		response.ParamError(c, "文件大小不能超过5MB")
		return
	}

	// 验证文件类型
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" {
		response.ParamError(c, "只支持 jpg/png 格式")
		return
	}

	// TODO: 上传到 OSS
	// 暂时返回占位 URL
	avatarURL := "https://cdn.example.com/avatars/placeholder.jpg"

	if err := h.userService.UpdateAvatar(userID, avatarURL); err != nil {
		response.ServerError(c, "")
		return
	}

	response.SuccessWithMessage(c, "上传成功", gin.H{
		"avatar_url": avatarURL,
	})
}
```

**Step 3: 验证编译**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add .
git commit -m "feat: add user service and handler

- Add GetProfile with quota info
- Add UpdateProfile for username and bio
- Add UploadAvatar placeholder"
```

---

## Task 5: Analysis Repository

**Files:**
- Create: `internal/repository/analysis_repo.go`

**Step 1: 创建 internal/repository/analysis_repo.go**

```go
package repository

import (
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type AnalysisRepository struct {
	db *gorm.DB
}

func NewAnalysisRepository(db *gorm.DB) *AnalysisRepository {
	return &AnalysisRepository{db: db}
}

func (r *AnalysisRepository) Create(analysis *model.Analysis) error {
	return r.db.Create(analysis).Error
}

func (r *AnalysisRepository) GetByID(id int64) (*model.Analysis, error) {
	var analysis model.Analysis
	err := r.db.Where("id = ?", id).First(&analysis).Error
	if err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (r *AnalysisRepository) GetByIDWithUser(id int64) (*model.Analysis, error) {
	var analysis model.Analysis
	err := r.db.Preload("User").Where("id = ?", id).First(&analysis).Error
	if err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (r *AnalysisRepository) Update(analysis *model.Analysis) error {
	return r.db.Save(analysis).Error
}

func (r *AnalysisRepository) UpdateFields(id int64, fields map[string]interface{}) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).Updates(fields).Error
}

func (r *AnalysisRepository) UpdateStatus(id int64, status string) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).Update("status", status).Error
}

func (r *AnalysisRepository) Delete(id int64) error {
	return r.db.Delete(&model.Analysis{}, id).Error
}

// ListByUserID 获取用户的分析列表
func (r *AnalysisRepository) ListByUserID(userID int64, page, pageSize int, search, status string) ([]*model.Analysis, int64, error) {
	var analyses []*model.Analysis
	var total int64

	query := r.db.Model(&model.Analysis{}).Where("user_id = ?", userID)

	if search != "" {
		query = query.Where("title LIKE ?", "%"+search+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&analyses).Error; err != nil {
		return nil, 0, err
	}

	return analyses, total, nil
}

// ListPublic 获取公开的分析列表
func (r *AnalysisRepository) ListPublic(page, pageSize int, sortBy string, tags []string) ([]*model.Analysis, int64, error) {
	var analyses []*model.Analysis
	var total int64

	query := r.db.Model(&model.Analysis{}).Preload("User").Where("is_public = ?", true)

	// TODO: 标签过滤

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	switch sortBy {
	case "hot":
		query = query.Order("(like_count * 3 + comment_count * 2 + view_count) DESC")
	default: // latest
		query = query.Order("shared_at DESC")
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&analyses).Error; err != nil {
		return nil, 0, err
	}

	return analyses, total, nil
}

// IncrementViewCount 增加浏览数
func (r *AnalysisRepository) IncrementViewCount(id int64) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).
		Update("view_count", gorm.Expr("view_count + 1")).Error
}

// IncrementLikeCount 增加点赞数
func (r *AnalysisRepository) IncrementLikeCount(id int64, delta int) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// IncrementBookmarkCount 增加收藏数
func (r *AnalysisRepository) IncrementBookmarkCount(id int64, delta int) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).
		Update("bookmark_count", gorm.Expr("bookmark_count + ?", delta)).Error
}

// IncrementCommentCount 增加评论数
func (r *AnalysisRepository) IncrementCommentCount(id int64, delta int) error {
	return r.db.Model(&model.Analysis{}).Where("id = ?", id).
		Update("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add analysis repository

- Add CRUD operations
- Add list with pagination and filters
- Add public list with sorting
- Add counter increment methods"
```

---

## Task 6: Job Repository

**Files:**
- Create: `internal/repository/job_repo.go`

**Step 1: 创建 internal/repository/job_repo.go**

```go
package repository

import (
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(job *model.AnalysisJob) error {
	return r.db.Create(job).Error
}

func (r *JobRepository) GetByID(id int64) (*model.AnalysisJob, error) {
	var job model.AnalysisJob
	err := r.db.Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobRepository) GetByAnalysisID(analysisID int64) (*model.AnalysisJob, error) {
	var job model.AnalysisJob
	err := r.db.Where("analysis_id = ?", analysisID).Order("created_at DESC").First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobRepository) Update(job *model.AnalysisJob) error {
	return r.db.Save(job).Error
}

func (r *JobRepository) UpdateStatus(id int64, status string) error {
	return r.db.Model(&model.AnalysisJob{}).Where("id = ?", id).Update("status", status).Error
}

func (r *JobRepository) UpdateStep(id int64, step string) error {
	return r.db.Model(&model.AnalysisJob{}).Where("id = ?", id).Update("current_step", step).Error
}

// GetPendingJobs 获取待处理的任务
func (r *JobRepository) GetPendingJobs(limit int) ([]*model.AnalysisJob, error) {
	var jobs []*model.AnalysisJob
	err := r.db.Where("status = ?", "queued").
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error
	return jobs, err
}

// CancelByAnalysisID 取消指定分析的任务
func (r *JobRepository) CancelByAnalysisID(analysisID int64) error {
	return r.db.Model(&model.AnalysisJob{}).
		Where("analysis_id = ? AND status IN ?", analysisID, []string{"queued", "processing"}).
		Update("status", "cancelled").Error
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add job repository

- Add CRUD operations
- Add status and step updates
- Add pending jobs query
- Add cancel by analysis ID"
```

---

## Task 7: Analysis DTOs

**Files:**
- Create: `internal/model/dto/analysis_dto.go`

**Step 1: 创建 internal/model/dto/analysis_dto.go**

```go
package dto

import "encoding/json"

// CreateAnalysisRequest 创建分析请求
type CreateAnalysisRequest struct {
	Title         string          `json:"title" binding:"required,max=200"`
	CreationType  string          `json:"creation_type" binding:"required,oneof=ai manual"`
	RepoURL       string          `json:"repo_url,omitempty" binding:"omitempty,url"`
	StartStruct   string          `json:"start_struct,omitempty" binding:"omitempty,max=100"`
	AnalysisDepth int             `json:"analysis_depth,omitempty" binding:"omitempty,min=1,max=10"`
	ModelName     string          `json:"model_name,omitempty" binding:"omitempty,max=50"`
	DiagramData   json.RawMessage `json:"diagram_data,omitempty"`
}

// CreateAnalysisResponse 创建分析响应
type CreateAnalysisResponse struct {
	AnalysisID int64 `json:"analysis_id"`
	JobID      int64 `json:"job_id,omitempty"`
}

// UpdateAnalysisRequest 更新分析请求
type UpdateAnalysisRequest struct {
	Title       *string         `json:"title,omitempty" binding:"omitempty,max=200"`
	Description *string         `json:"description,omitempty" binding:"omitempty,max=2000"`
	DiagramData json.RawMessage `json:"diagram_data,omitempty"`
}

// ShareAnalysisRequest 分享分析请求
type ShareAnalysisRequest struct {
	ShareTitle       string   `json:"share_title" binding:"required,max=200"`
	ShareDescription string   `json:"share_description,omitempty" binding:"omitempty,max=2000"`
	Tags             []string `json:"tags,omitempty" binding:"omitempty,max=5,dive,max=20"`
}

// AnalysisListItem 分析列表项
type AnalysisListItem struct {
	ID           int64    `json:"id"`
	Title        string   `json:"title"`
	CreationType string   `json:"creation_type"`
	Status       string   `json:"status"`
	IsPublic     bool     `json:"is_public"`
	ViewCount    int      `json:"view_count"`
	LikeCount    int      `json:"like_count"`
	CommentCount int      `json:"comment_count"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	Tags         []string `json:"tags,omitempty"`
}

// AnalysisDetail 分析详情
type AnalysisDetail struct {
	ID               int64    `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	CreationType     string   `json:"creation_type"`
	RepoURL          string   `json:"repo_url,omitempty"`
	StartStruct      string   `json:"start_struct,omitempty"`
	AnalysisDepth    int      `json:"analysis_depth,omitempty"`
	ModelName        string   `json:"model_name,omitempty"`
	DiagramOSSURL    string   `json:"diagram_oss_url,omitempty"`
	DiagramSize      int      `json:"diagram_size,omitempty"`
	Status           string   `json:"status"`
	ErrorMessage     string   `json:"error_message,omitempty"`
	IsPublic         bool     `json:"is_public"`
	ShareTitle       string   `json:"share_title,omitempty"`
	ShareDescription string   `json:"share_description,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	ViewCount        int      `json:"view_count"`
	LikeCount        int      `json:"like_count"`
	CommentCount     int      `json:"comment_count"`
	BookmarkCount    int      `json:"bookmark_count"`
	StartedAt        string   `json:"started_at,omitempty"`
	CompletedAt      string   `json:"completed_at,omitempty"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

// CommunityAnalysisItem 社区分析列表项
type CommunityAnalysisItem struct {
	ID               int64       `json:"id"`
	ShareTitle       string      `json:"share_title"`
	ShareDescription string      `json:"share_description"`
	Tags             []string    `json:"tags"`
	Author           *AuthorInfo `json:"author"`
	ViewCount        int         `json:"view_count"`
	LikeCount        int         `json:"like_count"`
	CommentCount     int         `json:"comment_count"`
	BookmarkCount    int         `json:"bookmark_count"`
	SharedAt         string      `json:"shared_at"`
}

// AuthorInfo 作者信息
type AuthorInfo struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio,omitempty"`
}

// CommunityAnalysisDetail 社区分析详情
type CommunityAnalysisDetail struct {
	ID               int64            `json:"id"`
	ShareTitle       string           `json:"share_title"`
	ShareDescription string           `json:"share_description"`
	Tags             []string         `json:"tags"`
	Author           *AuthorInfo      `json:"author"`
	DiagramOSSURL    string           `json:"diagram_oss_url"`
	CreationType     string           `json:"creation_type"`
	RepoURL          string           `json:"repo_url,omitempty"`
	ViewCount        int              `json:"view_count"`
	LikeCount        int              `json:"like_count"`
	CommentCount     int              `json:"comment_count"`
	BookmarkCount    int              `json:"bookmark_count"`
	SharedAt         string           `json:"shared_at"`
	UserInteraction  *UserInteraction `json:"user_interaction,omitempty"`
}

// UserInteraction 用户互动状态
type UserInteraction struct {
	Liked      bool `json:"liked"`
	Bookmarked bool `json:"bookmarked"`
}

// JobStatusResponse 任务状态响应
type JobStatusResponse struct {
	JobID          int64  `json:"job_id"`
	AnalysisID     int64  `json:"analysis_id"`
	Status         string `json:"status"`
	CurrentStep    string `json:"current_step,omitempty"`
	ElapsedSeconds int    `json:"elapsed_seconds,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add analysis DTOs

- Add create/update/share requests
- Add list and detail responses
- Add community related DTOs
- Add job status response"
```

---

## Task 8: Quota Service

**Files:**
- Create: `internal/service/quota_service.go`

**Step 1: 创建 internal/service/quota_service.go**

```go
package service

import (
	"errors"
	"time"

	"github.com/qs3c/anal_go_server/config"
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
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add quota service

- Add quota check and usage
- Add depth limit validation
- Add model permission check
- Add quota reset functionality"
```

---

## Task 9: Analysis Service

**Files:**
- Create: `internal/service/analysis_service.go`

**Step 1: 创建 internal/service/analysis_service.go**

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
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add analysis service

- Add create with quota/depth/model validation
- Add get, list, update, delete operations
- Add share/unshare functionality
- Add job status query"
```

---

## Task 10: Analysis Handler

**Files:**
- Create: `internal/api/handler/analysis.go`

**Step 1: 创建 internal/api/handler/analysis.go**

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

type AnalysisHandler struct {
	analysisService *service.AnalysisService
}

func NewAnalysisHandler(analysisService *service.AnalysisService) *AnalysisHandler {
	return &AnalysisHandler{
		analysisService: analysisService,
	}
}

// Create 创建分析
// POST /api/v1/analyses
func (h *AnalysisHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	var req dto.CreateAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.analysisService.Create(userID, &req)
	if err != nil {
		switch err {
		case service.ErrQuotaExceeded:
			response.QuotaError(c, err.Error())
		case service.ErrDepthExceeded:
			response.ParamError(c, err.Error())
		case service.ErrModelDenied:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "创建成功", resp)
}

// List 获取分析列表
// GET /api/v1/analyses
func (h *AnalysisHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	items, total, err := h.analysisService.List(userID, page, pageSize, search, status)
	if err != nil {
		response.ServerError(c, "")
		return
	}

	response.SuccessPage(c, total, page, pageSize, items)
}

// Get 获取分析详情
// GET /api/v1/analyses/:id
func (h *AnalysisHandler) Get(c *gin.Context) {
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

	detail, err := h.analysisService.GetByID(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.Success(c, detail)
}

// Update 更新分析
// PUT /api/v1/analyses/:id
func (h *AnalysisHandler) Update(c *gin.Context) {
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

	var req dto.UpdateAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	detail, err := h.analysisService.Update(userID, analysisID, &req)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "更新成功", detail)
}

// Delete 删除分析
// DELETE /api/v1/analyses/:id
func (h *AnalysisHandler) Delete(c *gin.Context) {
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

	if err := h.analysisService.Delete(userID, analysisID); err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// Share 分享到广场
// POST /api/v1/analyses/:id/share
func (h *AnalysisHandler) Share(c *gin.Context) {
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

	var req dto.ShareAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	if err := h.analysisService.Share(userID, analysisID, &req); err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		case service.ErrAnalysisNotComplete:
			response.ParamError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "分享成功", nil)
}

// Unshare 取消分享
// DELETE /api/v1/analyses/:id/share
func (h *AnalysisHandler) Unshare(c *gin.Context) {
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

	if err := h.analysisService.Unshare(userID, analysisID); err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "已取消分享", nil)
}

// GetJobStatus 获取任务状态
// GET /api/v1/analyses/:id/job-status
func (h *AnalysisHandler) GetJobStatus(c *gin.Context) {
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

	status, err := h.analysisService.GetJobStatus(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.Success(c, status)
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add analysis handler

- Add CRUD endpoints
- Add share/unshare endpoints
- Add job status endpoint"
```

---

## Task 11: Models Handler

**Files:**
- Create: `internal/api/handler/models.go`

**Step 1: 创建 internal/api/handler/models.go**

```go
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
)

type ModelsHandler struct {
	cfg *config.Config
}

func NewModelsHandler(cfg *config.Config) *ModelsHandler {
	return &ModelsHandler{cfg: cfg}
}

// List 获取模型列表
// GET /api/v1/models
func (h *ModelsHandler) List(c *gin.Context) {
	models := make([]map[string]interface{}, len(h.cfg.Models))

	for i, m := range h.cfg.Models {
		models[i] = map[string]interface{}{
			"name":           m.Name,
			"display_name":   m.DisplayName,
			"required_level": m.RequiredLevel,
			"description":    m.Description,
		}
	}

	response.Success(c, gin.H{
		"models": models,
	})
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add models handler

- Add models list endpoint
- Return configured models with levels"
```

---

## Task 12: Redis Queue

**Files:**
- Create: `internal/pkg/queue/queue.go`

**Step 1: 创建 internal/pkg/queue/queue.go**

```go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Queue struct {
	client    *redis.Client
	queueName string
}

type JobMessage struct {
	JobID      int64  `json:"job_id"`
	AnalysisID int64  `json:"analysis_id"`
	UserID     int64  `json:"user_id"`
	RepoURL    string `json:"repo_url"`
	StartStruct string `json:"start_struct"`
	Depth      int    `json:"depth"`
	ModelName  string `json:"model_name"`
}

func NewQueue(client *redis.Client, queueName string) *Queue {
	return &Queue{
		client:    client,
		queueName: queueName,
	}
}

// Push 将任务加入队列
func (q *Queue) Push(ctx context.Context, msg *JobMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return q.client.LPush(ctx, q.queueName, data).Err()
}

// Pop 从队列获取任务（阻塞）
func (q *Queue) Pop(ctx context.Context, timeout time.Duration) (*JobMessage, error) {
	result, err := q.client.BRPop(ctx, timeout, q.queueName).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 超时，无任务
		}
		return nil, fmt.Errorf("failed to pop from queue: %w", err)
	}

	if len(result) < 2 {
		return nil, nil
	}

	var msg JobMessage
	if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// Length 获取队列长度
func (q *Queue) Length(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.queueName).Result()
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add Redis queue

- Add job message struct
- Add push/pop operations
- Add queue length query"
```

---

## Task 13: WebSocket Hub

**Files:**
- Create: `internal/pkg/ws/hub.go`

**Step 1: 安装 WebSocket 依赖**

```bash
go get github.com/gorilla/websocket
```

**Step 2: 创建 internal/pkg/ws/hub.go**

```go
package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	connections map[int64]*websocket.Conn
	mu          sync.RWMutex
	register    chan *Client
	unregister  chan *Client
	broadcast   chan *Message
}

type Client struct {
	UserID int64
	Conn   *websocket.Conn
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[int64]*websocket.Conn),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *Message, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// 关闭旧连接
			if oldConn, ok := h.connections[client.UserID]; ok {
				oldConn.Close()
			}
			h.connections[client.UserID] = client.Conn
			h.mu.Unlock()
			log.Printf("User %d connected, total connections: %d", client.UserID, len(h.connections))

		case client := <-h.unregister:
			h.mu.Lock()
			if conn, ok := h.connections[client.UserID]; ok && conn == client.Conn {
				delete(h.connections, client.UserID)
				conn.Close()
			}
			h.mu.Unlock()
			log.Printf("User %d disconnected", client.UserID)
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// SendToUser 向指定用户发送消息
func (h *Hub) SendToUser(userID int64, msg *Message) error {
	h.mu.RLock()
	conn, ok := h.connections[userID]
	h.mu.RUnlock()

	if !ok {
		return nil // 用户不在线，忽略
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// IsOnline 检查用户是否在线
func (h *Hub) IsOnline(userID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.connections[userID]
	return ok
}

// ConnectionCount 获取在线连接数
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}
```

**Step 3: 验证编译**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add .
git commit -m "feat: add WebSocket hub

- Add connection management
- Add user message sending
- Add online status check"
```

---

## Task 14: WebSocket Handler

**Files:**
- Create: `internal/api/handler/websocket.go`

**Step 1: 创建 internal/api/handler/websocket.go**

```go
package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/qs3c/anal_go_server/internal/pkg/jwt"
	"github.com/qs3c/anal_go_server/internal/pkg/ws"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 生产环境需要验证 Origin
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebSocketHandler struct {
	hub       *ws.Hub
	jwtSecret string
}

func NewWebSocketHandler(hub *ws.Hub, jwtSecret string) *WebSocketHandler {
	return &WebSocketHandler{
		hub:       hub,
		jwtSecret: jwtSecret,
	}
}

// Handle WebSocket 连接处理
// GET /api/v1/ws?token=xxx
func (h *WebSocketHandler) Handle(c *gin.Context) {
	// 验证 JWT Token
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	claims, err := jwt.ParseToken(token, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// 升级连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &ws.Client{
		UserID: claims.UserID,
		Conn:   conn,
	}

	h.hub.Register(client)

	// 保持连接，读取消息（主要用于检测断开）
	go func() {
		defer h.hub.Unregister(client)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add WebSocket handler

- Add JWT token validation
- Add connection upgrade
- Add connection lifecycle management"
```

---

## Task 15: 更新 Router 添加新路由

**Files:**
- Modify: `internal/api/router.go`

**Step 1: 更新 internal/api/router.go**

```go
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/api/handler"
	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/pkg/ws"
)

type Router struct {
	authHandler      *handler.AuthHandler
	userHandler      *handler.UserHandler
	analysisHandler  *handler.AnalysisHandler
	modelsHandler    *handler.ModelsHandler
	websocketHandler *handler.WebSocketHandler
	cfg              *config.Config
}

func NewRouter(
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	analysisHandler *handler.AnalysisHandler,
	modelsHandler *handler.ModelsHandler,
	websocketHandler *handler.WebSocketHandler,
	cfg *config.Config,
) *Router {
	return &Router{
		authHandler:      authHandler,
		userHandler:      userHandler,
		analysisHandler:  analysisHandler,
		modelsHandler:    modelsHandler,
		websocketHandler: websocketHandler,
		cfg:              cfg,
	}
}

func (r *Router) Setup() *gin.Engine {
	if r.cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS(r.cfg.CORS))

	api := engine.Group("/api/v1")
	{
		// WebSocket
		api.GET("/ws", r.websocketHandler.Handle)

		// 公开接口 - 认证
		auth := api.Group("/auth")
		{
			auth.POST("/register", r.authHandler.Register)
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/verify-email", r.authHandler.VerifyEmail)
			auth.GET("/github", r.authHandler.GithubAuth)
			auth.GET("/github/callback", r.authHandler.GithubCallback)
		}

		// 公开接口 - 模型
		api.GET("/models", r.modelsHandler.List)

		// 需要认证的接口
		authenticated := api.Group("")
		authenticated.Use(middleware.Auth(r.cfg.JWT.Secret))
		{
			// 用户
			user := authenticated.Group("/user")
			{
				user.GET("/profile", r.userHandler.GetProfile)
				user.PUT("/profile", r.userHandler.UpdateProfile)
				user.POST("/avatar", r.userHandler.UploadAvatar)
			}

			// 分析
			analyses := authenticated.Group("/analyses")
			{
				analyses.POST("", r.analysisHandler.Create)
				analyses.GET("", r.analysisHandler.List)
				analyses.GET("/:id", r.analysisHandler.Get)
				analyses.PUT("/:id", r.analysisHandler.Update)
				analyses.DELETE("/:id", r.analysisHandler.Delete)
				analyses.POST("/:id/share", r.analysisHandler.Share)
				analyses.DELETE("/:id/share", r.analysisHandler.Unshare)
				analyses.GET("/:id/job-status", r.analysisHandler.GetJobStatus)
			}
		}

		// 公开接口 - 社区（可选认证）
		community := api.Group("/community")
		community.Use(middleware.OptionalAuth(r.cfg.JWT.Secret))
		{
			// TODO: Community APIs
		}
	}

	return engine
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: update router with all Phase 2 routes

- Add GitHub OAuth routes
- Add user profile routes
- Add analysis CRUD routes
- Add WebSocket route
- Add models route"
```

---

## Task 16: 更新 Server Main

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: 更新 cmd/server/main.go 注入所有依赖**

```go
package main

import (
	"fmt"
	"log"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/api"
	"github.com/qs3c/anal_go_server/internal/api/handler"
	"github.com/qs3c/anal_go_server/internal/database"
	"github.com/qs3c/anal_go_server/internal/pkg/queue"
	"github.com/qs3c/anal_go_server/internal/pkg/ws"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/service"
)

func main() {
	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := database.NewMySQL(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	log.Println("Database connected")

	// 初始化 Redis
	rdb, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}
	log.Println("Redis connected")

	// 初始化 Queue
	jobQueue := queue.NewQueue(rdb, cfg.Queue.AnalysisQueue)
	_ = jobQueue // TODO: 传给 analysis service

	// 初始化 WebSocket Hub
	wsHub := ws.NewHub()
	go wsHub.Run()
	log.Println("WebSocket hub started")

	// 初始化 Repository
	userRepo := repository.NewUserRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)

	// 初始化 Service
	authService := service.NewAuthService(userRepo, cfg)
	userService := service.NewUserService(userRepo, cfg)
	quotaService := service.NewQuotaService(userRepo, cfg)
	analysisService := service.NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, cfg)

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	analysisHandler := handler.NewAnalysisHandler(analysisService)
	modelsHandler := handler.NewModelsHandler(cfg)
	websocketHandler := handler.NewWebSocketHandler(wsHub, cfg.JWT.Secret)

	// 初始化 Router
	router := api.NewRouter(
		authHandler,
		userHandler,
		analysisHandler,
		modelsHandler,
		websocketHandler,
		cfg,
	)
	engine := router.Setup()

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: update server main with Phase 2 dependencies

- Add repository initialization
- Add service initialization
- Add handler initialization
- Add WebSocket hub
- Wire up all dependencies"
```

---

## Task 17: Worker Main 基础结构

**Files:**
- Modify: `cmd/worker/main.go`

**Step 1: 更新 cmd/worker/main.go**

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/database"
	"github.com/qs3c/anal_go_server/internal/pkg/queue"
	"github.com/qs3c/anal_go_server/internal/repository"
)

func main() {
	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := database.NewMySQL(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	log.Println("Database connected")

	// 初始化 Redis
	rdb, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}
	log.Println("Redis connected")

	// 初始化 Queue
	jobQueue := queue.NewQueue(rdb, cfg.Queue.AnalysisQueue)

	// 初始化 Repository
	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)

	// 创建 context 用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	log.Printf("Worker started, max workers: %d", cfg.Queue.MaxWorkers)

	// 启动 worker 循环
	for i := 0; i < cfg.Queue.MaxWorkers; i++ {
		go func(workerID int) {
			for {
				select {
				case <-ctx.Done():
					log.Printf("Worker %d shutting down", workerID)
					return
				default:
					// 从队列获取任务
					msg, err := jobQueue.Pop(ctx, 5*time.Second)
					if err != nil {
						if ctx.Err() != nil {
							return
						}
						log.Printf("Worker %d: failed to pop job: %v", workerID, err)
						continue
					}

					if msg == nil {
						continue // 超时，继续等待
					}

					log.Printf("Worker %d: processing job %d", workerID, msg.JobID)
					processJob(ctx, msg, analysisRepo, jobRepo, userRepo, cfg)
				}
			}
		}(i)
	}

	// 等待 context 取消
	<-ctx.Done()
	log.Println("Worker shutdown complete")
}

func processJob(
	ctx context.Context,
	msg *queue.JobMessage,
	analysisRepo *repository.AnalysisRepository,
	jobRepo *repository.JobRepository,
	userRepo *repository.UserRepository,
	cfg *config.Config,
) {
	job, err := jobRepo.GetByID(msg.JobID)
	if err != nil {
		log.Printf("Failed to get job %d: %v", msg.JobID, err)
		return
	}

	// 更新状态为处理中
	now := time.Now()
	job.Status = "processing"
	job.StartedAt = &now
	jobRepo.Update(job)

	analysisRepo.UpdateStatus(job.AnalysisID, "analyzing")

	// TODO: 实际的分析逻辑
	// 1. Clone 仓库
	// 2. 调用 anal_go_agent 分析
	// 3. 上传结果到 OSS
	// 4. 更新数据库

	// 模拟处理
	time.Sleep(2 * time.Second)

	// 标记完成（这里是模拟）
	job.Status = "completed"
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	job.ElapsedSeconds = int(completedAt.Sub(*job.StartedAt).Seconds())
	jobRepo.Update(job)

	analysisRepo.UpdateStatus(job.AnalysisID, "completed")

	log.Printf("Job %d completed in %d seconds", job.ID, job.ElapsedSeconds)
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add worker main with basic structure

- Add graceful shutdown handling
- Add multi-worker support
- Add job processing skeleton
- TODO: actual analysis implementation"
```

---

## Task 18: 整理依赖并最终验证

**Step 1: 整理 go.mod**

```bash
go mod tidy
```

**Step 2: 验证项目可以编译**

```bash
go build ./...
```

**Step 3: Final Commit**

```bash
git add .
git commit -m "chore: tidy dependencies and finalize Phase 2

Phase 2 complete:
- GitHub OAuth integration
- User profile management
- Analysis CRUD with quota/permission checks
- Redis queue for job processing
- WebSocket hub for real-time updates
- Worker process skeleton
- Models API

TODO for Phase 3:
- OSS client implementation
- Community/Comment/Interaction APIs
- Actual analysis worker implementation"
```

---

## Summary

Phase 2 完成后的功能：

| 功能 | 状态 |
|------|------|
| GitHub OAuth | ✅ |
| User Profile API | ✅ |
| Analysis CRUD | ✅ |
| Quota Service | ✅ |
| Analysis Service | ✅ |
| Redis Queue | ✅ |
| WebSocket Hub | ✅ |
| Worker Process | ✅ (skeleton) |
| Models API | ✅ |

**下一步（Phase 3）：**
- OSS Client 实现
- Community API（广场列表、详情）
- Interaction API（点赞、收藏）
- Comment API（评论列表、发表、删除）
- Worker 完整实现（调用 anal_go_agent）
