package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type CommunityHandler struct {
	communityService *service.CommunityService
}

func NewCommunityHandler(communityService *service.CommunityService) *CommunityHandler {
	return &CommunityHandler{
		communityService: communityService,
	}
}

// List 获取广场分析列表
// GET /api/v1/community/analyses
func (h *CommunityHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sort := c.DefaultQuery("sort", "latest")
	tags := c.Query("tags")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	items, total, err := h.communityService.ListPublicAnalyses(page, pageSize, sort, tags)
	if err != nil {
		response.ServerError(c, "")
		return
	}

	response.SuccessPage(c, total, page, pageSize, items)
}

// Get 获取广场分析详情
// GET /api/v1/community/analyses/:id
func (h *CommunityHandler) Get(c *gin.Context) {
	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	// 获取用户ID（可选）
	var userID *int64
	if id, ok := middleware.GetUserID(c); ok {
		userID = &id
	}

	detail, err := h.communityService.GetPublicAnalysis(analysisID, userID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisNotPublic:
			response.NotFoundError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.Success(c, detail)
}

// Like 点赞
// POST /api/v1/community/analyses/:id/like
func (h *CommunityHandler) Like(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	resp, err := h.communityService.Like(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisNotPublic:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "点赞成功", resp)
}

// Unlike 取消点赞
// DELETE /api/v1/community/analyses/:id/like
func (h *CommunityHandler) Unlike(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	resp, err := h.communityService.Unlike(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "已取消点赞", resp)
}

// Bookmark 收藏
// POST /api/v1/community/analyses/:id/bookmark
func (h *CommunityHandler) Bookmark(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	resp, err := h.communityService.Bookmark(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisNotPublic:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "收藏成功", resp)
}

// Unbookmark 取消收藏
// DELETE /api/v1/community/analyses/:id/bookmark
func (h *CommunityHandler) Unbookmark(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	resp, err := h.communityService.Unbookmark(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "已取消收藏", resp)
}
