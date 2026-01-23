package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func TestAnalysisRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)

	analysis := &model.Analysis{
		UserID:       user.ID,
		Title:        "Test Analysis",
		CreationType: "manual",
		Status:       "draft",
	}

	err := repo.Create(analysis)
	require.NoError(t, err)
	assert.NotZero(t, analysis.ID)
}

func TestAnalysisRepository_GetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)
	created := testutil.TestAnalysis(t, db, user.ID)

	found, err := repo.GetByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.Title, found.Title)
}

func TestAnalysisRepository_ListByUserID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)

	// 创建多个分析
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Analysis 1"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Analysis 2"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Analysis 3"))

	analyses, total, err := repo.ListByUserID(user.ID, 1, 10, "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, analyses, 3)
}

func TestAnalysisRepository_ListByUserID_WithSearch(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)

	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Go Project"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Python Project"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Go Analysis"))

	analyses, total, err := repo.ListByUserID(user.ID, 1, 10, "Go", "")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, analyses, 2)
}

func TestAnalysisRepository_ListByUserID_WithStatus(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)

	testutil.TestAnalysis(t, db, user.ID, testutil.WithStatus("completed"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithStatus("completed"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithStatus("draft"))

	analyses, total, err := repo.ListByUserID(user.ID, 1, 10, "", "completed")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, analyses, 2)
}

func TestAnalysisRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Original"))

	analysis.Title = "Updated"
	err := repo.Update(analysis)
	require.NoError(t, err)

	found, err := repo.GetByID(analysis.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", found.Title)
}

func TestAnalysisRepository_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	err := repo.Delete(analysis.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(analysis.ID)
	assert.Error(t, err)
}

func TestAnalysisRepository_ListPublic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)

	testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(false))

	analyses, total, err := repo.ListPublic(1, 10, "latest", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, analyses, 2)
}

func TestAnalysisRepository_IncrementViewCount(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	err := repo.IncrementViewCount(analysis.ID)
	require.NoError(t, err)

	found, err := repo.GetByID(analysis.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, found.ViewCount)
}

func TestAnalysisRepository_IncrementLikeCount(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewAnalysisRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	err := repo.IncrementLikeCount(analysis.ID, 1)
	require.NoError(t, err)

	found, err := repo.GetByID(analysis.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, found.LikeCount)

	err = repo.IncrementLikeCount(analysis.ID, -1)
	require.NoError(t, err)

	found, err = repo.GetByID(analysis.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, found.LikeCount)
}
