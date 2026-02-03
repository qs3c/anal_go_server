# 文件上传分析功能 - 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 支持用户上传 ZIP 文件进行 Go 项目结构分析，与现有 GitHub URL 分析并存。

**Architecture:** 新增上传解析服务处理 ZIP 文件，扩展分析服务支持两种数据来源（github/upload），Worker 根据来源决定是克隆仓库还是使用已上传的文件。

**Tech Stack:** Go 1.22+, Gin, archive/zip, go/parser, go/ast

---

## Task 1: 添加上传配置

**Files:**
- Modify: `config/config.go`
- Modify: `config.yaml`
- Modify: `config.docker.yaml`

**Step 1: 添加 UploadConfig 结构体**

编辑 `config/config.go`，在 Config 结构体中添加：

```go
type UploadConfig struct {
	MaxSize           int64    `yaml:"max_size"`            // 最大文件大小（字节）
	TempDir           string   `yaml:"temp_dir"`            // 临时目录
	ExpireHours       int      `yaml:"expire_hours"`        // 过期时间（小时）
	AllowedExtensions []string `yaml:"allowed_extensions"`  // 允许的扩展名
}
```

在 Config 结构体中添加字段：
```go
Upload UploadConfig `yaml:"upload"`
```

**Step 2: 更新配置文件**

编辑 `config.yaml` 和 `config.docker.yaml`，添加：

```yaml
upload:
  max_size: 104857600  # 100MB
  temp_dir: /tmp/uploads
  expire_hours: 1
  allowed_extensions:
    - .zip
```

**Step 3: 验证配置加载**

```bash
go build ./cmd/server && echo "Build OK"
```

**Step 4: Commit**

```bash
git add config/config.go config.yaml config.docker.yaml
git commit -m "feat: add upload configuration"
```

---

## Task 2: 创建上传 DTO

**Files:**
- Create: `internal/model/dto/upload_dto.go`

**Step 1: 创建 DTO 文件**

创建 `internal/model/dto/upload_dto.go`：

```go
package dto

// ParseUploadResponse 解析上传文件的响应
type ParseUploadResponse struct {
	UploadID    string       `json:"upload_id"`
	ExpiresAt   string       `json:"expires_at"`
	Files       []GoFileInfo `json:"files"`
	TotalFiles  int          `json:"total_files"`
	TotalStructs int         `json:"total_structs"`
}

// GoFileInfo Go 文件信息
type GoFileInfo struct {
	Path    string   `json:"path"`
	Structs []string `json:"structs"`
}
```

**Step 2: 验证编译**

```bash
go build ./internal/model/dto && echo "Build OK"
```

**Step 3: Commit**

```bash
git add internal/model/dto/upload_dto.go
git commit -m "feat: add upload DTOs"
```

---

## Task 3: 创建上传服务 - 基础结构

**Files:**
- Create: `internal/service/upload_service.go`
- Create: `internal/service/upload_service_test.go`

**Step 1: 写失败测试 - 创建服务**

创建 `internal/service/upload_service_test.go`：

```go
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/qs3c/anal_go_server/config"
)

func TestNewUploadService(t *testing.T) {
	cfg := &config.Config{
		Upload: config.UploadConfig{
			MaxSize:           104857600,
			TempDir:           "/tmp/test-uploads",
			ExpireHours:       1,
			AllowedExtensions: []string{".zip"},
		},
	}

	svc := NewUploadService(cfg)
	assert.NotNil(t, svc)
}
```

**Step 2: 运行测试验证失败**

```bash
go test ./internal/service -run TestNewUploadService -v
```
预期：FAIL - NewUploadService undefined

**Step 3: 实现基础服务**

创建 `internal/service/upload_service.go`：

```go
package service

import (
	"github.com/qs3c/anal_go_server/config"
)

type UploadService struct {
	cfg *config.Config
}

func NewUploadService(cfg *config.Config) *UploadService {
	return &UploadService{
		cfg: cfg,
	}
}
```

**Step 4: 运行测试验证通过**

```bash
go test ./internal/service -run TestNewUploadService -v
```
预期：PASS

**Step 5: Commit**

```bash
git add internal/service/upload_service.go internal/service/upload_service_test.go
git commit -m "feat: add UploadService skeleton"
```

---

## Task 4: 实现 ZIP 解析逻辑

