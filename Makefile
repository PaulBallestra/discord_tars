.PHONY: build test lint docker-up docker-down dev clean

# Variables
BINARY_NAME=discord-rag-agent
BINARY_PATH=./bin/
DOCKER_COMPOSE_FILE=deployments/docker/docker-compose.yml
DOCKER_DEV_COMPOSE_FILE=deployments/docker/docker-compose.dev.yml

# Development
dev:
    docker-compose -f $(DOCKER_DEV_COMPOSE_FILE) up --build

dev-infra:
    docker-compose -f $(DOCKER_DEV_COMPOSE_FILE) up -d postgres redis prometheus grafana jaeger

stop-dev:
    docker-compose -f $(DOCKER_DEV_COMPOSE_FILE) down

# Building
build: build-bot build-voice build-rag

build-bot:
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BINARY_PATH)bot cmd/bot/main.go

build-voice:
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BINARY_PATH)voice-processor cmd/voice-processor/main.go

build-rag:
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BINARY_PATH)rag-indexer cmd/rag-indexer/main.go

# Testing
test:
    go test -v -race ./...

test-coverage:
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

test-integration:
    go test -v -tags=integration ./tests/integration/...

# Code Quality
lint:
    golangci-lint run

fmt:
    go fmt ./...

vet:
    go vet ./...

# Dependencies
deps:
    go mod download
    go mod tidy

deps-upgrade:
    go get -u ./...
    go mod tidy

# Database
migrate-up:
    migrate -path migrations -database "$(shell grep POSTGRES_ .env | xargs -I {} echo {} | tr '\n' ' ' | sed 's/POSTGRES_HOST=/postgres:\/\//; s/POSTGRES_USER=/&/; s/POSTGRES_PASSWORD=/:/; s/POSTGRES_HOST=/&@/; s/POSTGRES_PORT=/:/; s/POSTGRES_DB=/\//; s/POSTGRES_SSL_MODE=/?sslmode=/; s/ //g')" up

migrate-down:
    migrate -path migrations -database "$(DATABASE_URL)" down

migrate-create:
    @read -p "Enter migration name: " name; \
    migrate create -ext sql -dir migrations $$name

# Docker
docker-build:
    docker-compose -f $(DOCKER_COMPOSE_FILE) build

docker-up:
    docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-down:
    docker-compose -f $(DOCKER_COMPOSE_FILE) down

docker-logs:
    docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

docker-clean:
    docker-compose -f $(DOCKER_COMPOSE_FILE) down -v
    docker system prune -af

# Setup
setup: deps
    cp .env.example .env
    @echo "Please edit .env with your configuration"
    @echo "Then run: make dev-infra to start infrastructure"

# Clean
clean:
    go clean
    rm -rf $(BINARY_PATH)
    rm -f coverage.out coverage.html

# Help
help:
    @echo "Available commands:"
    @echo "  setup          - Initial project setup"
    @echo "  dev            - Start development environment"
    @echo "  dev-infra      - Start only infrastructure (DB, Redis, etc.)"
    @echo "  build          - Build all binaries"
    @echo "  test           - Run tests"
    @echo "  lint           - Run linter"
    @echo "  docker-up      - Start with Docker"
    @echo "  clean          - Clean build artifacts"
