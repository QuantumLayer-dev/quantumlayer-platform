package providers

import (
	"context"
	"errors"
	
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	
	llm "github.com/QuantumLayer-dev/quantumlayer-platform/packages/llm-router"
)

// OpenAIClient implements the ProviderClient interface for OpenAI
type OpenAIClient struct {
	client *openai.Client
	logger *zap.Logger
	config *llm.ProviderConfig
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string, logger *zap.Logger) *OpenAIClient {
	return &OpenAIClient{
		client: openai.NewClient(apiKey),
		logger: logger,
	}
}

// Complete sends a completion request to OpenAI
func (c *OpenAIClient) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	// Convert our request to OpenAI format
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		}
	}
	
	openaiReq := openai.ChatCompletionRequest{
		Model:            c.mapModel(req.Model),
		Messages:         messages,
		MaxTokens:        req.MaxTokens,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		Stop:             req.Stop,
		PresencePenalty:  req.PresencePenalty,
		FrequencyPenalty: req.FrequencyPenalty,
		Stream:           false,
	}
	
	// Make the API call
	resp, err := c.client.CreateChatCompletion(ctx, openaiReq)
	if err != nil {
		c.logger.Error("OpenAI API error", zap.Error(err))
		return nil, err
	}
	
	// Convert response to our format
	choices := make([]llm.Choice, len(resp.Choices))
	for i, choice := range resp.Choices {
		choices[i] = llm.Choice{
			Index: choice.Index,
			Message: llm.Message{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
			FinishReason: string(choice.FinishReason),
		}
	}
	
	return &llm.Response{
		ID:       resp.ID,
		Object:   resp.Object,
		Created:  resp.Created,
		Model:    llm.Model(resp.Model),
		Provider: llm.ProviderOpenAI,
		Choices:  choices,
		Usage: llm.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// Stream sends a streaming completion request to OpenAI
func (c *OpenAIClient) Stream(ctx context.Context, req *llm.Request) (<-chan *llm.Response, error) {
	respChan := make(chan *llm.Response)
	
	go func() {
		defer close(respChan)
		
		messages := make([]openai.ChatCompletionMessage, len(req.Messages))
		for i, msg := range req.Messages {
			messages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
		
		openaiReq := openai.ChatCompletionRequest{
			Model:       c.mapModel(req.Model),
			Messages:    messages,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			Stream:      true,
		}
		
		stream, err := c.client.CreateChatCompletionStream(ctx, openaiReq)
		if err != nil {
			c.logger.Error("OpenAI stream error", zap.Error(err))
			return
		}
		defer stream.Close()
		
		for {
			response, err := stream.Recv()
			if errors.Is(err, context.Canceled) {
				return
			}
			if err != nil {
				c.logger.Error("Stream receive error", zap.Error(err))
				return
			}
			
			// Convert and send response
			if len(response.Choices) > 0 {
				resp := &llm.Response{
					ID:       response.ID,
					Object:   response.Object,
					Created:  response.Created,
					Model:    llm.Model(response.Model),
					Provider: llm.ProviderOpenAI,
					Choices: []llm.Choice{{
						Index: response.Choices[0].Index,
						Message: llm.Message{
							Role:    response.Choices[0].Delta.Role,
							Content: response.Choices[0].Delta.Content,
						},
						FinishReason: string(response.Choices[0].FinishReason),
					}},
				}
				
				select {
				case respChan <- resp:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	return respChan, nil
}

// Name returns the provider name
func (c *OpenAIClient) Name() llm.Provider {
	return llm.ProviderOpenAI
}

// IsAvailable checks if the provider is available
func (c *OpenAIClient) IsAvailable() bool {
	// Could implement a health check here
	return true
}

// GetCapabilities returns provider capabilities
func (c *OpenAIClient) GetCapabilities() llm.Capabilities {
	return llm.Capabilities{
		MaxTokens:        128000, // GPT-4 Turbo
		SupportStreaming: true,
		SupportFunctions: true,
		SupportVision:    true,
		Languages:        []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh"},
		Models: []llm.Model{
			llm.ModelGPT4Turbo,
			llm.ModelGPT4,
			llm.ModelGPT35Turbo,
		},
	}
}

// mapModel maps our model enum to OpenAI model string
func (c *OpenAIClient) mapModel(model llm.Model) string {
	switch model {
	case llm.ModelGPT4Turbo:
		return "gpt-4-turbo-preview"
	case llm.ModelGPT4:
		return "gpt-4"
	case llm.ModelGPT35Turbo:
		return "gpt-3.5-turbo"
	default:
		return "gpt-4-turbo-preview"
	}
}