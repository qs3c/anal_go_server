package handler

import (
	"errors"

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
			response.ServerError(c, "")
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
