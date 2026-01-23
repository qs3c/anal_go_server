# Phase 4 - 测试与部署

## 概述

完成测试框架搭建和部署配置。

## 任务列表

### Task 1: 测试工具和配置

创建测试辅助工具和配置：
- `internal/testutil/db.go` - 测试数据库辅助
- `internal/testutil/fixtures.go` - 测试数据生成

### Task 2: Repository 层单元测试

创建 Repository 测试：
- `internal/repository/user_repo_test.go`
- `internal/repository/analysis_repo_test.go`

### Task 3: Service 层单元测试

创建 Service 测试：
- `internal/service/auth_service_test.go`
- `internal/service/quota_service_test.go`

### Task 4: Handler 层 API 测试

创建 Handler 测试：
- `internal/api/handler/auth_test.go`
- `internal/api/handler/analysis_test.go`

### Task 5: Dockerfile - Server

创建 `Dockerfile.server`:
```dockerfile
FROM golang:1.22-alpine AS builder
...
FROM alpine:latest
...
```

### Task 6: Dockerfile - Worker

创建 `Dockerfile.worker`:
```dockerfile
FROM golang:1.22-alpine AS builder
...
FROM alpine:latest
...
```

### Task 7: docker-compose.yml

创建完整的 docker-compose 配置：
- server 服务
- worker 服务
- mysql 服务
- redis 服务
- 网络和卷配置

### Task 8: Makefile

创建 Makefile 包含常用命令：
- build, test, run
- docker-build, docker-up, docker-down
- migrate-up, migrate-down

### Task 9: 数据库迁移脚本

创建 SQL 迁移文件：
- `migrations/001_create_users.sql`
- `migrations/002_create_analyses.sql`
- `migrations/003_create_jobs.sql`
- `migrations/004_create_comments.sql`
- `migrations/005_create_interactions.sql`
- `migrations/006_create_subscriptions.sql`

### Task 10: 环境配置示例

创建配置文件示例：
- `config.example.yaml`
- `.env.example`

### Task 11: README 更新

更新 README.md 包含：
- 项目介绍
- 快速开始
- API 概览
- 部署说明

### Task 12: 依赖整理和验证

```bash
go mod tidy
go build ./...
go test ./...
```

## 验证

- [ ] go test ./... 通过
- [ ] docker-compose up 可启动
- [ ] 迁移脚本可执行
