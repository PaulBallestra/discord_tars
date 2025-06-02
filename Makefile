# Project Configuration
PROJECT_NAME := discord-rag-agent
GO_VERSION := 1.24.3
BINARY_PATH := bin
DOCKER_COMPOSE_FILE := deployments/docker/docker-compose.dev.yml

# Load environment variables from .env if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Go Configuration
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Binary names
BOT_BINARY := $(BINARY_PATH)/bot
VOICE_PROCESSOR_BINARY := $(BINARY_PATH)/voice-processor
RAG_INDEXER_BINARY := $(BINARY_PATH)/rag-indexer

# Database tool
MIGRATE_TOOL := $(shell go env GOPATH)/bin/migrate

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(shell git describe --tags --always --dirty) -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

# Default target
.PHONY: help
help: ## Show this help message
	@echo "$(PROJECT_NAME) - TARS Agent"
	@echo "Go version: $(GO_VERSION)"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Setup & Dependencies
.PHONY: setup
setup: deps ## Initial project setup
	@echo "🚀 Setting up $(PROJECT_NAME)..."
	@if [ ! -f .env ]; then \
        cp .env.example .env; \
        echo "✅ Created .env from .env.example"; \
        echo ""; \
        echo "⚠️  Next steps:"; \
        echo "   1. Edit .env with your configuration:"; \
        echo "      - Add your DISCORD_TOKEN"; \
        echo "      - Add your OPENAI_API_KEY"; \
        echo "      - Update POSTGRES_PASSWORD if needed"; \
        echo "   2. Run: make test-config"; \
        echo "   3. Run: make dev-infra"; \
        echo ""; \
    else \
        echo "✅ .env file already exists"; \
    fi

.PHONY: deps
deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "✅ Dependencies downloaded"

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "🔄 Updating dependencies..."
	@$(GOGET) -u ./...
	@$(GOMOD) tidy
	@echo "✅ Dependencies updated"

.PHONY: test-config
test-config: ## Test configuration loading
	@echo "🔧 Testing configuration..."
	@$(GOBUILD) -o $(BINARY_PATH)/config-test ./cmd/bot
	@./$(BINARY_PATH)/config-test || (echo "❌ Config test failed"; exit 1)
	@echo "✅ Configuration test passed"

##@ Building
.PHONY: build
build: build-bot build-voice-processor build-rag-indexer ## Build all binaries

.PHONY: build-bot
build-bot: ## Build Discord bot binary
	@echo "🔨 Building Discord bot..."
	@mkdir -p $(BINARY_PATH)
	@$(GOBUILD) $(LDFLAGS) -o $(BOT_BINARY) ./cmd/bot
	@echo "✅ Bot binary built: $(BOT_BINARY)"

.PHONY: build-voice-processor
build-voice-processor: ## Build voice processor binary
	@echo "🔨 Building voice processor..."
	@mkdir -p $(BINARY_PATH)
	@$(GOBUILD) $(LDFLAGS) -o $(VOICE_PROCESSOR_BINARY) ./cmd/voice-processor
	@echo "✅ Voice processor binary built: $(VOICE_PROCESSOR_BINARY)"

.PHONY: build-rag-indexer
build-rag-indexer: ## Build RAG indexer binary
	@echo "🔨 Building RAG indexer..."
	@mkdir -p $(BINARY_PATH)
	@$(GOBUILD) $(LDFLAGS) -o $(RAG_INDEXER_BINARY) ./cmd/rag-indexer
	@echo "✅ RAG indexer binary built: $(RAG_INDEXER_BINARY)"

.PHONY: build-linux
build-linux: ## Build binaries for Linux
	@echo "🔨 Building for Linux..."
	@mkdir -p $(BINARY_PATH)/linux
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)/linux/bot ./cmd/bot
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)/linux/voice-processor ./cmd/voice-processor
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)/linux/rag-indexer ./cmd/rag-indexer
	@echo "✅ Linux binaries built"

##@ Testing
.PHONY: test
test: ## Run tests
	@echo "🧪 Running tests..."
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "✅ Tests completed"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	@$(GOTEST) -v -tags=integration ./tests/integration/...
	@echo "✅ Integration tests completed"

.PHONY: test-load
test-load: ## Run load tests
	@echo "🧪 Running load tests..."
	@$(GOTEST) -v -tags=load ./tests/load/...
	@echo "✅ Load tests completed"

.PHONY: test-coverage
test-coverage: test ## Generate test coverage report
	@echo "📊 Generating coverage report..."
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

