# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Status

This is a **specification repository** containing detailed planning documents for a Go-based AI project structure analysis platform. Implementation code does not yet exist.

Key specification files:
- `backend-complete-guide.md` - Complete backend development specification (Chinese)
- `integration-specification.md` - Frontend-backend contract specification (Chinese)

## Planned Build Commands

```bash
# Development
go run cmd/server/main.go      # API server
go run cmd/worker/main.go      # Worker process

# Build & Test
make dev-server
make dev-worker
make build
make test

# Database migrations
make migrate-up
make migrate-down
make migrate-create name=<migration_name>

# Docker
docker-compose up -d
```

## Architecture Overview

Dual-process system with async job processing:

```
Frontend (React) → Nginx → API Server (Gin)
                              ↓
                    MySQL / Redis / OSS
                              ↓
                    Redis Queue → Worker → anal_go_agent
```

- **API Server** (`cmd/server/main.go`): HTTP REST + WebSocket for real-time updates
- **Worker** (`cmd/worker/main.go`): Processes analysis jobs from Redis queue
- **anal_go_agent**: External Go library for analyzing Go project struct dependencies

## Planned Project Structure

```
cmd/
  server/main.go, worker/main.go
internal/
  api/handler/       # HTTP handlers (auth, user, analysis, community, comment, websocket)
  api/middleware/    # JWT auth, quota, CORS, rate limiting
  service/           # Business logic
  repository/        # Data access (GORM)
  model/             # Entities + DTOs
  pkg/               # Utilities (oauth, jwt, oss, ws hub, queue, email)
  config/            # Configuration
migrations/          # SQL migration files
```

## Critical Conventions (from integration-specification.md)

**JSON field naming**: Always use `snake_case` in JSON tags
```go
type User struct {
    UserID    int64     `json:"user_id"`
    CreatedAt time.Time `json:"created_at"`
}
```

**DateTime format**: RFC3339/ISO 8601 (`"2025-01-20T10:30:00Z"`)

**Response format**:
```go
type Response struct {
    Code    int         `json:"code"`    // 0=success, 1000+=error
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}
```

**Error codes**:
- 0: Success
- 1000: Parameter error
- 1001: Auth failed
- 1002: Permission denied
- 1003: Resource not found
- 1004: Quota exceeded
- 1005: Duplicate action
- 5000: Server error

**Enums**: Lowercase strings only (`"free"`, `"basic"`, `"pro"`, `"completed"`)

## Tech Stack

- Go 1.22+, Gin, GORM, gorilla/websocket, go-redis
- MySQL 8.0, Redis 7.0, Aliyun OSS
- JWT auth + GitHub OAuth + WeChat OAuth
- OpenAI/Anthropic APIs for analysis

## Database Tables

6 core tables: `users`, `analyses`, `analysis_jobs`, `comments`, `interactions`, `subscriptions`

## Subscription Tiers

- **Free**: 5/day, depth ≤ 3
- **Basic** (¥19.9/mo): 30/day, depth ≤ 5
- **Pro** (¥49.9/mo): 100/day, depth ≤ 10
