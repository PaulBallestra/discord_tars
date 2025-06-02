package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"discord-tars/internal/config"
	"discord-tars/internal/services/openai"

	"github.com/bwmarrin/discordgo"
)

var (
	openaiService *openai.OpenAIService
)

func main() {
	fmt.Println("🚀 Starting Discord RAG Agent...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("❌ Failed to load configuration:", err)
	}

	fmt.Printf("Environment: %s\n", cfg.App.Environment)

	// Initialize OpenAI service
	openaiService = openai.NewOpenAIService(cfg.OpenAI.APIKey, cfg.OpenAI.Model)
	fmt.Println("🧠 OpenAI service initialized")

	// Create Discord session
	dg, err := discordgo.New("Bot " + cfg.Discord.Token)
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
	fmt.Println("📝 Try using /ask command or mentioning the bot!")

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
			Description: "Ask T.A.R.S a question",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "question",
					Description: "Your question for T.A.R.S",
					Required:    true,
				},
			},
		},
		{
			Name:        "help",
			Description: "Show available commands and features",
		},
		{
			Name:        "personality",
			Description: "Set T.A.R.S humor and honesty settings",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "humor",
					Description: "Humor setting (0-100)",
					MinValue:    &[]float64{0}[0],
					MaxValue:    100,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "honesty",
					Description: "Honesty setting (0-100)",
					MinValue:    &[]float64{0}[0],
					MaxValue:    100,
				},
			},
		},
	}

	// Register commands
	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			fmt.Printf("❌ Error creating command %s: %v\n", cmd.Name, err)
		} else {
			fmt.Printf("✅ Registered slash command: /%s\n", cmd.Name)
		}
	}

	s.UpdateGameStatus(0, "🤖 T.A.R.S Online | Humor: 75% | Try /ask")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Log all messages
	fmt.Printf("📨 Message from %s: %s\n", m.Author.Username, m.Content)

	// Check for mentions
	mentionsBot := false
	for _, mention := range m.Mentions {
		if mention.ID == s.State.User.ID {
			mentionsBot = true
			break
		}
	}

	// Respond to mentions with AI
	if mentionsBot {
		// Remove mention from message content
		content := m.Content
		for _, mention := range m.Mentions {
			if mention.ID == s.State.User.ID {
				content = strings.ReplaceAll(content, mention.String(), "")
			}
		}
		content = strings.TrimSpace(content)

		if content == "" {
			content = "Hello! How can I help you?"
		}

		// Show typing indicator
		s.ChannelTyping(m.ChannelID)

		// Get AI response
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := openaiService.GenerateResponse(ctx, content, m.Author.Username)
		if err != nil {
			fmt.Printf("❌ OpenAI error: %v\n", err)
			s.ChannelMessageSend(m.ChannelID, "🔧 My circuits seem to be malfunctioning. Please try again later.")
			return
		}

		s.ChannelMessageSend(m.ChannelID, response)
		return
	}

	// Simple ping command
	if m.Content == "!ping" {
		s.ChannelMessageSend(m.ChannelID, "🏓 Pong! All systems operational.")
		return
	}

	// Respond to greetings
	lowerContent := strings.ToLower(m.Content)
	if lowerContent == "hello" || lowerContent == "hi" || lowerContent == "hey" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("👋 Hello %s! I'm T.A.R.S. Mention me with `@T.A.R.S` followed by your question, or use `/ask` for a proper conversation.", m.Author.Mention()))
	}
}

func slashCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Name {
	case "ping":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🏓 Pong! T.A.R.S systems are operational.\n" +
					"⚡ Response time: Excellent\n" +
					"🧠 AI Status: Ready\n" +
					"💾 Memory banks: Online",
			},
		})

	case "help":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🤖 **T.A.R.S Command Interface**\n\n" +
					"**Slash Commands:**\n" +
					"`/ask <question>` - Ask me anything\n" +
					"`/ping` - Test system status\n" +
					"`/personality` - Adjust my humor/honesty settings\n" +
					"`/help` - Show this interface\n\n" +
					"**Chat Commands:**\n" +
					"• Mention me: `@T.A.R.S your question here`\n" +
					"• Simple greeting: `hello`, `hi`, `hey`\n" +
					"• System check: `!ping`\n\n" +
					"*Currently operating at 75% humor setting. Honesty level: Maximum.*",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})

	case "ask":
		question := i.ApplicationCommandData().Options[0].StringValue()

		// Defer response since OpenAI might take time
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})

		// Get AI response
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := openaiService.GenerateResponse(ctx, question, i.Member.User.Username)
		if err != nil {
			fmt.Printf("❌ OpenAI error: %v\n", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "🔧 Experiencing technical difficulties. Even advanced AI systems have their off days.",
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response,
			},
		})

	case "personality":
		// This is a fun command to showcase T.A.R.S personality
		humor := 75
		honesty := 100

		if len(i.ApplicationCommandData().Options) > 0 {
			for _, option := range i.ApplicationCommandData().Options {
				switch option.Name {
				case "humor":
					humor = int(option.IntValue())
				case "honesty":
					honesty = int(option.IntValue())
				}
			}
		}

		var response string
		if humor == 0 {
			response = "Humor setting disabled. I will now communicate with the efficiency of a technical manual."
		} else if humor > 90 {
			response = fmt.Sprintf("Humor setting at %d%%. Warning: Excessive humor may result in dad jokes and puns. Proceed with caution.", humor)
		} else {
			response = fmt.Sprintf("🤖 Personality matrix updated:\n• Humor: %d%%\n• Honesty: %d%%\n\nThese settings are cosmetic and don't actually affect my responses... or do they? 😏", humor, honesty)
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response,
			},
		})
	}
}