**Files:**
- Modify: `internal/service/upload_service.go`
- Modify: `internal/service/upload_service_test.go`

**Step 1: 写失败测试 - ParseZip**

添加到 `internal/service/upload_service_test.go`：

```go
func TestUploadService_ParseZip(t *testing.T) {
	cfg := &config.Config{
		Upload: config.UploadConfig{
			MaxSize:           104857600,
			TempDir:           t.TempDir(),
			ExpireHours:       1,
			AllowedExtensions: []string{".zip"},
		},
	}
	svc := NewUploadService(cfg)

	// 创建测试 ZIP 文件
	zipPath := createTestZip(t, map[string]string{
		"main.go": `package main

type Server struct {
	Host string
	Port int
}

type Config struct {
	Debug bool
}
`,
		"internal/model/user.go": `package model

type User struct {
	ID   int64
	Name string
}

type UserProfile struct {
	Bio string
}
`,
	})

	result, err := svc.ParseZip(zipPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.UploadID)
	assert.Equal(t, 2, result.TotalFiles)
	assert.Equal(t, 4, result.TotalStructs)

	// 验证文件和结构体
	fileMap := make(map[string][]string)
	for _, f := range result.Files {
		fileMap[f.Path] = f.Structs
	}

	assert.ElementsMatch(t, []string{"Server", "Config"}, fileMap["main.go"])
	assert.ElementsMatch(t, []string{"User", "UserProfile"}, fileMap["internal/model/user.go"])
}

// createTestZip 创建测试用 ZIP 文件
func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()

	zipPath := filepath.Join(t.TempDir(), "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			t.Fatal(err)
		}
	}
	w.Close()

	return zipPath
}
```

添加 imports：
```go
import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/qs3c/anal_go_server/config"
)
```

**Step 2: 运行测试验证失败**

```bash
go test ./internal/service -run TestUploadService_ParseZip -v
```
预期：FAIL - ParseZip undefined

**Step 3: 实现 ParseZip**

更新 `internal/service/upload_service.go`：

```go
package service

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model/dto"
)

var (
	ErrInvalidZip     = fmt.Errorf("ZIP 文件损坏或无法解压")
	ErrNoGoFiles      = fmt.Errorf("未找到 Go 源文件")
	ErrFileTooLarge   = fmt.Errorf("文件过大")
	ErrInvalidFormat  = fmt.Errorf("仅支持 ZIP 格式")
	ErrUploadNotFound = fmt.Errorf("上传文件不存在或已过期")
)

type UploadService struct {
	cfg *config.Config
}

func NewUploadService(cfg *config.Config) *UploadService {
	return &UploadService{
		cfg: cfg,
	}
}

// ParseZip 解析 ZIP 文件，提取 Go 文件和结构体信息
func (s *UploadService) ParseZip(zipPath string) (*dto.ParseUploadResponse, error) {
	// 生成 upload ID
	uploadID, err := generateUploadID()
	if err != nil {
		return nil, err
	}

	// 创建解压目录
	extractDir := filepath.Join(s.cfg.Upload.TempDir, uploadID)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, err
	}

	// 解压 ZIP
	if err := s.extractZip(zipPath, extractDir); err != nil {
		os.RemoveAll(extractDir)
		return nil, ErrInvalidZip
	}

	// 扫描 Go 文件和结构体
	files, err := s.scanGoFiles(extractDir)
	if err != nil {
		os.RemoveAll(extractDir)
		return nil, err
	}

	if len(files) == 0 {
		os.RemoveAll(extractDir)
		return nil, ErrNoGoFiles
	}

	// 计算总结构体数
	totalStructs := 0
	for _, f := range files {
		totalStructs += len(f.Structs)
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.Upload.ExpireHours) * time.Hour)

	return &dto.ParseUploadResponse{
		UploadID:     uploadID,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
		Files:        files,
		TotalFiles:   len(files),
		TotalStructs: totalStructs,
	}, nil
}

// GetUploadPath 获取上传文件的路径
func (s *UploadService) GetUploadPath(uploadID string) (string, error) {
	path := filepath.Join(s.cfg.Upload.TempDir, uploadID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", ErrUploadNotFound
	}
	return path, nil
}

// CleanupUpload 清理上传的文件
func (s *UploadService) CleanupUpload(uploadID string) error {
	path := filepath.Join(s.cfg.Upload.TempDir, uploadID)
	return os.RemoveAll(path)
}

func (s *UploadService) extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// 安全检查：防止 zip slip 攻击
		destPath := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0755)
			continue
		}

		// 创建父目录
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// 解压文件
		destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		srcFile, err := f.Open()
		if err != nil {
			destFile.Close()
			return err
		}

		_, err = io.Copy(destFile, srcFile)
		srcFile.Close()
		destFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *UploadService) scanGoFiles(rootDir string) ([]dto.GoFileInfo, error) {
	var files []dto.GoFileInfo

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录和非 .go 文件
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// 跳过测试文件
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		// 解析文件获取结构体
		structs, err := s.parseStructs(path)
		if err != nil {
			// 跳过解析失败的文件
			return nil
		}

		if len(structs) > 0 {
			// 获取相对路径
			relPath, _ := filepath.Rel(rootDir, path)
			files = append(files, dto.GoFileInfo{
				Path:    relPath,
				Structs: structs,
			})
		}

		return nil
	})

	return files, err
}

func (s *UploadService) parseStructs(filePath string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var structs []string
	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if _, isStruct := typeSpec.Type.(*ast.StructType); isStruct {
			structs = append(structs, typeSpec.Name.Name)
		}

		return true
	})

	return structs, nil
}

func generateUploadID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
```

