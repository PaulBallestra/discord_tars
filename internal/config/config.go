package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Discord    DiscordConfig
	OpenAI     OpenAIConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	App        AppConfig
	Monitoring MonitoringConfig
}

type DiscordConfig struct {
	Token   string
	GuildID string
}

type OpenAIConfig struct {
	APIKey         string
	Model          string
	EmbeddingModel string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type AppConfig struct {
	Environment string
	LogLevel    string
	HTTPPort    int
	GRPCPort    int
}

type MonitoringConfig struct {
	PrometheusPort int
	GrafanaPort    int
	JaegerEndpoint string
}

func LoadConfig() (*Config, error) {
	// Load .env file
	_ = godotenv.Load() // Don't fail if .env doesn't exist

	config := &Config{
		Discord: DiscordConfig{
			Token:   os.Getenv("DISCORD_TOKEN"),
			GuildID: os.Getenv("DISCORD_GUILD_ID"),
		},
		OpenAI: OpenAIConfig{
			APIKey:         os.Getenv("OPENAI_API_KEY"),
			Model:          getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
			EmbeddingModel: getEnvOrDefault("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		},
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
			Port:     getEnvIntOrDefault("POSTGRES_PORT", 5432),
			User:     getEnvOrDefault("POSTGRES_USER", "ragbot"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			DBName:   getEnvOrDefault("POSTGRES_DB", "tars_db"),
			SSLMode:  getEnvOrDefault("POSTGRES_SSL_MODE", "disable"),
		},
		App: AppConfig{
			Environment: getEnvOrDefault("ENVIRONMENT", "development"),
			LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
			HTTPPort:    getEnvIntOrDefault("HTTP_PORT", 8080),
			GRPCPort:    getEnvIntOrDefault("GRPC_PORT", 8081),
		},
	}

	return config, config.validate()
}

func (c *Config) validate() error {
	if c.Discord.Token == "" {
		return fmt.Errorf("DISCORD_TOKEN is required")
	}
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD is required")
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
