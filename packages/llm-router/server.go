package llmrouter

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Server represents the LLM Router HTTP server
type Server struct {
	router      *Router
	engine      *gin.Engine
	logger      *zap.Logger
	redisClient *redis.Client
	port        string
}

// NewServer creates a new LLM Router server
func NewServer(port string, logger *zap.Logger, redisClient *redis.Client) *Server {
	// Set Gin to release mode in production
	gin.SetMode(gin.ReleaseMode)
	
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(LoggerMiddleware(logger))
	engine.Use(CORSMiddleware())
	
	s := &Server{
		router:      NewRouter(logger),
		engine:      engine,
		logger:      logger,
		redisClient: redisClient,
		port:        port,
	}
	
	s.setupRoutes()
	s.initializeProviders()
	
	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health and metrics
	s.engine.GET("/health", s.handleHealth)
	s.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	s.engine.GET("/ready", s.handleReadiness)
	
	// API v1 routes
	v1 := s.engine.Group("/api/v1")
	{
		// Completion endpoints
		v1.POST("/complete", s.handleComplete)
		v1.POST("/stream", s.handleStream)
		
		// Provider management
		v1.GET("/providers", s.handleListProviders)
		v1.GET("/providers/:name/status", s.handleProviderStatus)
		
		// Model information
		v1.GET("/models", s.handleListModels)
		v1.GET("/models/:name", s.handleModelInfo)
		
		// Cost estimation
		v1.POST("/estimate", s.handleEstimateCost)
		
		// Usage and billing
		v1.GET("/usage", s.handleGetUsage)
	}
	
	// Admin routes (protected)
	admin := s.engine.Group("/admin")
	admin.Use(AuthMiddleware())
	{
		admin.POST("/providers/:name/enable", s.handleEnableProvider)
		admin.POST("/providers/:name/disable", s.handleDisableProvider)
		admin.PUT("/providers/:name/config", s.handleUpdateProviderConfig)
		admin.GET("/stats", s.handleGetStats)
	}
}

// initializeProviders sets up all LLM provider clients
func (s *Server) initializeProviders() {
	ctx := context.Background()
	
	// Initialize OpenAI
	if apiKey := getEnv("OPENAI_API_KEY", ""); apiKey != "" {
		client := NewOpenAIClient(apiKey, s.logger)
		config := &ProviderConfig{
			APIKey:             apiKey,
			Model:              ModelGPT4Turbo,
			MaxRetries:         3,
			Timeout:            30 * time.Second,
			RateLimiter:        NewRateLimiter(100, 1*time.Minute), // 100 req/min
			TokenBucket:        NewTokenBucket(1000000, 1*time.Hour), // 1M tokens/hour
			HealthChecker:      NewHealthChecker(),
			CostPerMillion:     10.0, // $10 per million tokens
			Priority:           8,
			IsQualityOptimized: true,
		}
		s.router.RegisterProvider(ProviderOpenAI, client, config)
		s.logger.Info("Initialized OpenAI provider")
	}
	
	// Initialize Anthropic
	if apiKey := getEnv("ANTHROPIC_API_KEY", ""); apiKey != "" {
		client := NewAnthropicClient(apiKey, s.logger)
		config := &ProviderConfig{
			APIKey:             apiKey,
			Model:              ModelClaude3Opus,
			MaxRetries:         3,
			Timeout:            30 * time.Second,
			RateLimiter:        NewRateLimiter(50, 1*time.Minute),
			TokenBucket:        NewTokenBucket(500000, 1*time.Hour),
			HealthChecker:      NewHealthChecker(),
			CostPerMillion:     15.0, // $15 per million tokens
			Priority:           9,
			IsQualityOptimized: true,
		}
		s.router.RegisterProvider(ProviderAnthropic, client, config)
		s.logger.Info("Initialized Anthropic provider")
	}
	
	// Initialize Groq (fast inference)
	if apiKey := getEnv("GROQ_API_KEY", ""); apiKey != "" {
		client := NewGroqClient(apiKey, s.logger)
		config := &ProviderConfig{
			APIKey:            apiKey,
			Model:             ModelLlama3_70B,
			MaxRetries:        3,
			Timeout:           10 * time.Second, // Faster timeout
			RateLimiter:       NewRateLimiter(200, 1*time.Minute), // Higher rate
			TokenBucket:       NewTokenBucket(2000000, 1*time.Hour),
			HealthChecker:     NewHealthChecker(),
			CostPerMillion:    0.7, // $0.70 per million tokens (much cheaper)
			Priority:          10,   // Highest priority for speed
			IsSpeedOptimized:  true,
		}
		s.router.RegisterProvider(ProviderGroq, client, config)
		s.logger.Info("Initialized Groq provider")
	}
	
	// Initialize AWS Bedrock
	if region := getEnv("AWS_BEDROCK_REGION", ""); region != "" {
		client := NewBedrockClient(region, s.logger)
		config := &ProviderConfig{
			Model:              ModelClaudeBedrock,
			MaxRetries:         3,
			Timeout:            30 * time.Second,
			RateLimiter:        NewRateLimiter(60, 1*time.Minute),
			TokenBucket:        NewTokenBucket(1000000, 1*time.Hour),
			HealthChecker:      NewHealthChecker(),
			CostPerMillion:     8.0,
			Priority:           6,
		}
		s.router.RegisterProvider(ProviderBedrock, client, config)
		s.logger.Info("Initialized AWS Bedrock provider")
	}
	
	// Cache warmup
	s.warmupCache(ctx)
}

