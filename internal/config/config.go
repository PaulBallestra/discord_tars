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
	DB       string `mapstructure:"db"`
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
}

type MonitoringConfig struct {
	JaegerEndpoint string `mapstructure:"jaeger_endpoint"`
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./")
	viper.AddConfigPath("../")

	// Environment variables
	viper.SetEnvPrefix("")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults() {
	// Discord defaults
	viper.SetDefault("discord.token", "")
	viper.SetDefault("discord.guild_id", "")

	// OpenAI defaults
	viper.SetDefault("openai.api_key", "")
	viper.SetDefault("openai.model", "gpt-4")
	viper.SetDefault("openai.embedding_model", "text-embedding-3-large")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "ragbot")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.db", "discord_rag")
	viper.SetDefault("database.ssl_mode", "disable")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// App defaults
	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("app.http_port", 8080)
	viper.SetDefault("app.grpc_port", 8081)
	viper.SetDefault("app.metrics_port", 9090)

	// Monitoring defaults
	viper.SetDefault("monitoring.jaeger_endpoint", "http://localhost:14268/api/traces")
}

func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DB,
		c.Database.SSLMode,
	)
}

func (c *Config) GetRedisURL() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}
