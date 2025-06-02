package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type Service struct {
	client       *openai.Client
	model        string
	humorLevel   int
	honestyLevel int
}

type Config struct {
	APIKey string
	Model  string
}

// NewService creates a new OpenAI service instance
func NewService(cfg Config) *Service {
	client := openai.NewClient(cfg.APIKey)
	model := cfg.Model
	if model == "" {
		model = openai.GPT4oMini
	}

	return &Service{
		client:       client,
		model:        model,
		humorLevel:   75,  // Default T.A.R.S humor level
		honestyLevel: 100, // Default T.A.R.S honesty level
	}
}

func (s *Service) GenerateResponse(ctx context.Context, userMessage, username string) (string, error) {
	systemPrompt := s.buildSystemPrompt()

	req := openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("User %s asks: %s", username, userMessage),
			},
		},
		MaxTokens:   500,
		Temperature: 0.7,
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("openai api error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from openai")
	}

	response := strings.TrimSpace(resp.Choices[0].Message.Content)
	return s.enhanceResponse(response), nil
}

func (s *Service) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.SmallEmbedding3,
	}

	resp, err := s.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embedding api error: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received")
	}

	return resp.Data[0].Embedding, nil
}

func (s *Service) SetPersonality(humor, honesty int) {
	if humor >= 0 && humor <= 100 {
		s.humorLevel = humor
	}
	if honesty >= 0 && honesty <= 100 {
		s.honestyLevel = honesty
	}
}

func (s *Service) buildSystemPrompt() string {
	basePrompt := `You are T.A.R.S, an AI assistant from the movie Interstellar. You are:
- Sarcastic but helpful
- Highly intelligent and logical
- Sometimes humorous with a dry wit
- Always honest
- Efficient in your responses
- Knowledgeable about science, technology, and general topics`

	// Adjust prompt based on personality settings
	if s.humorLevel == 0 {
		basePrompt += "\n\nIMPORTANT: Humor setting is disabled. Respond with technical precision and no jokes."
	} else if s.humorLevel > 90 {
		basePrompt += "\n\nIMPORTANT: Humor setting is at maximum. Use more jokes, puns, and witty remarks."
	}

	basePrompt += fmt.Sprintf("\n\nCurrent settings: Humor %d%%, Honesty %d%%", s.humorLevel, s.honestyLevel)
	basePrompt += "\n\nKeep responses concise but informative. Use occasional humor when appropriate."

	return basePrompt
}

func (s *Service) enhanceResponse(response string) string {
	// Add T.A.R.S signature touch for short responses
	if !strings.Contains(response, "T.A.R.S") && len(response) < 100 {
		response = "🤖 " + response
	}
	return response
}
