package main

import (
	"context"
	"log"
	"time"

	"discord-tars/internal/config"
	"discord-tars/internal/models"
	"discord-tars/internal/repository"
	"discord-tars/internal/repository/postgres"
	"discord-tars/internal/services/openai"
)

func main() {
	log.Println("🌱 Seeding database with test data...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := postgres.NewGormConnection(cfg.Database)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize AI service for embeddings
	aiSvc := openai.NewService(openai.Config{
		APIKey: cfg.OpenAI.APIKey,
		Model:  cfg.OpenAI.Model,
	})

	// Initialize message repository
	msgRepo := repository.NewMessageRepository(db)

	// Sample data
	guild := &models.Guild{
		ID:        1,
		Name:      "Test Guild",
		OwnerID:   1,
		CreatedAt: time.Now(),
	}

	channel := &models.Channel{
		ID:        1,
		GuildID:   1,
		Name:      "general",
		Type:      0,
		CreatedAt: time.Now(),
	}

	user := &models.User{
		ID:            1,
		Username:      "TestUser",
		Discriminator: "0001",
		Avatar:        "",
		CreatedAt:     time.Now(),
	}

	messages := []models.Message{
		{
			ID:        1,
			ChannelID: 1,
			UserID:    1,
			GuildID:   1,
			Content:   "Hello, this is a test message about coding in Go.",
			Timestamp: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        2,
			ChannelID: 1,
			UserID:    1,
			GuildID:   1,
			Content:   "I love programming with PostgreSQL and pgvector!",
			Timestamp: time.Now().Add(-30 * time.Minute),
			CreatedAt: time.Now(),
		},
		{
			ID:        3,
			ChannelID: 1,
			UserID:    1,
			GuildID:   1,
			Content:   "Does anyone know how to use OpenAI embeddings?",
			Timestamp: time.Now().Add(-10 * time.Minute),
			CreatedAt: time.Now(),
		},
	}

	// Store messages and embeddings
	ctx := context.Background()
	for _, msg := range messages {
		log.Printf("💾 Seeding message ID: %d", msg.ID)
		if err := msgRepo.StoreMessage(ctx, &msg, user, channel, guild); err != nil {
			log.Printf("❌ Failed to store message ID: %d: %v", msg.ID, err)
			continue
		}

		log.Printf("🧠 Generating embedding for message ID: %d", msg.ID)
		embedding, err := aiSvc.GenerateEmbedding(ctx, msg.Content)
		if err != nil {
			log.Printf("⚠️ Failed to generate embedding for message ID: %d: %v", msg.ID, err)
			continue
		}

		log.Printf("💾 Storing embedding for message ID: %d", msg.ID)
		if err := msgRepo.StoreEmbedding(ctx, msg.ID, embedding, "text-embedding-3-small"); err != nil {
			log.Printf("❌ Failed to store embedding for message ID: %d: %v", msg.ID, err)
			continue
		}
	}

	log.Println("✅ Database seeding completed")
}
