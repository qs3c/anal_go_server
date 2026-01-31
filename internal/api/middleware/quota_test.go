package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/pkg/response"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/service"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func setupQuotaService(t *testing.T) (*service.QuotaService, *gorm.DB, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)

	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free":  {DailyQuota: 5, MaxDepth: 3},
				"basic": {DailyQuota: 30, MaxDepth: 5},
				"pro":   {DailyQuota: 100, MaxDepth: 10},
			},
		},
	}

	userRepo := repository.NewUserRepository(db)
	quotaService := service.NewQuotaService(userRepo, cfg)

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return quotaService, db, cleanup
}

func TestQuotaCheck_Success(t *testing.T) {
	quotaService, db, cleanup := setupQuotaService(t)
	defer cleanup()

	// Create user with available quota
	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(0))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, user.ID)
		c.Next()
	})
	router.Use(QuotaCheck(quotaService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQuotaCheck_QuotaExceeded(t *testing.T) {
	quotaService, db, cleanup := setupQuotaService(t)
	defer cleanup()

	// Create user with quota already used up
	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(5)) // Free tier has 5 quota

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, user.ID)
		c.Next()
	})
	router.Use(QuotaCheck(quotaService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeQuotaExceeded, resp.Code)
}

func TestQuotaCheck_NoUserID(t *testing.T) {
	quotaService, _, cleanup := setupQuotaService(t)
	defer cleanup()

	router := gin.New()
	// No user ID set
	router.Use(QuotaCheck(quotaService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeAuthFailed, resp.Code)
}

func TestQuotaCheck_UserNotFound(t *testing.T) {
	quotaService, _, cleanup := setupQuotaService(t)
	defer cleanup()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, int64(99999)) // Non-existent user
		c.Next()
	})
	router.Use(QuotaCheck(quotaService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeServerError, resp.Code)
}

func TestQuotaCheck_BasicTier(t *testing.T) {
	quotaService, db, cleanup := setupQuotaService(t)
	defer cleanup()

	// Create basic tier user with some quota used
	user := testutil.TestUser(t, db, testutil.WithSubscription("basic", 30), testutil.WithQuotaUsed(10))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, user.ID)
		c.Next()
	})
	router.Use(QuotaCheck(quotaService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQuotaCheck_ProTier(t *testing.T) {
	quotaService, db, cleanup := setupQuotaService(t)
	defer cleanup()

	// Create pro tier user
	user := testutil.TestUser(t, db, testutil.WithSubscription("pro", 100), testutil.WithQuotaUsed(50))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, user.ID)
		c.Next()
	})
	router.Use(QuotaCheck(quotaService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQuotaCheck_BasicTierExceeded(t *testing.T) {
	quotaService, db, cleanup := setupQuotaService(t)
	defer cleanup()

	// Create basic tier user with quota exceeded
	user := testutil.TestUser(t, db, testutil.WithSubscription("basic", 30), testutil.WithQuotaUsed(30))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, user.ID)
		c.Next()
	})
	router.Use(QuotaCheck(quotaService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, response.CodeQuotaExceeded, resp.Code)
}

func TestQuotaCheck_LastQuota(t *testing.T) {
	quotaService, db, cleanup := setupQuotaService(t)
	defer cleanup()

	// Create user with exactly 1 quota remaining
	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(4)) // Free tier has 5 quota

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, user.ID)
		c.Next()
	})
	router.Use(QuotaCheck(quotaService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
