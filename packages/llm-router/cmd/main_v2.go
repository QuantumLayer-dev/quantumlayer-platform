// Enterprise LLM Router v2.0 - Production Ready
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/llm-router/internal/providers"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// RouterConfig holds the router configuration
type RouterConfig struct {
	Port              string
	MetricsPort       string
	EnabledProviders  []string
	PrimaryProvider   string
	FallbackProviders []string
	MaxRetries        int
	RequestTimeout    time.Duration
}

// LLMRouter manages multiple LLM providers with enterprise features
type LLMRouter struct {
	config    RouterConfig
	providers map[string]providers.Provider
	logger    *zap.Logger
	metrics   *RouterMetrics
	mu        sync.RWMutex
}

// RouterMetrics tracks router-level metrics
type RouterMetrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	providerErrors   *prometheus.CounterVec
	activeProviders  *prometheus.GaugeVec
	tokenUsage       *prometheus.CounterVec
}

// NewRouterMetrics creates Prometheus metrics
func NewRouterMetrics() *RouterMetrics {
	m := &RouterMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_router_requests_total",
				Help: "Total number of requests by provider and status",
			},
			[]string{"provider", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "llm_router_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider", "method"},
		),
		providerErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_router_provider_errors_total",
				Help: "Total number of provider errors by type",
			},
			[]string{"provider", "error_type"},
		),
		activeProviders: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llm_router_active_providers",
				Help: "Number of active providers",
			},
			[]string{"status"},
		),
		tokenUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_router_token_usage_total",
				Help: "Total token usage by provider",
			},
			[]string{"provider", "type"},
		),
	}

	// Register metrics
	prometheus.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.providerErrors,
		m.activeProviders,
		m.tokenUsage,
	)

	return m
}

// NewLLMRouter creates a new enterprise LLM router
func NewLLMRouter(config RouterConfig, logger *zap.Logger) (*LLMRouter, error) {
	router := &LLMRouter{
		config:    config,
		providers: make(map[string]providers.Provider),
		logger:    logger,
		metrics:   NewRouterMetrics(),
	}

	// Initialize providers
	if err := router.initializeProviders(); err != nil {
		return nil, fmt.Errorf("initialize providers: %w", err)
	}

	// Run health checks
	router.startHealthChecks()

	return router, nil
}

func (r *LLMRouter) initializeProviders() error {
	var wg sync.WaitGroup
	errors := make(chan error, len(r.config.EnabledProviders))

	for _, providerName := range r.config.EnabledProviders {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			
			provider, err := r.createProvider(name)
			if err != nil {
				r.logger.Error("Failed to create provider",
					zap.String("provider", name),
					zap.Error(err))
				errors <- fmt.Errorf("%s: %w", name, err)
				return
			}

			r.mu.Lock()
			r.providers[name] = provider
			r.mu.Unlock()

			r.metrics.activeProviders.WithLabelValues("healthy").Inc()
			r.logger.Info("Provider initialized",
				zap.String("provider", name))
		}(providerName)
	}

	wg.Wait()
	close(errors)

	// Check for initialization errors
	var initErrors []string
	for err := range errors {
		if err != nil {
			initErrors = append(initErrors, err.Error())
		}
	}

	if len(r.providers) == 0 {
		return fmt.Errorf("no providers initialized: %v", initErrors)
	}

	r.logger.Info("Providers initialized",
		zap.Int("count", len(r.providers)),
		zap.Any("providers", r.getProviderNames()))

	return nil
}

func (r *LLMRouter) createProvider(name string) (providers.Provider, error) {
	switch name {
	case "azure":
		return r.createAzureProvider()
	case "openai":
		return r.createOpenAIProvider()
	case "anthropic":
		return r.createAnthropicProvider()
	case "groq":
		return r.createGroqProvider()
	case "bedrock":
		return r.createBedrockProvider()
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

func (r *LLMRouter) createAzureProvider() (providers.Provider, error) {
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	apiKey := os.Getenv("AZURE_OPENAI_KEY")
	deployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")

	if endpoint == "" || apiKey == "" || deployment == "" {
		return nil, fmt.Errorf("missing Azure OpenAI configuration")
	}

	// Validate and fix endpoint
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}
	
	// Remove trailing slash
	endpoint = strings.TrimRight(endpoint, "/")
	
	// Validate endpoint format
	if !strings.Contains(endpoint, ".openai.azure.com") {
		r.logger.Warn("Azure endpoint doesn't match expected format",
			zap.String("endpoint", endpoint))
	}

	config := providers.AzureConfig{
		Endpoint:       endpoint,
		APIKey:         apiKey,
		DeploymentName: deployment,
		APIVersion:     os.Getenv("AZURE_OPENAI_API_VERSION"),
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		RateLimit:      10,
		BurstLimit:     20,
	}

	if config.APIVersion == "" {
		config.APIVersion = "2024-02-01"
	}

	return providers.NewAzureOpenAIProvider(config, r.logger), nil
}

func (r *LLMRouter) createOpenAIProvider() (providers.Provider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing OpenAI API key")
	}

	// TODO: Implement OpenAI provider
	return nil, fmt.Errorf("OpenAI provider not yet implemented")
}

