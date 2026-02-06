package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/pkg/oss"
	"github.com/qs3c/anal_go_server/internal/repository"
)

const reuploadInterval = 5 * time.Minute

// Reuploader 后台异步重传本地 diagram 到 OSS
type Reuploader struct {
	analysisRepo *repository.AnalysisRepository
	ossClient    *oss.Client
	cfg          *config.Config
}

// NewReuploader 创建重传器
func NewReuploader(
	analysisRepo *repository.AnalysisRepository,
	ossClient *oss.Client,
	cfg *config.Config,
) *Reuploader {
	return &Reuploader{
		analysisRepo: analysisRepo,
		ossClient:    ossClient,
		cfg:          cfg,
	}
}

// Start 启动后台重传循环
func (r *Reuploader) Start(ctx context.Context) {
	// 启动后先执行一次
	r.run()

	ticker := time.NewTicker(reuploadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Reuploader stopped")
			return
		case <-ticker.C:
			r.run()
		}
	}
}

func (r *Reuploader) run() {
	analyses, err := r.analysisRepo.ListLocalDiagrams()
	if err != nil {
		log.Printf("Reuploader: failed to query local diagrams: %v", err)
		return
	}

	if len(analyses) == 0 {
		return
	}

	log.Printf("Reuploader: found %d local diagrams to re-upload", len(analyses))

	for _, a := range analyses {
		localPath := filepath.Join(r.cfg.Upload.TempDir, "diagrams", fmt.Sprintf("%d.json", a.ID))
		data, err := os.ReadFile(localPath)
		if err != nil {
			log.Printf("Reuploader: failed to read local diagram %d: %v", a.ID, err)
			continue
		}

		ossURL, err := r.ossClient.UploadDiagramWithRetry(a.ID, data)
		if err != nil {
			log.Printf("Reuploader: failed to re-upload diagram %d: %v", a.ID, err)
			continue
		}

		// 更新 DB
		a.DiagramOSSURL = ossURL
		if err := r.analysisRepo.Update(a); err != nil {
			log.Printf("Reuploader: failed to update DB for diagram %d: %v", a.ID, err)
			continue
		}

		// 删除本地文件
		os.Remove(localPath)
		log.Printf("Reuploader: successfully re-uploaded diagram %d to OSS", a.ID)
	}
}
