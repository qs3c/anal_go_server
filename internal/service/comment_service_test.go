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

func setupCommentService(t *testing.T) (*CommentService, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)

	cfg := &config.Config{}

	service := NewCommentService(commentRepo, analysisRepo, userRepo, cfg)

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return service, cleanup
}

func TestCommentService_Create_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db, testutil.WithUsername("commenter"))
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	req := &dto.CreateCommentRequest{
		Content: "This is a test comment",
	}

	item, err := service.Create(user.ID, analysis.ID, req)
	require.NoError(t, err)
	assert.NotZero(t, item.ID)
	assert.Equal(t, "This is a test comment", item.Content)
	assert.NotNil(t, item.User)
	assert.Equal(t, "commenter", item.User.Username)
}

func TestCommentService_Create_Reply(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	// Create parent comment
	parent := testutil.TestComment(t, db, user.ID, analysis.ID, "Parent comment")

	req := &dto.CreateCommentRequest{
		Content:  "This is a reply",
		ParentID: &parent.ID,
	}

	item, err := service.Create(user.ID, analysis.ID, req)
	require.NoError(t, err)
	assert.NotZero(t, item.ID)
	assert.Equal(t, &parent.ID, item.ParentID)
}

func TestCommentService_Create_AnalysisNotFound(t *testing.T) {
	service, cleanup := setupCommentService(t)
	defer cleanup()

	req := &dto.CreateCommentRequest{
		Content: "Test comment",
	}

	_, err := service.Create(1, 99999, req)
	assert.Equal(t, ErrAnalysisNotFound, err)
}

func TestCommentService_Create_AnalysisNotPublic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(false))

	req := &dto.CreateCommentRequest{
		Content: "Test comment",
	}

	_, err := service.Create(user.ID, analysis.ID, req)
	assert.Equal(t, ErrAnalysisNotPublic, err)
}

func TestCommentService_Create_ParentNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	nonExistentID := int64(99999)
	req := &dto.CreateCommentRequest{
		Content:  "Test reply",
		ParentID: &nonExistentID,
	}

	_, err := service.Create(user.ID, analysis.ID, req)
	assert.Equal(t, ErrParentNotFound, err)
}

func TestCommentService_Delete_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))
	comment := testutil.TestComment(t, db, user.ID, analysis.ID, "To be deleted")

	err := service.Delete(user.ID, comment.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = commentRepo.GetByID(comment.ID)
	assert.Error(t, err)
}

func TestCommentService_Delete_NotFound(t *testing.T) {
	service, cleanup := setupCommentService(t)
	defer cleanup()

	err := service.Delete(1, 99999)
	assert.Equal(t, ErrCommentNotFound, err)
}

func TestCommentService_Delete_NoPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user1 := testutil.TestUser(t, db)
	user2 := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user1.ID, testutil.WithPublic(true))
	comment := testutil.TestComment(t, db, user1.ID, analysis.ID, "User1's comment")

	// User2 tries to delete User1's comment
	err := service.Delete(user2.ID, comment.ID)
	assert.Equal(t, ErrCommentPermission, err)
}

func TestCommentService_Delete_WithReplies(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))
	parent := testutil.TestComment(t, db, user.ID, analysis.ID, "Parent comment")
	testutil.TestReply(t, db, user.ID, analysis.ID, parent.ID, "Reply 1")
	testutil.TestReply(t, db, user.ID, analysis.ID, parent.ID, "Reply 2")

	err := service.Delete(user.ID, parent.ID)
	require.NoError(t, err)

	// Verify parent is deleted
	_, err = commentRepo.GetByID(parent.ID)
	assert.Error(t, err)
}

func TestCommentService_ListByAnalysisID_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	// Create comments
	testutil.TestComment(t, db, user.ID, analysis.ID, "Comment 1")
	testutil.TestComment(t, db, user.ID, analysis.ID, "Comment 2")
	testutil.TestComment(t, db, user.ID, analysis.ID, "Comment 3")

	items, total, err := service.ListByAnalysisID(analysis.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, items, 3)
}

func TestCommentService_ListByAnalysisID_WithReplies(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	// Create parent with replies
	parent := testutil.TestComment(t, db, user.ID, analysis.ID, "Parent comment")
	testutil.TestReply(t, db, user.ID, analysis.ID, parent.ID, "Reply 1")
	testutil.TestReply(t, db, user.ID, analysis.ID, parent.ID, "Reply 2")

	items, total, err := service.ListByAnalysisID(analysis.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total) // Only top-level comments
	assert.Len(t, items, 1)
	assert.Len(t, items[0].Replies, 2)
}

func TestCommentService_ListByAnalysisID_AnalysisNotFound(t *testing.T) {
	service, cleanup := setupCommentService(t)
	defer cleanup()

	_, _, err := service.ListByAnalysisID(99999, 1, 10)
	assert.Equal(t, ErrAnalysisNotFound, err)
}

func TestCommentService_ListByAnalysisID_NotPublic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(false))

	_, _, err := service.ListByAnalysisID(analysis.ID, 1, 10)
	assert.Equal(t, ErrAnalysisNotPublic, err)
}

func TestCommentService_ListByAnalysisID_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	commentRepo := repository.NewCommentRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)
	service := NewCommentService(commentRepo, analysisRepo, userRepo, &config.Config{})

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	items, total, err := service.ListByAnalysisID(analysis.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, items, 0)
}
