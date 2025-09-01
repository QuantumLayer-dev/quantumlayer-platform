package llmrouter

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"time"
)

// generateRequestID generates a unique request ID
func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "req_" + hex.EncodeToString(b)
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// sanitizeAPIKey removes sensitive parts from API key for logging
func sanitizeAPIKey(key string) string {
	if len(key) < 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// estimateTokens estimates token count from text (rough approximation)
func estimateTokens(text string) int {
	// Rough estimation: 1 token ≈ 4 characters
	// This is a simplification; actual tokenization is more complex
	return len(text) / 4
}

// truncateString truncates a string to specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// retryWithBackoff retries a function with exponential backoff
func retryWithBackoff(fn func() error, maxRetries int, initialDelay time.Duration) error {
	var err error
	delay := initialDelay
	
	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		
		if i < maxRetries-1 {
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}
	
	return err
}

// normalizeModel normalizes model names across providers
func normalizeModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	
	// Map common aliases
	aliases := map[string]string{
		"gpt-4-turbo":          "gpt-4-turbo-preview",
		"gpt-3.5":              "gpt-3.5-turbo",
		"claude-3":             "claude-3-opus",
		"claude":               "claude-3-opus",
		"llama-3-70b":          "llama3-70b-8192",
		"llama-3-8b":           "llama3-8b-8192",
		"mixtral":              "mixtral-8x7b-32768",
	}
	
	if normalized, ok := aliases[model]; ok {
		return normalized
	}
	
	return model
}

// selectModelForProvider selects the best model for a given provider
func selectModelForProvider(provider Provider, preferredModel Model) Model {
	// If preferred model is compatible with provider, use it
	if isModelCompatible(provider, preferredModel) {
		return preferredModel
	}
	
	// Otherwise, select default model for provider
	switch provider {
	case ProviderOpenAI:
		return ModelGPT4Turbo
	case ProviderAnthropic:
		return ModelClaude3Opus
	case ProviderGroq:
		return ModelLlama3_70B
	case ProviderBedrock:
		return ModelClaudeBedrock
	default:
		return ModelGPT4Turbo
	}
}

// isModelCompatible checks if a model is compatible with a provider
func isModelCompatible(provider Provider, model Model) bool {
	switch provider {
	case ProviderOpenAI:
		return strings.HasPrefix(string(model), "gpt")
	case ProviderAnthropic:
		return strings.HasPrefix(string(model), "claude")
	case ProviderGroq:
		return strings.Contains(string(model), "llama") || strings.Contains(string(model), "mixtral")
	case ProviderBedrock:
		return strings.Contains(string(model), "bedrock")
	default:
		return false
	}
}

// calculateCost calculates the cost of a request in cents
func calculateCost(tokens int, costPerMillion float64) float64 {
	return (float64(tokens) / 1000000.0) * costPerMillion * 100
}

// parseTimeout parses a timeout string (e.g., "30s", "1m")
func parseTimeout(s string) time.Duration {
	if s == "" {
		return 30 * time.Second // Default timeout
	}
	
	duration, err := time.ParseDuration(s)
	if err != nil {
		return 30 * time.Second
	}
	
	return duration
}

// mergeMessages merges system prompts with user messages
func mergeMessages(systemPrompt string, messages []Message) []Message {
	if systemPrompt == "" {
		return messages
	}
	
	// Prepend system message
	result := make([]Message, 0, len(messages)+1)
	result = append(result, Message{
		Role:    "system",
		Content: systemPrompt,
	})
	result = append(result, messages...)
	
	return result
}

// validateRequest validates an LLM request
func validateRequest(req *Request) error {
	if len(req.Messages) == 0 {
		return ErrInvalidRequest
	}
	
	// Validate temperature
	if req.Temperature < 0 || req.Temperature > 2 {
		req.Temperature = 0.7 // Default
	}
	
	// Validate max tokens
	if req.MaxTokens <= 0 {
		req.MaxTokens = 2048 // Default
	} else if req.MaxTokens > 128000 {
		req.MaxTokens = 128000 // Max limit
	}
	
	// Validate top_p
	if req.TopP < 0 || req.TopP > 1 {
		req.TopP = 1.0 // Default
	}
	
	return nil
}

// ResponseCache represents a simple response cache entry
type ResponseCache struct {
	Response  *Response
	ExpiresAt time.Time
}

// isExpired checks if a cache entry is expired
func (rc *ResponseCache) isExpired() bool {
	return time.Now().After(rc.ExpiresAt)
}

// hashRequest creates a hash key for caching requests
func hashRequest(req *Request) string {
	// Simplified hashing - in production, use proper hash function
	var parts []string
	
	parts = append(parts, string(req.Model))
	for _, msg := range req.Messages {
		parts = append(parts, msg.Role+":"+msg.Content)
	}
	
	return strings.Join(parts, "|")
}