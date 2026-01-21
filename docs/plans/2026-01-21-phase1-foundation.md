# Phase 1: 基础架构 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 搭建后端项目基础架构，包括项目结构、数据库、配置管理和完整的认证系统。

**Architecture:** 分层架构（Handler → Service → Repository → Model），使用 Gin 框架处理 HTTP 请求，GORM 作为 ORM，JWT 进行身份验证。所有 JSON 字段使用 snake_case 命名。

**Tech Stack:** Go 1.22+, Gin, GORM, MySQL, Redis, JWT, bcrypt, Viper

---

## Task 1: 项目初始化与目录结构

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`
- Create: `cmd/worker/main.go`
- Create: `Makefile`
- Create: `.env.example`

**Step 1: 初始化 Go module**

```bash
go mod init github.com/qs3c/anal_go_server
```

**Step 2: 创建目录结构**

```bash
mkdir -p cmd/server cmd/worker
mkdir -p internal/api/handler internal/api/middleware
mkdir -p internal/service internal/repository internal/model/dto
mkdir -p internal/pkg/jwt internal/pkg/response internal/pkg/oss internal/pkg/ws internal/pkg/queue
mkdir -p config migrations
```

**Step 3: 创建 cmd/server/main.go 占位文件**

```go
package main

func main() {
	println("API Server - TODO")
}
```

**Step 4: 创建 cmd/worker/main.go 占位文件**

```go
package main

func main() {
	println("Worker - TODO")
}
```

**Step 5: 创建 Makefile**

```makefile
.PHONY: dev-server dev-worker build test clean migrate-up migrate-down

# Development
dev-server:
	go run cmd/server/main.go

dev-worker:
	go run cmd/worker/main.go

# Build
build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/worker cmd/worker/main.go

# Test
test:
	go test -v ./...

# Clean
clean:
	rm -rf bin/

# Database migrations (requires golang-migrate)
migrate-up:
	migrate -path migrations -database "mysql://root:password@tcp(localhost:3306)/go_analyzer" up

migrate-down:
	migrate -path migrations -database "mysql://root:password@tcp(localhost:3306)/go_analyzer" down 1

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)
```

**Step 6: 创建 .env.example**

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
GIN_MODE=debug

# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=go_analyzer

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your_jwt_secret_key_change_in_production
JWT_EXPIRE_HOURS=168

# OSS (Aliyun)
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
OSS_ACCESS_KEY_ID=
OSS_ACCESS_KEY_SECRET=
OSS_BUCKET_NAME=go-analyzer
OSS_CDN_DOMAIN=https://cdn.example.com

# OAuth
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_REDIRECT_URI=http://localhost:8080/api/v1/auth/github/callback

# LLM API Keys
OPENAI_API_KEY=
ANTHROPIC_API_KEY=

# Email
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
EMAIL_FROM=noreply@example.com

# Frontend URL (for CORS)
FRONTEND_URL=http://localhost:3000
```

**Step 7: 验证项目可以编译**

```bash
go build ./...
```
Expected: 无错误输出

**Step 8: Commit**

```bash
git add .
git commit -m "chore: initialize project structure

- Add go.mod with module github.com/qs3c/anal_go_server
- Create directory structure (cmd, internal, config, migrations)
- Add Makefile with common commands
- Add .env.example with all configuration variables"
```

---

## Task 2: 配置管理

**Files:**
- Create: `config/config.go`
- Create: `config.yaml`

**Step 1: 安装依赖**

```bash
go get github.com/spf13/viper
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/mysql
go get github.com/go-redis/redis/v8
```

**Step 2: 创建 config/config.go**

```go
package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	OSS          OSSConfig          `mapstructure:"oss"`
	OAuth        OAuthConfig        `mapstructure:"oauth"`
	Email        EmailConfig        `mapstructure:"email"`
	Queue        QueueConfig        `mapstructure:"queue"`
	CORS         CORSConfig         `mapstructure:"cors"`
	Subscription SubscriptionConfig `mapstructure:"subscription"`
	Models       []ModelConfig      `mapstructure:"models"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type OSSConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	BucketName      string `mapstructure:"bucket_name"`
	CDNDomain       string `mapstructure:"cdn_domain"`
}

type OAuthConfig struct {
	Github GithubOAuthConfig `mapstructure:"github"`
	Wechat WechatOAuthConfig `mapstructure:"wechat"`
}

type GithubOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURI  string `mapstructure:"redirect_uri"`
}

type WechatOAuthConfig struct {
	AppID       string `mapstructure:"app_id"`
	AppSecret   string `mapstructure:"app_secret"`
	RedirectURI string `mapstructure:"redirect_uri"`
}

