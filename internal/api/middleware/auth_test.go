package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/internal/pkg/jwt"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const testJWTSecret = "test-secret-key-for-middleware"

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) response.Response {
	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}

func TestAuth_Success(t *testing.T) {
	router := gin.New()
	router.Use(Auth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		userID, ok := GetUserID(c)
		assert.True(t, ok)
		assert.Equal(t, int64(123), userID)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	token, err := jwt.GenerateToken(123, testJWTSecret, 24)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_MissingHeader(t *testing.T) {
	router := gin.New()
	router.Use(Auth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestAuth_InvalidFormat_NoBearer(t *testing.T) {
	router := gin.New()
	router.Use(Auth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "some-token-without-bearer")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	router := gin.New()
	router.Use(Auth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestAuth_WrongSecret(t *testing.T) {
	router := gin.New()
	router.Use(Auth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	// Generate token with different secret
	token, err := jwt.GenerateToken(123, "different-secret", 24)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestAuth_ExpiredToken(t *testing.T) {
	router := gin.New()
	router.Use(Auth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	// Generate token with 0 hours (immediately expired)
	token, err := jwt.GenerateToken(123, testJWTSecret, 0)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestOptionalAuth_WithValidToken(t *testing.T) {
	router := gin.New()
	router.Use(OptionalAuth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if ok {
			c.JSON(http.StatusOK, gin.H{"user_id": userID, "authenticated": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	token, err := jwt.GenerateToken(456, testJWTSecret, 24)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.True(t, result["authenticated"].(bool))
	assert.Equal(t, float64(456), result["user_id"])
}

func TestOptionalAuth_WithoutToken(t *testing.T) {
	router := gin.New()
	router.Use(OptionalAuth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if ok {
			c.JSON(http.StatusOK, gin.H{"user_id": userID, "authenticated": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.False(t, result["authenticated"].(bool))
}

func TestOptionalAuth_WithInvalidToken(t *testing.T) {
	router := gin.New()
	router.Use(OptionalAuth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if ok {
			c.JSON(http.StatusOK, gin.H{"user_id": userID, "authenticated": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	// Should still succeed but not authenticated
	assert.False(t, result["authenticated"].(bool))
}

func TestOptionalAuth_WithInvalidFormat(t *testing.T) {
	router := gin.New()
	router.Use(OptionalAuth(testJWTSecret))
	router.GET("/test", func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if ok {
			c.JSON(http.StatusOK, gin.H{"user_id": userID, "authenticated": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "no-bearer-prefix")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	// Should still succeed but not authenticated
	assert.False(t, result["authenticated"].(bool))
}

func TestGetUserID_NotSet(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		userID, ok := GetUserID(c)
		assert.False(t, ok)
		assert.Equal(t, int64(0), userID)
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserID_WrongType(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.Set(UserIDKey, "not-an-int64") // Wrong type
		userID, ok := GetUserID(c)
		assert.False(t, ok)
		assert.Equal(t, int64(0), userID)
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserID_ValidInt64(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.Set(UserIDKey, int64(789))
		userID, ok := GetUserID(c)
		assert.True(t, ok)
		assert.Equal(t, int64(789), userID)
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
