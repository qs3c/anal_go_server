package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/user/go-struct-analyzer/pkg/analyzer"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/pkg/oss"
	"github.com/qs3c/anal_go_server/internal/pkg/pubsub"
	"github.com/qs3c/anal_go_server/internal/pkg/queue"
	"github.com/qs3c/anal_go_server/internal/repository"
)

// Processor 任务处理器
type Processor struct {
	jobRepo      *repository.JobRepository
	analysisRepo *repository.AnalysisRepository
	ossClient    *oss.Client
	publisher    *pubsub.Publisher
	cfg          *config.Config
}

// NewProcessor 创建任务处理器
func NewProcessor(
	jobRepo *repository.JobRepository,
	analysisRepo *repository.AnalysisRepository,
	ossClient *oss.Client,
	publisher *pubsub.Publisher,
	cfg *config.Config,
) *Processor {
	return &Processor{
		jobRepo:      jobRepo,
		analysisRepo: analysisRepo,
		ossClient:    ossClient,
		publisher:    publisher,
		cfg:          cfg,
	}
}

// Process 处理分析任务
func (p *Processor) Process(ctx context.Context, msg *queue.JobMessage) error {
	job, err := p.jobRepo.GetByID(msg.JobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// 更新状态为处理中
	now := time.Now()
	job.Status = "processing"
	job.StartedAt = &now
	p.jobRepo.Update(job)
	p.analysisRepo.UpdateStatus(job.AnalysisID, "analyzing")

	// 定义进度推送辅助函数
	publishProgress := func(step, status string, errMsg string) {
		p.publisher.PublishProgress(ctx, &pubsub.ProgressMessage{
			UserID:     msg.UserID,
			AnalysisID: msg.AnalysisID,
			JobID:      msg.JobID,
			Status:     status,
			Step:       step,
			Error:      errMsg,
		})
	}

	// 定义失败处理函数
	handleError := func(step string, err error) error {
		errMsg := err.Error()
		job.Status = "failed"
		job.ErrorMessage = errMsg
		job.CurrentStep = step
		completedAt := time.Now()
		job.CompletedAt = &completedAt
		job.ElapsedSeconds = int(completedAt.Sub(*job.StartedAt).Seconds())
		p.jobRepo.Update(job)
		p.analysisRepo.UpdateStatus(job.AnalysisID, "failed")
		publishProgress(step, "failed", errMsg)
		return err
	}

	// 根据来源类型决定项目路径
	var projectPath string
	var needCleanup bool

	if msg.SourceType == "upload" {
		// Upload mode: use already uploaded files
		uploadRoot := filepath.Join(p.cfg.Upload.TempDir, msg.UploadID)
		if _, err := os.Stat(uploadRoot); os.IsNotExist(err) {
			return handleError(pubsub.StepCloning, fmt.Errorf("上传文件不存在或已过期"))
		}

		// Find the actual Go project directory (containing go.mod)
		projectPath = findGoProjectDir(uploadRoot)
		log.Printf("Job %d: upload root=%s, project dir=%s", job.ID, uploadRoot, projectPath)

		needCleanup = false // uploaded files managed by upload service

		// Skip cloning, go directly to parsing
		log.Printf("Job %d: using uploaded files at %s", job.ID, projectPath)
		job.CurrentStep = "正在解析项目结构"
		p.jobRepo.Update(job)
		publishProgress(pubsub.StepParsing, "processing", "")
	} else {
		// GitHub mode: clone repository
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

	// Cleanup only if we cloned
	if needCleanup {
		defer CleanupRepo(projectPath)
	}

	// Step 2: 解析项目
	log.Printf("Job %d: parsing project", job.ID)
	job.CurrentStep = "正在解析项目结构"
	p.jobRepo.Update(job)
	publishProgress(pubsub.StepParsing, "processing", "")

	// 获取 LLM 配置
	provider, apiKey := p.getModelConfig(msg.ModelName)

	// 创建分析器
	opts := analyzer.Options{
		ProjectPath: projectPath,
		StartStruct: msg.StartStruct,
		MaxDepth:    msg.Depth,
		LLMProvider: provider,
		LLMModel:    msg.ModelName,
		APIKey:      apiKey,
		EnableCache: true,
	}

	a, err := analyzer.New(opts)
	if err != nil {
		return handleError(pubsub.StepParsing, fmt.Errorf("failed to create analyzer: %w", err))
	}

	// Step 3: AI 分析
	log.Printf("Job %d: analyzing with model %s", job.ID, msg.ModelName)
	job.CurrentStep = "正在进行 AI 分析"
	p.jobRepo.Update(job)
	publishProgress(pubsub.StepAnalyzing, "processing", "")

	result, err := a.Analyze()
	if err != nil {
		return handleError(pubsub.StepAnalyzing, fmt.Errorf("analysis failed: %w", err))
	}

	// Step 4: 生成并上传结果
	log.Printf("Job %d: uploading results", job.ID)
	job.CurrentStep = "正在上传结果"
	p.jobRepo.Update(job)
	publishProgress(pubsub.StepUploading, "processing", "")

	// 生成可视化 JSON
	visualizerJSON, err := a.GenerateVisualizerJSON()
	if err != nil {
		return handleError(pubsub.StepUploading, fmt.Errorf("failed to generate visualizer JSON: %w", err))
	}

	// 上传到 OSS 或保存到本地
	var diagramURL string
	if p.ossClient != nil {
		diagramURL, err = p.ossClient.UploadDiagram(job.AnalysisID, []byte(visualizerJSON))
		if err != nil {
			return handleError(pubsub.StepUploading, fmt.Errorf("failed to upload diagram: %w", err))
		}
	} else {
		// 本地存储模式 - 保存到文件
		localDir := filepath.Join(p.cfg.Upload.TempDir, "diagrams")
		if err := os.MkdirAll(localDir, 0755); err != nil {
			return handleError(pubsub.StepUploading, fmt.Errorf("failed to create diagram dir: %w", err))
		}
		localPath := filepath.Join(localDir, fmt.Sprintf("%d.json", job.AnalysisID))
		if err := os.WriteFile(localPath, []byte(visualizerJSON), 0644); err != nil {
			return handleError(pubsub.StepUploading, fmt.Errorf("failed to save diagram locally: %w", err))
		}
		// 使用特殊前缀标记本地存储
		diagramURL = fmt.Sprintf("local://%d", job.AnalysisID)
		log.Printf("Job %d: saved diagram locally (OSS not configured)", job.ID)
	}

	// Step 5: 更新数据库
	log.Printf("Job %d: updating database", job.ID)

	// 更新 Analysis
	analysis, err := p.analysisRepo.GetByID(job.AnalysisID)
	if err != nil {
		return handleError(pubsub.StepDone, fmt.Errorf("failed to get analysis: %w", err))
	}

	analysis.Status = "completed"
	analysis.DiagramOSSURL = diagramURL
	analysis.DiagramSize = result.TotalStructs // 记录结构体数量
	if err := p.analysisRepo.Update(analysis); err != nil {
		return handleError(pubsub.StepDone, fmt.Errorf("failed to update analysis: %w", err))
	}

	// 更新 Job
	job.Status = "completed"
	job.CurrentStep = "分析完成"
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	job.ElapsedSeconds = int(completedAt.Sub(*job.StartedAt).Seconds())
	p.jobRepo.Update(job)

	// 推送完成消息
	publishProgress(pubsub.StepDone, "completed", "")

	log.Printf("Job %d: completed in %d seconds, found %d structs, %d deps",
		job.ID, job.ElapsedSeconds, result.TotalStructs, result.TotalDeps)

	return nil
}

// getModelConfig 根据模型名获取提供商和 API Key
func (p *Processor) getModelConfig(modelName string) (provider, apiKey string) {
	for _, m := range p.cfg.Models {
		if m.Name == modelName {
			return m.APIProvider, m.APIKey
		}
	}
	// 默认返回空，分析器会在没有 LLM 时跳过描述生成
	return "", ""
}

// findGoProjectDir finds the directory containing go.mod
// This handles the case where ZIP extraction creates a subdirectory
func findGoProjectDir(rootDir string) string {
	// First check if go.mod exists in the root directory
	if _, err := os.Stat(filepath.Join(rootDir, "go.mod")); err == nil {
		return rootDir
	}

	// Walk the directory tree to find go.mod
	var goModDir string
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == "go.mod" {
			goModDir = filepath.Dir(path)
			return filepath.SkipAll // Stop walking after finding the first go.mod
		}
		return nil
	})

	// If found, return the directory containing go.mod
	if goModDir != "" {
		return goModDir
	}

	// Fallback: if no go.mod found, check if there's only one subdirectory
	// (common case for ZIP files that contain a single project folder)
	entries, err := os.ReadDir(rootDir)
	if err == nil && len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(rootDir, entries[0].Name())
	}

	// Default to rootDir if nothing else found
	return rootDir
}