type EmailConfig struct {
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

type QueueConfig struct {
	AnalysisQueue string `mapstructure:"analysis_queue"`
	MaxWorkers    int    `mapstructure:"max_workers"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

type SubscriptionConfig struct {
	Levels map[string]SubscriptionLevel `mapstructure:"levels"`
}

type SubscriptionLevel struct {
	DailyQuota int     `mapstructure:"daily_quota"`
	MaxDepth   int     `mapstructure:"max_depth"`
	Price      float64 `mapstructure:"price"`
}

type ModelConfig struct {
	Name          string `mapstructure:"name"`
	DisplayName   string `mapstructure:"display_name"`
	RequiredLevel string `mapstructure:"required_level"`
	APIKey        string `mapstructure:"api_key"`
	APIProvider   string `mapstructure:"api_provider"`
	Description   string `mapstructure:"description"`
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 环境变量覆盖
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
```

**Step 3: 创建 config.yaml**

```yaml
server:
  host: 0.0.0.0
  port: 8080
  mode: debug

database:
  host: localhost
  port: 3306
  username: root
  password: password
  database: go_analyzer
  max_idle_conns: 10
  max_open_conns: 100

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 10

jwt:
  secret: your_jwt_secret_key_change_in_production
  expire_hours: 168

oss:
  endpoint: oss-cn-hangzhou.aliyuncs.com
  access_key_id: ""
  access_key_secret: ""
  bucket_name: go-analyzer
  cdn_domain: https://cdn.example.com

oauth:
  github:
    client_id: ""
    client_secret: ""
    redirect_uri: http://localhost:8080/api/v1/auth/github/callback
  wechat:
    app_id: ""
    app_secret: ""
    redirect_uri: http://localhost:8080/api/v1/auth/wechat/callback

email:
  smtp_host: smtp.gmail.com
  smtp_port: 587
  username: ""
  password: ""
  from: noreply@example.com

queue:
  analysis_queue: analysis_jobs
  max_workers: 5

cors:
  allowed_origins:
    - http://localhost:3000
    - http://localhost:5173
  allowed_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  allowed_headers:
    - Authorization
    - Content-Type

subscription:
  levels:
    free:
      daily_quota: 5
      max_depth: 3
    basic:
      daily_quota: 30
      max_depth: 5
      price: 19.9
    pro:
      daily_quota: 100
      max_depth: 10
      price: 49.9

models:
  - name: gpt-3.5-turbo
    display_name: GPT-3.5 Turbo
    required_level: free
    api_provider: openai
    description: 基础模型，适合简单分析
  - name: claude-haiku
    display_name: Claude Haiku
    required_level: free
    api_provider: anthropic
    description: 基础模型，快速分析
  - name: gpt-4o-mini
    display_name: GPT-4o Mini
    required_level: basic
    api_provider: openai
    description: 中级模型，平衡速度和质量
  - name: gpt-4
    display_name: GPT-4
    required_level: pro
    api_provider: openai
    description: 高级模型，适合复杂分析
  - name: claude-sonnet
    display_name: Claude Sonnet
    required_level: pro
    api_provider: anthropic
    description: 高级模型，高质量分析
```

**Step 4: 验证配置加载**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add configuration management

- Add config package with viper
- Support YAML config file with env var override
- Define all config structs matching requirements"
```

---

## Task 3: 统一响应封装

**Files:**
- Create: `internal/pkg/response/response.go`

**Step 1: 创建 internal/pkg/response/response.go**

```go
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误码定义
const (
	CodeSuccess          = 0
	CodeParamError       = 1000
	CodeAuthFailed       = 1001
	CodePermissionDenied = 1002
	CodeResourceNotFound = 1003
	CodeQuotaExceeded    = 1004
	CodeDuplicateAction  = 1005
	CodeServerError      = 5000
)

// 错误码对应的默认消息
var codeMessages = map[int]string{
	CodeSuccess:          "success",
	CodeParamError:       "参数错误",
	CodeAuthFailed:       "认证失败",
	CodePermissionDenied: "权限不足",
	CodeResourceNotFound: "资源不存在",
	CodeQuotaExceeded:    "配额不足",
	CodeDuplicateAction:  "重复操作",
	CodeServerError:      "服务器内部错误",
}

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PageData 分页数据结构
type PageData struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Items    interface{} `json:"items"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带自定义消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// SuccessPage 分页成功响应
func SuccessPage(c *gin.Context, total int64, page, pageSize int, items interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data: PageData{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
			Items:    items,
		},
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	if message == "" {
		message = codeMessages[code]
	}
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// ParamError 参数错误
func ParamError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeParamError]
	}
	Error(c, CodeParamError, message)
}

// AuthError 认证失败
func AuthError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeAuthFailed]
	}
	Error(c, CodeAuthFailed, message)
}

// PermissionError 权限不足
func PermissionError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodePermissionDenied]
	}
	Error(c, CodePermissionDenied, message)
}

// NotFoundError 资源不存在
func NotFoundError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeResourceNotFound]
	}
	Error(c, CodeResourceNotFound, message)
}

// QuotaError 配额不足
func QuotaError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeQuotaExceeded]
	}
	Error(c, CodeQuotaExceeded, message)
}

// DuplicateError 重复操作
func DuplicateError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeDuplicateAction]
	}
	Error(c, CodeDuplicateAction, message)
}

// ServerError 服务器错误
func ServerError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeServerError]
	}
	Error(c, CodeServerError, message)
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add unified response package

- Define error codes matching frontend contract
- Add Response struct with code/message/data
- Add helper functions for common responses
- Add pagination support"
```

---

## Task 4: 数据库模型定义

**Files:**
- Create: `internal/model/user.go`
- Create: `internal/model/analysis.go`
- Create: `internal/model/comment.go`
- Create: `internal/model/interaction.go`
- Create: `internal/model/job.go`
- Create: `internal/model/subscription.go`

**Step 1: 创建 internal/model/user.go**

```go
package model

import (
	"time"
)

type User struct {
	ID                    int64      `gorm:"primaryKey" json:"id"`
	Username              string     `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Email                 *string    `gorm:"size:100;uniqueIndex" json:"email,omitempty"`
	PasswordHash          *string    `gorm:"size:255" json:"-"`
	AvatarURL             string     `gorm:"size:500" json:"avatar_url"`
	Bio                   string     `gorm:"type:text" json:"bio"`
	GithubID              *string    `gorm:"size:50;uniqueIndex" json:"-"`
	WechatOpenID          *string    `gorm:"size:100;uniqueIndex" json:"-"`
	SubscriptionLevel     string     `gorm:"size:20;default:free" json:"subscription_level"`
	DailyQuota            int        `gorm:"default:5" json:"daily_quota"`
	QuotaUsedToday        int        `gorm:"default:0" json:"quota_used_today"`
	QuotaResetAt          *time.Time `json:"quota_reset_at,omitempty"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at,omitempty"`
	EmailVerified         bool       `gorm:"default:false" json:"email_verified"`
	VerificationCode      *string    `gorm:"size:100" json:"-"`
	VerificationExpiresAt *time.Time `json:"-"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
```

**Step 2: 创建 internal/model/analysis.go**

```go
package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// StringArray 用于 JSON 数组字段
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

type Analysis struct {
	ID               int64       `gorm:"primaryKey" json:"id"`
	UserID           int64       `gorm:"not null;index" json:"user_id"`
	Title            string      `gorm:"size:200;not null" json:"title"`
	Description      string      `gorm:"type:text" json:"description"`
	CreationType     string      `gorm:"size:20;not null" json:"creation_type"` // ai, manual
	RepoURL          string      `gorm:"size:500" json:"repo_url,omitempty"`
	StartStruct      string      `gorm:"size:100" json:"start_struct,omitempty"`
	AnalysisDepth    int         `json:"analysis_depth,omitempty"`
	ModelName        string      `gorm:"size:50" json:"model_name,omitempty"`
	DiagramOSSURL    string      `gorm:"size:500" json:"diagram_oss_url,omitempty"`
	DiagramSize      int         `json:"diagram_size,omitempty"`
	Status           string      `gorm:"size:20;default:draft;index" json:"status"` // draft, pending, analyzing, completed, failed
	ErrorMessage     string      `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt        *time.Time  `json:"started_at,omitempty"`
	CompletedAt      *time.Time  `json:"completed_at,omitempty"`
	IsPublic         bool        `gorm:"default:false;index" json:"is_public"`
	SharedAt         *time.Time  `gorm:"index" json:"shared_at,omitempty"`
	ShareTitle       string      `gorm:"size:200" json:"share_title,omitempty"`
	ShareDescription string      `gorm:"type:text" json:"share_description,omitempty"`
	Tags             StringArray `gorm:"type:json" json:"tags,omitempty"`
	ViewCount        int         `gorm:"default:0" json:"view_count"`
	LikeCount        int         `gorm:"default:0" json:"like_count"`
	CommentCount     int         `gorm:"default:0" json:"comment_count"`
	BookmarkCount    int         `gorm:"default:0" json:"bookmark_count"`
	CreatedAt        time.Time   `gorm:"index" json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Analysis) TableName() string {
	return "analyses"
}
```

**Step 3: 创建 internal/model/comment.go**

```go
package model

import (
	"time"
)

type Comment struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	UserID     int64     `gorm:"not null;index" json:"user_id"`
	AnalysisID int64     `gorm:"not null;index" json:"analysis_id"`
	ParentID   *int64    `gorm:"index" json:"parent_id,omitempty"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// 关联
	User    *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Replies []*Comment `gorm:"-" json:"replies,omitempty"`
}

func (Comment) TableName() string {
	return "comments"
}
```

**Step 4: 创建 internal/model/interaction.go**

```go
package model

import (
	"time"
)

type Interaction struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	UserID     int64     `gorm:"not null;index" json:"user_id"`
	AnalysisID int64     `gorm:"not null;index" json:"analysis_id"`
	Type       string    `gorm:"size:20;not null" json:"type"` // like, bookmark
	CreatedAt  time.Time `json:"created_at"`
}

