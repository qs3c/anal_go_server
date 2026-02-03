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
