package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model/dto"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func setupUserService(t *testing.T) (*UserService, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	userRepo := repository.NewUserRepository(db)

	cfg := &config.Config{}

	service := NewUserService(userRepo, nil, cfg)

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return service, cleanup
}

func TestUserService_GetProfile_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	service := NewUserService(userRepo, nil, &config.Config{})

	user := testutil.TestUser(t, db,
		testutil.WithUsername("profileuser"),
		testutil.WithSubscription("basic", 30),
	)

	profile, err := service.GetProfile(user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, profile.ID)
	assert.Equal(t, "profileuser", profile.Username)
	assert.Equal(t, "basic", profile.SubscriptionLevel)
	assert.NotNil(t, profile.QuotaInfo)
	assert.Equal(t, 30, profile.QuotaInfo.DailyQuota)
}

func TestUserService_GetProfile_NotFound(t *testing.T) {
	service, cleanup := setupUserService(t)
	defer cleanup()

	_, err := service.GetProfile(99999)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	service := NewUserService(userRepo, nil, &config.Config{})

	user := testutil.TestUser(t, db, testutil.WithUsername("oldname"))

	newUsername := "newname"
	newBio := "This is my bio"
	req := &dto.UpdateProfileRequest{
		Username: &newUsername,
		Bio:      &newBio,
	}

	profile, err := service.UpdateProfile(user.ID, req)
	require.NoError(t, err)
	assert.Equal(t, "newname", profile.Username)
	assert.Equal(t, "This is my bio", profile.Bio)
}

func TestUserService_UpdateProfile_UsernameOnly(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	service := NewUserService(userRepo, nil, &config.Config{})

	user := testutil.TestUser(t, db, testutil.WithUsername("oldname"))

	newUsername := "newname"
	req := &dto.UpdateProfileRequest{
		Username: &newUsername,
	}

	profile, err := service.UpdateProfile(user.ID, req)
	require.NoError(t, err)
	assert.Equal(t, "newname", profile.Username)
}

func TestUserService_UpdateProfile_BioOnly(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	service := NewUserService(userRepo, nil, &config.Config{})

	user := testutil.TestUser(t, db, testutil.WithUsername("unchanged"))

	newBio := "New bio"
	req := &dto.UpdateProfileRequest{
		Bio: &newBio,
	}

	profile, err := service.UpdateProfile(user.ID, req)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", profile.Username)
	assert.Equal(t, "New bio", profile.Bio)
}

func TestUserService_UpdateProfile_DuplicateUsername(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	service := NewUserService(userRepo, nil, &config.Config{})

	testutil.TestUser(t, db, testutil.WithUsername("existinguser"))
	user2 := testutil.TestUser(t, db, testutil.WithUsername("user2"))

	existingUsername := "existinguser"
	req := &dto.UpdateProfileRequest{
		Username: &existingUsername,
	}

	_, err := service.UpdateProfile(user2.ID, req)
	assert.Equal(t, ErrUsernameExists, err)
}

func TestUserService_UpdateProfile_SameUsername(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	service := NewUserService(userRepo, nil, &config.Config{})

	user := testutil.TestUser(t, db, testutil.WithUsername("myname"))

	// Update with same username should not fail
	sameUsername := "myname"
	newBio := "Updated bio"
	req := &dto.UpdateProfileRequest{
		Username: &sameUsername,
		Bio:      &newBio,
	}

	profile, err := service.UpdateProfile(user.ID, req)
	require.NoError(t, err)
	assert.Equal(t, "myname", profile.Username)
	assert.Equal(t, "Updated bio", profile.Bio)
}

func TestUserService_UpdateProfile_NotFound(t *testing.T) {
	service, cleanup := setupUserService(t)
	defer cleanup()

	newUsername := "newname"
	req := &dto.UpdateProfileRequest{
		Username: &newUsername,
	}

	_, err := service.UpdateProfile(99999, req)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_UpdateAvatar(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	service := NewUserService(userRepo, nil, &config.Config{})

	user := testutil.TestUser(t, db)

	err := service.UpdateAvatar(user.ID, "https://example.com/avatar.jpg")
	require.NoError(t, err)

	// Verify update
	updated, err := userRepo.GetByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/avatar.jpg", updated.AvatarURL)
}

func TestUserService_GetProfile_WithQuotaInfo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	service := NewUserService(userRepo, nil, &config.Config{})

	user := testutil.TestUser(t, db,
		testutil.WithSubscription("pro", 100),
		testutil.WithQuotaUsed(25),
	)

	profile, err := service.GetProfile(user.ID)
	require.NoError(t, err)

	assert.NotNil(t, profile.QuotaInfo)
	assert.Equal(t, 100, profile.QuotaInfo.DailyQuota)
	assert.Equal(t, 25, profile.QuotaInfo.QuotaUsedToday)
	assert.Equal(t, 75, profile.QuotaInfo.QuotaRemaining)
}
