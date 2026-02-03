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
