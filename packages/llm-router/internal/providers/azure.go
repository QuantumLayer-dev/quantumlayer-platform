package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// AzureOpenAIProvider implements enterprise-grade Azure OpenAI integration
type AzureOpenAIProvider struct {
	endpoint       string
	apiKey         string
	deploymentName string
	apiVersion     string
	client         *http.Client
	logger         *zap.Logger
	rateLimiter    *rate.Limiter
	circuitBreaker *gobreaker.CircuitBreaker
	metrics        *ProviderMetrics
}

// AzureConfig holds Azure OpenAI configuration
type AzureConfig struct {
	Endpoint       string
	APIKey         string
	DeploymentName string
	APIVersion     string
	MaxRetries     int
	Timeout        time.Duration
	RateLimit      rate.Limit
	BurstLimit     int
}

// NewAzureOpenAIProvider creates a new Azure OpenAI provider with enterprise features
func NewAzureOpenAIProvider(config AzureConfig, logger *zap.Logger) *AzureOpenAIProvider {
	if config.APIVersion == "" {
		config.APIVersion = "2024-02-01"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RateLimit == 0 {
		config.RateLimit = rate.Limit(10) // 10 requests per second
	}
	if config.BurstLimit == 0 {
		config.BurstLimit = 20
	}

	cbSettings := gobreaker.Settings{
		Name:        "AzureOpenAI",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("Circuit breaker state change",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()))
		},
	}

	return &AzureOpenAIProvider{
		endpoint:       strings.TrimRight(config.Endpoint, "/"),
		apiKey:         config.APIKey,
		deploymentName: config.DeploymentName,
		apiVersion:     config.APIVersion,
		client: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger:         logger,
		rateLimiter:    rate.NewLimiter(config.RateLimit, config.BurstLimit),
		circuitBreaker: gobreaker.NewCircuitBreaker(cbSettings),
		metrics:        NewProviderMetrics("azure_openai"),
	}
}

