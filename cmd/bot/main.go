package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"discord-tars/internal/config"
	discordService "discord-tars/internal/services/discord"
	openaiService "discord-tars/internal/services/openai"
)

func main() {
	log.Println("🚀 Starting Discord T.A.R.S...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Initialize services
	aiSvc := openaiService.NewService(openaiService.Config{
		APIKey: cfg.OpenAI.APIKey,
		Model:  cfg.OpenAI.Model,
	})

	// Initialize Discord bot
	bot, err := discordService.NewBot(discordService.BotConfig{
		Token:   cfg.Discord.Token,
		GuildID: cfg.Discord.GuildID,
	}, aiSvc)
	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v", err)
	}

	// Start bot
	if err := bot.Start(); err != nil {
		log.Fatalf("❌ Failed to start bot: %v", err)
	}
	defer bot.Stop()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("👋 Shutdown complete")
}
