# Go Struct Analyzer Backend

A Go backend service for AI-powered Go project structure analysis platform.

## Architecture

```
Frontend (React) → Nginx → API Server (Gin)
                              ↓
                    MySQL / Redis / OSS
                              ↓
                    Redis Queue → Worker → anal_go_agent
```

- **API Server**: HTTP REST API + WebSocket for real-time updates
- **Worker**: Background job processor for analysis tasks
- **anal_go_agent**: External Go library for struct dependency analysis

## Tech Stack

- Go 1.22+
- Gin Web Framework
- GORM (MySQL 8.0)
- Redis 7.0
- Aliyun OSS
- Docker & Docker Compose

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- MySQL 8.0 (or use Docker)
- Redis 7.0 (or use Docker)

### Development Setup

1. Clone the repository:
```bash
git clone https://github.com/qs3c/anal_go_server.git
cd anal_go_server
```

2. Copy environment config:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Start dependencies with Docker:
```bash
docker-compose up -d mysql redis
```

4. Run migrations:
```bash
make migrate-up
```

5. Run the server:
```bash
make dev-server
```

6. Run the worker (in a separate terminal):
```bash
make dev-worker
```

### Docker Deployment

Build and run all services:
```bash
docker-compose up -d
```

## Project Structure

```
cmd/
  server/main.go          # API server entrypoint
  worker/main.go          # Worker entrypoint
internal/
  api/handler/            # HTTP handlers
  api/middleware/         # JWT auth, quota, CORS, rate limiting
  service/                # Business logic
  repository/             # Data access layer
  model/                  # Entities + DTOs
  pkg/                    # Utilities (oauth, jwt, oss, ws, queue, email)
  config/                 # Configuration
  testutil/               # Test helpers
migrations/               # SQL migration files
```

## Available Commands

```bash
# Development
make dev-server           # Run API server
make dev-worker           # Run worker

# Build
make build                # Build all binaries
make build-server         # Build server only
make build-worker         # Build worker only

# Testing
make test                 # Run all tests
make test-coverage        # Run tests with coverage

# Code Quality
make lint                 # Run linter
make fmt                  # Format code
make vet                  # Run go vet

# Docker
make docker-build         # Build Docker images
make docker-up            # Start all services
make docker-down          # Stop all services
make docker-logs          # View logs

# Database
make migrate-up           # Run migrations
make migrate-down         # Rollback migration
make migrate-create name=xxx  # Create new migration

# Cleanup
make clean                # Clean build artifacts
make tidy                 # Tidy go modules
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Email registration
- `POST /api/v1/auth/login` - Email login
- `GET /api/v1/auth/github` - GitHub OAuth
- `GET /api/v1/auth/wechat` - WeChat OAuth
- `POST /api/v1/auth/verify-email` - Verify email

### User
- `GET /api/v1/user/profile` - Get profile
- `PUT /api/v1/user/profile` - Update profile
- `GET /api/v1/user/quota` - Get quota info

### Analysis
- `POST /api/v1/analyses` - Create analysis
- `GET /api/v1/analyses` - List user's analyses
- `GET /api/v1/analyses/:id` - Get analysis detail
- `PUT /api/v1/analyses/:id` - Update analysis
- `DELETE /api/v1/analyses/:id` - Delete analysis

### Community
- `GET /api/v1/community/analyses` - Browse public analyses
- `POST /api/v1/community/analyses/:id/like` - Like analysis
- `POST /api/v1/community/analyses/:id/bookmark` - Bookmark analysis
- `GET /api/v1/community/analyses/:id/comments` - Get comments
- `POST /api/v1/community/analyses/:id/comments` - Add comment

### WebSocket
- `GET /api/v1/ws` - Real-time analysis progress updates

## Subscription Tiers

| Tier | Daily Quota | Max Depth | Price |
|------|-------------|-----------|-------|
| Free | 5 | 3 | - |
| Basic | 30 | 5 | ¥19.9/mo |
| Pro | 100 | 10 | ¥49.9/mo |

## Environment Variables

See `.env.example` for all available configuration options.

## License

MIT
