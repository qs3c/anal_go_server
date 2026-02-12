package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CloneError 克隆错误，包含用户友好消息和原始错误
type CloneError struct {
	UserMessage string // 中文，给用户看
	RawError    error  // 原始错误，写日志
}

func (e *CloneError) Error() string {
	return e.UserMessage
}

func (e *CloneError) Unwrap() error {
	return e.RawError
}

// classifyCloneError 根据 git 输出分类错误，返回中文用户提示
func classifyCloneError(output string, err error) *CloneError {
	lower := strings.ToLower(output + " " + err.Error())

	switch {
	case strings.Contains(lower, "repository not found") ||
		strings.Contains(lower, "not found"):
		return &CloneError{
			UserMessage: "仓库不存在或无访问权限，请检查地址",
			RawError:    fmt.Errorf("%w, output: %s", err, output),
		}
	case strings.Contains(lower, "could not resolve host") ||
		strings.Contains(lower, "unable to access"):
		return &CloneError{
			UserMessage: "无法连接到代码托管平台，请稍后重试",
			RawError:    fmt.Errorf("%w, output: %s", err, output),
		}
	case strings.Contains(lower, "authentication") ||
		strings.Contains(lower, "403") ||
		strings.Contains(lower, "permission denied"):
		return &CloneError{
			UserMessage: "仓库访问被拒绝，请确认为公开仓库",
			RawError:    fmt.Errorf("%w, output: %s", err, output),
		}
	case strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "timed out"):
		return &CloneError{
			UserMessage: "克隆超时，仓库可能过大或网络不稳定",
			RawError:    fmt.Errorf("%w, output: %s", err, output),
		}
	case strings.Contains(lower, "empty repository"):
		return &CloneError{
			UserMessage: "仓库为空，请确认包含代码",
			RawError:    fmt.Errorf("%w, output: %s", err, output),
		}
	default:
		return &CloneError{
			UserMessage: "克隆仓库失败，请检查地址后重试",
			RawError:    fmt.Errorf("%w, output: %s", err, output),
		}
	}
}

// isTransient 判断克隆错误是否为暂时性错误（值得重试）
func isTransient(ce *CloneError) bool {
	// 仓库不存在、权限拒绝、仓库为空 → 不重试
	nonTransient := []string{
		"仓库不存在",
		"仓库访问被拒绝",
		"仓库为空",
	}
	for _, s := range nonTransient {
		if strings.Contains(ce.UserMessage, s) {
			return false
		}
	}
	return true
}

// CloneRepo 浅克隆仓库到指定目录，支持超时控制
func CloneRepo(ctx context.Context, repoURL, destDir string, timeoutSeconds int) *CloneError {
	// 确保目标目录不存在
	if _, err := os.Stat(destDir); err == nil {
		if err := os.RemoveAll(destDir); err != nil {
			return &CloneError{
				UserMessage: "克隆仓库失败，请检查地址后重试",
				RawError:    fmt.Errorf("failed to clean existing directory: %w", err),
			}
		}
	}

	// 创建父目录
	parentDir := filepath.Dir(destDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return &CloneError{
			UserMessage: "克隆仓库失败，请检查地址后重试",
			RawError:    fmt.Errorf("failed to create parent directory: %w", err),
		}
	}

	// 创建带超时的子 context
	if timeoutSeconds <= 0 {
		timeoutSeconds = 120
	}
	cloneCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// 执行浅克隆
	cmd := exec.CommandContext(cloneCtx, "git", "clone", "--depth", "1", repoURL, destDir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// 克隆失败，清理残留目录
		os.RemoveAll(destDir)
		return classifyCloneError(string(output), err)
	}

	return nil
}

// CloneRepoWithRetry 带重试的克隆，指数退避，非暂时性错误不重试
func CloneRepoWithRetry(ctx context.Context, repoURL, destDir string, timeoutSeconds, maxRetries int) error {
	if maxRetries <= 0 {
		maxRetries = 2
	}

	var lastErr *CloneError
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			log.Printf("Clone retry %d/%d after %v for %s", attempt, maxRetries, backoff, repoURL)
			select {
			case <-ctx.Done():
				return &CloneError{
					UserMessage: "克隆超时，仓库可能过大或网络不稳定",
					RawError:    ctx.Err(),
				}
			case <-time.After(backoff):
			}
		}

		lastErr = CloneRepo(ctx, repoURL, destDir, timeoutSeconds)
		if lastErr == nil {
			return nil
		}

		log.Printf("Clone attempt %d failed: %v", attempt+1, lastErr.RawError)

		// 非暂时性错误不重试
		if !isTransient(lastErr) {
			return lastErr
		}
	}

	return lastErr
}

// CleanupRepo 清理临时仓库目录
func CleanupRepo(dir string) error {
	if dir == "" {
		return nil
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

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
func ValidateRepoURL(repoURL string) error {
	if repoURL == "" {
		return &CloneError{
			UserMessage: "仓库地址不能为空",
		}
	}

	if strings.HasPrefix(repoURL, "git@") {
		// git@github.com:user/repo.git 格式
		return nil
	}

	if !strings.HasPrefix(repoURL, "https://") {
		return &CloneError{
			UserMessage: "仓库地址格式不正确，请使用 https:// 或 git@ 开头的地址",
		}
	}

	// 解析 https URL，检查至少有 host/user/repo 三段
	u, err := url.Parse(repoURL)
	if err != nil {
		return &CloneError{
			UserMessage: "仓库地址格式不正确，请检查后重试",
			RawError:    err,
		}
	}

	if u.Host == "" {
		return &CloneError{
			UserMessage: "仓库地址缺少域名，请检查后重试",
		}
	}

	// 路径至少需要 /user/repo
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return &CloneError{
			UserMessage: "仓库地址不完整，请提供完整的 用户名/仓库名 地址",
		}
	}

	return nil
}
