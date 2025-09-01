package llmrouter

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

var (
	ErrNoProvidersAvailable = errors.New("no LLM providers available")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrQuotaExceeded        = errors.New("quota exceeded")
	ErrProviderTimeout      = errors.New("provider timeout")
	ErrInvalidRequest       = errors.New("invalid request")
)

// Provider represents an LLM provider
type Provider string

const (
	ProviderOpenAI      Provider = "openai"
	ProviderAnthropic   Provider = "anthropic"
	ProviderGroq        Provider = "groq"
	ProviderBedrock     Provider = "bedrock"
	ProviderAzureOpenAI Provider = "azure-openai"
	ProviderVertexAI    Provider = "vertex-ai"
	ProviderCohere      Provider = "cohere"
	ProviderLocal       Provider = "local"
)

// Model represents an LLM model
type Model string

const (
	// OpenAI Models
	ModelGPT4Turbo   Model = "gpt-4-turbo-preview"
	ModelGPT4        Model = "gpt-4"
	ModelGPT35Turbo  Model = "gpt-3.5-turbo"
	
	// Anthropic Models
	ModelClaude3Opus   Model = "claude-3-opus-20240229"
	ModelClaude3Sonnet Model = "claude-3-sonnet-20240229"
	ModelClaude3Haiku  Model = "claude-3-haiku-20240307"
	
	// Groq Models (Fast inference)
	ModelLlama3_70B  Model = "llama3-70b-8192"
	ModelLlama3_8B   Model = "llama3-8b-8192"
	ModelMixtral8x7B Model = "mixtral-8x7b-32768"
	
	// Bedrock Models
	ModelClaudeBedrock Model = "anthropic.claude-v2"
	ModelLlamaBedrock  Model = "meta.llama2-70b-chat-v1"
)

// Request represents an LLM completion request
type Request struct {
	ID             string                 `json:"id"`
	Model          Model                  `json:"model,omitempty"`
	Messages       []Message              `json:"messages"`
	MaxTokens      int                    `json:"max_tokens,omitempty"`
	Temperature    float32                `json:"temperature,omitempty"`
	TopP           float32                `json:"top_p,omitempty"`
	Stream         bool                   `json:"stream,omitempty"`
	Stop           []string               `json:"stop,omitempty"`
	PresencePenalty float32               `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32              `json:"frequency_penalty,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	
	// Routing preferences
	PreferredProvider Provider `json:"preferred_provider,omitempty"`
	ForbiddenProviders []Provider `json:"forbidden_providers,omitempty"`
	RequireSpeed      bool     `json:"require_speed,omitempty"`
	RequireQuality    bool     `json:"require_quality,omitempty"`
	MaxCostCents      int      `json:"max_cost_cents,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Response represents an LLM completion response
type Response struct {
	ID        string    `json:"id"`
	Object    string    `json:"object"`
	Created   int64     `json:"created"`
	Model     Model     `json:"model"`
	Provider  Provider  `json:"provider"`
	Choices   []Choice  `json:"choices"`
	Usage     Usage     `json:"usage"`
	Metrics   Metrics   `json:"metrics"`
	Fallback  bool      `json:"fallback,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	FinishReason string   `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Metrics contains performance metrics
type Metrics struct {
	Latency      time.Duration `json:"latency_ms"`
	TimeToFirst  time.Duration `json:"time_to_first_token_ms,omitempty"`
	TokensPerSec float64       `json:"tokens_per_second,omitempty"`
	CostCents    float64       `json:"cost_cents"`
	CarbonGrams  float64       `json:"carbon_grams,omitempty"`
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	APIKey          string
	Endpoint        string
	Model           Model
	MaxRetries      int
	Timeout         time.Duration
	RateLimiter     *rate.Limiter
	TokenBucket     *TokenBucket
	HealthChecker   *HealthChecker
	CostPerMillion  float64 // Cost per million tokens
	Priority        int     // Higher priority = preferred
	IsSpeedOptimized bool
	IsQualityOptimized bool
}

// TokenBucket implements token bucket algorithm for quota management
type TokenBucket struct {
	capacity   int
	tokens     int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity int, refillRate time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Take attempts to take n tokens from the bucket
func (tb *TokenBucket) Take(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed
	elapsed := time.Since(tb.lastRefill)
	refills := int(elapsed / tb.refillRate)
	if refills > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+refills)
		tb.lastRefill = tb.lastRefill.Add(time.Duration(refills) * tb.refillRate)
	}

	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}
	return false
}

