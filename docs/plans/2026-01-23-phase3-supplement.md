# Phase 3 补充 - 缺失功能实现

## 概述

补充 Phase 2-3 中尚未实现的功能模块。

## 任务列表

### Task 1: OSS 客户端

创建 `internal/pkg/oss/client.go`

功能：
- 初始化阿里云 OSS 客户端
- 上传文件（框图 JSON、用户头像）
- 删除文件
- 生成访问 URL

```go
type OSSClient struct {
    client     *oss.Client
    bucket     *oss.Bucket
    bucketName string
    endpoint   string
    cdnDomain  string
}

func NewOSSClient(cfg *config.OSSConfig) (*OSSClient, error)
func (c *OSSClient) UploadDiagram(analysisID int64, data []byte) (string, error)
func (c *OSSClient) UploadAvatar(userID int64, data []byte, ext string) (string, error)
func (c *OSSClient) Delete(objectKey string) error
func (c *OSSClient) GetURL(objectKey string) string
```

### Task 2: 更新配置结构

在 `config/config.go` 中添加 OSS 和 Email 配置：

```go
type OSSConfig struct {
    Endpoint        string `yaml:"endpoint"`
    AccessKeyID     string `yaml:"access_key_id"`
    AccessKeySecret string `yaml:"access_key_secret"`
    BucketName      string `yaml:"bucket_name"`
    CDNDomain       string `yaml:"cdn_domain"`
}

type EmailConfig struct {
    SMTPHost     string `yaml:"smtp_host"`
    SMTPPort     int    `yaml:"smtp_port"`
    Username     string `yaml:"username"`
    Password     string `yaml:"password"`
    FromAddress  string `yaml:"from_address"`
    FromName     string `yaml:"from_name"`
}
```

### Task 3: 邮件服务

创建 `internal/pkg/email/email.go`

功能：
- SMTP 邮件发送
- 验证码邮件模板
- 密码重置邮件模板

```go
type EmailService struct {
    cfg *config.EmailConfig
}

func NewEmailService(cfg *config.EmailConfig) *EmailService
func (s *EmailService) SendVerificationCode(to, code string) error
func (s *EmailService) SendPasswordReset(to, resetLink string) error
```

### Task 4: 配额中间件

创建 `internal/api/middleware/quota.go`

功能：
- 检查用户当日配额
- 返回配额超限错误

```go
func QuotaCheck(quotaService *service.QuotaService) gin.HandlerFunc
```

### Task 5: 完善配额服务

更新 `internal/service/quota_service.go`

添加：
- 获取用户配额信息 API
- 配额使用记录
- 配额重置逻辑

```go
func (s *QuotaService) GetQuotaInfo(userID int64) (*dto.QuotaInfo, error)
func (s *QuotaService) UseQuota(userID int64) error
func (s *QuotaService) RefundQuota(userID int64) error
func (s *QuotaService) ResetDailyQuotas() error
```

### Task 6: 配额 DTO

创建 `internal/model/dto/quota_dto.go`

```go
type QuotaInfo struct {
    Tier         string `json:"tier"`
    DailyLimit   int    `json:"daily_limit"`
    DailyUsed    int    `json:"daily_used"`
    DailyRemain  int    `json:"daily_remain"`
    MaxDepth     int    `json:"max_depth"`
}
```

### Task 7: 配额 Handler

创建 `internal/api/handler/quota.go`

```go
// GET /api/v1/user/quota
func (h *QuotaHandler) GetQuota(c *gin.Context)
```

### Task 8: 更新 User Handler

在 `internal/api/handler/user.go` 中集成 OSS 上传头像功能。

### Task 9: 更新 Analysis Service

在 `internal/service/analysis_service.go` 中：
- 创建分析时检查配额
- 集成 OSS 上传框图

### Task 10: 定时任务 - 每日配额重置

创建 `internal/pkg/cron/cron.go`

```go
type CronService struct {
    quotaService *service.QuotaService
}

func NewCronService(quotaService *service.QuotaService) *CronService
func (c *CronService) Start()
func (c *CronService) resetDailyQuotas()
```

### Task 11: 更新路由和主入口

- 添加配额相关路由
- 初始化 OSS、Email、Cron 服务

### Task 12: 更新 config.yaml 示例

添加 OSS 和 Email 配置示例。

### Task 13: 依赖整理和验证

```bash
go mod tidy
go build ./...
```

## 验证

- [ ] go build ./... 通过
- [ ] OSS 配置正确加载
- [ ] Email 配置正确加载
- [ ] 配额中间件集成到路由
