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

func setupAnalysisService(t *testing.T) (*AnalysisService, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
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

	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	cleanup := func() {
		testutil.CleanupTestDB(t, db)
	}

	return service, cleanup
}

func TestAnalysisService_Create_Manual(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)

	req := &dto.CreateAnalysisRequest{
		Title:        "My Manual Analysis",
		CreationType: "manual",
	}

	resp, err := service.Create(user.ID, req)
	require.NoError(t, err)
	assert.NotZero(t, resp.AnalysisID)
}

func TestAnalysisService_Create_AI(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free": {DailyQuota: 5, MaxDepth: 3},
			},
		},
		Models: []config.ModelConfig{
			{Name: "gpt-3.5-turbo", RequiredLevel: "free"},
		},
	}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(0))

	req := &dto.CreateAnalysisRequest{
		Title:         "AI Analysis",
		CreationType:  "ai",
		RepoURL:       "https://github.com/example/repo",
		StartStruct:   "main.Config",
		AnalysisDepth: 3,
		ModelName:     "gpt-3.5-turbo",
	}

	resp, err := service.Create(user.ID, req)
	require.NoError(t, err)
	assert.NotZero(t, resp.AnalysisID)
	assert.NotZero(t, resp.JobID)
}

func TestAnalysisService_Create_AI_QuotaExceeded(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free": {DailyQuota: 5, MaxDepth: 3},
			},
		},
		Models: []config.ModelConfig{
			{Name: "gpt-3.5-turbo", RequiredLevel: "free"},
		},
	}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db, testutil.WithQuotaUsed(5)) // Quota exhausted

	req := &dto.CreateAnalysisRequest{
		Title:         "AI Analysis",
		CreationType:  "ai",
		RepoURL:       "https://github.com/example/repo",
		StartStruct:   "main.Config",
		AnalysisDepth: 3,
		ModelName:     "gpt-3.5-turbo",
	}

	_, err := service.Create(user.ID, req)
	assert.Equal(t, ErrQuotaExceeded, err)
}

func TestAnalysisService_Create_AI_DepthExceeded(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{
			Levels: map[string]config.SubscriptionLevel{
				"free": {DailyQuota: 5, MaxDepth: 3},
			},
		},
		Models: []config.ModelConfig{
			{Name: "gpt-3.5-turbo", RequiredLevel: "free"},
		},
	}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)

	req := &dto.CreateAnalysisRequest{
		Title:         "AI Analysis",
		CreationType:  "ai",
		RepoURL:       "https://github.com/example/repo",
		StartStruct:   "main.Config",
		AnalysisDepth: 10, // Exceeds free tier max depth (3)
		ModelName:     "gpt-3.5-turbo",
	}

	_, err := service.Create(user.ID, req)
	assert.Equal(t, ErrDepthExceeded, err)
}

func TestAnalysisService_GetByID_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Test Analysis"))

	detail, err := service.GetByID(user.ID, analysis.ID)
	require.NoError(t, err)
	assert.Equal(t, analysis.ID, detail.ID)
	assert.Equal(t, "Test Analysis", detail.Title)
}

func TestAnalysisService_GetByID_NotFound(t *testing.T) {
	service, cleanup := setupAnalysisService(t)
	defer cleanup()

	_, err := service.GetByID(1, 99999)
	assert.Equal(t, ErrAnalysisNotFound, err)
}

func TestAnalysisService_GetByID_NoPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user1 := testutil.TestUser(t, db)
	user2 := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user1.ID)

	_, err := service.GetByID(user2.ID, analysis.ID)
	assert.Equal(t, ErrAnalysisPermission, err)
}

func TestAnalysisService_List(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Analysis 1"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Analysis 2"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Analysis 3"))

	items, total, err := service.List(user.ID, 1, 10, "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, items, 3)
}

func TestAnalysisService_List_WithSearch(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Go Project"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Python Project"))
	testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Go Analysis"))

	items, total, err := service.List(user.ID, 1, 10, "Go", "")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)
}