func (Interaction) TableName() string {
	return "interactions"
}
```

**Step 5: 创建 internal/model/job.go**

```go
package model

import (
	"time"
)

type AnalysisJob struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	AnalysisID     int64      `gorm:"not null;index" json:"analysis_id"`
	UserID         int64      `gorm:"not null;index" json:"user_id"`
	RepoURL        string     `gorm:"size:500;not null" json:"repo_url"`
	StartStruct    string     `gorm:"size:100;not null" json:"start_struct"`
	Depth          int        `gorm:"not null" json:"depth"`
	ModelName      string     `gorm:"size:50;not null" json:"model_name"`
	Status         string     `gorm:"size:20;default:queued;index" json:"status"` // queued, processing, completed, failed, cancelled
	CurrentStep    string     `gorm:"size:200" json:"current_step,omitempty"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt      time.Time  `gorm:"index" json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ElapsedSeconds int        `json:"elapsed_seconds,omitempty"`
}

func (AnalysisJob) TableName() string {
	return "analysis_jobs"
}
```

**Step 6: 创建 internal/model/subscription.go**

```go
package model

import (
	"time"
)

type Subscription struct {
	ID            int64      `gorm:"primaryKey" json:"id"`
	UserID        int64      `gorm:"not null;index" json:"user_id"`
	Plan          string     `gorm:"size:20;not null" json:"plan"` // basic, pro
	Amount        float64    `gorm:"type:decimal(10,2)" json:"amount,omitempty"`
	DailyQuota    int        `json:"daily_quota"`
	StartedAt     time.Time  `gorm:"not null" json:"started_at"`
	ExpiresAt     time.Time  `gorm:"not null;index" json:"expires_at"`
	Status        string     `gorm:"size:20;default:active;index" json:"status"` // active, expired, cancelled
	PaymentMethod string     `gorm:"size:20" json:"payment_method,omitempty"`    // wechat, alipay
	TransactionID string     `gorm:"size:100" json:"transaction_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}
```

**Step 7: 验证编译**

```bash
go build ./...
```

**Step 8: Commit**

```bash
git add .
git commit -m "feat: add database models

- Add User model with auth and quota fields
- Add Analysis model with sharing and stats
- Add Comment model with reply support
- Add Interaction model for likes/bookmarks
- Add AnalysisJob model for task queue
- Add Subscription model for billing"
```

---

## Task 5: 数据库迁移文件

**Files:**
- Create: `migrations/000001_create_users.up.sql`
- Create: `migrations/000001_create_users.down.sql`
- Create: `migrations/000002_create_analyses.up.sql`
- Create: `migrations/000002_create_analyses.down.sql`
- Create: `migrations/000003_create_interactions.up.sql`
- Create: `migrations/000003_create_interactions.down.sql`
- Create: `migrations/000004_create_comments.up.sql`
- Create: `migrations/000004_create_comments.down.sql`
- Create: `migrations/000005_create_analysis_jobs.up.sql`
- Create: `migrations/000005_create_analysis_jobs.down.sql`
- Create: `migrations/000006_create_subscriptions.up.sql`
- Create: `migrations/000006_create_subscriptions.down.sql`

**Step 1: 创建 migrations/000001_create_users.up.sql**

```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL COMMENT '用户名',
    email VARCHAR(100) UNIQUE COMMENT '邮箱',
    password_hash VARCHAR(255) COMMENT '密码哈希',
    avatar_url VARCHAR(500) DEFAULT '' COMMENT '头像URL',
    bio TEXT COMMENT '个人简介',
    github_id VARCHAR(50) UNIQUE COMMENT 'GitHub ID',
    wechat_openid VARCHAR(100) UNIQUE COMMENT '微信OpenID',
    subscription_level ENUM('free', 'basic', 'pro') DEFAULT 'free' COMMENT '套餐级别',
    daily_quota INT DEFAULT 5 COMMENT '每日配额',
    quota_used_today INT DEFAULT 0 COMMENT '今日已用配额',
    quota_reset_at DATETIME COMMENT '配额重置时间',
    subscription_expires_at DATETIME COMMENT '订阅过期时间',
    email_verified BOOLEAN DEFAULT FALSE COMMENT '邮箱是否验证',
    verification_code VARCHAR(100) COMMENT '验证码',
    verification_expires_at DATETIME COMMENT '验证码过期时间',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_github_id (github_id),
    INDEX idx_wechat_openid (wechat_openid),
    INDEX idx_email (email),
    INDEX idx_verification_code (verification_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';
```

**Step 2: 创建 migrations/000001_create_users.down.sql**

```sql
DROP TABLE IF EXISTS users;
```

**Step 3: 创建 migrations/000002_create_analyses.up.sql**

```sql
CREATE TABLE analyses (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    title VARCHAR(200) NOT NULL COMMENT '项目名称',
    description TEXT COMMENT '项目描述',
    creation_type ENUM('ai', 'manual') NOT NULL COMMENT '创建方式',
    repo_url VARCHAR(500) COMMENT 'GitHub仓库地址',
    start_struct VARCHAR(100) COMMENT '起始结构体',
    analysis_depth INT COMMENT '分析深度',
    model_name VARCHAR(50) COMMENT '使用的模型',
    diagram_oss_url VARCHAR(500) COMMENT '框图JSON的OSS地址',
    diagram_size INT COMMENT '压缩后大小(bytes)',
    status ENUM('draft', 'pending', 'analyzing', 'completed', 'failed') DEFAULT 'draft' COMMENT '状态',
    error_message TEXT COMMENT '错误信息',
    started_at DATETIME COMMENT '分析开始时间',
    completed_at DATETIME COMMENT '分析完成时间',
    is_public BOOLEAN DEFAULT FALSE COMMENT '是否公开分享',
    shared_at DATETIME COMMENT '分享时间',
    share_title VARCHAR(200) COMMENT '分享标题',
    share_description TEXT COMMENT '分享描述',
    tags JSON COMMENT '标签数组',
    view_count INT DEFAULT 0 COMMENT '浏览数',
    like_count INT DEFAULT 0 COMMENT '点赞数',
    comment_count INT DEFAULT 0 COMMENT '评论数',
    bookmark_count INT DEFAULT 0 COMMENT '收藏数',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_is_public (is_public),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_shared_at (shared_at),
    FULLTEXT INDEX ft_share_title_desc (share_title, share_description)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分析项目表';
```

**Step 4: 创建 migrations/000002_create_analyses.down.sql**

```sql
DROP TABLE IF EXISTS analyses;
```

**Step 5: 创建 migrations/000003_create_interactions.up.sql**

```sql
CREATE TABLE interactions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    analysis_id BIGINT NOT NULL COMMENT '分析ID',
    type ENUM('like', 'bookmark') NOT NULL COMMENT '互动类型',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE,
    UNIQUE KEY uk_user_analysis_type (user_id, analysis_id, type),
    INDEX idx_analysis_type (analysis_id, type),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='互动表';
```

**Step 6: 创建 migrations/000003_create_interactions.down.sql**

```sql
DROP TABLE IF EXISTS interactions;
```

**Step 7: 创建 migrations/000004_create_comments.up.sql**

```sql
CREATE TABLE comments (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    analysis_id BIGINT NOT NULL COMMENT '分析ID',
    parent_id BIGINT COMMENT '父评论ID',
    content TEXT NOT NULL COMMENT '评论内容',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE,
    INDEX idx_analysis_id (analysis_id),
    INDEX idx_parent_id (parent_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评论表';
```

**Step 8: 创建 migrations/000004_create_comments.down.sql**

```sql
DROP TABLE IF EXISTS comments;
```

**Step 9: 创建 migrations/000005_create_analysis_jobs.up.sql**

```sql
CREATE TABLE analysis_jobs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    analysis_id BIGINT NOT NULL COMMENT '分析ID',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    repo_url VARCHAR(500) NOT NULL,
    start_struct VARCHAR(100) NOT NULL,
    depth INT NOT NULL,
    model_name VARCHAR(50) NOT NULL,
    status ENUM('queued', 'processing', 'completed', 'failed', 'cancelled') DEFAULT 'queued',
    current_step VARCHAR(200) COMMENT '当前步骤',
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME COMMENT '开始处理时间',
    completed_at DATETIME COMMENT '完成时间',
    elapsed_seconds INT COMMENT '耗时秒数',
    FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_status (status),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分析任务队列表';
```

**Step 10: 创建 migrations/000005_create_analysis_jobs.down.sql**

```sql
DROP TABLE IF EXISTS analysis_jobs;
```

**Step 11: 创建 migrations/000006_create_subscriptions.up.sql**

```sql
CREATE TABLE subscriptions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    plan ENUM('basic', 'pro') NOT NULL COMMENT '套餐',
    amount DECIMAL(10, 2) COMMENT '金额',
    daily_quota INT COMMENT '每日配额',
    started_at DATETIME NOT NULL COMMENT '生效时间',
    expires_at DATETIME NOT NULL COMMENT '过期时间',
    status ENUM('active', 'expired', 'cancelled') DEFAULT 'active',
    payment_method ENUM('wechat', 'alipay') COMMENT '支付方式',
    transaction_id VARCHAR(100) COMMENT '交易ID',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_expires_at (expires_at),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅记录表';
```

**Step 12: 创建 migrations/000006_create_subscriptions.down.sql**

```sql
DROP TABLE IF EXISTS subscriptions;
```

**Step 13: Commit**

```bash
git add .
git commit -m "feat: add database migration files

- Add users table migration
- Add analyses table with fulltext index
- Add interactions table for likes/bookmarks
- Add comments table with self-reference
- Add analysis_jobs table for task queue
- Add subscriptions table for billing"
```

---

## Task 6: JWT 工具包

**Files:**
- Create: `internal/pkg/jwt/jwt.go`

**Step 1: 安装 JWT 依赖**

```bash
go get github.com/golang-jwt/jwt/v5
```

**Step 2: 创建 internal/pkg/jwt/jwt.go**

```go
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID int64, secret string, expireHours int) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 解析并验证 JWT Token
func ParseToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
```

**Step 3: 验证编译**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add .
git commit -m "feat: add JWT utility package

- Add GenerateToken function
- Add ParseToken function with validation
- Define custom Claims struct with user_id"
```

---

## Task 7: 数据库初始化

**Files:**
- Create: `internal/database/database.go`

**Step 1: 创建 internal/database/database.go**

```go
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/qs3c/anal_go_server/config"
)

