package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
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

func TestUploadHandler_Parse_NoFile(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/parse", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/upload/parse", handler.Parse)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 1000, resp.Code) // ParamError
}

func TestUploadHandler_Parse_WrongExtension(t *testing.T) {
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

	// 创建带错误扩展名的请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("some content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/parse", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/upload/parse", handler.Parse)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 1000, resp.Code) // ParamError
	assert.Contains(t, resp.Message, "ZIP")
}

func TestUploadHandler_Parse_FileTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Upload: config.UploadConfig{
			MaxSize:           100, // 100 bytes
			TempDir:           t.TempDir(),
			ExpireHours:       1,
			AllowedExtensions: []string{".zip"},
		},
	}
	uploadService := service.NewUploadService(cfg)
	handler := NewUploadHandler(uploadService, cfg)

	// 创建测试 ZIP (会大于 100 bytes)
	zipContent := createTestZipContent(t, map[string]string{
		"main.go": `package main
type App struct { Name string }
`,
	})

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
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 1000, resp.Code) // ParamError
	assert.Contains(t, resp.Message, "过大")
}

func TestUploadHandler_Parse_NoGoFiles(t *testing.T) {
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

	// 创建不包含 Go 文件的 ZIP
	zipContent := createTestZipContent(t, map[string]string{
		"readme.txt": "hello",
	})

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
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 1000, resp.Code) // ParamError
	assert.Contains(t, resp.Message, "Go")
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
