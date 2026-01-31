package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
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

func setupUserHandler(t *testing.T) (*UserHandler, *testContext, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	cfg := &config.Config{}

	// ossClient is nil for tests (uploads will fail gracefully)
	userService := service.NewUserService(userRepo, nil, cfg)
	handler := NewUserHandler(userService)

	ctx := &testContext{
		DB: db,
	}

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return handler, ctx, cleanup
}

func TestUserHandler_GetProfile_Success(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB, testutil.WithUsername("profileuser"))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.GET("/profile", handler.GetProfile)

	req := httptest.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "profileuser", data["username"])
}

func TestUserHandler_GetProfile_Unauthorized(t *testing.T) {
	handler, _, cleanup := setupUserHandler(t)
	defer cleanup()

	router := gin.New()
	// No auth middleware
	router.GET("/profile", handler.GetProfile)

	req := httptest.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestUserHandler_UpdateProfile_Success(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB, testutil.WithUsername("oldname"))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.PUT("/profile", handler.UpdateProfile)

	newUsername := "newname"
	newBio := "New bio"
	reqBody := dto.UpdateProfileRequest{
		Username: &newUsername,
		Bio:      &newBio,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "newname", data["username"])
	assert.Equal(t, "New bio", data["bio"])
}

func TestUserHandler_UpdateProfile_Unauthorized(t *testing.T) {
	handler, _, cleanup := setupUserHandler(t)
	defer cleanup()

	router := gin.New()
	// No auth middleware
	router.PUT("/profile", handler.UpdateProfile)

	newUsername := "newname"
	reqBody := dto.UpdateProfileRequest{
		Username: &newUsername,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestUserHandler_UpdateProfile_DuplicateUsername(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user1 := testutil.TestUser(t, ctx.DB, testutil.WithUsername("existinguser"))
	user2 := testutil.TestUser(t, ctx.DB, testutil.WithUsername("anotheruser"))

	router := gin.New()
	router.Use(mockAuth(user2.ID))
	router.PUT("/profile", handler.UpdateProfile)

	// Try to use existing username
	duplicateName := user1.Username
	reqBody := dto.UpdateProfileRequest{
		Username: &duplicateName,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestUserHandler_UpdateProfile_InvalidRequest(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.PUT("/profile", handler.UpdateProfile)

	// Invalid JSON
	req := httptest.NewRequest("PUT", "/profile", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestUserHandler_UpdateProfile_OnlyBio(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB, testutil.WithUsername("keepname"))

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.PUT("/profile", handler.UpdateProfile)

	newBio := "Just updating bio"
	reqBody := dto.UpdateProfileRequest{
		Bio: &newBio,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "keepname", data["username"])
	assert.Equal(t, "Just updating bio", data["bio"])
}

func TestUserHandler_UploadAvatar_Unauthorized(t *testing.T) {
	handler, _, cleanup := setupUserHandler(t)
	defer cleanup()

	router := gin.New()
	// No auth middleware
	router.POST("/avatar", handler.UploadAvatar)

	req := httptest.NewRequest("POST", "/avatar", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestUserHandler_UploadAvatar_NoFile(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/avatar", handler.UploadAvatar)

	req := httptest.NewRequest("POST", "/avatar", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestUserHandler_UploadAvatar_InvalidContentType(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/avatar", handler.UploadAvatar)

	// Create multipart form with invalid content type
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	io.WriteString(part, "test content")
	writer.Close()

	req := httptest.NewRequest("POST", "/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestUserHandler_UploadAvatar_FileTooLarge(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/avatar", handler.UploadAvatar)

	// Create multipart form with large file (simulate > 5MB)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="large.jpg"`}
	h["Content-Type"] = []string{"image/jpeg"}
	part, _ := writer.CreatePart(h)

	// Write more than 5MB
	largeData := make([]byte, 6*1024*1024)
	part.Write(largeData)
	writer.Close()

	req := httptest.NewRequest("POST", "/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func createMultipartFileRequest(t *testing.T, fieldName, fileName, contentType string, content []byte) (*http.Request, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="` + fieldName + `"; filename="` + fileName + `"`}
	h["Content-Type"] = []string{contentType}
	part, err := writer.CreatePart(h)
	require.NoError(t, err)

	_, err = part.Write(content)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/avatar", body)
	return req, writer.FormDataContentType()
}

func TestUserHandler_UploadAvatar_ValidJPEG(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/avatar", handler.UploadAvatar)

	// Create valid JPEG file (small)
	content := make([]byte, 1024)
	req, contentType := createMultipartFileRequest(t, "file", "avatar.jpg", "image/jpeg", content)
	req.Header.Set("Content-Type", contentType)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	// Will fail because OSS client is nil, but should get past validation
	assert.Equal(t, http.StatusOK, w.Code)
	// Expect server error since OSS is not configured
	assert.Equal(t, response.CodeServerError, resp.Code)
}

func TestUserHandler_UploadAvatar_ValidPNG(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/avatar", handler.UploadAvatar)

	content := make([]byte, 1024)
	req, contentType := createMultipartFileRequest(t, "file", "avatar.png", "image/png", content)
	req.Header.Set("Content-Type", contentType)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	// Expect server error since OSS is not configured
	assert.Equal(t, response.CodeServerError, resp.Code)
}

func TestUserHandler_UploadAvatar_ValidWebP(t *testing.T) {
	handler, ctx, cleanup := setupUserHandler(t)
	defer cleanup()

	user := testutil.TestUser(t, ctx.DB)

	router := gin.New()
	router.Use(mockAuth(user.ID))
	router.POST("/avatar", handler.UploadAvatar)

	content := make([]byte, 1024)
	req, contentType := createMultipartFileRequest(t, "file", "avatar.webp", "image/webp", content)
	req.Header.Set("Content-Type", contentType)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	// Expect server error since OSS is not configured
	assert.Equal(t, response.CodeServerError, resp.Code)
}
