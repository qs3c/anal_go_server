package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func TestCommentRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	comment := &model.Comment{
		UserID:     user.ID,
		AnalysisID: analysis.ID,
		Content:    "This is a test comment",
	}

	err := repo.Create(comment)
	require.NoError(t, err)
	assert.NotZero(t, comment.ID)
}

func TestCommentRepository_GetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)
	created := testutil.TestComment(t, db, user.ID, analysis.ID, "Test comment")

	found, err := repo.GetByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Test comment", found.Content)
}

func TestCommentRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)

	_, err := repo.GetByID(99999)
	assert.Error(t, err)
}

func TestCommentRepository_GetByIDWithUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)
	user := testutil.TestUser(t, db, testutil.WithUsername("testuser"))
	analysis := testutil.TestAnalysis(t, db, user.ID)
	created := testutil.TestComment(t, db, user.ID, analysis.ID, "Test comment")

	found, err := repo.GetByIDWithUser(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.NotNil(t, found.User)
	assert.Equal(t, "testuser", found.User.Username)
}

func TestCommentRepository_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)
	comment := testutil.TestComment(t, db, user.ID, analysis.ID, "Test comment")

	err := repo.Delete(comment.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(comment.ID)
	assert.Error(t, err)
}

func TestCommentRepository_DeleteByParentID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	// Create parent comment
	parent := testutil.TestComment(t, db, user.ID, analysis.ID, "Parent comment")

	// Create replies
	testutil.TestReply(t, db, user.ID, analysis.ID, parent.ID, "Reply 1")
	testutil.TestReply(t, db, user.ID, analysis.ID, parent.ID, "Reply 2")
	testutil.TestReply(t, db, user.ID, analysis.ID, parent.ID, "Reply 3")

	// Delete replies
	deleted, err := repo.DeleteByParentID(parent.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)

	// Verify parent still exists
	_, err = repo.GetByID(parent.ID)
	require.NoError(t, err)
}

func TestCommentRepository_ListByAnalysisID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	// Create top-level comments
	parent1 := testutil.TestComment(t, db, user.ID, analysis.ID, "Comment 1")
	testutil.TestComment(t, db, user.ID, analysis.ID, "Comment 2")
	testutil.TestComment(t, db, user.ID, analysis.ID, "Comment 3")

	// Create a reply (should not be included in list)
	testutil.TestReply(t, db, user.ID, analysis.ID, parent1.ID, "Reply to Comment 1")

	comments, total, err := repo.ListByAnalysisID(analysis.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, comments, 3)
}

func TestCommentRepository_ListByAnalysisID_Pagination(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	// Create 5 comments
	for i := 0; i < 5; i++ {
		testutil.TestComment(t, db, user.ID, analysis.ID, "Comment")
	}

	// Get page 1
	comments, total, err := repo.ListByAnalysisID(analysis.ID, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, comments, 2)

	// Get page 2
	comments, total, err = repo.ListByAnalysisID(analysis.ID, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, comments, 2)
}

func TestCommentRepository_GetRepliesByParentIDs(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	// Create parent comments
	parent1 := testutil.TestComment(t, db, user.ID, analysis.ID, "Parent 1")
	parent2 := testutil.TestComment(t, db, user.ID, analysis.ID, "Parent 2")

	// Create replies
	testutil.TestReply(t, db, user.ID, analysis.ID, parent1.ID, "Reply 1-1")
	testutil.TestReply(t, db, user.ID, analysis.ID, parent1.ID, "Reply 1-2")
	testutil.TestReply(t, db, user.ID, analysis.ID, parent2.ID, "Reply 2-1")

	replies, err := repo.GetRepliesByParentIDs([]int64{parent1.ID, parent2.ID})
	require.NoError(t, err)
	assert.Len(t, replies, 3)
}

func TestCommentRepository_GetRepliesByParentIDs_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)

	replies, err := repo.GetRepliesByParentIDs([]int64{})
	require.NoError(t, err)
	assert.Nil(t, replies)
}

func TestCommentRepository_CountByAnalysisID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewCommentRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	// Create comments
	parent := testutil.TestComment(t, db, user.ID, analysis.ID, "Parent")
	testutil.TestComment(t, db, user.ID, analysis.ID, "Comment 2")
	testutil.TestReply(t, db, user.ID, analysis.ID, parent.ID, "Reply")

	count, err := repo.CountByAnalysisID(analysis.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count) // All comments including replies
}
