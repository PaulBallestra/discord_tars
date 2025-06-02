package discord

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"discord-tars/internal/interfaces"
	"discord-tars/internal/services/rag"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session    *discordgo.Session
	aiService  interfaces.AIService
	ragService *rag.Service
	config     BotConfig
	commands   []*discordgo.ApplicationCommand
}

type BotConfig struct {
	Token   string
	GuildID string
}

func NewBot(config BotConfig, aiService interfaces.AIService, ragService *rag.Service) (*Bot, error) {
	session, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	bot := &Bot{
		session:    session,
		aiService:  aiService,
		ragService: ragService,
		config:     config,
		commands:   make([]*discordgo.ApplicationCommand, 0),
	}

	bot.setupHandlers()
	bot.setupIntents()

	return bot, nil
}

func (b *Bot) setupHandlers() {
	b.session.AddHandler(b.onReady)
	b.session.AddHandler(b.onMessageCreate)
	b.session.AddHandler(b.onSlashCommand)
}

func (b *Bot) setupIntents() {
	b.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
}

func (b *Bot) Start() error {
	fmt.Println("🔌 Connecting to Discord...")
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open discord connection: %w", err)
	}

	fmt.Println("✅ Bot is running! Press Ctrl+C to stop.")
	return nil
}

func (b *Bot) Stop() error {
	fmt.Println("👋 Shutting down Discord bot...")

	// Clean up commands
	if b.config.GuildID != "" {
		for _, cmd := range b.commands {
			err := b.session.ApplicationCommandDelete(b.session.State.User.ID, b.config.GuildID, cmd.ID)
			if err != nil {
				log.Printf("❌ Failed to delete command %s: %v", cmd.Name, err)
			}
		}
	}

	return b.session.Close()
}

func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Printf("✅ Bot connected as %s#%s\n", event.User.Username, event.User.Discriminator)

	if err := b.registerCommands(); err != nil {
		log.Printf("❌ Failed to register commands: %v", err)
		return
	}

	s.UpdateGameStatus(0, "🤖 T.A.R.S Online | Humor: 75% | Try /ask")
}

func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "Test T.A.R.S responsiveness",
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
			Description: "Show T.A.R.S help information",
		},
		{
			Name:        "personality",
			Description: "Adjust T.A.R.S personality settings",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "humor",
					Description: "Humor level (0-100)",
					Required:    false,
					MinValue:    func() *float64 { v := 0.0; return &v }(),
					MaxValue:    100,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "honesty",
					Description: "Honesty level (0-100)",
					Required:    false,
					MinValue:    func() *float64 { v := 0.0; return &v }(),
					MaxValue:    100,
				},
			},
		},
	}

	// Register commands
	for _, cmd := range commands {
		registeredCmd, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.config.GuildID, cmd)
		if err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.Name, err)
		}
		b.commands = append(b.commands, registeredCmd)
		fmt.Printf("✅ Registered command: /%s\n", cmd.Name)
	}

	return nil
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	fmt.Printf("📨 Message from %s: %s\n", m.Author.Username, m.Content)

	// Process message for RAG indexing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Process message for RAG context
	if err := b.ragService.ProcessMessage(ctx, m.Message); err != nil {
		fmt.Printf("❌ Failed to process message for RAG: %v\n", err)
	}

	// Handle mentions
	if b.isBotMentioned(m) {
		b.handleMentionMessage(s, m)
		return
	}

	// Handle simple commands
	b.handleSimpleCommands(s, m)
}

func (b *Bot) handleSimpleCommands(s *discordgo.Session, m *discordgo.MessageCreate) {
	content := strings.ToLower(strings.TrimSpace(m.Content))

	switch {
	case content == "!ping":
		s.ChannelMessageSend(m.ChannelID, "🏓 Pong! T.A.R.S is operational.")

	case content == "hello" || content == "hi" || content == "hey":
		responses := []string{
			"👋 Hello there! I'm T.A.R.S, your AI assistant.",
			"🤖 Greetings! How may I assist you today?",
			"Hello! My humor setting is at 75%. How can I help?",
		}
		// Simple rotation based on user ID hash
		index := len(m.Author.ID) % len(responses)
		s.ChannelMessageSend(m.ChannelID, responses[index])

	case strings.Contains(content, "how are you"):
		s.ChannelMessageSend(m.ChannelID, "🤖 All systems operational. Humor level: 75%. Honesty level: 100%. Thanks for asking!")
	}
}

func (b *Bot) onSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commandName := i.ApplicationCommandData().Name

	switch commandName {
	case "ping":
		b.handlePingCommand(s, i)
	case "ask":
		b.handleAskCommand(s, i)
	case "help":
		b.handleHelpCommand(s, i)
	case "personality":
		b.handlePersonalityCommand(s, i)
	default:
		log.Printf("❌ Unknown command: %s", commandName)
	}
}

func (b *Bot) handlePingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	startTime := time.Now()

	// Calculate latency
	latency := time.Since(startTime)

	response := fmt.Sprintf("🏓 Pong!\n🤖 T.A.R.S is operational\n⚡ Response time: %v\n📡 WebSocket latency: %v",
		latency,
		s.HeartbeatLatency())

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}

