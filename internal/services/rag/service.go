package rag

import (
	"context"
	"fmt"
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
	// Skip empty messages or bot messages
	if strings.TrimSpace(discordMsg.Content) == "" || discordMsg.Author.Bot {
		return nil
	}

	// Convert Discord message to our models
	userID, _ := strconv.ParseInt(discordMsg.Author.ID, 10, 64)
	channelID, _ := strconv.ParseInt(discordMsg.ChannelID, 10, 64)
	messageID, _ := strconv.ParseInt(discordMsg.ID, 10, 64)
	guildID, _ := strconv.ParseInt(discordMsg.GuildID, 10, 64)

	timestamp, err := time.Parse(time.RFC3339, discordMsg.Timestamp.Format(time.RFC3339))
	if err != nil {
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
		if err == nil && channel != nil {
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
		if err == nil && guild != nil {
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
	if err := s.msgRepo.StoreMessage(ctx, message, user, channel, guild); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// Generate and store embedding for non-empty content
	if len(strings.TrimSpace(discordMsg.Content)) > 10 {
		embedding, err := s.aiService.GenerateEmbedding(ctx, discordMsg.Content)
		if err != nil {
			// Log error but don't fail the entire process
			fmt.Printf("⚠️ Failed to generate embedding for message %s: %v\n", discordMsg.ID, err)
			return nil
		}

		if err := s.msgRepo.StoreEmbedding(ctx, messageID, embedding, "text-embedding-3-small"); err != nil {
			return fmt.Errorf("failed to store embedding: %w", err)
		}

		fmt.Printf("✅ Stored message and embedding for: %s\n", discordMsg.Content[:min(50, len(discordMsg.Content))])
	}

	return nil
}

// SearchContext finds relevant messages for RAG context
func (s *Service) SearchContext(ctx context.Context, query string, channelID int64, maxResults int) ([]models.SearchResult, error) {
	// Generate embedding for the query
	queryEmbedding, err := s.aiService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search for similar messages (similarity threshold of 0.7)
	results, err := s.msgRepo.SearchSimilarMessages(ctx, queryEmbedding, maxResults, 0.7)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar messages: %w", err)
	}

	// If no similar messages found, get some recent messages for context
	if len(results) == 0 {
		results, err = s.msgRepo.GetRecentMessages(ctx, channelID, min(maxResults, 5))
		if err != nil {
			return nil, fmt.Errorf("failed to get recent messages: %w", err)
		}
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