// handleComplete handles completion requests
func (s *Server) handleComplete(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Generate request ID if not provided
	if req.ID == "" {
		req.ID = generateRequestID()
	}
	
	// Check cache first
	if cached := s.checkCache(c.Request.Context(), &req); cached != nil {
		c.JSON(http.StatusOK, cached)
		return
	}
	
	// Route to provider
	resp, err := s.router.Route(c.Request.Context(), &req)
	if err != nil {
		s.logger.Error("Failed to route request",
			zap.String("request_id", req.ID),
			zap.Error(err),
		)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": err.Error(),
			"request_id": req.ID,
		})
		return
	}
	
	// Cache successful responses
	s.cacheResponse(c.Request.Context(), &req, resp)
	
	// Record usage
	s.recordUsage(c, resp)
	
	c.JSON(http.StatusOK, resp)
}

// handleStream handles streaming completion requests
func (s *Server) handleStream(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	req.Stream = true
	
	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	
	// Create response channel
	provider := s.router.selectProvider(&req)
	if provider == "" {
		c.SSEvent("error", "No providers available")
		return
	}
	
	client := s.router.providers[provider]
	respChan, err := client.Stream(c.Request.Context(), &req)
	if err != nil {
		c.SSEvent("error", err.Error())
		return
	}
	
	// Stream responses
	for resp := range respChan {
		c.SSEvent("message", resp)
		c.Writer.Flush()
	}
	
	c.SSEvent("done", "")
}

// handleListProviders returns available providers
func (s *Server) handleListProviders(c *gin.Context) {
	providers := []gin.H{}
	
	s.router.mu.RLock()
	defer s.router.mu.RUnlock()
	
	for name, client := range s.router.providers {
		config := s.router.configs[name]
		providers = append(providers, gin.H{
			"name":       string(name),
			"available":  client.IsAvailable(),
			"healthy":    config.HealthChecker.IsHealthy(),
			"priority":   config.Priority,
			"speed_optimized": config.IsSpeedOptimized,
			"quality_optimized": config.IsQualityOptimized,
			"capabilities": client.GetCapabilities(),
		})
	}
	
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// handleProviderStatus returns detailed provider status
func (s *Server) handleProviderStatus(c *gin.Context) {
	providerName := Provider(c.Param("name"))
	
	s.router.mu.RLock()
	client, exists := s.router.providers[providerName]
	config := s.router.configs[providerName]
	s.router.mu.RUnlock()
	
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}
	
	status := gin.H{
		"name":      string(providerName),
		"available": client.IsAvailable(),
		"healthy":   config.HealthChecker.IsHealthy(),
		"metrics":   s.router.metrics.GetProviderMetrics(providerName),
	}
	
	c.JSON(http.StatusOK, status)
}