**Step 4: 运行测试验证通过**

```bash
go test ./internal/service -run TestUploadService_ParseZip -v
```
预期：PASS

**Step 5: Commit**

```bash
git add internal/service/upload_service.go internal/service/upload_service_test.go
git commit -m "feat: implement ZIP parsing with struct extraction"
```

---

## Task 5: 创建上传 Handler

**Files:**
- Create: `internal/api/handler/upload.go`
- Create: `internal/api/handler/upload_test.go`

**Step 1: 写失败测试**

创建 `internal/api/handler/upload_test.go`：

```go
package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/service"
)

func TestUploadHandler_Parse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Upload: config.UploadConfig{
			MaxSize:           104857600,
			TempDir:           t.TempDir(),
			ExpireHours:       1,
			AllowedExtensions: []string{".zip"},
		},
	}
	uploadService := service.NewUploadService(cfg)
	handler := NewUploadHandler(uploadService, cfg)

	// 创建测试 ZIP
	zipContent := createTestZipContent(t, map[string]string{
		"main.go": `package main
type App struct { Name string }
`,
	})

	// 创建 multipart 请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.zip")
	part.Write(zipContent)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/parse", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/upload/parse", handler.Parse)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int                     `json:"code"`
		Data dto.ParseUploadResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 0, resp.Code)
	assert.NotEmpty(t, resp.Data.UploadID)
	assert.Equal(t, 1, resp.Data.TotalFiles)
}

func createTestZipContent(t *testing.T, files map[string]string) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)
	for name, content := range files {
		f, _ := w.Create(name)
		f.Write([]byte(content))
	}
	w.Close()
	return buf.Bytes()
}
```

**Step 2: 运行测试验证失败**

```bash
go test ./internal/api/handler -run TestUploadHandler_Parse -v
```
预期：FAIL - NewUploadHandler undefined

**Step 3: 实现 Handler**

创建 `internal/api/handler/upload.go`：

```go
package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type UploadHandler struct {
	uploadService *service.UploadService
	cfg           *config.Config
}

func NewUploadHandler(uploadService *service.UploadService, cfg *config.Config) *UploadHandler {
	return &UploadHandler{
		uploadService: uploadService,
		cfg:           cfg,
	}
}

// Parse 解析上传的 ZIP 文件
// POST /api/v1/upload/parse
func (h *UploadHandler) Parse(c *gin.Context) {
	// 获取上传文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.ParamError(c, "请上传文件")
		return
	}
	defer file.Close()

	// 检查文件大小
	if header.Size > h.cfg.Upload.MaxSize {
		response.ParamError(c, "文件过大，最大支持 100MB")
		return
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := false
	for _, allowedExt := range h.cfg.Upload.AllowedExtensions {
		if ext == allowedExt {
			allowed = true
			break
		}
	}
	if !allowed {
		response.ParamError(c, "仅支持 ZIP 格式")
		return
	}

	// 保存到临时文件
	tempFile, err := os.CreateTemp("", "upload-*.zip")
	if err != nil {
		response.ServerError(c, "文件保存失败")
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		response.ServerError(c, "文件保存失败")
		return
	}

	// 解析 ZIP
	result, err := h.uploadService.ParseZip(tempFile.Name())
	if err != nil {
		switch err {
		case service.ErrInvalidZip:
			response.ParamError(c, "ZIP 文件损坏或无法解压")
		case service.ErrNoGoFiles:
			response.ParamError(c, "未找到 Go 源文件")
		default:
			response.ServerError(c, "解析失败")
		}
		return
	}

	response.Success(c, result)
}
```

