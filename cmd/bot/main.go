package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("🚀 Starting Discord RAG Agent...")

	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found, using environment variables")
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("❌ DISCORD_TOKEN environment variable is required")
	}

	// Create Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("❌ Error creating Discord session:", err)
	}

	// Add handlers
	dg.AddHandler(messageCreate)
	dg.AddHandler(ready)
	dg.AddHandler(slashCommandHandler)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	// Open connection
	fmt.Println("🔌 Connecting to Discord...")
	err = dg.Open()
	if err != nil {
		log.Fatal("❌ Error opening connection:", err)
	}
	defer dg.Close()

	fmt.Println("✅ Bot is running! Press Ctrl+C to stop.")
	fmt.Println("📝 Try typing a message in a channel where the bot has access!")

	// Wait for interrupt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("👋 Shutting down gracefully...")
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Printf("✅ Bot connected as %s#%s\n", event.User.Username, event.User.Discriminator)

	// Register slash commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "Test if the bot is responsive",
		},
		{
			Name:        "ask",
			Description: "Ask the AI assistant a question",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "question",
					Description: "Your question for the AI",
					Required:    true,
				},
			},
		},
		{
			Name:        "help",
			Description: "Show available commands and features",
		},
	}

	// Register commands globally (takes up to 1 hour)
	// For faster testing, register to a specific guild instead
	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			fmt.Printf("Error creating command %s: %v\n", cmd.Name, err)
		} else {
			fmt.Printf("✅ Registered slash command: /%s\n", cmd.Name)
		}
	}

	s.UpdateGameStatus(0, "🤖 T.A.R.S Online | Try /help")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Log all messages
	fmt.Printf("📨 Message from %s: %s\n", m.Author.Username, m.Content)

	// Respond to ping
	if m.Content == "!ping" {
		s.ChannelMessageSend(m.ChannelID, "🏓 Pong!")
		return
	}

	// Respond to mentions
	for _, mention := range m.Mentions {
		if mention.ID == s.State.User.ID {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🤖 Hello %s! I'm T.A.R.S, your AI assistant. Try typing `!ping` to test me!", m.Author.Mention()))
			return
		}
	}

	// Respond to hello/hi messages
	if m.Content == "hello" || m.Content == "hi" || m.Content == "hey" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("👋 Hello there, %s!", m.Author.Mention()))
	}
}

// Add this function
func slashCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Name {
	case "ping":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🏓 Pong! T.A.R.S is operational.",
			},
		})

	case "help":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🤖 **T.A.R.S Commands:**\n" +
					"`/ping` - Test bot responsiveness\n" +
					"`/ask <question>` - Ask me anything\n" +
					"`/help` - Show this help message\n\n" +
					"You can also mention me in messages!",
			},
		})

	case "ask":
		question := i.ApplicationCommandData().Options[0].StringValue()

		// For now, a simple response (we'll add OpenAI later)
		response := fmt.Sprintf("🤔 You asked: \"%s\"\n\nI received your question! OpenAI integration coming soon...", question)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response,
			},
		})
	}
}