// HealthChecker monitors provider health
type HealthChecker struct {
	failures       int
	successes      int
	lastCheck      time.Time
	isHealthy      bool
	backoffUntil   time.Time
	mu             sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		isHealthy: true,
		lastCheck: time.Now(),
	}
}

// RecordSuccess records a successful request
func (hc *HealthChecker) RecordSuccess() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.successes++
	hc.failures = 0
	hc.isHealthy = true
	hc.backoffUntil = time.Time{}
}

// RecordFailure records a failed request and applies backoff
func (hc *HealthChecker) RecordFailure() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.failures++
	if hc.failures >= 3 {
		hc.isHealthy = false
		// Exponential backoff: 1s, 2s, 4s, 8s, ...
		backoffSeconds := 1 << min(hc.failures-3, 6) // Max 64 seconds
		hc.backoffUntil = time.Now().Add(time.Duration(backoffSeconds) * time.Second)
	}
}

// IsHealthy checks if the provider is healthy
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	if !hc.isHealthy && time.Now().After(hc.backoffUntil) {
		// Reset health after backoff period
		hc.isHealthy = true
		hc.failures = 0
	}
	
	return hc.isHealthy
}

// Router manages multiple LLM providers with intelligent routing
type Router struct {
	providers     map[Provider]ProviderClient
	configs       map[Provider]*ProviderConfig
	fallbackChain []Provider
	logger        *zap.Logger
	metrics       *MetricsCollector
	mu            sync.RWMutex
}

// ProviderClient interface for LLM providers
type ProviderClient interface {
	Complete(ctx context.Context, req *Request) (*Response, error)
	Stream(ctx context.Context, req *Request) (<-chan *Response, error)
	Name() Provider
	IsAvailable() bool
	GetCapabilities() Capabilities
}

// Capabilities describes what a provider can do
type Capabilities struct {
	MaxTokens        int
	SupportStreaming bool
	SupportFunctions bool
	SupportVision    bool
	Languages        []string
	Models           []Model
}

// NewRouter creates a new LLM router
func NewRouter(logger *zap.Logger) *Router {
	return &Router{
		providers:     make(map[Provider]ProviderClient),
		configs:       make(map[Provider]*ProviderConfig),
		fallbackChain: []Provider{
			ProviderGroq,       // Fastest
			ProviderOpenAI,     // Most reliable
			ProviderAnthropic,  // High quality
			ProviderBedrock,    // Fallback
		},
		logger:  logger,
		metrics: NewMetricsCollector(),
	}
}

// RegisterProvider registers a new provider
func (r *Router) RegisterProvider(provider Provider, client ProviderClient, config *ProviderConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.providers[provider] = client
	r.configs[provider] = config
	
	r.logger.Info("Registered LLM provider",
		zap.String("provider", string(provider)),
		zap.Bool("speed_optimized", config.IsSpeedOptimized),
		zap.Bool("quality_optimized", config.IsQualityOptimized),
	)
}

// Route intelligently routes a request to the best available provider
func (r *Router) Route(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()
	
	// Select provider based on request requirements
	provider := r.selectProvider(req)
	
	// Try primary provider
	if provider != "" {
		if resp, err := r.tryProvider(ctx, provider, req); err == nil {
			r.recordSuccess(provider, time.Since(start))
			return resp, nil
		} else {
			r.logger.Warn("Primary provider failed",
				zap.String("provider", string(provider)),
				zap.Error(err),
			)
			r.recordFailure(provider, err)
		}
	}
	
	// Fallback chain
	for _, fallback := range r.fallbackChain {
		if r.shouldSkipProvider(fallback, req) {
			continue
		}
		
		if resp, err := r.tryProvider(ctx, fallback, req); err == nil {
			resp.Fallback = true
			r.recordSuccess(fallback, time.Since(start))
			return resp, nil
		} else {
			r.logger.Warn("Fallback provider failed",
				zap.String("provider", string(fallback)),
				zap.Error(err),
			)
			r.recordFailure(fallback, err)
		}
	}
	
	return nil, ErrNoProvidersAvailable
}