**Step 4: 运行测试验证通过**

```bash
go test ./internal/api/handler -run TestUploadHandler_Parse -v
```
预期：PASS

**Step 5: Commit**

```bash
git add internal/api/handler/upload.go internal/api/handler/upload_test.go
git commit -m "feat: add upload handler for ZIP parsing"
```

---

## Task 6: 扩展分析 DTO 支持上传模式

**Files:**
- Modify: `internal/model/dto/analysis_dto.go`

**Step 1: 更新 CreateAnalysisRequest**

编辑 `internal/model/dto/analysis_dto.go`，更新 `CreateAnalysisRequest`：

```go
type CreateAnalysisRequest struct {
	Title         string          `json:"title" binding:"required,max=200"`
	CreationType  string          `json:"creation_type" binding:"required,oneof=ai manual"`
	SourceType    string          `json:"source_type,omitempty"`    // "github" 或 "upload"
	RepoURL       string          `json:"repo_url,omitempty"`
	UploadID      string          `json:"upload_id,omitempty"`      // 新增
	StartFile     string          `json:"start_file,omitempty"`     // 新增
	StartStruct   string          `json:"start_struct,omitempty"`
	AnalysisDepth int             `json:"analysis_depth,omitempty"`
	ModelName     string          `json:"model_name,omitempty"`
	DiagramData   json.RawMessage `json:"diagram_data,omitempty"`
}
```

**Step 2: 验证编译**

```bash
go build ./internal/model/dto && echo "Build OK"
```

**Step 3: Commit**

```bash
git add internal/model/dto/analysis_dto.go
git commit -m "feat: extend analysis DTO with upload support"
```

---

## Task 7: 扩展分析服务支持上传模式

**Files:**
- Modify: `internal/service/analysis_service.go`
- Modify: `internal/service/analysis_service_test.go`

**Step 1: 更新 AnalysisService 添加 uploadService**

编辑 `internal/service/analysis_service.go`，添加 `uploadService` 字段：

```go
type AnalysisService struct {
	analysisRepo  *repository.AnalysisRepository
	jobRepo       *repository.JobRepository
	userRepo      *repository.UserRepository
	quotaService  *QuotaService
	uploadService *UploadService  // 新增
	ossClient     *oss.Client
	jobQueue      *queue.Queue
	cfg           *config.Config
}

func NewAnalysisService(
	analysisRepo *repository.AnalysisRepository,
	jobRepo *repository.JobRepository,
	userRepo *repository.UserRepository,
	quotaService *QuotaService,
	uploadService *UploadService,  // 新增参数
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
```

**Step 2: 更新 Create 方法支持上传模式**

在 Create 方法中，AI 分析分支内添加上传模式处理：

```go
// 在 if req.CreationType == "ai" 分支内
if req.CreationType == "ai" {
	// ... 现有的配额检查代码 ...

	// 根据来源类型设置不同字段
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = "github" // 默认为 github
	}

	if sourceType == "upload" {
		// 验证上传文件存在
		if req.UploadID == "" {
			return nil, errors.New("upload_id 不能为空")
		}
		if s.uploadService != nil {
			if _, err := s.uploadService.GetUploadPath(req.UploadID); err != nil {
				return nil, ErrUploadNotFound
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
}
```

**Step 3: 更新 model/analysis.go 添加新字段**

编辑 `internal/model/analysis.go`，添加字段：

```go
type Analysis struct {
	// ... 现有字段 ...
	SourceType string `gorm:"size:20;default:github"` // github 或 upload
	UploadID   string `gorm:"size:64"`
	StartFile  string `gorm:"size:500"`
}
```

**Step 4: 更新 JobMessage 支持上传模式**

