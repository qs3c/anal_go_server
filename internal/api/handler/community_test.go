package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/service"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func setupCommunityHandler(t *testing.T) (*CommunityHandler, *testContext, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)

	cfg := &config.Config{}

	communityService := service.NewCommunityService(analysisRepo, interactionRepo, cfg)
	handler := NewCommunityHandler(communityService)

	ctx := &testContext{
		DB: db,
	}

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return handler, ctx, cleanup
}

func TestCommunityHandler_List_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true), testutil.WithTitle("Public 1"))
	testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true), testutil.WithTitle("Public 2"))
	testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(false), testutil.WithTitle("Private"))

	router := gin.New()
	router.GET("/community/analyses", handler.List)

	req := httptest.NewRequest("GET", "/community/analyses?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), data["total"])
}

func TestCommunityHandler_List_Empty(t *testing.T) {
	handler, _, cleanup := setupCommunityHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/community/analyses", handler.List)

	req := httptest.NewRequest("GET", "/community/analyses", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(0), data["total"])
}

func TestCommunityHandler_List_Pagination(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	for i := 0; i < 25; i++ {
		testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true), testutil.WithTitle(fmt.Sprintf("Analysis %d", i)))
	}

	router := gin.New()
	router.GET("/community/analyses", handler.List)

	// Page 1
	req := httptest.NewRequest("GET", "/community/analyses?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(25), data["total"])
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(10), data["page_size"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 10)

	// Page 3
	req = httptest.NewRequest("GET", "/community/analyses?page=3&page_size=10", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp = parseResponse(t, w)
	data, _ = resp.Data.(map[string]interface{})
	items, _ = data["items"].([]interface{})
	assert.Len(t, items, 5) // Last 5 items
}

func TestCommunityHandler_List_InvalidPagination(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true))

	router := gin.New()
	router.GET("/community/analyses", handler.List)

	// Invalid page (should default to 1)
	req := httptest.NewRequest("GET", "/community/analyses?page=-1&page_size=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(20), data["page_size"]) // Default page_size
}

