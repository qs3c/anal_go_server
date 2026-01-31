package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}

func TestSuccess(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Success(c, gin.H{"key": "value"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeSuccess, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", data["key"])
}

func TestSuccess_NilData(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Success(c, nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeSuccess, resp.Code)
	assert.Nil(t, resp.Data)
}

func TestSuccessWithMessage(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		SuccessWithMessage(c, "操作成功", gin.H{"result": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeSuccess, resp.Code)
	assert.Equal(t, "操作成功", resp.Message)
}

func TestSuccessPage(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		items := []string{"item1", "item2", "item3"}
		SuccessPage(c, 100, 1, 10, items)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(100), data["total"])
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(10), data["page_size"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 3)
}

func TestSuccessPage_EmptyItems(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		SuccessPage(c, 0, 1, 10, []string{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(0), data["total"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 0)
}

func TestError(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Error(c, CodeServerError, "自定义错误消息")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeServerError, resp.Code)
	assert.Equal(t, "自定义错误消息", resp.Message)
	assert.Nil(t, resp.Data)
}

func TestError_DefaultMessage(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Error(c, CodeServerError, "")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, CodeServerError, resp.Code)
	assert.Equal(t, "服务器内部错误", resp.Message)
}

func TestParamError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantMessage string
	}{
		{
			name:        "with custom message",
			message:     "参数格式不正确",
			wantMessage: "参数格式不正确",
		},
		{
			name:        "with empty message",
			message:     "",
			wantMessage: "参数错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				ParamError(c, tt.message)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := parseResponse(t, w)
			assert.Equal(t, CodeParamError, resp.Code)
			assert.Equal(t, tt.wantMessage, resp.Message)
		})
	}
}

func TestAuthError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantMessage string
	}{
		{
			name:        "with custom message",
			message:     "token已过期",
			wantMessage: "token已过期",
		},
		{
			name:        "with empty message",
			message:     "",
			wantMessage: "认证失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				AuthError(c, tt.message)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := parseResponse(t, w)
			assert.Equal(t, CodeAuthFailed, resp.Code)
			assert.Equal(t, tt.wantMessage, resp.Message)
		})
	}
}

func TestPermissionError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantMessage string
	}{
		{
			name:        "with custom message",
			message:     "无权访问此资源",
			wantMessage: "无权访问此资源",
		},
		{
			name:        "with empty message",
			message:     "",
			wantMessage: "权限不足",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				PermissionError(c, tt.message)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := parseResponse(t, w)
			assert.Equal(t, CodePermissionDenied, resp.Code)
			assert.Equal(t, tt.wantMessage, resp.Message)
		})
	}
}

func TestNotFoundError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantMessage string
	}{
		{
			name:        "with custom message",
			message:     "用户不存在",
			wantMessage: "用户不存在",
		},
		{
			name:        "with empty message",
			message:     "",
			wantMessage: "资源不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				NotFoundError(c, tt.message)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := parseResponse(t, w)
			assert.Equal(t, CodeResourceNotFound, resp.Code)
			assert.Equal(t, tt.wantMessage, resp.Message)
		})
	}
}

func TestQuotaError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantMessage string
	}{
		{
			name:        "with custom message",
			message:     "今日额度已用完",
			wantMessage: "今日额度已用完",
		},
		{
			name:        "with empty message",
			message:     "",
			wantMessage: "配额不足",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				QuotaError(c, tt.message)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := parseResponse(t, w)
			assert.Equal(t, CodeQuotaExceeded, resp.Code)
			assert.Equal(t, tt.wantMessage, resp.Message)
		})
	}
}

func TestDuplicateError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantMessage string
	}{
		{
			name:        "with custom message",
			message:     "已经点赞过了",
			wantMessage: "已经点赞过了",
		},
		{
			name:        "with empty message",
			message:     "",
			wantMessage: "重复操作",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				DuplicateError(c, tt.message)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := parseResponse(t, w)
			assert.Equal(t, CodeDuplicateAction, resp.Code)
			assert.Equal(t, tt.wantMessage, resp.Message)
		})
	}
}

func TestServerError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantMessage string
	}{
		{
			name:        "with custom message",
			message:     "数据库连接失败",
			wantMessage: "数据库连接失败",
		},
		{
			name:        "with empty message",
			message:     "",
			wantMessage: "服务器内部错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				ServerError(c, tt.message)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := parseResponse(t, w)
			assert.Equal(t, CodeServerError, resp.Code)
			assert.Equal(t, tt.wantMessage, resp.Message)
		})
	}
}

func TestError_UnknownCode(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Error(c, 9999, "") // Unknown code
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, 9999, resp.Code)
	assert.Empty(t, resp.Message) // Unknown code has no default message
}
