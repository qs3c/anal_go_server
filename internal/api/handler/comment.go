package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

// List 获取评论列表
// GET /api/v1/analyses/:id/comments
func (h *CommentHandler) List(c *gin.Context) {
	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	items, total, err := h.commentService.ListByAnalysisID(analysisID, page, pageSize)
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

	response.SuccessPage(c, total, page, pageSize, items)
}

// Create 发表评论
// POST /api/v1/analyses/:id/comments
func (h *CommentHandler) Create(c *gin.Context) {
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

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	comment, err := h.commentService.Create(userID, analysisID, &req)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisNotPublic:
			response.PermissionError(c, err.Error())
		case service.ErrParentNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrParentNotInAnalysis:
			response.ParamError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "评论成功", comment)
}

// Delete 删除评论
// DELETE /api/v1/comments/:id
func (h *CommentHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的评论ID")
		return
	}

	if err := h.commentService.Delete(userID, commentID); err != nil {
		switch err {
		case service.ErrCommentNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrCommentPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}
