package llmrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
	
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"go.uber.org/zap"
)

// BedrockRealClient implements actual AWS Bedrock integration
type BedrockRealClient struct {
	client *bedrockruntime.Client
	logger *zap.Logger
	model  string
}

// ClaudeRequest represents the request format for Claude on Bedrock
type ClaudeRequest struct {
	Prompt            string   `json:"prompt"`
	MaxTokensToSample int      `json:"max_tokens_to_sample"`
	Temperature       float32  `json:"temperature,omitempty"`
	TopP              float32  `json:"top_p,omitempty"`
	TopK              int      `json:"top_k,omitempty"`
	StopSequences     []string `json:"stop_sequences,omitempty"`
}

// ClaudeResponse represents the response from Claude on Bedrock
type ClaudeResponse struct {
	Completion string `json:"completion"`
	StopReason string `json:"stop_reason"`
}

// NewBedrockRealClient creates a real AWS Bedrock client
func NewBedrockRealClient(logger *zap.Logger) (*BedrockRealClient, error) {
	// Get configuration from environment
	region := getEnv("AWS_BEDROCK_REGION", "us-east-1")
	model := getEnv("AWS_BEDROCK_MODEL", "anthropic.claude-3-haiku-20240307-v1:0")
	
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	// Create Bedrock Runtime client
	client := bedrockruntime.NewFromConfig(cfg)
	
	logger.Info("Initialized AWS Bedrock client", 
		zap.String("region", region),
		zap.String("model", model))
	
	return &BedrockRealClient{
		client: client,
		logger: logger,
		model:  model,
	}, nil
}

// Complete sends a real completion request to AWS Bedrock
func (c *BedrockRealClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	c.logger.Info("Sending request to AWS Bedrock", 
		zap.String("model", c.model),
		zap.Int("message_count", len(req.Messages)))
	
	// Convert messages to Claude prompt format
	prompt := c.formatPrompt(req.Messages)
	
	// Prepare Claude request
	claudeReq := ClaudeRequest{
		Prompt:            prompt,
		MaxTokensToSample: 2000,
		Temperature:       0.7,
		TopP:              0.9,
	}
	
	if req.MaxTokens > 0 {
		claudeReq.MaxTokensToSample = req.MaxTokens
	}
	if req.Temperature > 0 {
		claudeReq.Temperature = req.Temperature
	}
	
	// Marshal request to JSON
	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Call AWS Bedrock
	output, err := c.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(c.model),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		c.logger.Error("AWS Bedrock API call failed", zap.Error(err))
		return nil, fmt.Errorf("bedrock API call failed: %w", err)
	}
	
	// Parse response
	var claudeResp ClaudeResponse
	if err := json.Unmarshal(output.Body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	c.logger.Info("Received response from AWS Bedrock", 
		zap.Int("response_length", len(claudeResp.Completion)),
		zap.String("stop_reason", claudeResp.StopReason))
	
	// Convert to our response format
	return &Response{
		ID:       generateRequestID(),
		Object:   "chat.completion",
		Created:  time.Now().Unix(),
		Model:    Model(c.model),
		Provider: ProviderBedrock,
		Choices: []Choice{{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: claudeResp.Completion,
			},
			FinishReason: claudeResp.StopReason,
		}},
		Usage: Usage{
			PromptTokens:     len(prompt) / 4,     // Rough estimate
			CompletionTokens: len(claudeResp.Completion) / 4,
			TotalTokens:      (len(prompt) + len(claudeResp.Completion)) / 4,
		},
	}, nil
}

// formatPrompt converts chat messages to Claude prompt format
func (c *BedrockRealClient) formatPrompt(messages []Message) string {
	var prompt bytes.Buffer
	
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			prompt.WriteString(fmt.Sprintf("\n\n%s", msg.Content))
		case "user":
			prompt.WriteString(fmt.Sprintf("\n\nHuman: %s", msg.Content))
		case "assistant":
			prompt.WriteString(fmt.Sprintf("\n\nAssistant: %s", msg.Content))
		}
	}
	
	// Claude expects prompts to end with "\n\nAssistant:"
	prompt.WriteString("\n\nAssistant:")
	
	return prompt.String()
}

// Stream is not yet implemented for Bedrock
func (c *BedrockRealClient) Stream(ctx context.Context, req *Request) (<-chan *Response, error) {
	return nil, fmt.Errorf("streaming not yet implemented for Bedrock")
}

// Name returns the provider name
func (c *BedrockRealClient) Name() Provider {
	return ProviderBedrock
}

// IsAvailable checks if the provider is available
func (c *BedrockRealClient) IsAvailable() bool {
	// Check if AWS credentials are configured
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	return accessKey != "" && secretKey != ""
}

// GetCapabilities returns provider capabilities
func (c *BedrockRealClient) GetCapabilities() Capabilities {
	return Capabilities{
		MaxTokens:        200000, // Claude 3 on Bedrock
		SupportStreaming: false,  // Not implemented yet
		SupportFunctions: false,
		SupportVision:    true,
		Languages:        []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh"},
		Models: []Model{
			Model("anthropic.claude-3-haiku-20240307-v1:0"),
			Model("anthropic.claude-3-sonnet-20240229-v1:0"),
			Model("anthropic.claude-v2"),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}