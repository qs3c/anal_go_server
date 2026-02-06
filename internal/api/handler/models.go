package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
)

type ModelsHandler struct {
	cfg *config.Config
}

func NewModelsHandler(cfg *config.Config) *ModelsHandler {
	return &ModelsHandler{cfg: cfg}
}

// List 获取模型列表
// GET /api/v1/models
func (h *ModelsHandler) List(c *gin.Context) {
	models := make([]map[string]interface{}, len(h.cfg.Models))

	for i, m := range h.cfg.Models {
		models[i] = map[string]interface{}{
			"name":           m.Name,
			"display_name":   m.DisplayName,
			"required_level": m.RequiredLevel,
			"description":    m.Description,
			"available":      m.APIKey != "",
		}
	}

	response.Success(c, gin.H{
		"models": models,
	})
}
