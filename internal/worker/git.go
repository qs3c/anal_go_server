package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneRepo 浅克隆仓库到指定目录
func CloneRepo(ctx context.Context, repoURL, destDir string) error {
	// 确保目标目录不存在
	if _, err := os.Stat(destDir); err == nil {
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("failed to clean existing directory: %w", err)
		}
	}

	// 创建父目录
	parentDir := filepath.Dir(destDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// 执行浅克隆
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, destDir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0") // 禁用交互式提示

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	return nil
}

// CleanupRepo 清理临时仓库目录
func CleanupRepo(dir string) error {
	if dir == "" {
		return nil
	}

	// 安全检查：确保不会删除根目录或重要目录
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 确保目录在 /tmp 或系统临时目录下
	tempDir := os.TempDir()
	if !strings.HasPrefix(absDir, tempDir) && !strings.HasPrefix(absDir, "/tmp") {
		return fmt.Errorf("refusing to delete directory outside temp: %s", absDir)
	}

	return os.RemoveAll(absDir)
}

// GetTempDir 获取任务的临时目录路径
func GetTempDir(jobID int64) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("analysis_%d", jobID))
}

// ValidateRepoURL 验证仓库 URL 格式
func ValidateRepoURL(url string) error {
	if url == "" {
		return fmt.Errorf("repo URL is empty")
	}

	// 支持的格式：
	// - https://github.com/user/repo
	// - https://github.com/user/repo.git
	// - git@github.com:user/repo.git
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "git@") {
		return fmt.Errorf("invalid repo URL format, must start with https:// or git@")
	}

	return nil
}
