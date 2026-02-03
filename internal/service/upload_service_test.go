package service

import (
	"archive/zip"
	"os"
	"path/filepath"
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
