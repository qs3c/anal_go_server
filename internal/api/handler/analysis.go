package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/service"
)

type AnalysisHandler struct {
	analysisService *service.AnalysisService
}

func NewAnalysisHandler(analysisService *service.AnalysisService) *AnalysisHandler {
	return &AnalysisHandler{
		analysisService: analysisService,
	}
}

// Create 创建分析
// POST /api/v1/analyses
func (h *AnalysisHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	var req dto.CreateAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.analysisService.Create(userID, &req)
	if err != nil {
		switch err {
		case service.ErrQuotaExceeded:
			response.QuotaError(c, err.Error())
		case service.ErrDepthExceeded:
			response.ParamError(c, err.Error())
		case service.ErrModelDenied:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "创建成功", resp)
}

// List 获取分析列表
// GET /api/v1/analyses
func (h *AnalysisHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.AuthError(c, "")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	items, total, err := h.analysisService.List(userID, page, pageSize, search, status)
	if err != nil {
		response.ServerError(c, "")
		return
	}

	response.SuccessPage(c, total, page, pageSize, items)
}

// Get 获取分析详情
// GET /api/v1/analyses/:id
func (h *AnalysisHandler) Get(c *gin.Context) {
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

	detail, err := h.analysisService.GetByID(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.Success(c, detail)
}

// Update 更新分析
// PUT /api/v1/analyses/:id
func (h *AnalysisHandler) Update(c *gin.Context) {
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

	var req dto.UpdateAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	detail, err := h.analysisService.Update(userID, analysisID, &req)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "更新成功", detail)
}

// Delete 删除分析
// DELETE /api/v1/analyses/:id
func (h *AnalysisHandler) Delete(c *gin.Context) {
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

	if err := h.analysisService.Delete(userID, analysisID); err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// Share 分享到广场
// POST /api/v1/analyses/:id/share
func (h *AnalysisHandler) Share(c *gin.Context) {
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

	var req dto.ShareAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	if err := h.analysisService.Share(userID, analysisID, &req); err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		case service.ErrAnalysisNotComplete:
			response.ParamError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "分享成功", nil)
}

// Unshare 取消分享
// DELETE /api/v1/analyses/:id/share
func (h *AnalysisHandler) Unshare(c *gin.Context) {
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

	if err := h.analysisService.Unshare(userID, analysisID); err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.SuccessWithMessage(c, "已取消分享", nil)
}

// GetJobStatus 获取任务状态
// GET /api/v1/analyses/:id/job-status
func (h *AnalysisHandler) GetJobStatus(c *gin.Context) {
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

	status, err := h.analysisService.GetJobStatus(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			response.ServerError(c, "")
		}
		return
	}

	response.Success(c, status)
}

// GetDiagram 获取图表数据
// GET /api/v1/analyses/:id/diagram
func (h *AnalysisHandler) GetDiagram(c *gin.Context) {
	// 尝试获取用户ID（公开分析可以无需登录访问）
	userID, _ := middleware.GetUserID(c)

	analysisID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "无效的分析ID")
		return
	}

	data, err := h.analysisService.GetDiagramData(userID, analysisID)
	if err != nil {
		switch err {
		case service.ErrAnalysisNotFound:
			response.NotFoundError(c, err.Error())
		case service.ErrAnalysisPermission:
			response.PermissionError(c, err.Error())
		default:
			// 如果是 OSS 存储，返回提示使用 URL
			if err.Error() == "diagram stored in OSS, use diagram_oss_url directly" {
				response.ParamError(c, "请使用 diagram_oss_url 字段获取图表")
			} else {
				response.ServerError(c, err.Error())
			}
		}
		return
	}

	// 直接返回 JSON 数据
	c.Data(200, "application/json", data)
}
