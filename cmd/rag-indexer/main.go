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

	fmt.Printf("✅ RAG Indexer started successfully!\n")
	fmt.Printf("Environment: %s\n", cfg.App.Environment)
	fmt.Printf("OpenAI API Key: %s\n", maskToken(cfg.OpenAI.APIKey))
	fmt.Printf("Database URL: %s\n", cfg.Database.GetDatabaseURL())
	fmt.Printf("HTTP Port: %d\n", cfg.App.HTTPPort)
	fmt.Printf("gRPC Port: %d\n", cfg.App.GRPCPort)

	// TODO: Implement RAG indexer service
	fmt.Println("RAG Indexer is ready to receive requests.")
}

func maskToken(token string) string {
	if len(token) < 10 {
		return "***"
	}
	return token[:6] + "..." + token[len(token)-4:]
}
