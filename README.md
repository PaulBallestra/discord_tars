Discord RAG Agent 🤖
An intelligent Discord bot that acts as your server's AI-powered companion, capable of real-time voice conversations, contextual awareness, and serving as a brainstorming partner for work and life discussions.

✨ Features
🎤 Voice Interaction
Real-time Voice Chat: Continuous listening in voice channels with multi-user support
User Recognition: Identifies who's speaking in multi-user voice channels
Text-to-Speech: Responds with natural voice using OpenAI TTS
Speech-to-Text: Converts voice input using OpenAI Whisper
🧠 Contextual Intelligence
Server Memory: Learns from all server conversations and interactions
Real-time Awareness: Knows current server state (user count, online status, etc.)
Smart Summarization: "What happened while I was away?" functionality
RAG-Powered: Retrieval-Augmented Generation for contextual responses
🤝 AI Personality
Multi-faceted Partner: Work collaborator, life advisor, business brainstormer
Adaptive Personality: Context-aware responses based on conversation type
Server Culture: Understands server-specific memes, inside jokes, and culture
🏗️ Tech Stack
Core Technologies
Language: Go 1.24.3
Discord Library: DiscordGo
AI Services: OpenAI (GPT-4, Whisper, TTS, Embeddings)
Vector Database: PostgreSQL with pgvector extension
Cache & Messaging: Redis Cluster
Communication: gRPC + REST APIs
Infrastructure & DevOps
Containerization: Docker + Docker Compose
Orchestration: Kubernetes (production)
Monitoring: Prometheus + Grafana + Jaeger + Loki
CI/CD: GitHub Actions
Database Migrations: golang-migrate
Development Tools
Linting: golangci-lint
Testing: Go testing + Testify
Logging: Zerolog
Configuration: Viper
Documentation: OpenAPI + Protocol Buffers
📁 Project Structure
discord-rag-agent/
├── .github/                      # GitHub workflows and templates
│   ├── workflows/
│   │   ├── ci.yml               # Tests, linting, security scans
│   │   ├── cd.yml               # Auto-deployment
│   │   └── dependabot.yml       # Dependency updates
│   └── ISSUE_TEMPLATE/          # Bug reports, feature requests
├── cmd/                         # Application entrypoints
│   ├── bot/                     # Main Discord bot service
│   ├── voice-processor/         # Voice processing microservice
│   └── rag-indexer/             # RAG indexing microservice
├── internal/                    # Private application code
│   ├── repository/             # Repositories
│   │   └── postgres/           # Postgres repository
│   ├── bot/
│   │   ├── handlers/           # Discord event handlers
│   │   ├── commands/           # Slash commands
│   │   └── voice/              # Voice channel logic
│   ├── rag/
│   │   ├── embeddings/         # OpenAI embeddings
│   │   ├── retrieval/          # Vector search
│   │   └── generation/         # LLM response generation
│   ├── services/
│   │   ├── discord/            # Discord API wrapper
│   │   ├── openai/             # OpenAI client
│   │   └── storage/            # Database operations
│   │   └── rag/                # Rag service
│   ├── monitoring/             # Metrics, tracing helpers
│   ├── models/                 # Data structures
│   └── config/                 # Configuration management
├── pkg/                        # Public APIs for other bots
├── api/
│   ├── proto/                  # gRPC service definitions
│   └── openapi/                # REST API specifications
├── deployments/
│   ├── docker/                 # Docker configurations
│   ├── k8s/                    # Kubernetes manifests
│   └── monitoring/             # Grafana dashboards, Prometheus config
├── scripts/                    # Utility scripts
├── docs/                       # Documentation
├── tests/                      # Test suites
│   ├── integration/
│   ├── load/
│   └── fixtures/
├── migrations/                 # Database migrations
├── .cursorrules               # Cursor AI development rules
├── .env.example              # Environment variables template
├── .golangci.yml             # Linter configuration
├── Makefile                  # Development commands
└── README.md

🚀 Setup Instructions
Prerequisites
Ensure you have the following installed:

Go 1.24.3+: Download Go
Docker & Docker Compose: Install Docker
PostgreSQL 15+: Install PostgreSQL
Redis 7+: Install Redis
Make: Usually pre-installed on macOS/Linux
Development Tools Setup
# Install Go development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install protoc (Protocol Buffer Compiler)
# macOS
brew install protobuf

# Linux
sudo apt-get install protobuf-compiler

# Install air for hot reloading (optional)
go install github.com/cosmtrek/air@latest

Project Setup
1. Clone and Initialize
# Clone the repository
git clone https://github.com/yourusername/discord-rag-agent.git
cd discord-rag-agent

