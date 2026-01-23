package testutil

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/qs3c/anal_go_server/internal/model"
)

// SetupTestDB 创建测试数据库（SQLite 内存模式）
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect test database: %v", err)
	}

	// 自动迁移所有模型
	err = db.AutoMigrate(
		&model.User{},
		&model.Analysis{},
		&model.AnalysisJob{},
		&model.Comment{},
		&model.Interaction{},
		&model.Subscription{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// SetupTestDBWithMySQL 使用 MySQL 测试数据库（需要环境变量）
func SetupTestDBWithMySQL(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping MySQL tests")
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect test database: %v", err)
	}

	return db
}

// CleanupTestDB 清理测试数据库
func CleanupTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	if err != nil {
		t.Logf("Warning: Failed to get underlying DB: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		t.Logf("Warning: Failed to close test database: %v", err)
	}
}

// TruncateTables 清空所有表数据
func TruncateTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	tables := []string{
		"interactions",
		"comments",
		"analysis_jobs",
		"analyses",
		"subscriptions",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			t.Logf("Warning: Failed to truncate table %s: %v", table, err)
		}
	}
}
