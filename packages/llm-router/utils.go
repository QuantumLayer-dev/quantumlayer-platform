package llmrouter

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

// LoggerMiddleware creates a Gin middleware for logging
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("Request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
		)
	}
}

// CORSMiddleware creates a Gin middleware for CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// AuthMiddleware creates a Gin middleware for authentication
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement proper authentication
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}

		// Set user context for downstream handlers
		c.Set("user_id", "user-123")
		c.Set("org_id", "org-456")
		c.Next()
	}
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if request is allowed
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	tokensToAdd := int(elapsed / r.refillRate)

	if tokensToAdd > 0 {
		r.tokens = min(r.tokens+tokensToAdd, r.maxTokens)
		r.lastRefill = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}

// TokenBucket implements token bucket for quota management
type TokenBucket struct {
	tokens     int64
	maxTokens  int64
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(maxTokens int64, refillRate time.Duration) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Consume attempts to consume tokens
func (t *TokenBucket) Consume(tokens int64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastRefill)
	if elapsed >= t.refillRate {
		t.tokens = t.maxTokens
		t.lastRefill = now
	}

	if t.tokens >= tokens {
		t.tokens -= tokens
		return true
	}

	return false
}

// HealthChecker monitors provider health
type HealthChecker struct {
	failures    int
	maxFailures int
	lastCheck   time.Time
	isHealthy   bool
	backoff     time.Duration
	mu          sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		maxFailures: 3,
		isHealthy:   true,
		backoff:     time.Second,
	}
}

// RecordSuccess records a successful request
func (h *HealthChecker) RecordSuccess() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.failures = 0
	h.isHealthy = true
	h.backoff = time.Second
	h.lastCheck = time.Now()
}

// RecordFailure records a failed request
func (h *HealthChecker) RecordFailure() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.failures++
	h.lastCheck = time.Now()

	if h.failures >= h.maxFailures {
		h.isHealthy = false
		// Exponential backoff
		h.backoff = min(h.backoff*2, 5*time.Minute)
	}
}

// IsHealthy checks if provider is healthy
func (h *HealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.isHealthy {
		// Check if enough time has passed for retry
		if time.Since(h.lastCheck) >= h.backoff {
			h.mu.RUnlock()
			h.mu.Lock()
			h.isHealthy = true
			h.failures = 0
			h.mu.Unlock()
			h.mu.RLock()
		}
	}

	return h.isHealthy
}

// min returns the minimum of two values
func min[T ~int | ~int64 | time.Duration](a, b T) T {
	if a < b {
		return a
	}
	return b
}