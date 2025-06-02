package repository

import (
	"context"
	"fmt"
	"strings"

	"discord-tars/internal/models"
	"discord-tars/internal/repository/postgres"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *postgres.GormDB
}

func NewMessageRepository(db *postgres.GormDB) *MessageRepository {
	return &MessageRepository{db: db}
}

// StoreMessage saves a message with its user and channel info
func (r *MessageRepository) StoreMessage(ctx context.Context, msg *models.Message, user *models.User, channel *models.Channel, guild *models.Guild) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Upsert guild
		if err := tx.Where("id = ?", guild.ID).
			Assign(models.Guild{Name: guild.Name}).
			FirstOrCreate(guild).Error; err != nil {
			return fmt.Errorf("failed to upsert guild: %w", err)
		}

		// Upsert channel
		if err := tx.Where("id = ?", channel.ID).
			Assign(models.Channel{
				GuildID: channel.GuildID,
				Name:    channel.Name,
				Type:    channel.Type,
			}).
			FirstOrCreate(channel).Error; err != nil {
			return fmt.Errorf("failed to upsert channel: %w", err)
		}

		// Upsert user
		if err := tx.Where("id = ?", user.ID).
			Assign(models.User{
				Username:      user.Username,
				Discriminator: user.Discriminator,
				Avatar:        user.Avatar,
			}).
			FirstOrCreate(user).Error; err != nil {
			return fmt.Errorf("failed to upsert user: %w", err)
		}

		// Upsert message
		if err := tx.Where("id = ?", msg.ID).
			Assign(models.Message{
				ChannelID:   msg.ChannelID,
				UserID:      msg.UserID,
				GuildID:     msg.GuildID,
				Content:     msg.Content,
				Embeds:      msg.Embeds,
				Attachments: msg.Attachments,
				Timestamp:   msg.Timestamp,
			}).
			FirstOrCreate(msg).Error; err != nil {
			return fmt.Errorf("failed to upsert message: %w", err)
		}

		return nil
	})
}

// StoreEmbedding saves the vector embedding for a message
func (r *MessageRepository) StoreEmbedding(ctx context.Context, messageID int64, embeddingData []float32, modelName string) error {
	if modelName == "" {
		modelName = "text-embedding-3-small"
	}

	// Convert []float32 to PostgreSQL vector format
	vectorStr := fmt.Sprintf("[%s]", strings.Trim(fmt.Sprintf("%v", embeddingData), "[]"))

	// Create or update embedding
	embeddingRecord := models.MessageEmbedding{
		MessageID: messageID,
		Embedding: vectorStr,
		ModelName: modelName,
	}

	result := r.db.WithContext(ctx).Where("message_id = ?", messageID).
		Assign(models.MessageEmbedding{
			Embedding: vectorStr,
			ModelName: modelName,
		}).
		FirstOrCreate(&embeddingRecord)

	if result.Error != nil {
		return fmt.Errorf("failed to store embedding: %w", result.Error)
	}

	return nil
}

// SearchSimilarMessages finds messages similar to the query using vector search
func (r *MessageRepository) SearchSimilarMessages(ctx context.Context, queryEmbedding []float32, limit int, similarity float64) ([]models.SearchResult, error) {
	// Convert query embedding to vector format
	vectorStr := fmt.Sprintf("[%s]", strings.Trim(fmt.Sprintf("%v", queryEmbedding), "[]"))

	var results []models.SearchResult

	// Execute raw SQL for vector similarity search
	query := `
		SELECT 
			m.id, m.channel_id, m.user_id, m.guild_id, m.content, m.timestamp,
			u.id as user_id, u.username, u.discriminator, u.avatar_url,
			c.id as channel_id, c.name as channel_name, c.type as channel_type,
			1 - (me.embedding <=> $1::vector) as similarity
		FROM message_embeddings me
		JOIN messages m ON me.message_id = m.id
		JOIN users u ON m.user_id = u.id
		JOIN channels c ON m.channel_id = c.id
		WHERE 1 - (me.embedding <=> $1::vector) > $2
		ORDER BY me.embedding <=> $1::vector
		LIMIT $3
	`

	rows, err := r.db.Raw(query, vectorStr, similarity, limit).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to search similar messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var result models.SearchResult
		var msg models.Message
		var user models.User
		var channel models.Channel

		err := rows.Scan(
			&msg.ID, &msg.ChannelID, &msg.UserID, &msg.GuildID, &msg.Content, &msg.Timestamp,
			&user.ID, &user.Username, &user.Discriminator, &user.Avatar,
			&channel.ID, &channel.Name, &channel.Type,
			&result.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		result.Message = msg
		result.User = user
		result.Channel = channel
		results = append(results, result)
	}

	return results, nil
}

// GetRecentMessages gets recent messages from a channel
func (r *MessageRepository) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]models.SearchResult, error) {
	var messages []models.Message
	var results []models.SearchResult

	// Get messages with preloaded relations
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Channel").
		Where("channel_id = ?", channelID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&messages).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}

	// Convert to search results
	for _, msg := range messages {
		result := models.SearchResult{
			Message:    msg,
			User:       msg.User,
			Channel:    msg.Channel,
			Similarity: 1.0, // Set similarity to 1.0 for recent messages (exact match)
		}
		results = append(results, result)
	}

	return results, nil
}
