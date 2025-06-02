package rag

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"discord-tars/internal/interfaces"
	"discord-tars/internal/models"
	"discord-tars/internal/repository"
)

type Service struct {
	aiService interfaces.AIService
	msgRepo   *repository.MessageRepository
	session   *discordgo.Session
}

func NewService(aiService interfaces.AIService, msgRepo *repository.MessageRepository, session *discordgo.Session) *Service {
	return &Service{
		aiService: aiService,
		msgRepo:   msgRepo,
		session:   session,
	}
}

// ProcessMessage stores a message and generates embeddings
func (s *Service) ProcessMessage(ctx context.Context, discordMsg *discordgo.Message) error {
	// Log message receipt
	log.Printf("📨 Processing message ID: %s from user: %s", discordMsg.ID, discordMsg.Author.Username)

	// Skip bot messages, but allow short messages
	if discordMsg.Author.Bot {
		log.Printf("ℹ️ Skipping bot message ID: %s", discordMsg.ID)
		return nil
	}

	// Convert Discord message to our models
	userID, err := strconv.ParseInt(discordMsg.Author.ID, 10, 64)
	if err != nil {
		log.Printf("❌ Failed to parse user ID: %v", err)
		return fmt.Errorf("failed to parse user ID: %w", err)
	}

	channelID, err := strconv.ParseInt(discordMsg.ChannelID, 10, 64)
	if err != nil {
		log.Printf("❌ Failed to parse channel ID: %v", err)
		return fmt.Errorf("failed to parse channel ID: %w", err)
	}

	messageID, err := strconv.ParseInt(discordMsg.ID, 10, 64)
	if err != nil {
		log.Printf("❌ Failed to parse message ID: %v", err)
		return fmt.Errorf("failed to parse message ID: %w", err)
	}

	guildID, err := strconv.ParseInt(discordMsg.GuildID, 10, 64)
	if err != nil && discordMsg.GuildID != "" {
		log.Printf("❌ Failed to parse guild ID: %v", err)
		return fmt.Errorf("failed to parse guild ID: %w", err)
	}

	timestamp, err := time.Parse(time.RFC3339, discordMsg.Timestamp.Format(time.RFC3339))
	if err != nil {
		log.Printf("⚠️ Failed to parse timestamp, using current time: %v", err)
		timestamp = time.Now()
	}

	user := &models.User{
		ID:            userID,
		Username:      discordMsg.Author.Username,
		Discriminator: discordMsg.Author.Discriminator,
		Avatar:        discordMsg.Author.Avatar,
	}

	// Get channel information from Discord API
	channelName := "unknown"
	channelType := 0

	if s.session != nil {
		channel, err := s.session.Channel(discordMsg.ChannelID)
		if err != nil {
			log.Printf("⚠️ Failed to fetch channel info: %v", err)
		} else if channel != nil {
			channelName = channel.Name
			channelType = int(channel.Type)
		}
	}

	channel := &models.Channel{
		ID:      channelID,
		GuildID: guildID,
		Name:    channelName,
		Type:    channelType,
	}

	// Get guild information from Discord API
	guildName := "unknown"

	if discordMsg.GuildID != "" && s.session != nil {
		guild, err := s.session.Guild(discordMsg.GuildID)
		if err != nil {
			log.Printf("⚠️ Failed to fetch guild info: %v", err)
		} else if guild != nil {
			guildName = guild.Name
		}
	}

	guild := &models.Guild{
		ID:   guildID,
		Name: guildName,
	}

	message := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		UserID:    userID,
		GuildID:   guildID,
		Content:   discordMsg.Content,
		Timestamp: timestamp,
	}

	// Store message
	log.Printf("💾 Storing message ID: %s", discordMsg.ID)
	if err := s.msgRepo.StoreMessage(ctx, message, user, channel, guild); err != nil {
		log.Printf("❌ Failed to store message ID: %s: %v", discordMsg.ID, err)
		return fmt.Errorf("failed to store message: %w", err)
	}

	// Generate and store embedding for non-empty content
	if strings.TrimSpace(discordMsg.Content) != "" {
		log.Printf("🧠 Generating embedding for message ID: %s", discordMsg.ID)
		embedding, err := s.aiService.GenerateEmbedding(ctx, discordMsg.Content)
		if err != nil {
			log.Printf("⚠️ Failed to generate embedding for message ID: %s: %v", discordMsg.ID, err)
			// Continue without embedding to avoid blocking message storage
			return nil
		}

		log.Printf("💾 Storing embedding for message ID: %s", discordMsg.ID)
		if err := s.msgRepo.StoreEmbedding(ctx, messageID, embedding, "text-embedding-3-small"); err != nil {
			log.Printf("❌ Failed to store embedding for message ID: %s: %v", discordMsg.ID, err)
			return fmt.Errorf("failed to store embedding: %w", err)
		}

		log.Printf("✅ Successfully stored message and embedding for ID: %s, content: %s",
			discordMsg.ID, discordMsg.Content[:min(50, len(discordMsg.Content))])
	} else {
		log.Printf("ℹ️ Skipping embedding for empty message ID: %s", discordMsg.ID)
	}

	return nil
}

// SearchContext finds relevant messages for RAG context
func (s *Service) SearchContext(ctx context.Context, query string, channelID int64, maxResults int) ([]models.SearchResult, error) {
	log.Printf("🔍 Searching context for query: %s", query[:min(50, len(query))])

	// Generate embedding for the query
	queryEmbedding, err := s.aiService.GenerateEmbedding(ctx, query)
	if err != nil {
		log.Printf("❌ Failed to generate query embedding: %v", err)
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search for similar messages
	results, err := s.msgRepo.SearchSimilarMessages(ctx, queryEmbedding, maxResults, 0.7)
	if err != nil {
		log.Printf("❌ Failed to search similar messages: %v", err)
		return nil, fmt.Errorf("failed to search similar messages: %w", err)
	}

	log.Printf("📊 Found %d similar messages", len(results))

	// If no similar messages found, get recent messages
	if len(results) == 0 {
		log.Printf("ℹ️ No similar messages found, fetching recent messages for channel ID: %d", channelID)
		results, err = s.msgRepo.GetRecentMessages(ctx, channelID, min(maxResults, 5))
		if err != nil {
			log.Printf("❌ Failed to get recent messages: %v", err)
			return nil, fmt.Errorf("failed to get recent messages: %w", err)
		}
		log.Printf("📊 Found %d recent messages", len(results))
	}

	return results, nil
}

// BuildRAGPrompt creates a prompt with relevant context
func (s *Service) BuildRAGPrompt(userQuery string, context []models.SearchResult) string {
	var contextBuilder strings.Builder

	contextBuilder.WriteString("Here is some relevant context from previous conversations:\n\n")

	for _, result := range context {
		contextBuilder.WriteString(fmt.Sprintf("**%s**: %s\n",
			result.User.Username,
			result.Message.Content))

		if result.Similarity < 1.0 {
			contextBuilder.WriteString(fmt.Sprintf("(similarity: %.2f)\n", result.Similarity))
		}
		contextBuilder.WriteString("\n")
	}

	contextBuilder.WriteString(fmt.Sprintf("\nUser's current question: %s", userQuery))

	return contextBuilder.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
