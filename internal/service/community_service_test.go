package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func setupCommunityService(t *testing.T) (*CommunityService, *repository.AnalysisRepository, *repository.InteractionRepository, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)

	cfg := &config.Config{}

	service := NewCommunityService(analysisRepo, interactionRepo, cfg)

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return service, analysisRepo, interactionRepo, cleanup
}

func TestCommunityService_ListPublicAnalyses(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)

	// Create public and private analyses
	testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(false))

	items, total, err := service.ListPublicAnalyses(1, 10, "latest", "")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)
}

func TestCommunityService_ListPublicAnalyses_Empty(t *testing.T) {
	service, _, _, cleanup := setupCommunityService(t)
	defer cleanup()

	items, total, err := service.ListPublicAnalyses(1, 10, "latest", "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, items, 0)
}

func TestCommunityService_GetPublicAnalysis_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db, testutil.WithUsername("author"))
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true), testutil.WithTitle("Public Analysis"))

	detail, err := service.GetPublicAnalysis(analysis.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, analysis.ID, detail.ID)
	assert.NotNil(t, detail.Author)
	assert.Equal(t, "author", detail.Author.Username)
}

func TestCommunityService_GetPublicAnalysis_NotFound(t *testing.T) {
	service, _, _, cleanup := setupCommunityService(t)
	defer cleanup()

	_, err := service.GetPublicAnalysis(99999, nil)
	assert.Equal(t, ErrAnalysisNotFound, err)
}

func TestCommunityService_GetPublicAnalysis_NotPublic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(false))

	_, err := service.GetPublicAnalysis(analysis.ID, nil)
	assert.Equal(t, ErrAnalysisNotPublic, err)
}

func TestCommunityService_GetPublicAnalysis_WithUserInteraction(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	// Create interactions
	testutil.TestInteraction(t, db, user.ID, analysis.ID, "like")
	testutil.TestInteraction(t, db, user.ID, analysis.ID, "bookmark")

	detail, err := service.GetPublicAnalysis(analysis.ID, &user.ID)
	require.NoError(t, err)
	assert.NotNil(t, detail.UserInteraction)
	assert.True(t, detail.UserInteraction.Liked)
	assert.True(t, detail.UserInteraction.Bookmarked)
}

func TestCommunityService_Like_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	resp, err := service.Like(user.ID, analysis.ID)
	require.NoError(t, err)
	assert.True(t, resp.Liked)
	assert.Equal(t, 1, resp.LikeCount)
}

func TestCommunityService_Like_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	// First like
	_, err := service.Like(user.ID, analysis.ID)
	require.NoError(t, err)

	// Second like (idempotent)
	resp, err := service.Like(user.ID, analysis.ID)
	require.NoError(t, err)
	assert.True(t, resp.Liked)
}

func TestCommunityService_Like_NotPublic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(false))

	_, err := service.Like(user.ID, analysis.ID)
	assert.Equal(t, ErrAnalysisNotPublic, err)
}

func TestCommunityService_Unlike_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	// Like first
	_, err := service.Like(user.ID, analysis.ID)
	require.NoError(t, err)

	// Unlike
	resp, err := service.Unlike(user.ID, analysis.ID)
	require.NoError(t, err)
	assert.False(t, resp.Liked)
	assert.Equal(t, 0, resp.LikeCount)
}

func TestCommunityService_Unlike_NotLiked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	// Unlike without liking (idempotent)
	resp, err := service.Unlike(user.ID, analysis.ID)
	require.NoError(t, err)
	assert.False(t, resp.Liked)
}

func TestCommunityService_Bookmark_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	resp, err := service.Bookmark(user.ID, analysis.ID)
	require.NoError(t, err)
	assert.True(t, resp.Bookmarked)
	assert.Equal(t, 1, resp.BookmarkCount)
}

func TestCommunityService_Bookmark_NotPublic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(false))

	_, err := service.Bookmark(user.ID, analysis.ID)
	assert.Equal(t, ErrAnalysisNotPublic, err)
}

func TestCommunityService_Unbookmark_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	service := NewCommunityService(analysisRepo, interactionRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	// Bookmark first
	_, err := service.Bookmark(user.ID, analysis.ID)
	require.NoError(t, err)

	// Unbookmark
	resp, err := service.Unbookmark(user.ID, analysis.ID)
	require.NoError(t, err)
	assert.False(t, resp.Bookmarked)
	assert.Equal(t, 0, resp.BookmarkCount)
}