func NewMySQL(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add database initialization

- Add MySQL connection with GORM
- Configure connection pool settings"
```

---

## Task 8: Redis 初始化

**Files:**
- Create: `internal/database/redis.go`

**Step 1: 创建 internal/database/redis.go**

```go
package database

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"

	"github.com/qs3c/anal_go_server/config"
)

func NewRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	return client, nil
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add Redis initialization

- Add Redis client creation
- Configure connection pool"
```

---

## Task 9: User Repository

**Files:**
- Create: `internal/repository/user_repo.go`

**Step 1: 创建 internal/repository/user_repo.go**

```go
package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/internal/model"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetByID(id int64) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByGithubID(githubID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("github_id = ?", githubID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByVerificationCode(code string) (*model.User, error) {
	var user model.User
	err := r.db.Where("verification_code = ?", code).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) UpdateFields(id int64, fields map[string]interface{}) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(fields).Error
}

func (r *UserRepository) IncrementQuotaUsed(id int64) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("quota_used_today", gorm.Expr("quota_used_today + 1")).Error
}

func (r *UserRepository) DecrementQuotaUsed(id int64) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("quota_used_today", gorm.Expr("GREATEST(quota_used_today - 1, 0)")).Error
}

func (r *UserRepository) ResetQuota(id int64, nextResetAt time.Time) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"quota_used_today": 0,
		"quota_reset_at":   nextResetAt,
	}).Error
}

