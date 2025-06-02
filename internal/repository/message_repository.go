package repository

import (
	"context"
	"fmt"
	"strings"

	"discord-tars/internal/models"
	"discord-tars/internal/repository/postgres"
)

type MessageRepository struct {
	db *postgres.DB
}

func NewMessageRepository(db *postgres.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// StoreMessage saves a message with its user and channel info
func (r *MessageRepository) StoreMessage(ctx context.Context, msg *models.Message, user *models.User, channel *models.Channel, guild *models.Guild) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Upsert guild
	_, err = tx.ExecContext(ctx, `
        INSERT INTO guilds (id, name) 
        VALUES ($1, $2) 
        ON CONFLICT (id) DO UPDATE SET 
            name = EXCLUDED.name,
            updated_at = NOW()`,
		guild.ID, guild.Name)
	if err != nil {
		return fmt.Errorf("failed to upsert guild: %w", err)
	}

	// Upsert channel
	_, err = tx.ExecContext(ctx, `
        INSERT INTO channels (id, guild_id, name, type) 
        VALUES ($1, $2, $3, $4) 
        ON CONFLICT (id) DO UPDATE SET 
            name = EXCLUDED.name,
            type = EXCLUDED.type,
            updated_at = NOW()`,
		channel.ID, channel.GuildID, channel.Name, channel.Type)
	if err != nil {
		return fmt.Errorf("failed to upsert channel: %w", err)
	}

	// Upsert user
	_, err = tx.ExecContext(ctx, `
        INSERT INTO users (id, username, discriminator, avatar) 
        VALUES ($1, $2, $3, $4) 
        ON CONFLICT (id) DO UPDATE SET 
            username = EXCLUDED.username,
            discriminator = EXCLUDED.discriminator,
            avatar = EXCLUDED.avatar,
            updated_at = NOW()`,
		user.ID, user.Username, user.Discriminator, user.Avatar)
	if err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	// Insert message
	_, err = tx.ExecContext(ctx, `
        INSERT INTO messages (id, channel_id, user_id, content, embeds, attachments, timestamp) 
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (id) DO UPDATE SET 
            content = EXCLUDED.content,
            embeds = EXCLUDED.embeds,
            attachments = EXCLUDED.attachments,
            updated_at = NOW()`,
		msg.ID, msg.ChannelID, msg.UserID, msg.Content, msg.Embeds, msg.Attachments, msg.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return tx.Commit()
}

// StoreEmbedding saves the vector embedding for a message
func (r *MessageRepository) StoreEmbedding(ctx context.Context, messageID int64, embedding []float32, chunkIndex int) error {
	// Convert []float32 to PostgreSQL vector format
	vectorStr := fmt.Sprintf("[%s]", strings.Trim(fmt.Sprintf("%v", embedding), "[]"))

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO message_embeddings (message_id, embedding, chunk_index) 
        VALUES ($1, $2::vector, $3)`,
		messageID, vectorStr, chunkIndex)
	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	return nil
}

// SearchSimilarMessages finds messages similar to the query using vector search
func (r *MessageRepository) SearchSimilarMessages(ctx context.Context, queryEmbedding []float32, limit int, similarity float64) ([]models.SearchResult, error) {
	// Convert query embedding to vector format
	vectorStr := fmt.Sprintf("[%s]", strings.Trim(fmt.Sprintf("%v", queryEmbedding), "[]"))

	query := `
        SELECT 
            m.id, m.channel_id, m.user_id, m.content, m.timestamp,
            u.username, u.discriminator, u.avatar,
            c.name as channel_name, c.type as channel_type,
            1 - (me.embedding <=> $1::vector) as similarity
        FROM message_embeddings me
        JOIN messages m ON me.message_id = m.id
        JOIN users u ON m.user_id = u.id
        JOIN channels c ON m.channel_id = c.id
        WHERE 1 - (me.embedding <=> $1::vector) > $2
        ORDER BY me.embedding <=> $1::vector
        LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, vectorStr, similarity, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar messages: %w", err)
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var result models.SearchResult
		err := rows.Scan(
			&result.Message.ID, &result.Message.ChannelID, &result.Message.UserID,
			&result.Message.Content, &result.Message.Timestamp,
			&result.User.Username, &result.User.Discriminator, &result.User.Avatar,
			&result.Channel.Name, &result.Channel.Type,
			&result.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// GetRecentMessages gets recent messages from a channel
func (r *MessageRepository) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]models.SearchResult, error) {
	query := `
        SELECT 
            m.id, m.channel_id, m.user_id, m.content, m.timestamp,
            u.username, u.discriminator, u.avatar,
            c.name as channel_name, c.type as channel_type
        FROM messages m
        JOIN users u ON m.user_id = u.id
        JOIN channels c ON m.channel_id = c.id
        WHERE m.channel_id = $1
        ORDER BY m.timestamp DESC
        LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var result models.SearchResult
		err := rows.Scan(
			&result.Message.ID, &result.Message.ChannelID, &result.Message.UserID,
			&result.Message.Content, &result.Message.Timestamp,
			&result.User.Username, &result.User.Discriminator, &result.User.Avatar,
			&result.Channel.Name, &result.Channel.Type,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		// Set similarity to 1.0 for recent messages (exact match)
		result.Similarity = 1.0
		results = append(results, result)
	}

	return results, nil
}
