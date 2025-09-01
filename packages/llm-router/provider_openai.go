package llmrouter

import (
	"context"
	"errors"
	
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// OpenAIClient implements the ProviderClient interface for OpenAI
type OpenAIClient struct {
	client *openai.Client
	logger *zap.Logger
	config *ProviderConfig
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string, logger *zap.Logger) *OpenAIClient {
	return &OpenAIClient{
		client: openai.NewClient(apiKey),
		logger: logger,
	}
}

// Complete sends a completion request to OpenAI
func (c *OpenAIClient) Complete(ctx context.Context, req *Request) (*Response, error) {
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
	choices := make([]Choice, len(resp.Choices))
	for i, choice := range resp.Choices {
		choices[i] = Choice{
			Index: choice.Index,
			Message: Message{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
			FinishReason: string(choice.FinishReason),
		}
	}
	
	return &Response{
		ID:       resp.ID,
		Object:   resp.Object,
		Created:  resp.Created,
		Model:    Model(resp.Model),
		Provider: ProviderOpenAI,
		Choices:  choices,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// Stream sends a streaming completion request to OpenAI
func (c *OpenAIClient) Stream(ctx context.Context, req *Request) (<-chan *Response, error) {
	respChan := make(chan *Response)
	
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
				resp := &Response{
					ID:       response.ID,
					Object:   response.Object,
					Created:  response.Created,
					Model:    Model(response.Model),
					Provider: ProviderOpenAI,
					Choices: []Choice{{
						Index: response.Choices[0].Index,
						Message: Message{
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
func (c *OpenAIClient) Name() Provider {
	return ProviderOpenAI
}

// IsAvailable checks if the provider is available
func (c *OpenAIClient) IsAvailable() bool {
	// Could implement a health check here
	return true
}

// GetCapabilities returns provider capabilities
func (c *OpenAIClient) GetCapabilities() Capabilities {
	return Capabilities{
		MaxTokens:        128000, // GPT-4 Turbo
		SupportStreaming: true,
		SupportFunctions: true,
		SupportVision:    true,
		Languages:        []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh"},
		Models: []Model{
			ModelGPT4Turbo,
			ModelGPT4,
			ModelGPT35Turbo,
		},
	}
}

// mapModel maps our model enum to OpenAI model string
func (c *OpenAIClient) mapModel(model Model) string {
	switch model {
	case ModelGPT4Turbo:
		return "gpt-4-turbo-preview"
	case ModelGPT4:
		return "gpt-4"
	case ModelGPT35Turbo:
		return "gpt-3.5-turbo"
	default:
		return "gpt-4-turbo-preview"
	}
}