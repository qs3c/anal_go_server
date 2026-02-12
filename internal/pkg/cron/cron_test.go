package cron

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/service"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.User{})
	require.NoError(t, err)

	return db
}

func setupCronService(t *testing.T) (*Service, *gorm.DB, func()) {
	t.Helper()

	db := setupTestDB(t)

	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free": {DailyQuota: 5, MaxDepth: 3},
			},
		},
	}

	userRepo := repository.NewUserRepository(db)
	quotaService := service.NewQuotaService(userRepo, cfg)
	cronService := NewService(quotaService, nil, "", 1)

	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}

	return cronService, db, cleanup
}

func TestNewService(t *testing.T) {
	_, _, cleanup := setupCronService(t)
	defer cleanup()

	// Test with nil quotaService
	svc := NewService(nil, nil, "", 1)
	assert.NotNil(t, svc)
	assert.Nil(t, svc.quotaService)
	assert.NotNil(t, svc.stopChan)
}

func TestService_StartAndStop(t *testing.T) {
	svc, _, cleanup := setupCronService(t)
	defer cleanup()

	// Start should not panic
	svc.Start()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Stop should not panic
	svc.Stop()

	// Give it a moment to stop
	time.Sleep(10 * time.Millisecond)
}

func stringPtr(s string) *string {
	return &s
}

func TestService_RunNow(t *testing.T) {
	svc, db, cleanup := setupCronService(t)
	defer cleanup()

	// Create a user with some quota used
	user := &model.User{
		Username:          "testuser",
		GithubID:          stringPtr("12345"),
		SubscriptionLevel: "free",
		DailyQuota:        5,
		QuotaUsedToday:    3,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	// Run the reset
	err = svc.RunNow()
	assert.NoError(t, err)

	// Verify quota was reset
	var updatedUser model.User
	err = db.First(&updatedUser, user.ID).Error
	require.NoError(t, err)
	assert.Equal(t, 0, updatedUser.QuotaUsedToday)
}

func TestService_RunNow_MultipleUsers(t *testing.T) {
	svc, db, cleanup := setupCronService(t)
	defer cleanup()

	// Create multiple users with quota used
	users := []model.User{
		{Username: "user1", GithubID: stringPtr("1"), SubscriptionLevel: "free", DailyQuota: 5, QuotaUsedToday: 5},
		{Username: "user2", GithubID: stringPtr("2"), SubscriptionLevel: "free", DailyQuota: 5, QuotaUsedToday: 3},
		{Username: "user3", GithubID: stringPtr("3"), SubscriptionLevel: "free", DailyQuota: 5, QuotaUsedToday: 1},
	}

	for _, u := range users {
		err := db.Create(&u).Error
		require.NoError(t, err)
	}

	// Run the reset
	err := svc.RunNow()
	assert.NoError(t, err)

	// Verify all quotas were reset
	var allUsers []model.User
	err = db.Find(&allUsers).Error
	require.NoError(t, err)

	for _, u := range allUsers {
		assert.Equal(t, 0, u.QuotaUsedToday, "User %s should have quota reset", u.Username)
	}
}

func TestService_RunNow_NoUsers(t *testing.T) {
	svc, _, cleanup := setupCronService(t)
	defer cleanup()

	// Run the reset with no users - should not error
	err := svc.RunNow()
	assert.NoError(t, err)
}

func TestService_StopBeforeStart(t *testing.T) {
	svc, _, cleanup := setupCronService(t)
	defer cleanup()

	// Stop before start should not panic
	// (channel close on unstarted goroutine is fine)
	svc.Stop()
}

func TestService_MultipleStarts(t *testing.T) {
	svc, _, cleanup := setupCronService(t)
	defer cleanup()

	// Multiple starts should be safe (each starts a new goroutine)
	svc.Start()
	svc.Start()

	time.Sleep(10 * time.Millisecond)

	svc.Stop()
}

func TestService_Structure(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	cfg := &config.Config{}
	userRepo := repository.NewUserRepository(db)
	quotaService := service.NewQuotaService(userRepo, cfg)

	svc := NewService(quotaService, nil, "", 1)

	assert.Equal(t, quotaService, svc.quotaService)
	assert.NotNil(t, svc.stopChan)
}