func (r *UserRepository) ResetAllQuotas(nextResetAt time.Time) error {
	return r.db.Model(&model.User{}).Where("1 = 1").Updates(map[string]interface{}{
		"quota_used_today": 0,
		"quota_reset_at":   nextResetAt,
	}).Error
}

func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add UserRepository

- Add CRUD operations
- Add quota management methods
- Add existence checks"
```

---

## Task 10: Auth DTO

**Files:**
- Create: `internal/model/dto/auth_dto.go`

**Step 1: 创建 internal/model/dto/auth_dto.go**

```go
package dto

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=32"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	UserID int64 `json:"user_id"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string    `json:"token"`
	User  *UserInfo `json:"user"`
}

// VerifyEmailRequest 邮箱验证请求
type VerifyEmailRequest struct {
	Code string `json:"code" binding:"required"`
}

// UserInfo 用户信息（返回给前端）
type UserInfo struct {
	ID                int64   `json:"id"`
	Username          string  `json:"username"`
	Email             string  `json:"email,omitempty"`
	AvatarURL         string  `json:"avatar_url"`
	Bio               string  `json:"bio"`
	SubscriptionLevel string  `json:"subscription_level"`
	EmailVerified     bool    `json:"email_verified,omitempty"`
	QuotaInfo         *QuotaInfo `json:"quota_info,omitempty"`
	CreatedAt         string  `json:"created_at,omitempty"`
}

