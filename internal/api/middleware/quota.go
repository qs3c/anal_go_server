package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

// QuotaCheck 配额检查中间件
func QuotaCheck(quotaService *service.QuotaService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if !ok {
			response.AuthError(c, "")
			c.Abort()
			return
		}

		hasQuota, err := quotaService.CheckQuota(userID)
		if err != nil {
			response.ServerError(c, "配额检查失败")
			c.Abort()
			return
		}

		if !hasQuota {
			response.QuotaError(c, "今日分析配额已用完")
			c.Abort()
			return
		}

		c.Next()
	}
}
