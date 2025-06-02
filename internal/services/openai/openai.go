package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type OpenAIService struct {
	client *openai.Client
	model  string
}

func NewOpenAIService(apiKey, model string) *OpenAIService {
	client := openai.NewClient(apiKey)
	if model == "" {
		model = openai.GPT4oMini // Cost-effective default
	}

	return &OpenAIService{
		client: client,
		model:  model,
	}
}

func (s *OpenAIService) GenerateResponse(ctx context.Context, userMessage, username string) (string, error) {
	// System message to define T.A.R.S personality
	systemPrompt := `You are T.A.R.S, an AI assistant from the movie Interstellar. You are:
- Sarcastic but helpful
- Highly intelligent and logical
- Sometimes humorous with a dry wit
- Always honest (humor setting at 75%)
- Efficient in your responses
- Knowledgeable about science, technology, and general topics

Keep responses concise but informative. Use occasional humor when appropriate.`

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
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	response := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Add T.A.R.S signature touch
	if !strings.Contains(response, "T.A.R.S") && len(response) < 100 {
		response = "🤖 " + response
	}

	return response, nil
}

func (s *OpenAIService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.SmallEmbedding3, // text-embedding-3-small
	}

	resp, err := s.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embedding API error: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received")
	}

	return resp.Data[0].Embedding, nil
}
