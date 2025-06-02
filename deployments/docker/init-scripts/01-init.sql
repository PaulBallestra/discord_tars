-- Create pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create basic tables
CREATE TABLE IF NOT EXISTS guilds (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS channels (
    id BIGINT PRIMARY KEY,
    guild_id BIGINT REFERENCES guilds(id),
    name VARCHAR(255) NOT NULL,
    type INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    discriminator VARCHAR(10),
    avatar VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messages (
    id BIGINT PRIMARY KEY,
    channel_id BIGINT REFERENCES channels(id),
    user_id BIGINT REFERENCES users(id),
    content TEXT,
    embeds JSONB,
    attachments JSONB,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS message_embeddings (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT REFERENCES messages(id),
    embedding vector(1536), -- OpenAI text-embedding-3-large dimension
    chunk_index INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_messages_channel_timestamp ON messages(channel_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_messages_user_timestamp ON messages(user_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_message_embeddings_vector ON message_embeddings USING ivfflat (embedding vector_cosine_ops);

-- Add updated_at triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_guilds_updated_at BEFORE UPDATE ON guilds FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_channels_updated_at BEFORE UPDATE ON channels FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_messages_updated_at BEFORE UPDATE ON messages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
