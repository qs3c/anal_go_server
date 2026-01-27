package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func setupAuthService(t *testing.T) (*AuthService, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:      "test-secret-key-for-testing",
			ExpireHours: 24,
		},
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free":  {DailyQuota: 5, MaxDepth: 3},
				"basic": {DailyQuota: 30, MaxDepth: 5},
				"pro":   {DailyQuota: 100, MaxDepth: 10},
			},
		},
		OAuth: config.OAuthConfig{
			Github: config.GithubOAuthConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURI:  "http://localhost:8080/callback",
			},
		},
	}

	service := NewAuthService(userRepo, cfg)

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return service, cleanup
}

func TestAuthService_Register_Success(t *testing.T) {
	service, cleanup := setupAuthService(t)
	defer cleanup()

	req := &dto.RegisterRequest{
		Email:    "newuser@example.com",
		Username: "newuser",
		Password: "password123",
	}

	resp, err := service.Register(req)
	require.NoError(t, err)
	assert.NotZero(t, resp.UserID)
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	service, cleanup := setupAuthService(t)
	defer cleanup()

	// First registration
	req := &dto.RegisterRequest{
		Email:    "duplicate@example.com",
		Username: "user1",
		Password: "password123",
	}
	_, err := service.Register(req)
	require.NoError(t, err)

	// Second registration with same email
	req2 := &dto.RegisterRequest{
		Email:    "duplicate@example.com",
		Username: "user2",
		Password: "password123",
	}
	_, err = service.Register(req2)
	assert.Equal(t, ErrEmailExists, err)
}

func TestAuthService_Register_DuplicateUsername(t *testing.T) {
	service, cleanup := setupAuthService(t)
	defer cleanup()

	// First registration
	req := &dto.RegisterRequest{
		Email:    "user1@example.com",
		Username: "sameusername",
		Password: "password123",
	}
	_, err := service.Register(req)
	require.NoError(t, err)

	// Second registration with same username
	req2 := &dto.RegisterRequest{
		Email:    "user2@example.com",
		Username: "sameusername",
		Password: "password123",
	}
	_, err = service.Register(req2)
	assert.Equal(t, ErrUsernameExists, err)
}

func TestAuthService_Login_EmailNotVerified(t *testing.T) {
	service, cleanup := setupAuthService(t)
	defer cleanup()

	// Register user (not verified by default in Register flow)
	regReq := &dto.RegisterRequest{
		Email:    "unverified@example.com",
		Username: "unverified",
		Password: "password123",
	}
	_, err := service.Register(regReq)
	require.NoError(t, err)

	// Try to login
	loginReq := &dto.LoginRequest{
		Email:    "unverified@example.com",
		Password: "password123",
	}
	_, err = service.Login(loginReq)
	assert.Equal(t, ErrEmailNotVerified, err)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	service, cleanup := setupAuthService(t)
	defer cleanup()

	loginReq := &dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}
	_, err := service.Login(loginReq)
	assert.Equal(t, ErrInvalidCredentials, err)
}

func TestAuthService_VerifyEmail_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:      "test-secret-key-for-testing",
			ExpireHours: 24,
		},
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free": {DailyQuota: 5, MaxDepth: 3},
			},
		},
		OAuth: config.OAuthConfig{
			Github: config.GithubOAuthConfig{},
		},
	}
	service := NewAuthService(userRepo, cfg)

	// Create user
	user := testutil.TestUser(t, db)

	// Manually update verification code
	verifyCode := "testverifycode123456789012"
	expiresAt := time.Now().Add(24 * time.Hour)
	db.Model(user).Updates(map[string]interface{}{
		"email_verified":          false,
		"verification_code":       verifyCode,
		"verification_expires_at": expiresAt,
	})

	resp, err := service.VerifyEmail(verifyCode)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.NotNil(t, resp.User)
}

func TestAuthService_VerifyEmail_InvalidCode(t *testing.T) {
	service, cleanup := setupAuthService(t)
	defer cleanup()

	_, err := service.VerifyEmail("invalidcode")
	assert.Equal(t, ErrInvalidVerifyCode, err)
}

func TestAuthService_GetUserByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		OAuth: config.OAuthConfig{Github: config.GithubOAuthConfig{}},
	}
	service := NewAuthService(userRepo, cfg)

	user := testutil.TestUser(t, db, testutil.WithUsername("testuser"))

	found, err := service.GetUserByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "testuser", found.Username)
}

func TestAuthService_GetUserByID_NotFound(t *testing.T) {
	service, cleanup := setupAuthService(t)
	defer cleanup()

	_, err := service.GetUserByID(99999)
	assert.Error(t, err)
}

func TestAuthService_GetGithubAuthURL(t *testing.T) {
	service, cleanup := setupAuthService(t)
	defer cleanup()

	url := service.GetGithubAuthURL("test-state")
	assert.Contains(t, url, "github.com")
	assert.Contains(t, url, "test-state")
}

