.PHONY: all build test clean dev-server dev-worker docker-build docker-up docker-down migrate-up migrate-down migrate-create lint fmt help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Binary names
SERVER_BINARY=server
WORKER_BINARY=worker

# Build directories
BUILD_DIR=bin
CMD_DIR=cmd

# Migration parameters
MIGRATE_DIR=migrations
DB_URL?=mysql://root:password@tcp(localhost:3306)/anal_go

# Default target
all: build

## Build

build: build-server build-worker ## Build all binaries

build-server: ## Build server binary
	@echo "Building server..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(SERVER_BINARY) ./$(CMD_DIR)/server

build-worker: ## Build worker binary
	@echo "Building worker..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(WORKER_BINARY) ./$(CMD_DIR)/worker

## Development

dev-server: ## Run server in development mode
	@echo "Starting server..."
	$(GOCMD) run ./$(CMD_DIR)/server

dev-worker: ## Run worker in development mode
	@echo "Starting worker..."
	$(GOCMD) run ./$(CMD_DIR)/worker

## Testing

test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

test-short: ## Run tests without race detector
	@echo "Running tests (short)..."
	$(GOTEST) -v -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Code Quality

lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## Dependencies

tidy: ## Tidy go modules
	@echo "Tidying modules..."
	$(GOMOD) tidy

download: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

## Docker

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	docker-compose build

docker-up: ## Start all services
	@echo "Starting services..."
	docker-compose up -d

docker-down: ## Stop all services
	@echo "Stopping services..."
	docker-compose down

docker-logs: ## View logs
	docker-compose logs -f

docker-clean: ## Remove containers and volumes
	@echo "Cleaning up Docker..."
	docker-compose down -v --remove-orphans

## Database Migrations

migrate-up: ## Run all migrations
	@echo "Running migrations..."
	migrate -path $(MIGRATE_DIR) -database "$(DB_URL)" up

migrate-down: ## Rollback last migration
	@echo "Rolling back migration..."
	migrate -path $(MIGRATE_DIR) -database "$(DB_URL)" down 1

migrate-drop: ## Drop all tables (DANGEROUS)
	@echo "Dropping all tables..."
	migrate -path $(MIGRATE_DIR) -database "$(DB_URL)" drop -f

migrate-create: ## Create new migration (usage: make migrate-create name=migration_name)
	@echo "Creating migration: $(name)"
	@test -n "$(name)" || (echo "Usage: make migrate-create name=migration_name" && exit 1)
	migrate create -ext sql -dir $(MIGRATE_DIR) -seq $(name)

migrate-version: ## Show current migration version
	migrate -path $(MIGRATE_DIR) -database "$(DB_URL)" version

## Cleanup

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## Help

help: ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