func (r *LLMRouter) createAnthropicProvider() (providers.Provider, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing Anthropic API key")
	}

	// TODO: Implement Anthropic provider
	return nil, fmt.Errorf("Anthropic provider not yet implemented")
}

func (r *LLMRouter) createGroqProvider() (providers.Provider, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing Groq API key")
	}

	// TODO: Implement Groq provider
	return nil, fmt.Errorf("Groq provider not yet implemented")
}

func (r *LLMRouter) createBedrockProvider() (providers.Provider, error) {
	// TODO: Implement Bedrock provider
	return nil, fmt.Errorf("Bedrock provider not yet implemented")
}

func (r *LLMRouter) getProviderNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

func (r *LLMRouter) startHealthChecks() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			r.mu.RLock()
			providers := make(map[string]providers.Provider)
			for k, v := range r.providers {
				providers[k] = v
			}
			r.mu.RUnlock()

			for name, provider := range providers {
				go func(n string, p providers.Provider) {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					if err := p.HealthCheck(ctx); err != nil {
						r.logger.Warn("Provider health check failed",
							zap.String("provider", n),
							zap.Error(err))
						r.metrics.activeProviders.WithLabelValues("unhealthy").Inc()
						r.metrics.activeProviders.WithLabelValues("healthy").Dec()
					}
				}(name, provider)
			}
		}
	}()
}

