package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/qs3c/anal_go_server/config"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000", "https://example.com"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}

	router := gin.New()
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_NotAllowedOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	router := gin.New()
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Origin header should NOT be set for disallowed origins
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	// Other headers should still be set
	assert.Equal(t, "GET, POST", w.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORS_NoOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	router := gin.New()
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No Origin header
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_OptionsRequest(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}

	router := gin.New()
	router.Use(CORS(cfg))
	router.OPTIONS("/test", func(c *gin.Context) {
		// This should not be reached due to middleware abort
		c.JSON(http.StatusOK, gin.H{})
	})
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_MultipleOrigins(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000", "https://app.example.com", "https://admin.example.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	router := gin.New()
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	// Test first allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))

	// Test second allowed origin
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))

	// Test third allowed origin
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, "https://admin.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_EmptyConfig(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{},
		AllowedMethods: []string{},
		AllowedHeaders: []string{},
	}

	router := gin.New()
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Headers"))
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: "",
		},
		{
			name:     "single element",
			input:    []string{"GET"},
			expected: "GET",
		},
		{
			name:     "multiple elements",
			input:    []string{"GET", "POST", "PUT"},
			expected: "GET, POST, PUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
