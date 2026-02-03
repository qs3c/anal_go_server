package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/api/middleware"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/service"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

// testContext 本地测试上下文
type testContext struct {
	DB *gorm.DB
}

func setupAnalysisHandler(t *testing.T) (*AnalysisHandler, *testContext, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)

	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free":  {DailyQuota: 5, MaxDepth: 3},
				"basic": {DailyQuota: 30, MaxDepth: 5},
				"pro":   {DailyQuota: 100, MaxDepth: 10},
			},
		},
		Models: []config.ModelConfig{
			{Name: "gpt-3.5-turbo", RequiredLevel: "free"},
		},
	}

	quotaService := service.NewQuotaService(userRepo, cfg)
	analysisService := service.NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, nil, nil, cfg)
	handler := NewAnalysisHandler(analysisService)

	ctx := &testContext{
		DB: db,
	}

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return handler, ctx, cleanup
}

// mockAuth 模拟认证中间件
func mockAuth(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
}

func TestAnalysisHandler_Create_Manual_Success(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/analyses", handler.Create)

	req := dto.CreateAnalysisRequest{
		Title:        "Test Analysis",
		CreationType: "manual",
	}

	w := performRequest(router, "POST", "/analyses", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	// Verify response data
	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotZero(t, data["analysis_id"])
}

func TestAnalysisHandler_Create_AI_Success(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB, testutil.WithQuotaUsed(0))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/analyses", handler.Create)

	req := dto.CreateAnalysisRequest{
		Title:         "AI Analysis",
		CreationType:  "ai",
		RepoURL:       "https://github.com/example/repo",
		StartStruct:   "main.Config",
		AnalysisDepth: 3,
		ModelName:     "gpt-3.5-turbo",
	}

	w := performRequest(router, "POST", "/analyses", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestAnalysisHandler_Create_QuotaExceeded(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB, testutil.WithQuotaUsed(5))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/analyses", handler.Create)

	req := dto.CreateAnalysisRequest{
		Title:         "AI Analysis",
		CreationType:  "ai",
		RepoURL:       "https://github.com/example/repo",
		StartStruct:   "main.Config",
		AnalysisDepth: 3,
		ModelName:     "gpt-3.5-turbo",
	}

	w := performRequest(router, "POST", "/analyses", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeQuotaExceeded, resp.Code)
}

func TestAnalysisHandler_Create_Unauthorized(t *testing.T) {
	handler, _, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	router := gin.New()
	// No auth middleware
	router.POST("/analyses", handler.Create)

	req := dto.CreateAnalysisRequest{
		Title:        "Test Analysis",
		CreationType: "manual",
	}

	w := performRequest(router, "POST", "/analyses", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestAnalysisHandler_List_Success(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithTitle("Analysis 1"))
	testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithTitle("Analysis 2"))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.GET("/analyses", handler.List)

	req := httptest.NewRequest("GET", "/analyses?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), data["total"])
}

func TestAnalysisHandler_List_WithSearch(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithTitle("Go Project"))
	testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithTitle("Python Project"))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.GET("/analyses", handler.List)

	req := httptest.NewRequest("GET", "/analyses?search=Go", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["total"])
}

func TestAnalysisHandler_Get_Success(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithTitle("Test Analysis"))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.GET("/analyses/:id", handler.Get)

	req := httptest.NewRequest("GET", fmt.Sprintf("/analyses/%d", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestAnalysisHandler_Get_NotFound(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.GET("/analyses/:id", handler.Get)

	req := httptest.NewRequest("GET", "/analyses/99999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestAnalysisHandler_Get_NoPermission(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user1 := testutil.TestUser(t, ctx.DB)
	user2 := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user1.ID)

	router := gin.New()
	router.Use(mockAuth(user2.ID))
	router.GET("/analyses/:id", handler.Get)

	req := httptest.NewRequest("GET", fmt.Sprintf("/analyses/%d", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodePermissionDenied, resp.Code)
}

func TestAnalysisHandler_Update_Success(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithTitle("Original"))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.PUT("/analyses/:id", handler.Update)

	newTitle := "Updated Title"
	req := dto.UpdateAnalysisRequest{
		Title: &newTitle,
	}

	jsonBytes, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("PUT", fmt.Sprintf("/analyses/%d", analysis.ID), bytes.NewBuffer(jsonBytes))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestAnalysisHandler_Delete_Success(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.DELETE("/analyses/:id", handler.Delete)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/analyses/%d", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestAnalysisHandler_Delete_NotFound(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.DELETE("/analyses/:id", handler.Delete)

	req := httptest.NewRequest("DELETE", "/analyses/99999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestAnalysisHandler_Share_Success(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithStatus("completed"))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/analyses/:id/share", handler.Share)

	req := dto.ShareAnalysisRequest{
		ShareTitle:       "Shared Analysis",
		ShareDescription: "Description",
		Tags:             []string{"go", "test"},
	}

	jsonBytes, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", fmt.Sprintf("/analyses/%d/share", analysis.ID), bytes.NewBuffer(jsonBytes))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestAnalysisHandler_Share_NotCompleted(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithStatus("draft"))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/analyses/:id/share", handler.Share)

	req := dto.ShareAnalysisRequest{
		ShareTitle: "Title",
	}

	jsonBytes, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", fmt.Sprintf("/analyses/%d/share", analysis.ID), bytes.NewBuffer(jsonBytes))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestAnalysisHandler_Unshare_Success(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.DELETE("/analyses/:id/share", handler.Unshare)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/analyses/%d/share", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestAnalysisHandler_GetJobStatus_Success(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID)
	testutil.TestJob(t, ctx.DB, user.ID, analysis.ID, "processing")

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.GET("/analyses/:id/job-status", handler.GetJobStatus)

	req := httptest.NewRequest("GET", fmt.Sprintf("/analyses/%d/job-status", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestAnalysisHandler_InvalidID(t *testing.T) {
	handler, ctx, cleanup := setupAnalysisHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.GET("/analyses/:id", handler.Get)

	req := httptest.NewRequest("GET", "/analyses/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}
