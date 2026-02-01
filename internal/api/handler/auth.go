package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register 用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.authService.Register(&req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailExists):
			response.ParamError(c, err.Error())
		case errors.Is(err, service.ErrUsernameExists):
			response.ParamError(c, err.Error())
		default:
			// 记录详细错误信息用于调试
			fmt.Printf("Register error: %v\n", err)
			response.ServerError(c, err.Error())
		}
		return
	}

	response.SuccessWithMessage(c, "注册成功，请查收验证邮件", resp)
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			response.AuthError(c, err.Error())
		case errors.Is(err, service.ErrEmailNotVerified):
			response.AuthError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "登录成功", resp)
}

// VerifyEmail 验证邮箱
// POST /api/v1/auth/verify-email
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.authService.VerifyEmail(req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidVerifyCode):
			response.ParamError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "邮箱验证成功", resp)
}

// GithubAuth GitHub OAuth 登录
// GET /api/v1/auth/github
func (h *AuthHandler) GithubAuth(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		state = "random_state" // 生产环境应该使用随机值并存储验证
	}
	authURL := h.authService.GetGithubAuthURL(state)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// GithubCallback GitHub OAuth 回调
// GET /api/v1/auth/github/callback
func (h *AuthHandler) GithubCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.ParamError(c, "missing code parameter")
		return
	}

	resp, err := h.authService.GithubCallback(c.Request.Context(), code)
	if err != nil {
		response.ServerError(c, "GitHub 登录失败")
		return
	}

	// 重定向到前端，携带 token
	frontendURL := c.Query("redirect_uri")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, resp.Token)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}
