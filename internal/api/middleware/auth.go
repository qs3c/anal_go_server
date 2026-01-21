package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/pkg/jwt"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
)

const (
	UserIDKey = "userID"
)

// Auth JWT 认证中间件
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.AuthError(c, "请提供认证信息")
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			response.AuthError(c, "认证格式错误")
			c.Abort()
			return
		}

		claims, err := jwt.ParseToken(tokenString, jwtSecret)
		if err != nil {
			response.AuthError(c, "认证失败或已过期")
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

// OptionalAuth 可选认证中间件（不强制要求登录）
func OptionalAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.Next()
			return
		}

		claims, err := jwt.ParseToken(tokenString, jwtSecret)
		if err == nil {
			c.Set(UserIDKey, claims.UserID)
		}

		c.Next()
	}
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) (int64, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}
	id, ok := userID.(int64)
	return id, ok
}
