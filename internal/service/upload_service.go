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
	return &UploadService{cfg: cfg}
}

// ParseZip 解析 ZIP 文件，提取 Go 文件和结构体信息
func (s *UploadService) ParseZip(zipPath string) (*dto.ParseUploadResponse, error) {
	uploadID, err := generateUploadID()
	if err != nil {
		return nil, err
	}

	extractDir := filepath.Join(s.cfg.Upload.TempDir, uploadID)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, err
	}

	if err := s.extractZip(zipPath, extractDir); err != nil {
		os.RemoveAll(extractDir)
		return nil, ErrInvalidZip
	}

	files, err := s.scanGoFiles(extractDir)
	if err != nil {
		os.RemoveAll(extractDir)
		return nil, err
	}

	if len(files) == 0 {
		os.RemoveAll(extractDir)
		return nil, ErrNoGoFiles
	}

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
		destPath := filepath.Join(destDir, f.Name)
		// Security: prevent zip slip attack
		if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

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

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		structs, err := s.parseStructs(path)
		if err != nil {
			return nil // skip files that fail to parse
		}

		if len(structs) > 0 {
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
