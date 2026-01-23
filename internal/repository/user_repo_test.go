package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/internal/testutil"
)

func TestUserRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	_ = NewUserRepository(db)

	email := "test@example.com"
	user := testutil.TestUser(t, db, testutil.WithEmail(email))

	assert.NotZero(t, user.ID)
	assert.Equal(t, email, *user.Email)
}

func TestUserRepository_GetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	// 创建测试用户
	created := testutil.TestUser(t, db)

	// 查询用户
	found, err := repo.GetByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.Username, found.Username)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	_, err := repo.GetByID(99999)
	assert.Error(t, err)
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	email := "unique@example.com"
	testutil.TestUser(t, db, testutil.WithEmail(email))

	found, err := repo.GetByEmail(email)
	require.NoError(t, err)
	assert.Equal(t, email, *found.Email)
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	email := "exists@example.com"
	testutil.TestUser(t, db, testutil.WithEmail(email))

	exists, err := repo.ExistsByEmail(email)
	require.NoError(t, err)
	assert.True(t, exists)

	notExists, err := repo.ExistsByEmail("notexists@example.com")
	require.NoError(t, err)
	assert.False(t, notExists)
}

func TestUserRepository_ExistsByUsername(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	username := "uniqueuser"
	testutil.TestUser(t, db, testutil.WithUsername(username))

	exists, err := repo.ExistsByUsername(username)
	require.NoError(t, err)
	assert.True(t, exists)

	notExists, err := repo.ExistsByUsername("notexistsuser")
	require.NoError(t, err)
	assert.False(t, notExists)
}

func TestUserRepository_IncrementQuotaUsed(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(0))

	err := repo.IncrementQuotaUsed(user.ID)
	require.NoError(t, err)

	updated, err := repo.GetByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updated.QuotaUsedToday)
}

func TestUserRepository_DecrementQuotaUsed(t *testing.T) {
	// Skip this test for SQLite as it uses MySQL-specific GREATEST function
	t.Skip("Skipping: uses MySQL-specific GREATEST function not supported by SQLite")

	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewUserRepository(db)

	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(5))

	err := repo.DecrementQuotaUsed(user.ID)
	require.NoError(t, err)

	updated, err := repo.GetByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, 4, updated.QuotaUsedToday)
}