.PHONY: test-watch
test-watch: ## Watch and run tests on file changes
	@echo "👀 Watching for changes..."
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	@air -c .air-test.toml

##@ Code Quality
.PHONY: lint
lint: ## Run linter
	@echo "🔍 Running linter..."
	@which $(GOLINT) > /dev/null || (echo "Installing golangci-lint..." && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin)
	@$(GOLINT) run ./...
	@echo "✅ Linting completed"

.PHONY: fmt
fmt: ## Format code
	@echo "🎨 Formatting code..."
	@$(GOFMT) -s -w .
	@echo "✅ Code formatted"

.PHONY: fmt-check
fmt-check: ## Check code formatting
	@echo "🔍 Checking code formatting..."
	@test -z "$$($(GOFMT) -s -l . | tee /dev/stderr)" || (echo "❌ Code is not formatted"; exit 1)
	@echo "✅ Code formatting is correct"

.PHONY: vet
vet: ## Run go vet
	@echo "🔍 Running go vet..."
	@$(GOCMD) vet ./...
	@echo "✅ Vet completed"

.PHONY: check
check: fmt-check vet lint test ## Run all checks (format, vet, lint, test)

##@ Database
.PHONY: install-migrate
install-migrate: ## Install migrate tool
	@echo "📦 Installing migrate tool..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "✅ Migrate tool installed at $(MIGRATE_TOOL)"

.PHONY: db-migrate-up
db-migrate-up: ## Run database migrations up
	@echo "⬆️ Running database migrations..."
	@if [ ! -f "$(MIGRATE_TOOL)" ]; then \
        echo "Installing migrate tool..."; \
        $(MAKE) install-migrate; \
    fi
	@echo "Running migrations..."
	@$(MIGRATE_TOOL) -path migrations -database "$$POSTGRES_URL" up
	@echo "✅ Migrations completed"

.PHONY: db-migrate-down
db-migrate-down: ## Run database migrations down
	@echo "⬇️ Rolling back database migrations..."
	@if [ ! -f "$(MIGRATE_TOOL)" ]; then \
        $(MAKE) install-migrate; \
    fi
	@$(MIGRATE_TOOL) -path migrations -database "$$POSTGRES_URL" down
	@echo "✅ Migrations rolled back"

.PHONY: db-migrate-create
db-migrate-create: ## Create new migration (usage: make db-migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then echo "❌ NAME is required. Usage: make db-migrate-create NAME=migration_name"; exit 1; fi
	@echo "📝 Creating migration: $(NAME)..."
	@if [ ! -f "$(MIGRATE_TOOL)" ]; then \
        $(MAKE) install-migrate; \
    fi
	@$(MIGRATE_TOOL) create -ext sql -dir migrations $(NAME)
	@echo "✅ Migration created"

.PHONY: db-reset
db-reset: ## Reset database (down + up)
	@echo "🔄 Resetting database..."
	@$(MAKE) db-migrate-down || true
	@$(MAKE) db-migrate-up
	@echo "✅ Database reset completed"

.PHONY: db-status
db-status: ## Show migration status
	@echo "📊 Checking migration status..."
	@if [ ! -f "$(MIGRATE_TOOL)" ]; then \
        $(MAKE) install-migrate; \
    fi
	@$(MIGRATE_TOOL) -path migrations -database "$$POSTGRES_URL" version

##@ Docker & Infrastructure
.PHONY: dev-infra
dev-infra:
	@echo "🏗️ Starting development infrastructure..." 
	docker-compose -f deployments/docker/docker-compose.dev.yml up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo ""
	@echo "✅ Development infrastructure started!"
	@echo "📊 Access points:"
	@echo "   - Grafana:    http://localhost:3000 (admin/admin)"
	@echo "   - Prometheus: http://localhost:9090"
	@echo "   - Jaeger:     http://localhost:16686"
	@echo "   - PostgreSQL: localhost:5432"
	@echo "   - Redis:      localhost:6379"
	@echo ""
	@echo "🗄️ Database initialized with pgvector and T.A.R.S schema"

.PHONY: dev-infra-down
dev-infra-down:
	@echo "🛑 Stopping development infrastructure..."
	docker-compose -f deployments/docker/docker-compose.dev.yml down -v
	@echo "✅ Development infrastructure stopped!"

.PHONY: dev
dev: dev-infra build-bot ## Start full development environment
	@echo "🚀 Starting Discord bot..."
	@./$(BOT_BINARY)

.PHONY: dev-watch
dev-watch: dev-infra ## Start development with hot reload
	@echo "👀 Starting development with hot reload..."
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	@air -c .air.toml