func TestCommunityHandler_Get_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true), testutil.WithTitle("Public Analysis"))

	router := gin.New()
	router.GET("/community/analyses/:id", handler.Get)

	req := httptest.NewRequest("GET", fmt.Sprintf("/community/analyses/%d", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestCommunityHandler_Get_NotFound(t *testing.T) {
	handler, _, cleanup := setupCommunityHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/community/analyses/:id", handler.Get)

	req := httptest.NewRequest("GET", "/community/analyses/99999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestCommunityHandler_Get_NotPublic(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(false))

	router := gin.New()
	router.GET("/community/analyses/:id", handler.Get)

	req := httptest.NewRequest("GET", fmt.Sprintf("/community/analyses/%d", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestCommunityHandler_Get_InvalidID(t *testing.T) {
	handler, _, cleanup := setupCommunityHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/community/analyses/:id", handler.Get)

	req := httptest.NewRequest("GET", "/community/analyses/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestCommunityHandler_Get_WithUserInteraction(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	viewer := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	// Add like interaction
	testutil.TestInteraction(t, ctx.DB, viewer.ID, analysis.ID, "like")

	router := gin.New()
	router.Use(mockAuth(viewer.ID))
	router.GET("/community/analyses/:id", handler.Get)

	req := httptest.NewRequest("GET", fmt.Sprintf("/community/analyses/%d", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	userInteraction, ok := data["user_interaction"].(map[string]interface{})
	require.True(t, ok)
	assert.True(t, userInteraction["liked"].(bool))
}

func TestCommunityHandler_Like_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	liker := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	router.Use(mockAuth(liker.ID))
	router.POST("/community/analyses/:id/like", handler.Like)

	req := httptest.NewRequest("POST", fmt.Sprintf("/community/analyses/%d/like", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, data["liked"].(bool))
}

func TestCommunityHandler_Like_Unauthorized(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true))

	router := gin.New()
	// No auth middleware
	router.POST("/community/analyses/:id/like", handler.Like)

	req := httptest.NewRequest("POST", fmt.Sprintf("/community/analyses/%d/like", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestCommunityHandler_Like_NotFound(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/community/analyses/:id/like", handler.Like)

	req := httptest.NewRequest("POST", "/community/analyses/99999/like", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestCommunityHandler_Like_NotPublic(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	liker := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(false))

	router := gin.New()
	router.Use(mockAuth(liker.ID))
	router.POST("/community/analyses/:id/like", handler.Like)

	req := httptest.NewRequest("POST", fmt.Sprintf("/community/analyses/%d/like", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodePermissionDenied, resp.Code)
}

func TestCommunityHandler_Like_Idempotent(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	liker := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	router.Use(mockAuth(liker.ID))
	router.POST("/community/analyses/:id/like", handler.Like)

	// First like
	req := httptest.NewRequest("POST", fmt.Sprintf("/community/analyses/%d/like", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, response.CodeSuccess, parseResponse(t, w).Code)

	// Second like (should still succeed - idempotent)
	req = httptest.NewRequest("POST", fmt.Sprintf("/community/analyses/%d/like", analysis.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, data["liked"].(bool))
}

func TestCommunityHandler_Unlike_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	liker := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))
	testutil.TestInteraction(t, ctx.DB, liker.ID, analysis.ID, "like")

	router := gin.New()
	router.Use(mockAuth(liker.ID))
	router.DELETE("/community/analyses/:id/like", handler.Unlike)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/community/analyses/%d/like", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.False(t, data["liked"].(bool))
}

func TestCommunityHandler_Unlike_Unauthorized(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true))

	router := gin.New()
	// No auth middleware
	router.DELETE("/community/analyses/:id/like", handler.Unlike)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/community/analyses/%d/like", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestCommunityHandler_Unlike_NotLiked(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.DELETE("/community/analyses/:id/like", handler.Unlike)

	// Unlike without liking first (should be idempotent)
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/community/analyses/%d/like", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.False(t, data["liked"].(bool))
}

func TestCommunityHandler_Bookmark_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	bookmarker := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	router.Use(mockAuth(bookmarker.ID))
	router.POST("/community/analyses/:id/bookmark", handler.Bookmark)

	req := httptest.NewRequest("POST", fmt.Sprintf("/community/analyses/%d/bookmark", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, data["bookmarked"].(bool))
}

func TestCommunityHandler_Bookmark_Unauthorized(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true))

	router := gin.New()
	// No auth middleware
	router.POST("/community/analyses/:id/bookmark", handler.Bookmark)

	req := httptest.NewRequest("POST", fmt.Sprintf("/community/analyses/%d/bookmark", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestCommunityHandler_Bookmark_NotPublic(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	bookmarker := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(false))

	router := gin.New()
	router.Use(mockAuth(bookmarker.ID))
	router.POST("/community/analyses/:id/bookmark", handler.Bookmark)

	req := httptest.NewRequest("POST", fmt.Sprintf("/community/analyses/%d/bookmark", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodePermissionDenied, resp.Code)
}

func TestCommunityHandler_Unbookmark_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	bookmarker := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))
	testutil.TestInteraction(t, ctx.DB, bookmarker.ID, analysis.ID, "bookmark")

	router := gin.New()
	router.Use(mockAuth(bookmarker.ID))
	router.DELETE("/community/analyses/:id/bookmark", handler.Unbookmark)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/community/analyses/%d/bookmark", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.False(t, data["bookmarked"].(bool))
}

func TestCommunityHandler_Unbookmark_Unauthorized(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, user.ID, testutil.WithPublic(true))

	router := gin.New()
	// No auth middleware
	router.DELETE("/community/analyses/:id/bookmark", handler.Unbookmark)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/community/analyses/%d/bookmark", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestCommunityHandler_Unbookmark_NotBookmarked(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	user := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.DELETE("/community/analyses/:id/bookmark", handler.Unbookmark)

	// Unbookmark without bookmarking first (should be idempotent)
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/community/analyses/%d/bookmark", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.False(t, data["bookmarked"].(bool))
}

func TestCommunityHandler_Like_InvalidID(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/community/analyses/:id/like", handler.Like)

	req := httptest.NewRequest("POST", "/community/analyses/invalid/like", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestCommunityHandler_Bookmark_InvalidID(t *testing.T) {
	handler, ctx, cleanup := setupCommunityHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/community/analyses/:id/bookmark", handler.Bookmark)

	req := httptest.NewRequest("POST", "/community/analyses/invalid/bookmark", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}
