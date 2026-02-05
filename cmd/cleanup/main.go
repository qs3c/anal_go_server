package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/model"
)

var (
	dryRun         = flag.Bool("dry-run", true, "Dry run mode, don't actually delete files")
	uploadExpire   = flag.Int("upload-expire", 24, "Hours to keep uploaded source files")
	diagramExpire  = flag.Int("diagram-expire", 7, "Days to keep local diagram files")
	cleanUploads   = flag.Bool("clean-uploads", true, "Clean expired upload files")
	cleanDiagrams  = flag.Bool("clean-diagrams", true, "Clean diagrams migrated to OSS")
)

func main() {
	flag.Parse()

	log.Println("🧹 Starting cleanup task...")
	log.Printf("Mode: dry-run=%v", *dryRun)

	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	db, err := connectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	uploadDir := cfg.Upload.TempDir
	totalSize := int64(0)
	deletedSize := int64(0)
	totalFiles := 0
	deletedFiles := 0

	// 1. 清理过期的上传文件
	if *cleanUploads {
		log.Printf("\n📦 Cleaning expired upload files (older than %d hours)...", *uploadExpire)
		size, count := cleanExpiredUploads(uploadDir, *uploadExpire, *dryRun)
		deletedSize += size
		deletedFiles += count
	}

	// 2. 清理已迁移到OSS的diagram文件
	if *cleanDiagrams {
		log.Printf("\n📊 Cleaning diagrams migrated to OSS...")
		size, count := cleanMigratedDiagrams(db, uploadDir, *diagramExpire, *dryRun)
		deletedSize += size
		deletedFiles += count
	}

	// 3. 统计当前占用
	log.Println("\n📈 Scanning current disk usage...")
	filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
			totalFiles++
		}
		return nil
	})

	// 输出统计
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("📊 Cleanup Summary")
	log.Println(strings.Repeat("=", 60))
	log.Printf("Total files: %d", totalFiles)
	log.Printf("Total size: %s", formatSize(totalSize))
	log.Printf("Deleted files: %d", deletedFiles)
	log.Printf("Freed space: %s", formatSize(deletedSize))
	if *dryRun {
		log.Println("\n⚠️  DRY RUN MODE - No files were actually deleted")
		log.Println("   Run with -dry-run=false to actually delete files")
	} else {
		log.Println("\n✅ Cleanup completed!")
	}
	log.Println(strings.Repeat("=", 60))
}

// cleanExpiredUploads 清理过期的上传文件
func cleanExpiredUploads(uploadDir string, expireHours int, dryRun bool) (int64, int) {
	expireTime := time.Now().Add(-time.Duration(expireHours) * time.Hour)
	var totalSize int64
	var count int

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		log.Printf("Failed to read upload dir: %v", err)
		return 0, 0
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 跳过diagrams目录
		if entry.Name() == "diagrams" {
			continue
		}

		dirPath := filepath.Join(uploadDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 检查是否过期
		if info.ModTime().Before(expireTime) {
			size := getDirSize(dirPath)
			totalSize += size

			log.Printf("  - %s (%.2f MB, %s old)",
				entry.Name(),
				float64(size)/1024/1024,
				time.Since(info.ModTime()).Round(time.Hour))

			if !dryRun {
				if err := os.RemoveAll(dirPath); err != nil {
					log.Printf("    ❌ Failed to delete: %v", err)
				} else {
					count++
				}
			} else {
				count++
			}
		}
	}

	log.Printf("Found %d expired upload directories (total: %s)",
		count, formatSize(totalSize))

	return totalSize, count
}

// cleanMigratedDiagrams 清理已迁移到OSS的diagram文件
func cleanMigratedDiagrams(db *gorm.DB, uploadDir string, keepDays int, dryRun bool) (int64, int) {
	diagramDir := filepath.Join(uploadDir, "diagrams")
	var totalSize int64
	var count int

	// 获取所有已迁移到OSS的分析记录
	var analyses []model.Analysis
	err := db.Where("diagram_oss_url LIKE ?", "https://%").
		Find(&analyses).Error
	if err != nil {
		log.Printf("Failed to query analyses: %v", err)
		return 0, 0
	}

	log.Printf("Found %d analyses migrated to OSS", len(analyses))

	// 为了安全，只删除超过N天的旧文件
	expireTime := time.Now().Add(-time.Duration(keepDays) * 24 * time.Hour)

	for _, analysis := range analyses {
		localPath := filepath.Join(diagramDir, fmt.Sprintf("%d.json", analysis.ID))

		// 检查文件是否存在
		info, err := os.Stat(localPath)
		if os.IsNotExist(err) {
			continue // 文件不存在，跳过
		}
		if err != nil {
			log.Printf("  ⚠️  Failed to stat %d.json: %v", analysis.ID, err)
			continue
		}

		// 只删除超过指定天数的文件（安全措施）
		if info.ModTime().Before(expireTime) {
			totalSize += info.Size()

			log.Printf("  - %d.json (%.2f KB, migrated to OSS, %s old)",
				analysis.ID,
				float64(info.Size())/1024,
				time.Since(info.ModTime()).Round(time.Hour))

			if !dryRun {
				if err := os.Remove(localPath); err != nil {
					log.Printf("    ❌ Failed to delete: %v", err)
				} else {
					count++
				}
			} else {
				count++
			}
		}
	}

	log.Printf("Found %d diagram files to clean (total: %s)",
		count, formatSize(totalSize))

	return totalSize, count
}

// getDirSize 计算目录大小
func getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// formatSize 格式化文件大小
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// connectDB 连接数据库
func connectDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
