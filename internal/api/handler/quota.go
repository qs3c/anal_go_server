package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type QuotaHandler struct {
	quotaService *service.QuotaService
}

func NewQuotaHandler(quotaService *service.QuotaService) *QuotaHandler {
	return &QuotaHandler{
		quotaService: quotaService,
	}
}

// GetQuota 获取当前用户配额信息
// GET /api/v1/user/quota
func (h *QuotaHandler) GetQuota(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	info, err := h.quotaService.GetQuotaInfo(userID)
	if err != nil {
		response.ServerError(c, "")
		return
	}

	response.Success(c, info)
}