.PHONY: docker-build
docker-build: ## Build Docker images
	@echo "🐳 Building Docker images..."
	@docker build -f deployments/docker/Dockerfile.bot -t $(PROJECT_NAME)/bot:latest .
	@docker build -f deployments/docker/Dockerfile.voice-processor -t $(PROJECT_NAME)/voice-processor:latest .
	@docker build -f deployments/docker/Dockerfile.rag-indexer -t $(PROJECT_NAME)/rag-indexer:latest .
	@echo "✅ Docker images built"

.PHONY: docker-up
docker-up: ## Start with Docker Compose
	@echo "🐳 Starting with Docker Compose..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
	@echo "✅ All services started with Docker"

.PHONY: docker-down
docker-down: ## Stop Docker Compose services
	@echo "🐳 Stopping Docker Compose services..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down
	@echo "✅ Services stopped"

.PHONY: docker-logs
docker-logs: ## Show Docker Compose logs
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

.PHONY: docker-clean
docker-clean: ## Clean Docker resources
	@echo "🧹 Cleaning Docker resources..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down -v
	@docker system prune -af
	@echo "✅ Docker resources cleaned"

##@ Generation & Protobuf
.PHONY: proto
proto: ## Generate protobuf files
	@echo "🔧 Generating protobuf files..."
	@which protoc > /dev/null || (echo "❌ protoc is required. Install Protocol Buffers compiler"; exit 1)
	@protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        api/proto/*.proto
	@echo "✅ Protobuf files generated"

.PHONY: mocks
mocks: ## Generate mocks
	@echo "🎭 Generating mocks..."
	@which mockgen > /dev/null || (echo "Installing mockgen..." && go install github.com/golang/mock/mockgen@latest)
	@$(GOCMD) generate ./...
	@echo "✅ Mocks generated"

.PHONY: seed-data
seed-data: ## Seed development data
	@echo "🌱 Seeding development data..."
	@$(GOCMD) run scripts/seed/main.go
	@echo "✅ Data seeded"

##@ Deployment
.PHONY: deploy-staging
deploy-staging: ## Deploy to staging
	@echo "🚀 Deploying to staging..."
	@kubectl apply -f deployments/k8s/staging/
	@echo "✅ Deployed to staging"

.PHONY: deploy-prod
deploy-prod: ## Deploy to production
	@echo "🚀 Deploying to production..."
	@echo "⚠️  Are you sure? This will deploy to production! (Press Ctrl+C to cancel)"
	@read -p "Type 'yes' to continue: " confirm && [ "$$confirm" = "yes" ] && kubectl apply -f deployments/k8s/production/
	@echo "✅ Deployed to production"

##@ Utilities
.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	@$(GOCLEAN)
	@rm -rf $(BINARY_PATH)
	@rm -f coverage.out coverage.html
	@echo "✅ Cleaned"

.PHONY: version
version: ## Show version information
	@echo "Project: $(PROJECT_NAME)"
	@echo "Go version: $$(go version)"
	@echo "Git commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Git tag: $$(git describe --tags --always 2>/dev/null || echo 'unknown')"

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "🔧 Installing development tools..."
	@$(GOCMD) install github.com/cosmtrek/air@latest
	@$(GOCMD) install github.com/golang/mock/mockgen@latest
	@$(GOCMD) install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	@echo "✅ Development tools installed"

.PHONY: check-env
check-env: ## Check environment variables
	@echo "🔍 Checking environment variables..."
	@echo "DISCORD_TOKEN: $$(if [ -n "$$DISCORD_TOKEN" ]; then echo "✅ Set"; else echo "❌ Missing"; fi)"
	@echo "OPENAI_API_KEY: $$(if [ -n "$$OPENAI_API_KEY" ]; then echo "✅ Set"; else echo "❌ Missing"; fi)"
	@echo "POSTGRES_URL: $$(if [ -n "$$POSTGRES_URL" ]; then echo "✅ Set"; else echo "❌ Missing"; fi)"
	@echo "POSTGRES_USER: $$POSTGRES_USER"
	@echo "POSTGRES_DB: $$POSTGRES_DB"

##@ Quick Start
.PHONY: quick-start
quick-start: setup install-tools ## Quick start for new developers
	@echo ""
	@echo "🎉 Quick start completed!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Edit .env with your tokens: nano .env"
	@echo "2. Test configuration: make test-config"
	@echo "3. Start infrastructure: make dev-infra"
	@echo "4. Run the bot: make dev"
	@echo ""
	@echo "Or use hot reload: make dev-watch"

# Default target when just running 'make'
.DEFAULT_GOAL := help