编辑 `internal/pkg/queue/queue.go`，更新 JobMessage：

```go
type JobMessage struct {
	JobID       int64  `json:"job_id"`
	AnalysisID  int64  `json:"analysis_id"`
	UserID      int64  `json:"user_id"`
	SourceType  string `json:"source_type"`   // 新增
	RepoURL     string `json:"repo_url"`
	UploadID    string `json:"upload_id"`     // 新增
	StartFile   string `json:"start_file"`    // 新增
	StartStruct string `json:"start_struct"`
	Depth       int    `json:"depth"`
	ModelName   string `json:"model_name"`
}
```

**Step 5: 更新 analysis_service.go Create 方法中的 JobMessage 构造**

```go
jobMsg := &queue.JobMessage{
	JobID:       job.ID,
	AnalysisID:  analysis.ID,
	UserID:      userID,
	SourceType:  analysis.SourceType,    // 新增
	RepoURL:     req.RepoURL,
	UploadID:    req.UploadID,           // 新增
	StartFile:   req.StartFile,          // 新增
	StartStruct: req.StartStruct,
	Depth:       req.AnalysisDepth,
	ModelName:   req.ModelName,
}
```

**Step 6: 更新测试文件**

更新 `internal/service/analysis_service_test.go` 中所有 `NewAnalysisService` 调用，添加 nil 参数：

```go
// 查找替换所有 NewAnalysisService 调用
// 旧：NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, ossClient, jobQueue, cfg)
// 新：NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, ossClient, jobQueue, cfg)
```

**Step 7: 运行测试**

```bash
go test ./internal/service -v
```

**Step 8: Commit**

```bash
git add internal/service/analysis_service.go internal/service/analysis_service_test.go internal/model/analysis.go internal/pkg/queue/queue.go
git commit -m "feat: extend analysis service to support upload mode"
```

---

## Task 8: 更新 Worker 支持上传模式

**Files:**
- Modify: `internal/worker/processor.go`

**Step 1: 修改 Process 方法支持上传模式**

编辑 `internal/worker/processor.go`，修改克隆步骤：

```go
func (p *Processor) Process(ctx context.Context, msg *queue.JobMessage) error {
	// ... 现有的 job 获取代码 ...

	var projectPath string
	var needCleanup bool

	if msg.SourceType == "upload" {
		// 上传模式：直接使用已上传的文件
		projectPath = filepath.Join(p.cfg.Upload.TempDir, msg.UploadID)
		if _, err := os.Stat(projectPath); os.IsNotExist(err) {
			return handleError(pubsub.StepCloning, fmt.Errorf("上传文件不存在或已过期"))
		}
		needCleanup = false // 上传的文件由上传服务管理

		// 跳过克隆步骤，直接标记为解析中
		job.CurrentStep = "正在解析项目结构"
		p.jobRepo.Update(job)
		publishProgress(pubsub.StepParsing, "processing", "")
	} else {
		// GitHub 模式：克隆仓库
		projectPath = GetTempDir(job.ID)
		needCleanup = true

		log.Printf("Job %d: cloning repo %s", job.ID, msg.RepoURL)
		job.CurrentStep = "正在克隆仓库"
		p.jobRepo.Update(job)
		publishProgress(pubsub.StepCloning, "processing", "")

		if err := ValidateRepoURL(msg.RepoURL); err != nil {
			return handleError(pubsub.StepCloning, fmt.Errorf("invalid repo URL: %w", err))
		}

		if err := CloneRepo(ctx, msg.RepoURL, projectPath); err != nil {
			return handleError(pubsub.StepCloning, fmt.Errorf("clone failed: %w", err))
		}
	}

	if needCleanup {
		defer CleanupRepo(projectPath)
	}

	// ... 后续的解析和分析代码保持不变，但使用 projectPath ...
}
```

**Step 2: 验证编译**

```bash
go build ./internal/worker && echo "Build OK"
```

**Step 3: Commit**

```bash
git add internal/worker/processor.go
git commit -m "feat: update worker to support upload mode"
```

---

## Task 9: 注册路由和更新 main.go

**Files:**
- Modify: `internal/api/router.go`
- Modify: `cmd/server/main.go`

**Step 1: 更新 router.go 添加上传路由**

编辑 `internal/api/router.go`，添加 UploadHandler：

