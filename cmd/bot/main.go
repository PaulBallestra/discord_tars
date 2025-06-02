package main

import (
	"fmt"
	"log"

	"discord-tars/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	fmt.Printf("✅ Configuration loaded successfully!\n")
	fmt.Printf("Discord Token: %s\n", maskToken(cfg.Discord.Token))
	fmt.Printf("OpenAI API Key: %s\n", maskToken(cfg.OpenAI.APIKey))
	fmt.Printf("Database URL: %s\n", cfg.Database.GetDatabaseURL())
	fmt.Printf("Redis Address: %s\n", cfg.Redis.GetRedisAddr())
	fmt.Printf("Environment: %s\n", cfg.App.Environment)
}

func maskToken(token string) string {
	if len(token) < 10 {
		return "***"
	}
	return token[:6] + "..." + token[len(token)-4:]
}
