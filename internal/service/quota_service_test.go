package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func setupQuotaService(t *testing.T) (*QuotaService, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
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
			{Name: "gpt-4o-mini", RequiredLevel: "basic"},
			{Name: "gpt-4", RequiredLevel: "pro"},
		},
	}

	service := NewQuotaService(userRepo, cfg)

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return service, cleanup
}

func TestQuotaService_CheckQuota_HasQuota(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free": {DailyQuota: 5, MaxDepth: 3},
			},
		},
	}
	service := NewQuotaService(userRepo, cfg)

	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(2))

	hasQuota, err := service.CheckQuota(user.ID)
	require.NoError(t, err)
	assert.True(t, hasQuota)
}

func TestQuotaService_CheckQuota_NoQuota(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free": {DailyQuota: 5, MaxDepth: 3},
			},
		},
	}
	service := NewQuotaService(userRepo, cfg)

	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(5))

	hasQuota, err := service.CheckQuota(user.ID)
	require.NoError(t, err)
	assert.False(t, hasQuota)
}

func TestQuotaService_CheckDepth_Valid(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free":  {DailyQuota: 5, MaxDepth: 3},
				"basic": {DailyQuota: 30, MaxDepth: 5},
			},
		},
	}
	service := NewQuotaService(userRepo, cfg)

	err := service.CheckDepth("free", 3)
	assert.NoError(t, err)

	err = service.CheckDepth("basic", 5)
	assert.NoError(t, err)
}

func TestQuotaService_CheckDepth_Exceeded(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free": {DailyQuota: 5, MaxDepth: 3},
			},
		},
	}
	service := NewQuotaService(userRepo, cfg)

	err := service.CheckDepth("free", 5)
	assert.Equal(t, ErrDepthExceeded, err)
}

func TestQuotaService_CheckModelPermission_Free(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{Name: "gpt-3.5-turbo", RequiredLevel: "free"},
			{Name: "gpt-4o-mini", RequiredLevel: "basic"},
			{Name: "gpt-4", RequiredLevel: "pro"},
		},
	}
	service := NewQuotaService(userRepo, cfg)

	// Free 用户只能使用 free 模型
	err := service.CheckModelPermission("free", "gpt-3.5-turbo")
	assert.NoError(t, err)

	err = service.CheckModelPermission("free", "gpt-4o-mini")
	assert.Equal(t, ErrModelDenied, err)
}

func TestQuotaService_CheckModelPermission_Basic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{Name: "gpt-3.5-turbo", RequiredLevel: "free"},
			{Name: "gpt-4o-mini", RequiredLevel: "basic"},
			{Name: "gpt-4", RequiredLevel: "pro"},
		},
	}
	service := NewQuotaService(userRepo, cfg)

	// Basic 用户可以使用 free 和 basic 模型
	err := service.CheckModelPermission("basic", "gpt-3.5-turbo")
	assert.NoError(t, err)

	err = service.CheckModelPermission("basic", "gpt-4o-mini")
	assert.NoError(t, err)

	err = service.CheckModelPermission("basic", "gpt-4")
	assert.Equal(t, ErrModelDenied, err)
}

func TestQuotaService_CheckModelPermission_Pro(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{Name: "gpt-3.5-turbo", RequiredLevel: "free"},
			{Name: "gpt-4o-mini", RequiredLevel: "basic"},
			{Name: "gpt-4", RequiredLevel: "pro"},
		},
	}
	service := NewQuotaService(userRepo, cfg)

	// Pro 用户可以使用所有模型
	err := service.CheckModelPermission("pro", "gpt-3.5-turbo")
	assert.NoError(t, err)

	err = service.CheckModelPermission("pro", "gpt-4o-mini")
	assert.NoError(t, err)

	err = service.CheckModelPermission("pro", "gpt-4")
	assert.NoError(t, err)
}

func TestQuotaService_GetMaxDepth(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free":  {DailyQuota: 5, MaxDepth: 3},
				"basic": {DailyQuota: 30, MaxDepth: 5},
				"pro":   {DailyQuota: 100, MaxDepth: 10},
			},
		},
	}
	service := NewQuotaService(userRepo, cfg)

	assert.Equal(t, 3, service.GetMaxDepth("free"))
	assert.Equal(t, 5, service.GetMaxDepth("basic"))
	assert.Equal(t, 10, service.GetMaxDepth("pro"))
	assert.Equal(t, 3, service.GetMaxDepth("unknown")) // 默认返回 free 级别
}

func TestQuotaService_GetQuotaInfo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free": {DailyQuota: 5, MaxDepth: 3},
			},
		},
	}
	service := NewQuotaService(userRepo, cfg)

	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(2))

	info, err := service.GetQuotaInfo(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "free", info.Tier)
	assert.Equal(t, 5, info.DailyLimit)
	assert.Equal(t, 2, info.DailyUsed)
	assert.Equal(t, 3, info.DailyRemain)
	assert.Equal(t, 3, info.MaxDepth)
}
