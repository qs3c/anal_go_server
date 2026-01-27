package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qs3c/anal_go_server/internal/model"
	"github.com/qs3c/anal_go_server/internal/testutil"
)

func TestJobRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	job := &model.AnalysisJob{
		AnalysisID:  analysis.ID,
		UserID:      user.ID,
		RepoURL:     "https://github.com/example/repo",
		StartStruct: "main.Config",
		Depth:       3,
		ModelName:   "gpt-3.5-turbo",
		Status:      "queued",
	}

	err := repo.Create(job)
	require.NoError(t, err)
	assert.NotZero(t, job.ID)
}

func TestJobRepository_GetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)
	created := testutil.TestJob(t, db, user.ID, analysis.ID, "queued")

	found, err := repo.GetByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "queued", found.Status)
}

func TestJobRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)

	_, err := repo.GetByID(99999)
	assert.Error(t, err)
}

func TestJobRepository_GetByAnalysisID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	// Create multiple jobs for the same analysis
	testutil.TestJob(t, db, user.ID, analysis.ID, "completed")
	latest := testutil.TestJob(t, db, user.ID, analysis.ID, "queued")

	found, err := repo.GetByAnalysisID(analysis.ID)
	require.NoError(t, err)
	assert.Equal(t, latest.ID, found.ID) // Should return the latest
}

func TestJobRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)
	job := testutil.TestJob(t, db, user.ID, analysis.ID, "queued")

	job.Status = "processing"
	job.CurrentStep = "downloading"
	err := repo.Update(job)
	require.NoError(t, err)

	found, err := repo.GetByID(job.ID)
	require.NoError(t, err)
	assert.Equal(t, "processing", found.Status)
	assert.Equal(t, "downloading", found.CurrentStep)
}

func TestJobRepository_UpdateStatus(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)
	job := testutil.TestJob(t, db, user.ID, analysis.ID, "queued")

	err := repo.UpdateStatus(job.ID, "processing")
	require.NoError(t, err)

	found, err := repo.GetByID(job.ID)
	require.NoError(t, err)
	assert.Equal(t, "processing", found.Status)
}

func TestJobRepository_UpdateStep(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)
	job := testutil.TestJob(t, db, user.ID, analysis.ID, "processing")

	err := repo.UpdateStep(job.ID, "analyzing")
	require.NoError(t, err)

	found, err := repo.GetByID(job.ID)
	require.NoError(t, err)
	assert.Equal(t, "analyzing", found.CurrentStep)
}

func TestJobRepository_GetPendingJobs(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)
	user := testutil.TestUser(t, db)

	// Create jobs with different statuses
	analysis1 := testutil.TestAnalysis(t, db, user.ID)
	analysis2 := testutil.TestAnalysis(t, db, user.ID)
	analysis3 := testutil.TestAnalysis(t, db, user.ID)
	analysis4 := testutil.TestAnalysis(t, db, user.ID)

	testutil.TestJob(t, db, user.ID, analysis1.ID, "queued")
	testutil.TestJob(t, db, user.ID, analysis2.ID, "queued")
	testutil.TestJob(t, db, user.ID, analysis3.ID, "processing")
	testutil.TestJob(t, db, user.ID, analysis4.ID, "completed")

	jobs, err := repo.GetPendingJobs(10)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)

	for _, job := range jobs {
		assert.Equal(t, "queued", job.Status)
	}
}

func TestJobRepository_GetPendingJobs_WithLimit(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)
	user := testutil.TestUser(t, db)

	// Create 5 queued jobs
	for i := 0; i < 5; i++ {
		analysis := testutil.TestAnalysis(t, db, user.ID)
		testutil.TestJob(t, db, user.ID, analysis.ID, "queued")
	}

	jobs, err := repo.GetPendingJobs(3)
	require.NoError(t, err)
	assert.Len(t, jobs, 3)
}

func TestJobRepository_CancelByAnalysisID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)
	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	// Create jobs with different statuses
	job1 := testutil.TestJob(t, db, user.ID, analysis.ID, "queued")
	job2 := testutil.TestJob(t, db, user.ID, analysis.ID, "processing")
	job3 := testutil.TestJob(t, db, user.ID, analysis.ID, "completed")

	err := repo.CancelByAnalysisID(analysis.ID)
	require.NoError(t, err)

	// Check job statuses
	found1, _ := repo.GetByID(job1.ID)
	found2, _ := repo.GetByID(job2.ID)
	found3, _ := repo.GetByID(job3.ID)

	assert.Equal(t, "cancelled", found1.Status)
	assert.Equal(t, "cancelled", found2.Status)
	assert.Equal(t, "completed", found3.Status) // Already completed, should not change
}
