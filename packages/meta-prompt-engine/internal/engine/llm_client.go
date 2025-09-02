package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// RealLLMClient implements the LLMClient interface with actual LLM Router calls
type RealLLMClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewRealLLMClient creates a new LLM Router client
func NewRealLLMClient(baseURL string, logger *logrus.Logger) *RealLLMClient {
	if baseURL == "" {
		baseURL = "http://llm-router.quantumlayer.svc.cluster.local:8080"
	}
	
	return &RealLLMClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Complete sends a completion request to the LLM Router
func (c *RealLLMClient) Complete(ctx context.Context, prompt string, model string) (string, int, error) {
	// Build request
	req := map[string]interface{}{
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": 2000,
	}
	
	// Add model if specified
	if model != "" {
		req["model"] = model
	}
	
	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal LLM request")
		return "", 0, err
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/complete", bytes.NewBuffer(reqBody))
	if err != nil {
		c.logger.WithError(err).Error("Failed to create HTTP request")
		return "", 0, err
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Make the request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.WithError(err).Error("Failed to call LLM Router")
		return "", 0, err
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithError(err).Error("Failed to read response")
		return "", 0, err
	}
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		c.logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("LLM Router returned error")
		return "", 0, fmt.Errorf("LLM Router returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var llmResp struct {
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
	
	if err := json.Unmarshal(body, &llmResp); err != nil {
		c.logger.WithError(err).Error("Failed to parse LLM response")
		return "", 0, err
	}
	
	// Extract response and token count
	if len(llmResp.Choices) > 0 {
		content := llmResp.Choices[0].Message.Content
		tokens := llmResp.Usage.TotalTokens
		
		c.logger.WithFields(logrus.Fields{
			"model":  llmResp.Model,
			"tokens": tokens,
		}).Debug("Successfully got LLM response")
		
		return content, tokens, nil
	}
	
	return "", 0, fmt.Errorf("no content in LLM response")
}