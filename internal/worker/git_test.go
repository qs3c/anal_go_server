package worker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRepoURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https url",
			url:     "https://github.com/user/repo",
			wantErr: false,
		},
		{
			name:    "valid https url with .git",
			url:     "https://github.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid git@ url",
			url:     "git@github.com:user/repo.git",
			wantErr: false,
		},
		{
			name:    "empty url",
			url:     "",
			wantErr: true,
		},
		{
			name:    "invalid http url",
			url:     "http://github.com/user/repo",
			wantErr: true,
		},
		{
			name:    "invalid ftp url",
			url:     "ftp://github.com/user/repo",
			wantErr: true,
		},
		{
			name:    "invalid plain text",
			url:     "just-some-text",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepoURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetTempDir(t *testing.T) {
	tests := []struct {
		name  string
		jobID int64
	}{
		{name: "job id 1", jobID: 1},
		{name: "job id 12345", jobID: 12345},
		{name: "large job id", jobID: 9999999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := GetTempDir(tt.jobID)

			// Should be under system temp dir
			assert.True(t, strings.HasPrefix(dir, os.TempDir()))

			// Should contain job ID
			assert.Contains(t, dir, "analysis_")

			// Should be unique per job ID
			dir2 := GetTempDir(tt.jobID + 1)
			assert.NotEqual(t, dir, dir2)
		})
	}
}

func TestCleanupRepo(t *testing.T) {
	t.Run("cleanup empty path", func(t *testing.T) {
		err := CleanupRepo("")
		assert.NoError(t, err)
	})

	t.Run("cleanup temp directory", func(t *testing.T) {
		// Create a temp directory
		tempDir := filepath.Join(os.TempDir(), "test_cleanup_"+time.Now().Format("20060102150405"))
		err := os.MkdirAll(tempDir, 0755)
		require.NoError(t, err)

		// Create a file inside
		testFile := filepath.Join(tempDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)

		// Cleanup should succeed
		err = CleanupRepo(tempDir)
		assert.NoError(t, err)

		// Directory should no longer exist
		_, err = os.Stat(tempDir)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("refuse to delete outside temp", func(t *testing.T) {
		// Try to delete a path outside temp directory
		err := CleanupRepo("/usr/local/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refusing to delete")
	})

	t.Run("refuse to delete home directory", func(t *testing.T) {
		homeDir, _ := os.UserHomeDir()
		err := CleanupRepo(homeDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refusing to delete")
	})
}

func TestCloneRepo(t *testing.T) {
	// Skip in CI or when git is not available
	if os.Getenv("CI") != "" {
		t.Skip("Skipping clone test in CI")
	}

	t.Run("clone small public repo", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tempDir := filepath.Join(os.TempDir(), "test_clone_"+time.Now().Format("20060102150405"))
		defer CleanupRepo(tempDir)

		// Clone a small public repo
		err := CloneRepo(ctx, "https://github.com/octocat/Hello-World.git", tempDir)
		assert.NoError(t, err)

		// Verify .git directory exists
		gitDir := filepath.Join(tempDir, ".git")
		_, err = os.Stat(gitDir)
		assert.NoError(t, err)
	})

	t.Run("clone with context timeout", func(t *testing.T) {
		// Use an already-cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		tempDir := filepath.Join(os.TempDir(), "test_clone_timeout_"+time.Now().Format("20060102150405"))
		defer CleanupRepo(tempDir)

		err := CloneRepo(ctx, "https://github.com/octocat/Hello-World.git", tempDir)
		assert.Error(t, err)
	})

	t.Run("clone invalid url", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tempDir := filepath.Join(os.TempDir(), "test_clone_invalid_"+time.Now().Format("20060102150405"))
		defer CleanupRepo(tempDir)

		err := CloneRepo(ctx, "https://github.com/nonexistent/repo12345678.git", tempDir)
		assert.Error(t, err)
	})

	t.Run("clone to existing directory", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tempDir := filepath.Join(os.TempDir(), "test_clone_existing_"+time.Now().Format("20060102150405"))

		// Create the directory first
		err := os.MkdirAll(tempDir, 0755)
		require.NoError(t, err)

		// Create a file inside to ensure it gets cleaned
		testFile := filepath.Join(tempDir, "existing.txt")
		err = os.WriteFile(testFile, []byte("existing content"), 0644)
		require.NoError(t, err)

		defer CleanupRepo(tempDir)

		// Clone should succeed (clean and recreate)
		err = CloneRepo(ctx, "https://github.com/octocat/Hello-World.git", tempDir)
		assert.NoError(t, err)

		// Old file should be gone
		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))

		// .git should exist
		gitDir := filepath.Join(tempDir, ".git")
		_, err = os.Stat(gitDir)
		assert.NoError(t, err)
	})
}

func TestCleanupRepo_NonExistentDirectory(t *testing.T) {
	// Cleanup non-existent directory should succeed
	tempDir := filepath.Join(os.TempDir(), "non_existent_dir_12345")
	err := CleanupRepo(tempDir)
	assert.NoError(t, err)
}

func TestCleanupRepo_NestedDirectory(t *testing.T) {
	// Create nested temp directory
	tempDir := filepath.Join(os.TempDir(), "test_nested_"+time.Now().Format("20060102150405"))
	nestedDir := filepath.Join(tempDir, "level1", "level2", "level3")
	err := os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)

	// Create files at each level
	for _, dir := range []string{tempDir, filepath.Join(tempDir, "level1"), filepath.Join(tempDir, "level1", "level2"), nestedDir} {
		testFile := filepath.Join(dir, "test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Cleanup should remove everything
	err = CleanupRepo(tempDir)
	assert.NoError(t, err)

	// Nothing should remain
	_, err = os.Stat(tempDir)
	assert.True(t, os.IsNotExist(err))
}

func TestValidateRepoURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "https with port",
			url:     "https://github.com:443/user/repo",
			wantErr: false,
		},
		{
			name:    "https gitlab",
			url:     "https://gitlab.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "https bitbucket",
			url:     "https://bitbucket.org/user/repo.git",
			wantErr: false,
		},
		{
			name:    "git@ gitlab",
			url:     "git@gitlab.com:user/repo.git",
			wantErr: false,
		},
		{
			name:    "https with subgroup",
			url:     "https://gitlab.com/group/subgroup/repo.git",
			wantErr: false,
		},
		{
			name:    "whitespace only",
			url:     "   ",
			wantErr: true,
		},
		{
			name:    "file protocol",
			url:     "file:///path/to/repo",
			wantErr: true,
		},
		{
			name:    "ssh protocol",
			url:     "ssh://git@github.com/user/repo.git",
			wantErr: true, // not supported, only git@ format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepoURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetTempDir_Consistency(t *testing.T) {
	// Same job ID should always return same path
	jobID := int64(42)
	dir1 := GetTempDir(jobID)
	dir2 := GetTempDir(jobID)
	assert.Equal(t, dir1, dir2)
}

func TestGetTempDir_ZeroID(t *testing.T) {
	dir := GetTempDir(0)
	assert.Contains(t, dir, "analysis_0")
	assert.True(t, strings.HasPrefix(dir, os.TempDir()))
}

func TestGetTempDir_NegativeID(t *testing.T) {
	// Negative ID should still work (though unusual)
	dir := GetTempDir(-1)
	assert.Contains(t, dir, "analysis_-1")
	assert.True(t, strings.HasPrefix(dir, os.TempDir()))
}
