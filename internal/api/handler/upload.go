package handler

import (
	"io"
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
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.ParamError(c, "请上传文件")
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > h.cfg.Upload.MaxSize {
		response.ParamError(c, "文件过大，最大支持 100MB")
		return
	}

	// Check extension
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

	// Save to temp file
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

	// Parse ZIP
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
