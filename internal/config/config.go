package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Discord    DiscordConfig    `mapstructure:"discord"`
	OpenAI     OpenAIConfig     `mapstructure:"openai"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	App        AppConfig        `mapstructure:"app"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

type DiscordConfig struct {
	Token   string `mapstructure:"token"`
	GuildID string `mapstructure:"guild_id"`
}

type OpenAIConfig struct {
	APIKey         string `mapstructure:"api_key"`
	Model          string `mapstructure:"model"`
	EmbeddingModel string `mapstructure:"embedding_model"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AppConfig struct {
	LogLevel    string `mapstructure:"log_level"`
	HTTPPort    int    `mapstructure:"http_port"`
	GRPCPort    int    `mapstructure:"grpc_port"`
	MetricsPort int    `mapstructure:"metrics_port"`
	Environment string `mapstructure:"environment"`
}

type MonitoringConfig struct {
	JaegerEndpoint string `mapstructure:"jaeger_endpoint"`
}

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() (*Config, error) {
	// Set default values
	viper.SetDefault("discord.token", "test_discord_token_value")
	viper.SetDefault("discord.guild_id", "123456789012345678")
	viper.SetDefault("openai.api_key", "test_openai_api_key_value")
	viper.SetDefault("openai.model", "gpt-4")
	viper.SetDefault("openai.embedding_model", "text-embedding-3-large")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "ragbot")
	viper.SetDefault("database.password", "secure_password")
	viper.SetDefault("database.database", "discord_rag")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("app.http_port", 8080)
	viper.SetDefault("app.grpc_port", 8081)
	viper.SetDefault("app.metrics_port", 9090)
	viper.SetDefault("app.environment", "development")
	viper.SetDefault("monitoring.jaeger_endpoint", "http://localhost:14268/api/traces")

	// Configure viper
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./")

	// Enable reading from environment variables
	viper.AutomaticEnv()

	// Replace dots and dashes in env var names
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Map environment variables to config structure
	viper.BindEnv("discord.token", "DISCORD_TOKEN")
	viper.BindEnv("discord.guild_id", "DISCORD_GUILD_ID")
	viper.BindEnv("openai.api_key", "OPENAI_API_KEY")
	viper.BindEnv("openai.model", "OPENAI_MODEL")
	viper.BindEnv("openai.embedding_model", "OPENAI_EMBEDDING_MODEL")
	viper.BindEnv("database.host", "POSTGRES_HOST")
	viper.BindEnv("database.port", "POSTGRES_PORT")
	viper.BindEnv("database.user", "POSTGRES_USER")
	viper.BindEnv("database.password", "POSTGRES_PASSWORD")
	viper.BindEnv("database.database", "POSTGRES_DB")
	viper.BindEnv("database.ssl_mode", "POSTGRES_SSL_MODE")
	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.db", "REDIS_DB")
	viper.BindEnv("app.log_level", "LOG_LEVEL")
	viper.BindEnv("app.http_port", "HTTP_PORT")
	viper.BindEnv("app.grpc_port", "GRPC_PORT")
	viper.BindEnv("app.metrics_port", "METRICS_PORT")
	viper.BindEnv("app.environment", "ENVIRONMENT")
	viper.BindEnv("monitoring.jaeger_endpoint", "JAEGER_ENDPOINT")

	// Try to read .env file (it's okay if it doesn't exist)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validateConfig validates that required configuration values are set
func validateConfig(config *Config) error {
	if config.Discord.Token == "" {
		return fmt.Errorf("DISCORD_TOKEN is required")
	}

	if config.Discord.Token == "your_discord_bot_token" {
		return fmt.Errorf("please replace the placeholder DISCORD_TOKEN with your actual Discord bot token")
	}

	if config.OpenAI.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	if config.OpenAI.APIKey == "your_openai_api_key" {
		return fmt.Errorf("please replace the placeholder OPENAI_API_KEY with your actual OpenAI API key")
	}

	if config.Database.Password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD is required")
	}

	return nil
}

// GetDatabaseURL returns a formatted database connection URL
func (c *DatabaseConfig) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

// GetRedisAddr returns formatted Redis address
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsProduction returns true if running in production environment
func (c *AppConfig) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// IsDevelopment returns true if running in development environment
func (c *AppConfig) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}
