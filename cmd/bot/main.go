package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"discord-tars/internal/config"

	"github.com/bwmarrin/discordgo"
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "Show all available commands and features",
		},
		{
			Name:        "ping",
			Description: "Test bot responsiveness",
		},
		{
			Name:        "status",
			Description: "Show bot status and health",
		},
		{
			Name:        "about",
			Description: "Learn about this AI assistant",
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
			Name:        "summarize",
			Description: "Summarize recent chat messages",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "count",
					Description: "Number of recent messages to summarize (default: 20)",
					Required:    false,
					MinValue:    &[]float64{5}[0],
					MaxValue:    100,
				},
			},
		},
		{
			Name:        "voice",
			Description: "Voice channel commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "action",
					Description: "What to do with voice",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "join",
							Value: "join",
						},
						{
							Name:  "leave",
							Value: "leave",
						},
						{
							Name:  "status",
							Value: "status",
						},
					},
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"help":      handleHelpCommand,
		"ping":      handlePingCommand,
		"status":    handleStatusCommand,
		"about":     handleAboutCommand,
		"ask":       handleAskCommand,
		"summarize": handleSummarizeCommand,
		"voice":     handleVoiceCommand,
	}
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	fmt.Printf("🚀 Starting Discord RAG Agent...\n")
	fmt.Printf("Environment: %s\n", cfg.App.Environment)

	// Validate Discord token
	if cfg.Discord.Token == "" || cfg.Discord.Token == "your_discord_bot_token_here" {
		log.Fatal("❌ DISCORD_TOKEN is required. Please set it in your .env file")
	}

	// Create Discord session
	dg, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	// Register event handlers
	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)
	dg.AddHandler(interactionCreate)

	// Set intents
	dg.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildVoiceStates

	// Open connection
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}
	defer dg.Close()

	// Register slash commands
	fmt.Println("📝 Registering slash commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, cmd := range commands {
		registeredCommand, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Cannot create '%v' command: %v", cmd.Name, err)
		} else {
			registeredCommands[i] = registeredCommand
			fmt.Printf("  ✅ Registered /%s\n", cmd.Name)
		}
	}

	fmt.Println("✅ Bot is now running! Press CTRL-C to exit.")
	fmt.Println("💡 Try these slash commands in Discord:")
	fmt.Println("   /help - Show all commands")
	fmt.Println("   /ping - Test responsiveness")
	fmt.Println("   /ask question:What is AI? - Ask a question")
	fmt.Println("   @mention me for natural conversation!")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	fmt.Println("🛑 Shutting down gracefully...")

	// Clean up commands
	fmt.Println("🧹 Cleaning up slash commands...")
	for _, cmd := range registeredCommands {
		if cmd != nil {
			err := dg.ApplicationCommandDelete(dg.State.User.ID, "", cmd.ID)
			if err != nil {
				log.Printf("Cannot delete '%v' command: %v", cmd.Name, err)
			}
		}
	}
}

// Event handler for when bot is ready
func ready(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Printf("✅ Bot is ready! Logged in as: %s#%s\n", event.User.Username, event.User.Discriminator)

	// Set bot status
	err := s.UpdateGameStatus(0, "🤖 AI Assistant | Use /help")
	if err != nil {
		log.Printf("Error setting status: %v", err)
	}
}

// Event handler for slash command interactions
func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if handler, exists := commandHandlers[i.ApplicationCommandData().Name]; exists {
		handler(s, i)
	}
}

// Event handler for message creation (mentions only)
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Ignore messages from other bots
	if m.Author.Bot {
		return
	}

	content := strings.TrimSpace(m.Content)

	// Only handle mentions (no more prefix commands)
	if strings.Contains(content, "<@"+s.State.User.ID+">") || strings.Contains(content, "<@!"+s.State.User.ID+">") {
		handleMention(s, m, content)
		return
	}
}

// Slash command handlers
func handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 Discord RAG Agent Commands",
		Description: "Your intelligent AI assistant for Discord",
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "📋 Basic Commands",
				Value: "`/help` - Show this help message\n" +
					"`/ping` - Test bot responsiveness\n" +
					"`/status` - Show bot status\n" +
					"`/about` - About this bot",
				Inline: false,
			},
			{
				Name: "🧠 AI Features",
				Value: "`/ask question:<text>` - Ask the AI assistant\n" +
					"`/summarize count:<num>` - Summarize recent chat\n" +
					"`/voice action:<join|leave|status>` - Voice commands",
				Inline: false,
			},
			{
				Name:   "💬 Natural Conversation",
				Value:  "**Mention me** (<@" + s.State.User.ID + ">) to chat naturally!",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Built with Go + DiscordGo + OpenAI",
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to help command: %v", err)
	}
}

func handlePingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🏓 Pong! Bot is responsive and ready.",
		},
	})
	if err != nil {
		log.Printf("Error responding to ping command: %v", err)
	}
}

func handleStatusCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title: "📊 Bot Status",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🟢 Status",
				Value:  "Online and ready",
				Inline: true,
			},
			{
				Name:   "🔧 Version",
				Value:  "Development v0.1.0",
				Inline: true,
			},
			{
				Name:   "🎯 Features",
				Value:  "✅ Slash commands\n✅ Mention responses\n⏳ AI integration",
				Inline: false,
			},
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to status command: %v", err)
	}
}

func handleAboutCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 Discord RAG Agent",
		Description: "An AI-powered Discord bot designed to be your server's intelligent companion!",
		Color:       0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "✅ Current Features",
				Value: "• Modern slash commands\n" +
					"• Mention-based conversations\n" +
					"• Embedded help system\n" +
					"• Health monitoring",
				Inline: false,
			},
			{
				Name: "⏳ Coming Soon",
				Value: "• OpenAI integration\n" +
					"• Voice chat capabilities\n" +
					"• Context awareness\n" +
					"• Document processing",
				Inline: false,
			},
			{
				Name:   "🛠️ Built With",
				Value:  "Go + DiscordGo + OpenAI + PostgreSQL",
				Inline: false,
			},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: s.State.User.AvatarURL(""),
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to about command: %v", err)
	}
}

func handleAskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	question := options[0].StringValue()

	embed := &discordgo.MessageEmbed{
		Title:       "🔮 AI Question",
		Description: fmt.Sprintf("**Your Question:** %s", question),
		Color:       0xff9900,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🚧 Status",
				Value:  "AI integration coming soon! This will be powered by OpenAI for intelligent responses.",
				Inline: false,
			},
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to ask command: %v", err)
	}
}

func handleSummarizeCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	count := 20 // default
	if len(i.ApplicationCommandData().Options) > 0 {
		count = int(i.ApplicationCommandData().Options[0].IntValue())
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📝 Chat Summary",
		Description: fmt.Sprintf("Summarizing last %d messages", count),
		Color:       0x9900ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🚧 Status",
				Value:  "Message summarization coming soon! This will analyze recent chat history and provide intelligent summaries.",
				Inline: false,
			},
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to summarize command: %v", err)
	}
}

func handleVoiceCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action := i.ApplicationCommandData().Options[0].StringValue()

	var response string
	var color int

	switch action {
	case "join":
		response = "🎤 Voice join functionality coming soon! I'll be able to join voice channels for real-time conversations."
		color = 0x00ff00
	case "leave":
		response = "👋 Voice leave functionality coming soon!"
		color = 0xff9900
	case "status":
		response = "📊 Voice Status: Not connected\n🚧 Voice capabilities are in development."
		color = 0x0099ff
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎵 Voice Commands",
		Description: response,
		Color:       color,
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to voice command: %v", err)
	}
}

func handleMention(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	// Remove mentions from content
	cleanContent := content
	cleanContent = strings.ReplaceAll(cleanContent, "<@"+s.State.User.ID+">", "")
	cleanContent = strings.ReplaceAll(cleanContent, "<@!"+s.State.User.ID+">", "")
	cleanContent = strings.TrimSpace(cleanContent)

	if cleanContent == "" {
		embed := &discordgo.MessageEmbed{
			Title:       "👋 Hello there!",
			Description: fmt.Sprintf("Hi %s! How can I help you today?", m.Author.Mention()),
			Color:       0x00ff99,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "💡 Tip",
					Value:  "Use `/help` to see all available commands!",
					Inline: false,
				},
			},
		}

		_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
		if err != nil {
			log.Printf("Error sending mention response: %v", err)
		}
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💬 Natural Conversation",
		Description: fmt.Sprintf("**You said:** %s", cleanContent),
		Color:       0xff6b6b,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🚧 Coming Soon",
				Value:  "AI-powered conversations are in development! Soon I'll be able to have natural conversations powered by OpenAI.",
				Inline: false,
			},
			{
				Name:   "🔧 For Now",
				Value:  "Try the `/ask` command to see what questions will look like!",
				Inline: false,
			},
		},
	}

	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		log.Printf("Error sending mention response: %v", err)
	}
}
