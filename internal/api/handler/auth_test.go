package handler

import (
	"bytes"
	"encoding/json"
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

func init() {
	gin.SetMode(gin.TestMode)
}

func setupAuthHandler(t *testing.T) (*AuthHandler, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:      "test-secret-key",
			ExpireHours: 24,
		},
	}

	authService := service.NewAuthService(userRepo, cfg)
	handler := NewAuthHandler(authService)

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return handler, cleanup
}

func performRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBytes)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) response.Response {
	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}

func TestAuthHandler_Register_Success(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	router := gin.New()
	router.POST("/register", handler.Register)

	req := dto.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}

	w := performRequest(router, "POST", "/register", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeSuccess, resp.Code)
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	router := gin.New()
	router.POST("/register", handler.Register)

	req := dto.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser1",
		Password: "password123",
	}

	// First registration
	w := performRequest(router, "POST", "/register", req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Duplicate email
	req.Username = "testuser2"
	w = performRequest(router, "POST", "/register", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestAuthHandler_Register_InvalidRequest(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	router := gin.New()
	router.POST("/register", handler.Register)

	// Missing required fields
	req := map[string]string{
		"email": "invalid-email",
	}

	w := performRequest(router, "POST", "/register", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	router := gin.New()
	router.POST("/register", handler.Register)
	router.POST("/login", handler.Login)

	// Register first
	registerReq := dto.RegisterRequest{
		Email:    "login@example.com",
		Username: "loginuser",
		Password: "password123",
	}
	w := performRequest(router, "POST", "/register", registerReq)
	require.Equal(t, http.StatusOK, w.Code)

	// Login (note: email verification is required in production)
	loginReq := dto.LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	}
	w = performRequest(router, "POST", "/login", loginReq)
	resp := parseResponse(t, w)

	// Will fail because email is not verified
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	router := gin.New()
	router.POST("/login", handler.Login)

	req := dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "wrongpassword",
	}

	w := performRequest(router, "POST", "/login", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestAuthHandler_Login_InvalidRequest(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	router := gin.New()
	router.POST("/login", handler.Login)

	// Invalid request body
	req := map[string]string{}

	w := performRequest(router, "POST", "/login", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestAuthHandler_VerifyEmail_InvalidCode(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	router := gin.New()
	router.POST("/verify-email", handler.VerifyEmail)

	req := dto.VerifyEmailRequest{
		Code: "invalid-code",
	}

	w := performRequest(router, "POST", "/verify-email", req)
	resp := parseResponse(t, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}

func TestAuthHandler_GithubAuth_Redirect(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/github", handler.GithubAuth)

	req := httptest.NewRequest("GET", "/github?state=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
}

func TestAuthHandler_GithubCallback_MissingCode(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/callback", handler.GithubCallback)

	req := httptest.NewRequest("GET", "/callback", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeParamError, resp.Code)
}