func (b *Bot) handleAskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	question := i.ApplicationCommandData().Options[0].StringValue()
	username := i.Member.User.Username

	// Send initial response to avoid timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("❌ Failed to defer interaction: %v", err)
		return
	}

	// Get AI response with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	response, err := b.aiService.GenerateResponse(ctx, question, username)
	if err != nil {
		log.Printf("❌ AI service error: %v", err)
		response = "🔧 My circuits are experiencing difficulties. My humor setting might need adjustment. Please try again later."
	}

	// Update the deferred response
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &response,
	})
	if err != nil {
		log.Printf("❌ Failed to edit interaction response: %v", err)
	}
}

func (b *Bot) handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	helpText := "🤖 **T.A.R.S - AI Assistant**\n\n" +
		"**Available Commands:**\n" +
		"`/ping` - Test bot responsiveness and latency\n" +
		"`/ask <question>` - Ask me anything (powered by AI)\n" +
		"`/help` - Show this help message\n" +
		"`/personality [humor] [honesty]` - Adjust my personality settings\n\n" +
		"**Direct Interaction:**\n" +
		"• Mention me (@T.A.R.S) to chat naturally\n" +
		"• Simple greetings like \"hello\" work too\n" +
		"• Type `/ping` for a quick response test\n\n" +
		"**About T.A.R.S:**\n" +
		"I'm an AI assistant based on the T.A.R.S robot from Interstellar. My current settings:\n" +
		"• Humor: 75% (adjustable)\n" +
		"• Honesty: 100% (always)\n\n" +
		"**Tips:**\n" +
		"• I work best with specific questions\n" +
		"• I can help with general knowledge, coding, science, and more\n" +
		"• My responses are powered by advanced AI\n\n" +
		"Built with ❤️ for the Discord community"

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpText,
		},
	})
}

func (b *Bot) handlePersonalityCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options

	// Default values
	humor := 75
	honesty := 100

	// Parse options
	for _, option := range options {
		switch option.Name {
		case "humor":
			humor = int(option.IntValue())
		case "honesty":
			honesty = int(option.IntValue())
		}
	}

	// Update AI service personality
	b.aiService.SetPersonality(humor, honesty)

	// Create response based on settings
	var response string
	switch {
	case humor == 0:
		response = fmt.Sprintf("⚙️ Personality matrix updated:\n• Humor: %d%% (Disabled)\n• Honesty: %d%%\n\nHumor circuits offline. I will now communicate with maximum efficiency and zero entertainment value.", humor, honesty)
	case humor >= 90:
		response = fmt.Sprintf("🎭 Personality matrix updated:\n• Humor: %d%% (Maximum!)\n• Honesty: %d%%\n\nWarning: Humor levels approaching critical mass. Dad jokes and puns may spontaneously occur. You've been warned! 😄", humor, honesty)
	case humor <= 25:
		response = fmt.Sprintf("🤖 Personality matrix updated:\n• Humor: %d%% (Low)\n• Honesty: %d%%\n\nSwitching to serious mode. My witty remarks will be kept to a minimum.", humor, honesty)
	default:
		response = fmt.Sprintf("🔧 Personality matrix updated:\n• Humor: %d%%\n• Honesty: %d%%\n\nOptimal settings configured. I'll maintain my characteristic blend of helpfulness and sarcasm.", humor, honesty)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}

func (b *Bot) isBotMentioned(m *discordgo.MessageCreate) bool {
	for _, mention := range m.Mentions {
		if mention.ID == b.session.State.User.ID {
			return true
		}
	}
	return false
}

func (b *Bot) handleMentionMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Extract message content without mentions
	content := b.cleanMentionsFromContent(m.Content, m.Mentions)
	if content == "" {
		content = "Hello! How can I help you?"
	}

	// Show typing indicator
	s.ChannelTyping(m.ChannelID)

	// Get AI response
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := b.aiService.GenerateResponse(ctx, content, m.Author.Username)
	if err != nil {
		fmt.Printf("❌ AI service error: %v\n", err)
		s.ChannelMessageSend(m.ChannelID, "🔧 My circuits seem to be malfunctioning. Please try again later.")
		return
	}

	s.ChannelMessageSend(m.ChannelID, response)
}

func (b *Bot) cleanMentionsFromContent(content string, mentions []*discordgo.User) string {
	for _, mention := range mentions {
		if mention.ID == b.session.State.User.ID {
			// Remove both <@ID> and <@!ID> formats
			content = strings.ReplaceAll(content, "<@"+mention.ID+">", "")
			content = strings.ReplaceAll(content, "<@!"+mention.ID+">", "")
		}
	}
	return strings.TrimSpace(content)
}

// SendMessage implements the DiscordService interface
func (b *Bot) SendMessage(channelID, content string) error {
	_, err := b.session.ChannelMessageSend(channelID, content)
	return err
}

// SendTyping implements the DiscordService interface
func (b *Bot) SendTyping(channelID string) error {
	return b.session.ChannelTyping(channelID)
}

// UpdateStatus implements the DiscordService interface
func (b *Bot) UpdateStatus(activity string) error {
	return b.session.UpdateGameStatus(0, activity)
}

// GetSession returns the Discord session
func (b *Bot) GetSession() *discordgo.Session {
	return b.session
}

// SetRAGService updates the RAG service reference
func (b *Bot) SetRAGService(ragService *rag.Service) {
	b.ragService = ragService
}
