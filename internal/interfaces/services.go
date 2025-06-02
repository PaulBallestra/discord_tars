package interfaces

import (
	"context"
	"discord-tars/internal/config"
)

// AIService defines the interface for AI-powered responses
type AIService interface {
	GenerateResponse(ctx context.Context, userMessage, username string) (string, error)
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	SetPersonality(humor, honesty int)
}

// DiscordService defines the interface for Discord operations
type DiscordService interface {
	SendMessage(channelID, content string) error
	SendTyping(channelID string) error
	UpdateStatus(activity string) error
}

// ConfigService defines the interface for configuration management
type ConfigService interface {
	GetDiscordConfig() config.DiscordConfig
	GetOpenAIConfig() config.OpenAIConfig
	GetDatabaseConfig() config.DatabaseConfig
	Validate() error
}