// selectProvider selects the best provider for a request
func (r *Router) selectProvider(req *Request) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Use preferred provider if specified
	if req.PreferredProvider != "" {
		if r.isProviderAvailable(req.PreferredProvider) {
			return req.PreferredProvider
		}
	}
	
	// Select based on requirements
	var candidates []Provider
	
	for provider, config := range r.configs {
		if r.shouldSkipProvider(provider, req) {
			continue
		}
		
		if !r.isProviderAvailable(provider) {
			continue
		}
		
		// Filter by requirements
		if req.RequireSpeed && !config.IsSpeedOptimized {
			continue
		}
		
		if req.RequireQuality && !config.IsQualityOptimized {
			continue
		}
		
		// Check cost constraints
		if req.MaxCostCents > 0 {
			estimatedCost := r.estimateCost(req, config)
			if estimatedCost > float64(req.MaxCostCents) {
				continue
			}
		}
		
		candidates = append(candidates, provider)
	}
	
	// Select from candidates based on priority
	if len(candidates) > 0 {
		return r.selectByPriority(candidates)
	}
	
	return ""
}

// selectByPriority selects provider with highest priority
func (r *Router) selectByPriority(providers []Provider) Provider {
	var best Provider
	var highestPriority int = -1
	
	for _, provider := range providers {
		if config := r.configs[provider]; config != nil {
			if config.Priority > highestPriority {
				highestPriority = config.Priority
				best = provider
			}
		}
	}
	
	// If same priority, randomly select to distribute load
	if highestPriority > 0 {
		samePriority := []Provider{}
		for _, provider := range providers {
			if config := r.configs[provider]; config != nil && config.Priority == highestPriority {
				samePriority = append(samePriority, provider)
			}
		}
		if len(samePriority) > 1 {
			return samePriority[rand.Intn(len(samePriority))]
		}
	}
	
	return best
}

// tryProvider attempts to use a specific provider
func (r *Router) tryProvider(ctx context.Context, provider Provider, req *Request) (*Response, error) {
	r.mu.RLock()
	client, ok := r.providers[provider]
	config := r.configs[provider]
	r.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("provider %s not registered", provider)
	}
	
	// Check rate limit
	if config.RateLimiter != nil && !config.RateLimiter.Allow() {
		return nil, ErrRateLimitExceeded
	}
	
	// Check token bucket (quota)
	if config.TokenBucket != nil && !config.TokenBucket.Take(1) {
		return nil, ErrQuotaExceeded
	}
	
	// Set timeout
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}
	
	// Make request
	start := time.Now()
	resp, err := client.Complete(ctx, req)
	
	if err != nil {
		config.HealthChecker.RecordFailure()
		return nil, err
	}
	
	// Update metrics
	resp.Metrics.Latency = time.Since(start)
	resp.Provider = provider
	
	// Calculate cost
	if config.CostPerMillion > 0 {
		resp.Metrics.CostCents = (float64(resp.Usage.TotalTokens) / 1000000.0) * config.CostPerMillion * 100
	}
	
	config.HealthChecker.RecordSuccess()
	return resp, nil
}

// shouldSkipProvider checks if a provider should be skipped
func (r *Router) shouldSkipProvider(provider Provider, req *Request) bool {
	// Check forbidden list
	for _, forbidden := range req.ForbiddenProviders {
		if provider == forbidden {
			return true
		}
	}
	return false
}

// isProviderAvailable checks if a provider is available
func (r *Router) isProviderAvailable(provider Provider) bool {
	if client, ok := r.providers[provider]; ok {
		if !client.IsAvailable() {
			return false
		}
	} else {
		return false
	}
	
	if config, ok := r.configs[provider]; ok {
		if config.HealthChecker != nil && !config.HealthChecker.IsHealthy() {
			return false
		}
	}
	
	return true
}

// estimateCost estimates the cost of a request
func (r *Router) estimateCost(req *Request, config *ProviderConfig) float64 {
	// Rough estimation based on message length
	totalChars := 0
	for _, msg := range req.Messages {
		totalChars += len(msg.Content)
	}
	
	// Estimate tokens (rough: 1 token ≈ 4 chars)
	estimatedTokens := totalChars / 4
	if req.MaxTokens > 0 {
		estimatedTokens += req.MaxTokens
	}
	
	return (float64(estimatedTokens) / 1000000.0) * config.CostPerMillion * 100
}

// recordSuccess records a successful request
func (r *Router) recordSuccess(provider Provider, latency time.Duration) {
	r.metrics.RecordSuccess(provider, latency)
}

// recordFailure records a failed request
func (r *Router) recordFailure(provider Provider, err error) {
	r.metrics.RecordFailure(provider, err)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}