// AzureRequest represents the Azure OpenAI API request
type AzureRequest struct {
	Messages      []Message `json:"messages"`
	MaxTokens     int       `json:"max_tokens,omitempty"`
	Temperature   float32   `json:"temperature,omitempty"`
	TopP          float32   `json:"top_p,omitempty"`
	Stream        bool      `json:"stream"`
	Stop          []string  `json:"stop,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat specifies the output format
type ResponseFormat struct {
	Type string `json:"type"` // "text" or "json_object"
}

// AzureResponse represents the Azure OpenAI API response
type AzureResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	Error   *Error   `json:"error,omitempty"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Error represents an API error
type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// GenerateCode generates code using Azure OpenAI with enterprise features
func (p *AzureOpenAIProvider) GenerateCode(ctx context.Context, request CodeGenerationRequest) (*CodeGenerationResponse, error) {
	// Rate limiting
	if err := p.rateLimiter.Wait(ctx); err != nil {
		p.metrics.RecordError("rate_limit")
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	// Circuit breaker
	response, err := p.circuitBreaker.Execute(func() (interface{}, error) {
		return p.executeRequest(ctx, request)
	})

	if err != nil {
		p.metrics.RecordError("circuit_breaker")
		return nil, err
	}

	return response.(*CodeGenerationResponse), nil
}

func (p *AzureOpenAIProvider) executeRequest(ctx context.Context, request CodeGenerationRequest) (*CodeGenerationResponse, error) {
	startTime := time.Now()
	
	// Build messages with proper code generation prompt
	messages := []Message{
		{
			Role: "system",
			Content: p.buildSystemPrompt(request),
		},
		{
			Role: "user",
			Content: request.Prompt,
		},
	}

	// Create request
	azureReq := AzureRequest{
		Messages:    messages,
		MaxTokens:   request.MaxTokens,
		Temperature: 0.7,
		TopP:        0.95,
		Stream:      false,
	}

	// For code generation, request JSON format when possible
	if strings.Contains(strings.ToLower(request.Language), "json") {
		azureReq.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}

	body, err := json.Marshal(azureReq)
	if err != nil {
		p.metrics.RecordError("marshal")
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.endpoint, p.deploymentName, p.apiVersion)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		p.metrics.RecordError("request_creation")
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

	// Execute request
	p.logger.Debug("Calling Azure OpenAI",
		zap.String("deployment", p.deploymentName),
		zap.Int("max_tokens", request.MaxTokens))

	resp, err := p.client.Do(req)
	if err != nil {
		p.metrics.RecordError("http_request")
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.metrics.RecordError("read_response")
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		p.metrics.RecordError(fmt.Sprintf("http_%d", resp.StatusCode))
		var azureResp AzureResponse
		if err := json.Unmarshal(respBody, &azureResp); err == nil && azureResp.Error != nil {
			return nil, fmt.Errorf("Azure API error: %s (code: %s)", azureResp.Error.Message, azureResp.Error.Code)
		}
		return nil, fmt.Errorf("Azure API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var azureResp AzureResponse
	if err := json.Unmarshal(respBody, &azureResp); err != nil {
		p.metrics.RecordError("unmarshal")
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Extract code
	if len(azureResp.Choices) == 0 {
		p.metrics.RecordError("no_choices")
		return nil, fmt.Errorf("no choices in response")
	}

	content := azureResp.Choices[0].Message.Content
	
	// Clean and extract code
	code := p.extractCode(content)
	
	// Record metrics
	p.metrics.RecordLatency(time.Since(startTime))
	p.metrics.RecordTokens(azureResp.Usage.TotalTokens)
	p.metrics.RecordSuccess()

	return &CodeGenerationResponse{
		Code:     code,
		Language: request.Language,
		Provider: "azure-openai",
		Model:    p.deploymentName,
		Usage: TokenUsage{
			PromptTokens:     azureResp.Usage.PromptTokens,
			CompletionTokens: azureResp.Usage.CompletionTokens,
			TotalTokens:      azureResp.Usage.TotalTokens,
		},
		Latency: time.Since(startTime),
	}, nil
}

func (p *AzureOpenAIProvider) buildSystemPrompt(request CodeGenerationRequest) string {
	var prompt strings.Builder
	
	prompt.WriteString("You are an expert software engineer and code generator. ")
	prompt.WriteString("Generate production-ready, well-structured code following best practices. ")
	prompt.WriteString("IMPORTANT: Return ONLY code without any explanations, markdown formatting, or conversational text. ")
	prompt.WriteString("Do not include code block markers (```) or language tags. ")
	prompt.WriteString("Generate pure, executable code only.\n\n")
	
	if request.Language != "" {
		prompt.WriteString(fmt.Sprintf("Language: %s\n", request.Language))
	}
	if request.Framework != "" {
		prompt.WriteString(fmt.Sprintf("Framework: %s\n", request.Framework))
	}
	if request.Type != "" {
		prompt.WriteString(fmt.Sprintf("Type: %s application\n", request.Type))
	}
	
	prompt.WriteString("\nRequirements:\n")
	prompt.WriteString("- Include proper error handling\n")
	prompt.WriteString("- Follow language idioms and conventions\n")
	prompt.WriteString("- Add necessary imports/dependencies\n")
	prompt.WriteString("- Ensure code is complete and runnable\n")
	prompt.WriteString("- Use meaningful variable and function names\n")
	
	return prompt.String()
}

func (p *AzureOpenAIProvider) extractCode(content string) string {
	// Remove markdown code blocks if present
	content = strings.TrimSpace(content)
	
	// Remove code block markers
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		var codeLines []string
		inCodeBlock := false
		
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inCodeBlock = !inCodeBlock
				continue
			}
			if inCodeBlock || !strings.HasPrefix(line, "```") {
				codeLines = append(codeLines, line)
			}
		}
		content = strings.Join(codeLines, "\n")
	}
	
	return strings.TrimSpace(content)
}

// HealthCheck verifies Azure OpenAI connectivity
func (p *AzureOpenAIProvider) HealthCheck(ctx context.Context) error {
	// Simple health check with minimal tokens
	request := CodeGenerationRequest{
		Prompt:    "Write a hello world function",
		Language:  "python",
		MaxTokens: 50,
	}
	
	_, err := p.GenerateCode(ctx, request)
	return err
}

// GetMetrics returns provider metrics
func (p *AzureOpenAIProvider) GetMetrics() ProviderMetrics {
	return *p.metrics
}