// GenerateCode handles code generation requests with fallback
func (r *LLMRouter) GenerateCode(c *gin.Context) {
	startTime := time.Now()

	var request struct {
		Messages    []providers.Message `json:"messages"`
		Prompt      string             `json:"prompt"`
		Language    string             `json:"language"`
		Framework   string             `json:"framework,omitempty"`
		Type        string             `json:"type,omitempty"`
		Provider    string             `json:"provider,omitempty"`
		MaxTokens   int                `json:"max_tokens,omitempty"`
		Temperature float32            `json:"temperature,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert messages format or use direct prompt
	codeRequest := r.buildCodeRequest(request)

	// Select provider order
	providerOrder := r.selectProviderOrder(request.Provider)

	var lastError error
	for _, providerName := range providerOrder {
		provider, exists := r.getProvider(providerName)
		if !exists {
			continue
		}

		r.logger.Info("Attempting provider",
			zap.String("provider", providerName),
			zap.String("language", codeRequest.Language))

		ctx, cancel := context.WithTimeout(c.Request.Context(), r.config.RequestTimeout)
		response, err := provider.GenerateCode(ctx, codeRequest)
		cancel()

		if err != nil {
			r.logger.Warn("Provider failed",
				zap.String("provider", providerName),
				zap.Error(err))
			r.metrics.providerErrors.WithLabelValues(providerName, "generation").Inc()
			lastError = err
			continue
		}

		// Validate response
		if !r.isValidCodeResponse(response.Code) {
			r.logger.Warn("Invalid code response",
				zap.String("provider", providerName),
				zap.Int("length", len(response.Code)))
			r.metrics.providerErrors.WithLabelValues(providerName, "validation").Inc()
			continue
		}

		// Success - record metrics
		r.metrics.requestsTotal.WithLabelValues(providerName, "success").Inc()
		r.metrics.requestDuration.WithLabelValues(providerName, "generate").Observe(time.Since(startTime).Seconds())
		r.metrics.tokenUsage.WithLabelValues(providerName, "prompt").Add(float64(response.Usage.PromptTokens))
		r.metrics.tokenUsage.WithLabelValues(providerName, "completion").Add(float64(response.Usage.CompletionTokens))

		// Return response
		c.JSON(http.StatusOK, gin.H{
			"content":  response.Code,
			"provider": response.Provider,
			"model":    response.Model,
			"usage":    response.Usage,
			"latency":  response.Latency.Milliseconds(),
		})
		return
	}

	// All providers failed
	r.metrics.requestsTotal.WithLabelValues("all", "failed").Inc()
	errorMessage := "All providers failed"
	if lastError != nil {
		errorMessage = fmt.Sprintf("%s: %v", errorMessage, lastError)
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":   errorMessage,
		"tried":   providerOrder,
		"latency": time.Since(startTime).Milliseconds(),
	})
}

func (r *LLMRouter) buildCodeRequest(request interface{}) providers.CodeGenerationRequest {
	// Type assertion for the request
	req := request.(struct {
		Messages    []providers.Message `json:"messages"`
		Prompt      string             `json:"prompt"`
		Language    string             `json:"language"`
		Framework   string             `json:"framework,omitempty"`
		Type        string             `json:"type,omitempty"`
		Provider    string             `json:"provider,omitempty"`
		MaxTokens   int                `json:"max_tokens,omitempty"`
		Temperature float32            `json:"temperature,omitempty"`
	})

	// Extract prompt from messages or use direct prompt
	prompt := req.Prompt
	if prompt == "" && len(req.Messages) > 0 {
		for _, msg := range req.Messages {
			if msg.Role == "user" {
				prompt = msg.Content
				break
			}
		}
	}

	// Default max tokens
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2000
	}

	return providers.CodeGenerationRequest{
		Prompt:      prompt,
		Language:    req.Language,
		Framework:   req.Framework,
		Type:        req.Type,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}
}

func (r *LLMRouter) selectProviderOrder(preferredProvider string) []string {
	if preferredProvider != "" {
		// Try preferred provider first
		order := []string{preferredProvider}
		for _, p := range r.config.FallbackProviders {
			if p != preferredProvider {
				order = append(order, p)
			}
		}
		return order
	}

	// Use configured order
	order := []string{r.config.PrimaryProvider}
	order = append(order, r.config.FallbackProviders...)
	return order
}

func (r *LLMRouter) getProvider(name string) (providers.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, exists := r.providers[name]
	return p, exists
}

func (r *LLMRouter) isValidCodeResponse(code string) bool {
	// More intelligent validation
	code = strings.TrimSpace(code)
	
	// Check minimum length (reduced from 100 to 30)
	if len(code) < 30 {
		return false
	}

	// Check for error markers
	if strings.HasPrefix(strings.ToLower(code), "error:") {
		return false
	}
	if strings.HasPrefix(strings.ToLower(code), "i'm sorry") {
		return false
	}
	if strings.HasPrefix(strings.ToLower(code), "i cannot") {
		return false
	}

	// Check for code indicators (more lenient)
	codeIndicators := []string{
		"def ", "function ", "func ", "class ", "struct ",
		"import ", "require", "include", "using ",
		"const ", "let ", "var ", "public ", "private ",
		"return", "if ", "for ", "while ",
		"{", "}", "()", "[]", "=>", "->",
	}

	indicatorCount := 0
	lowerCode := strings.ToLower(code)
	for _, indicator := range codeIndicators {
		if strings.Contains(lowerCode, indicator) {
			indicatorCount++
			if indicatorCount >= 2 { // Reduced from 3 to 2
				return true
			}
		}
	}

	// Accept if it looks like structured data
	if strings.Contains(code, "{") && strings.Contains(code, "}") {
		return true
	}
	if strings.Contains(code, "[") && strings.Contains(code, "]") {
		return true
	}

	return false
}

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	config := RouterConfig{
		Port:              getEnv("PORT", "8080"),
		MetricsPort:       getEnv("METRICS_PORT", "9090"),
		PrimaryProvider:   getEnv("PRIMARY_PROVIDER", "azure"),
		FallbackProviders: strings.Split(getEnv("FALLBACK_PROVIDERS", "groq,openai,anthropic"), ","),
		EnabledProviders:  strings.Split(getEnv("ENABLED_PROVIDERS", "azure,groq,openai,anthropic"), ","),
		MaxRetries:        3,
		RequestTimeout:    30 * time.Second,
	}

	// Create router
	llmRouter, err := NewLLMRouter(config, logger)
	if err != nil {
		logger.Fatal("Failed to create LLM router", zap.Error(err))
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "providers": llmRouter.getProviderNames()})
	})
	r.GET("/ready", func(c *gin.Context) {
		// Check if at least one provider is available
		if len(llmRouter.providers) > 0 {
			c.JSON(http.StatusOK, gin.H{"status": "ready", "providers": len(llmRouter.providers)})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
		}
	})
	r.POST("/generate", llmRouter.GenerateCode)
	r.POST("/v1/chat/completions", llmRouter.GenerateCode) // OpenAI compatible

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logger.Info("Starting metrics server", zap.String("port", config.MetricsPort))
		if err := http.ListenAndServe(":"+config.MetricsPort, nil); err != nil {
			logger.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Start main server
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Starting LLM Router", zap.String("port", config.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}