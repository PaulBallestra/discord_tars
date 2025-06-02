package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"discord-tars/internal/config"
	"discord-tars/internal/repository"
	"discord-tars/internal/repository/postgres"
	discordService "discord-tars/internal/services/discord"
	openaiService "discord-tars/internal/services/openai"
	ragService "discord-tars/internal/services/rag"
)

func main() {
	log.Println("🚀 Starting Discord T.A.R.S...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Initialize GORM database
	db, err := postgres.NewGormConnection(cfg.Database)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("✅ Database connected with GORM")

	// Initialize repositories
	msgRepo := repository.NewMessageRepository(db)

	// Initialize AI service
	aiSvc := openaiService.NewService(openaiService.Config{
		APIKey: cfg.OpenAI.APIKey,
		Model:  cfg.OpenAI.Model,
	})

	// Initialize RAG service without session first (we'll update it after bot creation)
	ragSvc := ragService.NewService(aiSvc, msgRepo, nil)

	// Initialize Discord bot with RAG capability
	bot, err := discordService.NewBot(discordService.BotConfig{
		Token:   cfg.Discord.Token,
		GuildID: cfg.Discord.GuildID,
	}, aiSvc, ragSvc)
	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v", err)
	}

	// Get Discord session from bot and update RAG service
	discordSession := bot.GetSession()
	ragSvc = ragService.NewService(aiSvc, msgRepo, discordSession)
	bot.SetRAGService(ragSvc)

	// Start bot
	if err := bot.Start(); err != nil {
		log.Fatalf("❌ Failed to start bot: %v", err)
	}
	defer bot.Stop()

	log.Println("🤖 T.A.R.S is now online with RAG capabilities!")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("👋 Shutdown complete")
}