// QuotaInfo 配额信息
type QuotaInfo struct {
	DailyQuota     int    `json:"daily_quota"`
	QuotaUsedToday int    `json:"quota_used_today"`
	QuotaRemaining int    `json:"quota_remaining"`
	QuotaResetAt   string `json:"quota_reset_at,omitempty"`
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	Username *string `json:"username,omitempty" binding:"omitempty,min=3,max=50"`
	Bio      *string `json:"bio,omitempty" binding:"omitempty,max=500"`
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add auth DTOs

- Add RegisterRequest/Response
- Add LoginRequest/Response
- Add VerifyEmailRequest
- Add UserInfo and QuotaInfo"
```

---

## Task 11: Auth Service

**Files:**
- Create: `internal/service/auth_service.go`

**Step 1: 安装 bcrypt 依赖**

```bash
go get golang.org/x/crypto/bcrypt
```

**Step 2: 创建 internal/service/auth_service.go**

```go
package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/jwt"
	"github.com/qs3c/anal_go_server/internal/repository"
)

var (
	ErrEmailExists         = errors.New("邮箱已被注册")
	ErrUsernameExists      = errors.New("用户名已被使用")
	ErrInvalidCredentials  = errors.New("邮箱或密码错误")
	ErrEmailNotVerified    = errors.New("邮箱尚未验证")
	ErrInvalidVerifyCode   = errors.New("验证码无效或已过期")
	ErrUserNotFound        = errors.New("用户不存在")
)

