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
	voiceService "discord-tars/internal/services/voice"
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

	// Verify pgvector extension
	var extensionExists bool
	err = db.Raw("SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&extensionExists).Error
	if err != nil || !extensionExists {
		log.Fatalf("❌ pgvector extension not enabled: %v", err)
	}
	log.Println("✅ pgvector extension verified")

	// Initialize repositories
	msgRepo := repository.NewMessageRepository(db)

	// Initialize AI service
	aiSvc := openaiService.NewService(openaiService.Config{
		APIKey: cfg.OpenAI.APIKey,
		Model:  cfg.OpenAI.Model,
	})

	// Initialize voice service
	voiceSvc := voiceService.NewService(voiceService.Config{
		OpenAIAPIKey: cfg.OpenAI.APIKey,
		TTSModel:     cfg.OpenAI.TTSModel,
	})

	// Initialize Discord bot
	bot, err := discordService.NewBot(discordService.BotConfig{
		Token:   cfg.Discord.Token,
		GuildID: cfg.Discord.GuildID,
	}, aiSvc, nil, voiceSvc)
	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v", err)
	}

	// Initialize RAG service with bot session
	ragSvc := ragService.NewService(aiSvc, msgRepo, bot.GetSession())
	bot.SetRAGService(ragSvc)

	// Start bot
	if err := bot.Start(); err != nil {
		log.Fatalf("❌ Failed to start bot: %v", err)
	}
	defer bot.Stop()

	log.Println("🤖 T.A.R.S is now online with RAG and voice capabilities!")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("👋 Shutdown complete")
}
