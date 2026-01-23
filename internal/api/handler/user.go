package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetProfile 获取当前用户信息
// GET /api/v1/user/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		response.ServerError(c, "")
		return
	}

	response.Success(c, profile)
}

// UpdateProfile 更新用户信息
// PUT /api/v1/user/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	profile, err := h.userService.UpdateProfile(userID, &req)
	if err != nil {
		if err == service.ErrUsernameExists {
			response.ParamError(c, err.Error())
			return
		}
		response.ServerError(c, "")
		return
	}

	response.SuccessWithMessage(c, "更新成功", profile)
}

// UploadAvatar 上传头像
// POST /api/v1/user/avatar
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.ParamError(c, "请选择文件")
		return
	}

	// 验证文件大小 (5MB)
	if file.Size > 5*1024*1024 {
		response.ParamError(c, "文件大小不能超过5MB")
		return
	}

	// 验证文件类型
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		response.ParamError(c, "只支持 jpg/png/webp 格式")
		return
	}

	// 打开文件
	f, err := file.Open()
	if err != nil {
		response.ServerError(c, "文件读取失败")
		return
	}
	defer f.Close()

	// 上传到 OSS
	avatarURL, err := h.userService.UploadAvatar(userID, f, file.Filename)
	if err != nil {
		response.ServerError(c, "上传失败")
		return
	}

	response.SuccessWithMessage(c, "上传成功", gin.H{
		"avatar_url": avatarURL,
	})
}
