package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误码定义
const (
	CodeSuccess          = 0
	CodeParamError       = 1000
	CodeAuthFailed       = 1001
	CodePermissionDenied = 1002
	CodeResourceNotFound = 1003
	CodeQuotaExceeded    = 1004
	CodeDuplicateAction  = 1005
	CodeServerError      = 5000
)

// 错误码对应的默认消息
var codeMessages = map[int]string{
	CodeSuccess:          "success",
	CodeParamError:       "参数错误",
	CodeAuthFailed:       "认证失败",
	CodePermissionDenied: "权限不足",
	CodeResourceNotFound: "资源不存在",
	CodeQuotaExceeded:    "配额不足",
	CodeDuplicateAction:  "重复操作",
	CodeServerError:      "服务器内部错误",
}

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PageData 分页数据结构
type PageData struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Items    interface{} `json:"items"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带自定义消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// SuccessPage 分页成功响应
func SuccessPage(c *gin.Context, total int64, page, pageSize int, items interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data: PageData{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
			Items:    items,
		},
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	if message == "" {
		message = codeMessages[code]
	}
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// ParamError 参数错误
func ParamError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeParamError]
	}
	Error(c, CodeParamError, message)
}

// AuthError 认证失败
func AuthError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeAuthFailed]
	}
	Error(c, CodeAuthFailed, message)
}

// PermissionError 权限不足
func PermissionError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodePermissionDenied]
	}
	Error(c, CodePermissionDenied, message)
}

// NotFoundError 资源不存在
func NotFoundError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeResourceNotFound]
	}
	Error(c, CodeResourceNotFound, message)
}

// QuotaError 配额不足
func QuotaError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeQuotaExceeded]
	}
	Error(c, CodeQuotaExceeded, message)
}

// DuplicateError 重复操作
func DuplicateError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeDuplicateAction]
	}
	Error(c, CodeDuplicateAction, message)
}

// ServerError 服务器错误
func ServerError(c *gin.Context, message string) {
	if message == "" {
		message = codeMessages[CodeServerError]
	}
	Error(c, CodeServerError, message)
}
