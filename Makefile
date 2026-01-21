.PHONY: dev-server dev-worker build test clean migrate-up migrate-down

# Development
dev-server:
	go run cmd/server/main.go

dev-worker:
	go run cmd/worker/main.go

# Build
build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/worker cmd/worker/main.go

# Test
test:
	go test -v ./...

# Clean
clean:
	rm -rf bin/

# Database migrations (requires golang-migrate)
migrate-up:
	migrate -path migrations -database "mysql://root:password@tcp(localhost:3306)/go_analyzer" up

migrate-down:
	migrate -path migrations -database "mysql://root:password@tcp(localhost:3306)/go_analyzer" down 1

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)