# Initialize Go modules
go mod init github.com/yourusername/discord-rag-agent
go mod tidy

2. Environment Configuration
# Copy environment template
cp .env.example .env

# Edit with your credentials
nano .env

Required environment variables:

# Discord Configuration
DISCORD_TOKEN=your_discord_bot_token
DISCORD_GUILD_ID=your_test_guild_id

# OpenAI Configuration
OPENAI_API_KEY=your_openai_api_key
OPENAI_MODEL=gpt-4
OPENAI_EMBEDDING_MODEL=text-embedding-3-large

# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=ragbot
POSTGRES_PASSWORD=secure_password
POSTGRES_DB=discord_rag
POSTGRES_SSL_MODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Monitoring Configuration
PROMETHEUS_PORT=9090
GRAFANA_PORT=3000
JAEGER_ENDPOINT=http://localhost:14268/api/traces

# Application Configuration
LOG_LEVEL=info
HTTP_PORT=8080
GRPC_PORT=8081

3. Infrastructure Setup
# Start infrastructure services
make docker-infra-up

# This starts:
# - PostgreSQL with pgvector extension
# - Redis cluster
# - Prometheus
# - Grafana
# - Jaeger

4. Database Setup
# Run database migrations
make migrate-up

# Seed initial data (optional)
make seed-data

5. Discord Bot Setup
Create Discord Application:

Go to Discord Developer Portal
Create new application
Go to "Bot" section and create bot
Copy bot token to .env
Set Bot Permissions:

Required permissions:
- Read Messages/View Channels
- Send Messages
- Use Slash Commands
- Connect (Voice)
- Speak (Voice)
- Use Voice Activity

Invite Bot to Server:

https://discord.com/api/oauth2/authorize?client_id=YOUR_CLIENT_ID&permissions=3145728&scope=bot%20applications.commands

6. Build and Run
# Development mode (with hot reload)
make dev

# Or build and run manually
make build
./bin/bot

Development Commands
# Development
make dev              # Start development environment
make dev-voice        # Start only voice processing service
make dev-rag          # Start only RAG indexing service

# Building
make build            # Build all services
make build-bot        # Build main bot service
make build-docker     # Build Docker images

# Testing
make test             # Run unit tests
make test-integration # Run integration tests
make test-load        # Run load tests
make test-coverage    # Generate test coverage report

# Code Quality
make lint             # Run linter
make fmt              # Format code
make vet              # Run go vet
make security-scan    # Run security analysis

# Database
make migrate-up       # Apply migrations
make migrate-down     # Rollback migrations
make migrate-create   # Create new migration
make seed-data        # Seed test data

# Docker
make docker-up        # Start all services
make docker-down      # Stop all services
make docker-logs      # View logs
make docker-clean     # Clean up containers and volumes

# Monitoring
make grafana-setup    # Import Grafana dashboards
make prometheus-config # Update Prometheus configuration

Monitoring Access
Once started, access monitoring tools:

Grafana: http://localhost:3000 (admin/admin)
Prometheus: http://localhost:9090
Jaeger: http://localhost:16686
Bot Health: http://localhost:8080/health
Production Deployment
# Build production images
make docker-build-prod

# Deploy to Kubernetes
kubectl apply -f deployments/k8s/

# Or use Docker Compose
docker-compose -f deployments/docker/docker-compose.prod.yml up -d

📊 Architecture
Microservices Communication
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Discord Bot   │◄──►│  Voice Service  │    │  RAG Service    │
│   (Main)        │    │  (STT/TTS)      │◄──►│  (Embeddings)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 ▼
                    ┌─────────────────────────┐
                    │      Redis Cluster      │
                    │  - Message Queue        │
                    │  - Real-time Cache      │
                    │  - Session Management   │
                    └─────────────────────────┘
                                 │
                                 ▼
                    ┌─────────────────────────┐
                    │    PostgreSQL +        │
                    │    pgvector            │
                    │  - Message Storage      │
                    │  - Vector Embeddings    │
                    └─────────────────────────┘

🧪 Testing
# Run all tests
make test

# Run with coverage
make test-coverage

# Integration tests (requires running infrastructure)
make test-integration

# Load testing
make test-load

🤝 Contributing
Fork the repository
Create feature branch (git checkout -b feature/amazing-feature)
Commit changes (git commit -m 'Add amazing feature')
Push to branch (git push origin feature/amazing-feature)
Open Pull Request
📝 License
This project is licensed under the MIT License - see the LICENSE file for details.

🆘 Support
Documentation: docs/
Issues: GitHub Issues
Discussions: GitHub Discussions
Built with ❤️ for the Discord community