type AuthService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Register 用户注册
func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// 检查邮箱是否存在
	exists, err := s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}

	// 检查用户名是否存在
	exists, err = s.userRepo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 生成验证码
	verifyCode, err := generateRandomCode(32)
	if err != nil {
		return nil, err
	}

	passwordStr := string(hashedPassword)
	expiresAt := time.Now().Add(24 * time.Hour)
	resetAt := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)

	user := &model.User{
		Username:              req.Username,
		Email:                 &req.Email,
		PasswordHash:          &passwordStr,
		SubscriptionLevel:     "free",
		DailyQuota:            s.cfg.Subscription.Levels["free"].DailyQuota,
		QuotaResetAt:          &resetAt,
		VerificationCode:      &verifyCode,
		VerificationExpiresAt: &expiresAt,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// TODO: 发送验证邮件

	return &dto.RegisterResponse{
		UserID: user.ID,
	}, nil
}

// Login 用户登录
func (s *AuthService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// 检查邮箱是否验证
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	// 验证密码
	if user.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 生成 Token
	token, err := jwt.GenerateToken(user.ID, s.cfg.JWT.Secret, s.cfg.JWT.ExpireHours)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: token,
		User:  s.buildUserInfo(user),
	}, nil
}

// VerifyEmail 验证邮箱
func (s *AuthService) VerifyEmail(code string) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByVerificationCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidVerifyCode
		}
		return nil, err
	}

	// 检查验证码是否过期
	if user.VerificationExpiresAt == nil || time.Now().After(*user.VerificationExpiresAt) {
		return nil, ErrInvalidVerifyCode
	}

	// 更新用户状态
	user.EmailVerified = true
	user.VerificationCode = nil
	user.VerificationExpiresAt = nil
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	// 生成 Token
	token, err := jwt.GenerateToken(user.ID, s.cfg.JWT.Secret, s.cfg.JWT.ExpireHours)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: token,
		User:  s.buildUserInfo(user),
	}, nil
}

// GetUserByID 根据 ID 获取用户
func (s *AuthService) GetUserByID(id int64) (*model.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *AuthService) buildUserInfo(user *model.User) *dto.UserInfo {
	info := &dto.UserInfo{
		ID:                user.ID,
		Username:          user.Username,
		AvatarURL:         user.AvatarURL,
		Bio:               user.Bio,
		SubscriptionLevel: user.SubscriptionLevel,
		EmailVerified:     user.EmailVerified,
	}

	if user.Email != nil {
		info.Email = *user.Email
	}

	return info
}

