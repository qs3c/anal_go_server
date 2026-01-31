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

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/service"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func setupCommentHandler(t *testing.T) (*CommentHandler, *testContext, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)

	cfg := &config.Config{}

	commentService := service.NewCommentService(commentRepo, analysisRepo, userRepo, cfg)
	handler := NewCommentHandler(commentService)

	ctx := &testContext{
		DB: db,
	}

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return handler, ctx, cleanup
}

func TestCommentHandler_List_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	testutil.TestComment(t, ctx.DB, commenter.ID, analysis.ID, "Comment 1")
	testutil.TestComment(t, ctx.DB, commenter.ID, analysis.ID, "Comment 2")

	router := gin.New()
	router.GET("/analyses/:id/comments", handler.List)

	req := httptest.NewRequest("GET", fmt.Sprintf("/analyses/%d/comments", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), data["total"])
}

func TestCommentHandler_List_Empty(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	router.GET("/analyses/:id/comments", handler.List)

	req := httptest.NewRequest("GET", fmt.Sprintf("/analyses/%d/comments", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(0), data["total"])
}

func TestCommentHandler_List_AnalysisNotFound(t *testing.T) {
	handler, _, cleanup := setupCommentHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/analyses/:id/comments", handler.List)

	req := httptest.NewRequest("GET", "/analyses/99999/comments", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestCommentHandler_List_AnalysisNotPublic(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(false))

	router := gin.New()
	router.GET("/analyses/:id/comments", handler.List)

	req := httptest.NewRequest("GET", fmt.Sprintf("/analyses/%d/comments", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestCommentHandler_List_InvalidID(t *testing.T) {
	handler, _, cleanup := setupCommentHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/analyses/:id/comments", handler.List)

	req := httptest.NewRequest("GET", "/analyses/invalid/comments", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestCommentHandler_List_Pagination(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	for i := 0; i < 25; i++ {
		testutil.TestComment(t, ctx.DB, commenter.ID, analysis.ID, fmt.Sprintf("Comment %d", i))
	}

	router := gin.New()
	router.GET("/analyses/:id/comments", handler.List)

	req := httptest.NewRequest("GET", fmt.Sprintf("/analyses/%d/comments?page=1&page_size=10", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(25), data["total"])
	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 10)
}

func TestCommentHandler_List_WithReplies(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	parentComment := testutil.TestComment(t, ctx.DB, commenter.ID, analysis.ID, "Parent comment")
	testutil.TestReply(t, ctx.DB, commenter.ID, analysis.ID, parentComment.ID, "Reply 1")
	testutil.TestReply(t, ctx.DB, commenter.ID, analysis.ID, parentComment.ID, "Reply 2")

	router := gin.New()
	router.GET("/analyses/:id/comments", handler.List)

	req := httptest.NewRequest("GET", fmt.Sprintf("/analyses/%d/comments", analysis.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	// Total should be 1 (only parent comments are counted in pagination)
	assert.Equal(t, float64(1), data["total"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 1)

	// Check that replies are included
	firstItem := items[0].(map[string]interface{})
	replies, ok := firstItem["replies"].([]interface{})
	require.True(t, ok)
	assert.Len(t, replies, 2)
}

func TestCommentHandler_Create_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	router.Use(mockAuth(commenter.ID))
	router.POST("/analyses/:id/comments", handler.Create)

	reqBody := dto.CreateCommentRequest{
		Content: "This is a test comment",
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", fmt.Sprintf("/analyses/%d/comments", analysis.ID), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "This is a test comment", data["content"])
	assert.NotZero(t, data["id"])
}

func TestCommentHandler_Create_Unauthorized(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	// No auth middleware
	router.POST("/analyses/:id/comments", handler.Create)

	reqBody := dto.CreateCommentRequest{
		Content: "Test comment",
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", fmt.Sprintf("/analyses/%d/comments", analysis.ID), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestCommentHandler_Create_AnalysisNotFound(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/analyses/:id/comments", handler.Create)

	reqBody := dto.CreateCommentRequest{
		Content: "Test comment",
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/analyses/99999/comments", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestCommentHandler_Create_AnalysisNotPublic(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(false))

	router := gin.New()
	router.Use(mockAuth(commenter.ID))
	router.POST("/analyses/:id/comments", handler.Create)

	reqBody := dto.CreateCommentRequest{
		Content: "Test comment",
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", fmt.Sprintf("/analyses/%d/comments", analysis.ID), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodePermissionDenied, resp.Code)
}

func TestCommentHandler_Create_InvalidRequest(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	router.Use(mockAuth(commenter.ID))
	router.POST("/analyses/:id/comments", handler.Create)

	// Empty content
	reqBody := dto.CreateCommentRequest{
		Content: "",
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", fmt.Sprintf("/analyses/%d/comments", analysis.ID), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestCommentHandler_Create_InvalidID(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/analyses/:id/comments", handler.Create)

	reqBody := dto.CreateCommentRequest{
		Content: "Test comment",
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/analyses/invalid/comments", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestCommentHandler_Create_Reply_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))
	parentComment := testutil.TestComment(t, ctx.DB, commenter.ID, analysis.ID, "Parent comment")

	router := gin.New()
	router.Use(mockAuth(commenter.ID))
	router.POST("/analyses/:id/comments", handler.Create)

	parentID := parentComment.ID
	reqBody := dto.CreateCommentRequest{
		Content:  "This is a reply",
		ParentID: &parentID,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", fmt.Sprintf("/analyses/%d/comments", analysis.ID), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "This is a reply", data["content"])
	assert.Equal(t, float64(parentID), data["parent_id"])
}

func TestCommentHandler_Create_Reply_ParentNotFound(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	router := gin.New()
	router.Use(mockAuth(commenter.ID))
	router.POST("/analyses/:id/comments", handler.Create)

	nonExistentParentID := int64(99999)
	reqBody := dto.CreateCommentRequest{
		Content:  "This is a reply",
		ParentID: &nonExistentParentID,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", fmt.Sprintf("/analyses/%d/comments", analysis.ID), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestCommentHandler_Create_Reply_ParentNotInAnalysis(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis1 := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))
	analysis2 := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	// Create comment on analysis1
	parentComment := testutil.TestComment(t, ctx.DB, commenter.ID, analysis1.ID, "Parent comment")

	router := gin.New()
	router.Use(mockAuth(commenter.ID))
	router.POST("/analyses/:id/comments", handler.Create)

	// Try to reply on analysis2 with parent from analysis1
	parentID := parentComment.ID
	reqBody := dto.CreateCommentRequest{
		Content:  "This is a reply",
		ParentID: &parentID,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", fmt.Sprintf("/analyses/%d/comments", analysis2.ID), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestCommentHandler_Delete_Success(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))
	comment := testutil.TestComment(t, ctx.DB, commenter.ID, analysis.ID, "Comment to delete")

	router := gin.New()
	router.Use(mockAuth(commenter.ID))
	router.DELETE("/comments/:id", handler.Delete)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/comments/%d", comment.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestCommentHandler_Delete_Unauthorized(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))
	comment := testutil.TestComment(t, ctx.DB, commenter.ID, analysis.ID, "Comment")

	router := gin.New()
	// No auth middleware
	router.DELETE("/comments/:id", handler.Delete)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/comments/%d", comment.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestCommentHandler_Delete_NotFound(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.DELETE("/comments/:id", handler.Delete)

	req := httptest.NewRequest("DELETE", "/comments/99999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeResourceNotFound, resp.Code)
}

func TestCommentHandler_Delete_NoPermission(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	otherUser := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))
	comment := testutil.TestComment(t, ctx.DB, commenter.ID, analysis.ID, "Comment")

	router := gin.New()
	router.Use(mockAuth(otherUser.ID)) // Different user trying to delete
	router.DELETE("/comments/:id", handler.Delete)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/comments/%d", comment.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodePermissionDenied, resp.Code)
}

func TestCommentHandler_Delete_InvalidID(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.DELETE("/comments/:id", handler.Delete)

	req := httptest.NewRequest("DELETE", "/comments/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestCommentHandler_Delete_WithReplies(t *testing.T) {
	handler, ctx, cleanup := setupCommentHandler(t)
	defer cleanup()

	author := testutil.TestUser(t, ctx.DB)
	commenter := testutil.TestUser(t, ctx.DB)
	analysis := testutil.TestAnalysis(t, ctx.DB, author.ID, testutil.WithPublic(true))

	parentComment := testutil.TestComment(t, ctx.DB, commenter.ID, analysis.ID, "Parent comment")
	testutil.TestReply(t, ctx.DB, commenter.ID, analysis.ID, parentComment.ID, "Reply 1")
	testutil.TestReply(t, ctx.DB, commenter.ID, analysis.ID, parentComment.ID, "Reply 2")

	router := gin.New()
	router.Use(mockAuth(commenter.ID))
	router.DELETE("/comments/:id", handler.Delete)

	// Delete parent comment (should also delete replies)
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/comments/%d", parentComment.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}
