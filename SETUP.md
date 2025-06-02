# Discord RAG Bot Setup Guide

This guide will help you set up the Discord RAG bot with improved GORM integration for database operations.

## Prerequisites

- Go 1.24.3 or later
- PostgreSQL 15+ with pgvector extension
- Discord Bot Token
- OpenAI API Key

## Environment Setup

1. Create a `.env` file in the root directory with the following variables:

```
# Discord Configuration
DISCORD_TOKEN=your_discord_bot_token
DISCORD_GUILD_ID=your_test_guild_id

# OpenAI Configuration
OPENAI_API_KEY=your_openai_api_key
OPENAI_MODEL=gpt-4o
OPENAI_EMBEDDING_MODEL=text-embedding-3-small

# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=ragbot
POSTGRES_PASSWORD=secure_password
POSTGRES_DB=tars_db
POSTGRES_SSL_MODE=disable

# Application Configuration
LOG_LEVEL=info
ENVIRONMENT=development
```

2. Install PostgreSQL and pgvector:

For macOS:
```bash
brew install postgresql@15
brew install pgvector
```

For Ubuntu:
```bash
sudo apt-get install postgresql-15
sudo apt-get install postgresql-15-pgvector
```

3. Create and set up the PostgreSQL database:

```bash
# Create database
createdb tars_db

# Enable pgvector extension
psql -d tars_db -c "CREATE EXTENSION IF NOT EXISTS vector;"

# Create user (optional)
psql -d tars_db -c "CREATE USER ragbot WITH PASSWORD 'secure_password';"
psql -d tars_db -c "GRANT ALL PRIVILEGES ON DATABASE tars_db TO ragbot;"
```

## Running the Bot

1. Install dependencies:

```bash
go mod tidy
```

2. Run the bot:

```bash
cd cmd/bot
go run main.go
```

## Testing the RAG Functionality

1. Invite the bot to your Discord server
2. Send some messages in channels where the bot has access
3. The bot will automatically process and store these messages with embeddings
4. Try mentioning the bot with a question related to previous conversations
5. The bot should respond with contextually relevant information

## Monitoring the Database

Check if messages and embeddings are being stored:

```bash
# Connect to database
psql -d tars_db

# Check message count
SELECT COUNT(*) FROM messages;

# Check embedding count
SELECT COUNT(*) FROM message_embeddings;

# View channel data
SELECT id, name, guild_id FROM channels;
```

## Troubleshooting

If the bot isn't storing messages or retrieving context properly:

1. Check logs for any errors
2. Verify database connection settings
3. Ensure the bot has proper permissions in Discord
4. Make sure the OpenAI API key is valid and has access to embedding models
5. Confirm pgvector extension is properly installed

## Recent Improvements

- Migrated from raw SQL to GORM for better database operations
- Added proper Discord session sharing with the RAG service
- Improved message processing to capture guild and channel information
- Enhanced embedding storage to match vector database requirements
- Updated models to support GORM's auto-migration feature

## Common Issues and Solutions

### GORM Migration Constraint Error

If you encounter the following error when starting the bot:

```
ERROR: constraint "uni_message_embeddings_message_id" of relation "message_embeddings" does not exist
```

This happens because GORM is trying to modify constraints on a table that was created with a different schema. The fix is already implemented in our code by disabling foreign key constraints during migration:

```go
gormConfig := &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info),
    // Disable foreign key constraints when migrating 
    DisableForeignKeyConstraintWhenMigrating: true,
}
```

If you're using a different version of the code, you can add this configuration when initializing GORM.

### Database Tables Not Being Created

If the database tables aren't being created, make sure:

1. You have the proper permissions to create tables in the database
2. The database exists and is accessible
3. The pgvector extension is installed and enabled
4. The GORM models match your expected schema

You can manually create the required tables using the SQL script in `deployments/docker/init-scripts/01-schema.sql`. 