func generateRandomCode(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
```

**Step 3: 验证编译**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add .
git commit -m "feat: add AuthService

- Add Register with email verification
- Add Login with password validation
- Add VerifyEmail functionality
- Use bcrypt for password hashing"
```

---

## Task 12: Auth Middleware

**Files:**
- Create: `internal/api/middleware/auth.go`

**Step 1: 创建 internal/api/middleware/auth.go**

```go
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/pkg/jwt"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
)

const (
	UserIDKey = "userID"
)

// Auth JWT 认证中间件
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.AuthError(c, "请提供认证信息")
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			response.AuthError(c, "认证格式错误")
			c.Abort()
			return
		}

		claims, err := jwt.ParseToken(tokenString, jwtSecret)
		if err != nil {
			response.AuthError(c, "认证失败或已过期")
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

// OptionalAuth 可选认证中间件（不强制要求登录）
func OptionalAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.Next()
			return
		}

		claims, err := jwt.ParseToken(tokenString, jwtSecret)
		if err == nil {
			c.Set(UserIDKey, claims.UserID)
		}

		c.Next()
	}
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) (int64, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}
	id, ok := userID.(int64)
	return id, ok
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add auth middleware

- Add Auth middleware for protected routes
- Add OptionalAuth for public routes with optional auth
- Add GetUserID helper function"
```

---

## Task 13: CORS Middleware

**Files:**
- Create: `internal/api/middleware/cors.go`

**Step 1: 创建 internal/api/middleware/cors.go**

```go
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/config"
)

// CORS 跨域中间件
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 检查是否在允许列表中
		allowed := false
		for _, allowedOrigin := range cfg.AllowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", joinStrings(cfg.AllowedMethods))
		c.Header("Access-Control-Allow-Headers", joinStrings(cfg.AllowedHeaders))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add CORS middleware

- Support configurable allowed origins
- Handle preflight OPTIONS requests"
```

---

## Task 14: Auth Handler

**Files:**
- Create: `internal/api/handler/auth.go`

**Step 1: 创建 internal/api/handler/auth.go**

```go
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register 用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.authService.Register(&req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailExists):
			response.ParamError(c, err.Error())
		case errors.Is(err, service.ErrUsernameExists):
			response.ParamError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "注册成功，请查收验证邮件", resp)
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			response.AuthError(c, err.Error())
		case errors.Is(err, service.ErrEmailNotVerified):
			response.AuthError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "登录成功", resp)
}

// VerifyEmail 验证邮箱
// POST /api/v1/auth/verify-email
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.authService.VerifyEmail(req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidVerifyCode):
			response.ParamError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "邮箱验证成功", resp)
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add auth handler

- Add Register endpoint
- Add Login endpoint
- Add VerifyEmail endpoint"
```

---

## Task 15: Router 配置

**Files:**
- Create: `internal/api/router.go`

**Step 1: 创建 internal/api/router.go**

```go
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/api/handler"
	"github.com/qs3c/anal_go_server/internal/api/middleware"
)

type Router struct {
	authHandler *handler.AuthHandler
	cfg         *config.Config
}

func NewRouter(authHandler *handler.AuthHandler, cfg *config.Config) *Router {
	return &Router{
		authHandler: authHandler,
		cfg:         cfg,
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
		// 公开接口 - 认证
		auth := api.Group("/auth")
		{
			auth.POST("/register", r.authHandler.Register)
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/verify-email", r.authHandler.VerifyEmail)
			// TODO: GitHub OAuth
			// TODO: WeChat OAuth
		}

		// 需要认证的接口
		authenticated := api.Group("")
		authenticated.Use(middleware.Auth(r.cfg.JWT.Secret))
		{
			// TODO: User APIs
			// TODO: Analysis APIs
			// TODO: Comment APIs
			// TODO: Quota APIs
		}

		// 公开接口 - 社区（可选认证）
		community := api.Group("/community")
		community.Use(middleware.OptionalAuth(r.cfg.JWT.Secret))
		{
			// TODO: Community APIs
		}

		// 公开接口 - 其他
		api.GET("/models", func(c *gin.Context) {
			// TODO: Models API
		})
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
git commit -m "feat: add router configuration

- Setup API routes structure
- Apply CORS and auth middleware
- Define route groups for public/authenticated/community"
```

---

## Task 16: Server Main

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: 更新 cmd/server/main.go**

```go
package main

import (
	"fmt"
	"log"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/api"
	"github.com/qs3c/anal_go_server/internal/api/handler"
	"github.com/qs3c/anal_go_server/internal/database"
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
	_ = rdb // TODO: 后续使用

	// 初始化 Repository
	userRepo := repository.NewUserRepository(db)

	// 初始化 Service
	authService := service.NewAuthService(userRepo, cfg)

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(authService)

	// 初始化 Router
	router := api.NewRouter(authHandler, cfg)
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
git commit -m "feat: implement server main entry

- Load configuration
- Initialize database and redis
- Wire up repositories, services, handlers
- Start HTTP server"
```

---

## Task 17: 整理依赖并最终验证

**Step 1: 整理 go.mod**

```bash
go mod tidy
```

**Step 2: 验证项目可以编译**

```bash
go build ./...
```

**Step 3: 验证服务可以启动（需要数据库）**

注意：此步骤需要先创建 MySQL 数据库并运行迁移。

```bash
# 创建数据库（手动执行）
# mysql -u root -p -e "CREATE DATABASE go_analyzer CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 运行迁移（需要安装 golang-migrate）
# make migrate-up

# 启动服务
# make dev-server
```

**Step 4: Final Commit**

```bash
git add .
git commit -m "chore: tidy dependencies and finalize phase 1

Phase 1 complete:
- Project structure initialized
- Configuration management with viper
- Database models and migrations
- JWT authentication
- User registration and login
- Auth middleware
- Router setup"
```

---

## Summary

Phase 1 完成后的功能：

| 功能 | 状态 |
|------|------|
| 项目结构 | ✅ |
| 配置管理 | ✅ |
| 统一响应 | ✅ |
| 数据库模型 | ✅ |
| 数据库迁移 | ✅ |
| JWT 工具 | ✅ |
| 用户注册 | ✅ |
| 用户登录 | ✅ |
| 邮箱验证 | ✅ |
| 认证中间件 | ✅ |
| CORS 中间件 | ✅ |
| 路由配置 | ✅ |
| Server 入口 | ✅ |

**下一步（Phase 2）：**
- GitHub OAuth
- User Profile API
- Analysis API
- Redis Queue
- WebSocket Hub
- Worker Process
- OSS Integration