// handleListModels returns available models
func (s *Server) handleListModels(c *gin.Context) {
	models := []gin.H{
		// OpenAI
		{"provider": "openai", "model": "gpt-4-turbo-preview", "context": 128000, "cost_per_million": 10.0},
		{"provider": "openai", "model": "gpt-4", "context": 8192, "cost_per_million": 30.0},
		{"provider": "openai", "model": "gpt-3.5-turbo", "context": 16385, "cost_per_million": 0.5},
		
		// Anthropic
		{"provider": "anthropic", "model": "claude-3-opus", "context": 200000, "cost_per_million": 15.0},
		{"provider": "anthropic", "model": "claude-3-sonnet", "context": 200000, "cost_per_million": 3.0},
		{"provider": "anthropic", "model": "claude-3-haiku", "context": 200000, "cost_per_million": 0.25},
		
		// Groq
		{"provider": "groq", "model": "llama3-70b", "context": 8192, "cost_per_million": 0.7},
		{"provider": "groq", "model": "llama3-8b", "context": 8192, "cost_per_million": 0.05},
		{"provider": "groq", "model": "mixtral-8x7b", "context": 32768, "cost_per_million": 0.27},
	}
	
	c.JSON(http.StatusOK, gin.H{"models": models})
}

// handleModelInfo returns detailed model information
func (s *Server) handleModelInfo(c *gin.Context) {
	modelName := c.Param("name")
	
	// Return model details
	c.JSON(http.StatusOK, gin.H{
		"model": modelName,
		"info": "Model information endpoint",
	})
}

// handleEstimateCost estimates the cost of a request
func (s *Server) handleEstimateCost(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	estimates := []gin.H{}
	
	s.router.mu.RLock()
	defer s.router.mu.RUnlock()
	
	for provider, config := range s.router.configs {
		cost := s.router.estimateCost(&req, config)
		estimates = append(estimates, gin.H{
			"provider": string(provider),
			"estimated_cost_cents": cost,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{"estimates": estimates})
}

// handleGetUsage returns usage statistics
func (s *Server) handleGetUsage(c *gin.Context) {
	// Get user/org from context (set by auth middleware)
	userID := c.GetString("user_id")
	orgID := c.GetString("org_id")
	
	usage := s.getUsageStats(c.Request.Context(), userID, orgID)
	c.JSON(http.StatusOK, usage)
}

// handleHealth returns service health status
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "llm-router",
		"timestamp": time.Now().Unix(),
	})
}

// handleReadiness checks if service is ready
func (s *Server) handleReadiness(c *gin.Context) {
	// Check if at least one provider is available
	hasProvider := false
	s.router.mu.RLock()
	for _, client := range s.router.providers {
		if client.IsAvailable() {
			hasProvider = true
			break
		}
	}
	s.router.mu.RUnlock()
	
	if !hasProvider {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"reason": "no providers available",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"providers_count": len(s.router.providers),
	})
}

// Admin handlers

func (s *Server) handleEnableProvider(c *gin.Context) {
	providerName := Provider(c.Param("name"))
	// Implementation for enabling provider
	c.JSON(http.StatusOK, gin.H{"message": "Provider enabled", "provider": providerName})
}

func (s *Server) handleDisableProvider(c *gin.Context) {
	providerName := Provider(c.Param("name"))
	// Implementation for disabling provider
	c.JSON(http.StatusOK, gin.H{"message": "Provider disabled", "provider": providerName})
}

func (s *Server) handleUpdateProviderConfig(c *gin.Context) {
	providerName := Provider(c.Param("name"))
	// Implementation for updating provider config
	c.JSON(http.StatusOK, gin.H{"message": "Config updated", "provider": providerName})
}

func (s *Server) handleGetStats(c *gin.Context) {
	stats := s.router.metrics.GetAllMetrics()
	c.JSON(http.StatusOK, stats)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting LLM Router server", zap.String("port", s.port))
	return s.engine.Run(":" + s.port)
}

// Helper functions

func (s *Server) checkCache(ctx context.Context, req *Request) *Response {
	// Implementation for cache checking
	return nil
}

func (s *Server) cacheResponse(ctx context.Context, req *Request, resp *Response) {
	// Implementation for caching responses
}

func (s *Server) warmupCache(ctx context.Context) {
	// Implementation for cache warmup
}

func (s *Server) recordUsage(c *gin.Context, resp *Response) {
	// Implementation for recording usage
}

func (s *Server) getUsageStats(ctx context.Context, userID, orgID string) gin.H {
	// Implementation for getting usage stats
	return gin.H{
		"user_id": userID,
		"org_id": orgID,
		"tokens_used": 0,
		"requests_count": 0,
		"cost_cents": 0,
	}
}