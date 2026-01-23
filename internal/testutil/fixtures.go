package testutil

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

// TestUser 创建测试用户
func TestUser(t *testing.T, db *gorm.DB, opts ...func(*model.User)) *model.User {
	t.Helper()

	email := fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())
	passwordHash := "$2a$10$abcdefghijklmnopqrstuvwxyz123456" // bcrypt hash placeholder
	user := &model.User{
		Username:          fmt.Sprintf("testuser_%d", time.Now().UnixNano()%10000),
		Email:             &email,
		PasswordHash:      &passwordHash,
		SubscriptionLevel: "free",
		DailyQuota:        5,
		QuotaUsedToday:    0,
		EmailVerified:     true,
	}

	for _, opt := range opts {
		opt(user)
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// WithUsername 设置用户名
func WithUsername(username string) func(*model.User) {
	return func(u *model.User) {
		u.Username = username
	}
}

// WithEmail 设置邮箱
func WithEmail(email string) func(*model.User) {
	return func(u *model.User) {
		u.Email = &email
	}
}

// WithSubscription 设置订阅级别
func WithSubscription(level string, quota int) func(*model.User) {
	return func(u *model.User) {
		u.SubscriptionLevel = level
		u.DailyQuota = quota
	}
}

// WithQuotaUsed 设置已使用配额
func WithQuotaUsed(used int) func(*model.User) {
	return func(u *model.User) {
		u.QuotaUsedToday = used
	}
}

// TestAnalysis 创建测试分析
func TestAnalysis(t *testing.T, db *gorm.DB, userID int64, opts ...func(*model.Analysis)) *model.Analysis {
	t.Helper()

	analysis := &model.Analysis{
		UserID:       userID,
		Title:        fmt.Sprintf("Test Analysis %d", time.Now().UnixNano()%10000),
		CreationType: "manual",
		Status:       "completed",
	}

	for _, opt := range opts {
		opt(analysis)
	}

	if err := db.Create(analysis).Error; err != nil {
		t.Fatalf("Failed to create test analysis: %v", err)
	}

	return analysis
}

// WithTitle 设置分析标题
func WithTitle(title string) func(*model.Analysis) {
	return func(a *model.Analysis) {
		a.Title = title
	}
}

// WithCreationType 设置创建类型
func WithCreationType(creationType string) func(*model.Analysis) {
	return func(a *model.Analysis) {
		a.CreationType = creationType
	}
}

// WithStatus 设置状态
func WithStatus(status string) func(*model.Analysis) {
	return func(a *model.Analysis) {
		a.Status = status
	}
}

// WithPublic 设置为公开
func WithPublic(isPublic bool) func(*model.Analysis) {
	return func(a *model.Analysis) {
		a.IsPublic = isPublic
		if isPublic {
			now := time.Now()
			a.SharedAt = &now
			a.ShareTitle = a.Title
		}
	}
}

// WithRepoURL 设置仓库 URL
func WithRepoURL(url string) func(*model.Analysis) {
	return func(a *model.Analysis) {
		a.RepoURL = url
	}
}

// TestComment 创建测试评论
func TestComment(t *testing.T, db *gorm.DB, userID, analysisID int64, content string) *model.Comment {
	t.Helper()

	comment := &model.Comment{
		UserID:     userID,
		AnalysisID: analysisID,
		Content:    content,
	}

	if err := db.Create(comment).Error; err != nil {
		t.Fatalf("Failed to create test comment: %v", err)
	}

	return comment
}

// TestReply 创建测试回复
func TestReply(t *testing.T, db *gorm.DB, userID, analysisID, parentID int64, content string) *model.Comment {
	t.Helper()

	comment := &model.Comment{
		UserID:     userID,
		AnalysisID: analysisID,
		ParentID:   &parentID,
		Content:    content,
	}

	if err := db.Create(comment).Error; err != nil {
		t.Fatalf("Failed to create test reply: %v", err)
	}

	return comment
}

// TestInteraction 创建测试互动
func TestInteraction(t *testing.T, db *gorm.DB, userID, analysisID int64, interactionType string) *model.Interaction {
	t.Helper()

	interaction := &model.Interaction{
		UserID:     userID,
		AnalysisID: analysisID,
		Type:       interactionType,
	}

	if err := db.Create(interaction).Error; err != nil {
		t.Fatalf("Failed to create test interaction: %v", err)
	}

	return interaction
}

// TestJob 创建测试任务
func TestJob(t *testing.T, db *gorm.DB, userID, analysisID int64, status string) *model.AnalysisJob {
	t.Helper()

	job := &model.AnalysisJob{
		AnalysisID:  analysisID,
		UserID:      userID,
		RepoURL:     "https://github.com/example/repo",
		StartStruct: "main.Config",
		Depth:       3,
		ModelName:   "gpt-3.5-turbo",
		Status:      status,
	}

	if err := db.Create(job).Error; err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	return job
}
