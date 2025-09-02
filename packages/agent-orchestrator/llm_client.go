package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// LLMClient handles communication with the LLM Router service
type LLMClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewLLMClient creates a new LLM Router client
func NewLLMClient(baseURL string, logger *zap.Logger) *LLMClient {
	if baseURL == "" {
		baseURL = "http://llm-router.quantumlayer.svc.cluster.local:8080"
	}
	
	return &LLMClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// LLMRequest represents a request to the LLM Router
type LLMRequest struct {
	Messages  []LLMMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens,omitempty"`
}

// LLMMessage represents a message in the conversation
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse represents the response from LLM Router
type LLMResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// GenerateCode calls the LLM Router to generate code
func (c *LLMClient) GenerateCode(ctx context.Context, prompt, language, framework string) (string, error) {
	// Build the prompt for code generation
	systemPrompt := fmt.Sprintf("You are an expert %s developer. Generate clean, production-ready code.", language)
	if framework != "" {
		systemPrompt += fmt.Sprintf(" Use the %s framework.", framework)
	}
	
	userPrompt := prompt
	if userPrompt == "" {
		userPrompt = fmt.Sprintf("Generate a %s application", language)
	}
	
	// Create the request
	req := LLMRequest{
		Messages: []LLMMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MaxTokens: 2000,
	}
	
	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		c.logger.Error("Failed to marshal LLM request", zap.Error(err))
		return "", err
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/complete", bytes.NewBuffer(reqBody))
	if err != nil {
		c.logger.Error("Failed to create HTTP request", zap.Error(err))
		return "", err
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Make the request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("Failed to call LLM Router", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response", zap.Error(err))
		return "", err
	}
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("LLM Router returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return "", fmt.Errorf("LLM Router returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		c.logger.Error("Failed to parse LLM response", zap.Error(err))
		return "", err
	}
	
	// Extract generated code
	if len(llmResp.Choices) > 0 {
		generatedCode := llmResp.Choices[0].Message.Content
		c.logger.Info("Successfully generated code",
			zap.String("model", llmResp.Model),
			zap.Int("tokens", llmResp.Usage.TotalTokens),
		)
		return generatedCode, nil
	}
	
	return "", fmt.Errorf("no content in LLM response")
}

// Complete sends a general completion request to the LLM Router
func (c *LLMClient) Complete(ctx context.Context, messages []LLMMessage, maxTokens int) (string, error) {
	if maxTokens == 0 {
		maxTokens = 1000
	}
	
	req := LLMRequest{
		Messages:  messages,
		MaxTokens: maxTokens,
	}
	
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/complete", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM Router returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return "", err
	}
	
	if len(llmResp.Choices) > 0 {
		return llmResp.Choices[0].Message.Content, nil
	}
	
	return "", fmt.Errorf("no content in response")
}