package repository

import (
	"context"
	"fmt"
	"log"
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
	log.Printf("💾 Storing message ID: %d in database", msg.ID)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Upsert guild
		log.Printf("💾 Upserting guild ID: %d", guild.ID)
		if err := tx.Where("id = ?", guild.ID).
			Assign(models.Guild{
				Name:    guild.Name,
				OwnerID: guild.OwnerID,
				IconURL: guild.IconURL,
			}).
			FirstOrCreate(guild).Error; err != nil {
			log.Printf("❌ Failed to upsert guild ID: %d: %v", guild.ID, err)
			return fmt.Errorf("failed to upsert guild: %w", err)
		}

		// Upsert channel
		log.Printf("💾 Upserting channel ID: %d", channel.ID)
		if err := tx.Where("id = ?", channel.ID).
			Assign(models.Channel{
				GuildID: channel.GuildID,
				Name:    channel.Name,
				Type:    channel.Type,
			}).
			FirstOrCreate(channel).Error; err != nil {
			log.Printf("❌ Failed to upsert channel ID: %d: %v", channel.ID, err)
			return fmt.Errorf("failed to upsert channel: %w", err)
		}

		// Upsert user
		log.Printf("💾 Upserting user ID: %d", user.ID)
		if err := tx.Where("id = ?", user.ID).
			Assign(models.User{
				Username:      user.Username,
				Discriminator: user.Discriminator,
				Avatar:        user.Avatar,
			}).
			FirstOrCreate(user).Error; err != nil {
			log.Printf("❌ Failed to upsert user ID: %d: %v", user.ID, err)
			return fmt.Errorf("failed to upsert user: %w", err)
		}

		// Upsert message
		log.Printf("💾 Upserting message ID: %d", msg.ID)
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
			log.Printf("❌ Failed to upsert message ID: %d: %v", msg.ID, err)
			return fmt.Errorf("failed to upsert message: %w", err)
		}

		log.Printf("✅ Successfully stored message ID: %d", msg.ID)
		return nil
	})
}

// StoreEmbedding saves the vector embedding for a message
func (r *MessageRepository) StoreEmbedding(ctx context.Context, messageID int64, embeddingData []float32, modelName string) error {
	if modelName == "" {
		modelName = "text-embedding-3-small"
	}

	// Convert []float32 to comma-separated string for pgvector
	var vectorParts []string
	for _, val := range embeddingData {
		vectorParts = append(vectorParts, fmt.Sprintf("%g", val))
	}
	vectorStr := fmt.Sprintf("[%s]", strings.Join(vectorParts, ","))

	log.Printf("💾 Storing embedding for message ID: %d, vector: %s", messageID, vectorStr[:min(100, len(vectorStr))]+"...")

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
		log.Printf("❌ Failed to store embedding for message ID: %d: %v", messageID, result.Error)
		return fmt.Errorf("failed to store embedding: %w", result.Error)
	}

	log.Printf("✅ Successfully stored embedding for message ID: %d", messageID)
	return nil
}

// SearchSimilarMessages finds messages similar to the query using vector search
func (r *MessageRepository) SearchSimilarMessages(ctx context.Context, queryEmbedding []float32, limit int, similarity float64) ([]models.SearchResult, error) {
	log.Printf("🔍 Performing vector search with limit: %d, similarity threshold: %.2f", limit, similarity)

	// Convert query embedding to vector format
	var vectorParts []string
	for _, val := range queryEmbedding {
		vectorParts = append(vectorParts, fmt.Sprintf("%g", val))
	}
	vectorStr := fmt.Sprintf("[%s]", strings.Join(vectorParts, ","))

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
		log.Printf("❌ Failed to execute vector search query: %v", err)
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
			log.Printf("❌ Failed to scan search result: %v", err)
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		result.Message = msg
		result.User = user
		result.Channel = channel
		results = append(results, result)
	}

	log.Printf("✅ Vector search returned %d results", len(results))
	return results, nil
}

// GetRecentMessages gets recent messages from a channel
func (r *MessageRepository) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]models.SearchResult, error) {
	log.Printf("🔍 Fetching recent messages for channel ID: %d, limit: %d", channelID, limit)

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
		log.Printf("❌ Failed to fetch recent messages: %v", err)
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}

	// Convert to search results
	for _, msg := range messages {
		result := models.SearchResult{
			Message:    msg,
			User:       msg.User,
			Channel:    msg.Channel,
			Similarity: 1.0, // Set similarity to 1.0 for recent messages
		}
		results = append(results, result)
	}

	log.Printf("✅ Fetched %d recent messages", len(results))
	return results, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
