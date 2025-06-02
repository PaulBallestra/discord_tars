-- T.A.R.S Discord Bot Database Schema
-- This script initializes the database with all necessary tables for RAG functionality

-- Enable pgvector extension for embeddings
CREATE EXTENSION IF NOT EXISTS vector;

-- Create guilds table
CREATE TABLE IF NOT EXISTS guilds (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    icon_url VARCHAR(500),
    owner_id BIGINT,
    member_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create channels table  
CREATE TABLE IF NOT EXISTS channels (
    id BIGINT PRIMARY KEY,
    guild_id BIGINT REFERENCES guilds(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type INTEGER NOT NULL DEFAULT 0,
    topic TEXT,
    position INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    discriminator VARCHAR(10),
    display_name VARCHAR(255),
    avatar_url VARCHAR(500),
    bot BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create messages table
CREATE TABLE IF NOT EXISTS messages (
    id BIGINT PRIMARY KEY,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    guild_id BIGINT REFERENCES guilds(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    message_type INTEGER DEFAULT 0,
    reply_to_id BIGINT REFERENCES messages(id) ON DELETE SET NULL,
    edited_at TIMESTAMP WITH TIME ZONE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create message_embeddings table for RAG functionality
CREATE TABLE IF NOT EXISTS message_embeddings (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT REFERENCES messages(id) ON DELETE CASCADE,
    embedding vector(1536), -- OpenAI embeddings are 1536 dimensions
    model_name VARCHAR(100) DEFAULT 'text-embedding-3-small',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uni_message_embeddings_message_id UNIQUE (message_id) -- Explicitly named constraint
);

-- Create conversation_context table for tracking conversations
CREATE TABLE IF NOT EXISTS conversation_context (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    context_data JSONB,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create bot_interactions table for tracking bot responses
CREATE TABLE IF NOT EXISTS bot_interactions (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    command VARCHAR(100),
    query TEXT,
    response TEXT,
    response_time_ms INTEGER,
    tokens_used INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_messages_channel_timestamp ON messages(channel_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_guild_timestamp ON messages(guild_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_user_timestamp ON messages(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_message_embeddings_message_id ON message_embeddings(message_id);
CREATE INDEX IF NOT EXISTS idx_conversation_context_channel ON conversation_context(channel_id);
CREATE INDEX IF NOT EXISTS idx_bot_interactions_channel ON bot_interactions(channel_id);

-- Create vector similarity search index (using cosine distance for embeddings)
CREATE INDEX IF NOT EXISTS idx_message_embeddings_vector ON message_embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Insert some initial data for testing
INSERT INTO guilds (id, name, owner_id) VALUES (0, 'T.A.R.S Test Guild', 0) ON CONFLICT (id) DO NOTHING;
INSERT INTO channels (id, guild_id, name, type) VALUES (0, 0, 'general', 0) ON CONFLICT (id) DO NOTHING;

-- Log successful initialization
DO $$ 
BEGIN
    RAISE NOTICE 'T.A.R.S Database Schema Initialized Successfully!';
    RAISE NOTICE 'Tables created: guilds, channels, users, messages, message_embeddings, conversation_context, bot_interactions';
    RAISE NOTICE 'pgvector extension enabled for RAG functionality';
END $$;
