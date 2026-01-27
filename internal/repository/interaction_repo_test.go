package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/internal/testutil"
)

func TestInteractionRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewInteractionRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	interaction := testutil.TestInteraction(t, db, user.ID, analysis.ID, "like")

	assert.NotZero(t, interaction.ID)
	assert.Equal(t, user.ID, interaction.UserID)
	assert.Equal(t, analysis.ID, interaction.AnalysisID)
	assert.Equal(t, "like", interaction.Type)

	// Verify using repo
	exists, err := repo.Exists(user.ID, analysis.ID, "like")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestInteractionRepository_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewInteractionRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	testutil.TestInteraction(t, db, user.ID, analysis.ID, "like")

	// Delete the interaction
	err := repo.Delete(user.ID, analysis.ID, "like")
	require.NoError(t, err)

	// Verify deletion
	exists, err := repo.Exists(user.ID, analysis.ID, "like")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestInteractionRepository_Exists(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewInteractionRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	// Should not exist initially
	exists, err := repo.Exists(user.ID, analysis.ID, "like")
	require.NoError(t, err)
	assert.False(t, exists)

	// Create interaction
	testutil.TestInteraction(t, db, user.ID, analysis.ID, "like")

	// Should exist now
	exists, err = repo.Exists(user.ID, analysis.ID, "like")
	require.NoError(t, err)
	assert.True(t, exists)

	// Different type should not exist
	exists, err = repo.Exists(user.ID, analysis.ID, "bookmark")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestInteractionRepository_GetByUserAndAnalysis(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewInteractionRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	// Create multiple interactions
	testutil.TestInteraction(t, db, user.ID, analysis.ID, "like")
	testutil.TestInteraction(t, db, user.ID, analysis.ID, "bookmark")

	interactions, err := repo.GetByUserAndAnalysis(user.ID, analysis.ID)
	require.NoError(t, err)
	assert.Len(t, interactions, 2)
}

func TestInteractionRepository_GetUserLikedAnalyses(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewInteractionRepository(db)
	user := testutil.TestUser(t, db)

	// Create multiple analyses and like them
	analysis1 := testutil.TestAnalysis(t, db, user.ID)
	analysis2 := testutil.TestAnalysis(t, db, user.ID)
	analysis3 := testutil.TestAnalysis(t, db, user.ID)

	testutil.TestInteraction(t, db, user.ID, analysis1.ID, "like")
	testutil.TestInteraction(t, db, user.ID, analysis2.ID, "like")
	testutil.TestInteraction(t, db, user.ID, analysis3.ID, "bookmark") // Not a like

	ids, total, err := repo.GetUserLikedAnalyses(user.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, ids, 2)
}

func TestInteractionRepository_GetUserBookmarkedAnalyses(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewInteractionRepository(db)
	user := testutil.TestUser(t, db)

	// Create multiple analyses and bookmark them
	analysis1 := testutil.TestAnalysis(t, db, user.ID)
	analysis2 := testutil.TestAnalysis(t, db, user.ID)
	analysis3 := testutil.TestAnalysis(t, db, user.ID)

	testutil.TestInteraction(t, db, user.ID, analysis1.ID, "bookmark")
	testutil.TestInteraction(t, db, user.ID, analysis2.ID, "bookmark")
	testutil.TestInteraction(t, db, user.ID, analysis3.ID, "like") // Not a bookmark

	ids, total, err := repo.GetUserBookmarkedAnalyses(user.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, ids, 2)
}

func TestInteractionRepository_Pagination(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewInteractionRepository(db)
	user := testutil.TestUser(t, db)

	// Create 5 analyses and like them
	for i := 0; i < 5; i++ {
		analysis := testutil.TestAnalysis(t, db, user.ID)
		testutil.TestInteraction(t, db, user.ID, analysis.ID, "like")
	}

	// Get page 1 with size 2
	ids, total, err := repo.GetUserLikedAnalyses(user.ID, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, ids, 2)

	// Get page 2 with size 2
	ids, total, err = repo.GetUserLikedAnalyses(user.ID, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, ids, 2)

	// Get page 3 with size 2
	ids, total, err = repo.GetUserLikedAnalyses(user.ID, 3, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, ids, 1)
}