func TestAnalysisService_Update_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithTitle("Original Title"))

	newTitle := "Updated Title"
	newDesc := "New description"
	req := &dto.UpdateAnalysisRequest{
		Title:       &newTitle,
		Description: &newDesc,
	}

	detail, err := service.Update(user.ID, analysis.ID, req)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", detail.Title)
	assert.Equal(t, "New description", detail.Description)
}

func TestAnalysisService_Update_NotFound(t *testing.T) {
	service, cleanup := setupAnalysisService(t)
	defer cleanup()

	newTitle := "Title"
	req := &dto.UpdateAnalysisRequest{Title: &newTitle}
	_, err := service.Update(1, 99999, req)
	assert.Equal(t, ErrAnalysisNotFound, err)
}

func TestAnalysisService_Update_NoPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user1 := testutil.TestUser(t, db)
	user2 := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user1.ID)

	newTitle := "Title"
	req := &dto.UpdateAnalysisRequest{Title: &newTitle}
	_, err := service.Update(user2.ID, analysis.ID, req)
	assert.Equal(t, ErrAnalysisPermission, err)
}

func TestAnalysisService_Delete_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)

	err := service.Delete(user.ID, analysis.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = analysisRepo.GetByID(analysis.ID)
	assert.Error(t, err)
}

func TestAnalysisService_Delete_NotFound(t *testing.T) {
	service, cleanup := setupAnalysisService(t)
	defer cleanup()

	err := service.Delete(1, 99999)
	assert.Equal(t, ErrAnalysisNotFound, err)
}

func TestAnalysisService_Delete_NoPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user1 := testutil.TestUser(t, db)
	user2 := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user1.ID)

	err := service.Delete(user2.ID, analysis.ID)
	assert.Equal(t, ErrAnalysisPermission, err)
}

func TestAnalysisService_Share_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithStatus("completed"))

	req := &dto.ShareAnalysisRequest{
		ShareTitle:       "Shared Title",
		ShareDescription: "This is my shared analysis",
		Tags:             []string{"go", "architecture"},
	}

	err := service.Share(user.ID, analysis.ID, req)
	require.NoError(t, err)

	// Verify sharing
	shared, _ := analysisRepo.GetByID(analysis.ID)
	assert.True(t, shared.IsPublic)
	assert.Equal(t, "Shared Title", shared.ShareTitle)
	assert.NotNil(t, shared.SharedAt)
}

func TestAnalysisService_Share_NotCompleted(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithStatus("draft"))

	req := &dto.ShareAnalysisRequest{ShareTitle: "Title"}
	err := service.Share(user.ID, analysis.ID, req)
	assert.Equal(t, ErrAnalysisNotComplete, err)
}

func TestAnalysisService_Unshare_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID, testutil.WithPublic(true))

	err := service.Unshare(user.ID, analysis.ID)
	require.NoError(t, err)

	// Verify unsharing
	unshared, _ := analysisRepo.GetByID(analysis.ID)
	assert.False(t, unshared.IsPublic)
	assert.Nil(t, unshared.SharedAt)
}

func TestAnalysisService_GetJobStatus_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user.ID)
	job := testutil.TestJob(t, db, user.ID, analysis.ID, "processing")

	resp, err := service.GetJobStatus(user.ID, analysis.ID)
	require.NoError(t, err)
	assert.Equal(t, job.ID, resp.JobID)
	assert.Equal(t, "processing", resp.Status)
}

func TestAnalysisService_GetJobStatus_NotFound(t *testing.T) {
	service, cleanup := setupAnalysisService(t)
	defer cleanup()

	_, err := service.GetJobStatus(1, 99999)
	assert.Equal(t, ErrAnalysisNotFound, err)
}

func TestAnalysisService_GetJobStatus_NoPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	cfg := &config.Config{}
	quotaService := NewQuotaService(userRepo, cfg)
	service := NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, nil, cfg)

	user1 := testutil.TestUser(t, db)
	user2 := testutil.TestUser(t, db)
	analysis := testutil.TestAnalysis(t, db, user1.ID)
	testutil.TestJob(t, db, user1.ID, analysis.ID, "processing")

	_, err := service.GetJobStatus(user2.ID, analysis.ID)
	assert.Equal(t, ErrAnalysisPermission, err)
}