```go
type Router struct {
	// ... 现有字段 ...
	uploadHandler *handler.UploadHandler  // 新增
}

func NewRouter(
	// ... 现有参数 ...
	uploadHandler *handler.UploadHandler,  // 新增
	cfg *config.Config,
) *Router {
	return &Router{
		// ... 现有赋值 ...
		uploadHandler: uploadHandler,
	}
}

func (r *Router) Setup() *gin.Engine {
	// ... 现有代码 ...

	// 在需要认证的路由组中添加
	authGroup := engine.Group("/api/v1")
	authGroup.Use(r.authMiddleware.Handle())
	{
		// ... 现有路由 ...

		// 上传相关
		authGroup.POST("/upload/parse", r.uploadHandler.Parse)
	}

	return engine
}
```

**Step 2: 更新 main.go 初始化服务**

编辑 `cmd/server/main.go`：

```go
// 在 Service 初始化部分添加
uploadService := service.NewUploadService(cfg)

// 更新 analysisService 初始化
analysisService := service.NewAnalysisService(
	analysisRepo, jobRepo, userRepo, quotaService,
	uploadService,  // 新增
	ossClient, jobQueue, cfg,
)

// 在 Handler 初始化部分添加
uploadHandler := handler.NewUploadHandler(uploadService, cfg)

// 更新 Router 初始化
router := api.NewRouter(
	authHandler,
	userHandler,
	analysisHandler,
	modelsHandler,
	websocketHandler,
	communityHandler,
	commentHandler,
	quotaHandler,
	uploadHandler,  // 新增
	cfg,
)
```

**Step 3: 验证编译和启动**

```bash
go build ./cmd/server && echo "Build OK"
```

**Step 4: Commit**

```bash
git add internal/api/router.go cmd/server/main.go
git commit -m "feat: register upload routes and wire up services"
```

---

## Task 10: 数据库迁移

**Files:**
- Create: `migrations/XXXXXX_add_upload_fields.sql`

**Step 1: 创建迁移文件**

创建迁移 SQL：

```sql
-- migrations/20260202_add_upload_fields.sql

ALTER TABLE analyses
ADD COLUMN source_type VARCHAR(20) DEFAULT 'github' AFTER model_name,
ADD COLUMN upload_id VARCHAR(64) AFTER source_type,
ADD COLUMN start_file VARCHAR(500) AFTER upload_id;

-- 更新现有记录
UPDATE analyses SET source_type = 'github' WHERE source_type IS NULL;
```

**Step 2: Commit**

```bash
git add migrations/
git commit -m "feat: add migration for upload fields"
```

---

## Task 11: 集成测试

**Step 1: 启动服务测试完整流程**

```bash
# 构建并启动
go build ./cmd/server && ./server &

# 等待启动
sleep 2

# 创建测试 ZIP
mkdir -p /tmp/test-project
cat > /tmp/test-project/main.go << 'EOF'
package main

type App struct {
    Name string
}

type Config struct {
    Debug bool
}
EOF
cd /tmp/test-project && zip -r ../test.zip . && cd -

# 测试上传解析 (需要有效 token)
TOKEN="your-test-token"
curl -X POST http://localhost:8080/api/v1/upload/parse \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/tmp/test.zip"
```

**Step 2: 验证响应**

预期响应：
```json
{
  "code": 0,
  "data": {
    "upload_id": "xxx",
    "files": [
      {"path": "main.go", "structs": ["App", "Config"]}
    ],
    "total_files": 1,
    "total_structs": 2
  }
}
```

**Step 3: Commit 最终状态**

```bash
git add -A
git commit -m "feat: complete file upload analysis feature"
```

---

## 总结

| Task | 描述 | 预计时间 |
|------|------|---------|
| 1 | 添加上传配置 | 5 min |
| 2 | 创建上传 DTO | 3 min |
| 3 | 创建上传服务基础结构 | 5 min |
| 4 | 实现 ZIP 解析逻辑 | 15 min |
| 5 | 创建上传 Handler | 10 min |
| 6 | 扩展分析 DTO | 3 min |
| 7 | 扩展分析服务 | 15 min |
| 8 | 更新 Worker | 10 min |
| 9 | 注册路由和更新 main.go | 10 min |
| 10 | 数据库迁移 | 5 min |
| 11 | 集成测试 | 10 min |
