package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/meta-prompt-engine/internal/api"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/meta-prompt-engine/internal/engine"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/meta-prompt-engine/internal/templates"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// LLM Router URL from environment or default
	llmRouterURL := os.Getenv("LLM_ROUTER_URL")
	if llmRouterURL == "" {
		llmRouterURL = "http://llm-router.quantumlayer.svc.cluster.local:8080"
	}

	// Create real LLM client pointing to LLM Router service
	llmClient := engine.NewRealLLMClient(llmRouterURL, logger)
	logger.WithField("url", llmRouterURL).Info("Using real LLM Router client")

	// Initialize Meta Prompt Engine
	metaEngine := engine.NewMetaPromptEngine(llmClient, logger)

	// Register built-in templates
	for _, tmpl := range templates.GetBuiltinTemplates() {
		if err := metaEngine.RegisterTemplate(tmpl); err != nil {
			logger.WithError(err).WithField("template", tmpl.Name).Error("Failed to register template")
		}
	}

	logger.Info("Registered built-in prompt templates")

	// Setup Gin router
	router := setupRouter(metaEngine, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.WithField("port", port).Info("Starting Meta Prompt Engine server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Server exited")
}

func setupRouter(engine *engine.MetaPromptEngine, logger *logrus.Logger) *gin.Engine {
	router := gin.New()
	
	// Add middleware
	router.Use(gin.Recovery())
	// TODO: Add telemetry middleware when implemented
	// router.Use(telemetry.GinMiddleware())
	
	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// CRITICAL: Main enhancement endpoint expected by workflows
	router.POST("/enhance", api.EnhanceHandler(engine, logger))
	
	// API routes v1
	v1 := router.Group("/api/v1")
	{
		// Status endpoint
		v1.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service": "meta-prompt-engine",
				"version": "2.5.1",
				"status": "operational",
				"features": []string{
					"prompt_enhancement",
					"template_system",
					"optimization",
					"metrics",
				},
			})
		})
		
		// Enhancement endpoint (v1 version)
		v1.POST("/enhance", api.EnhanceHandler(engine, logger))
		
		// Template management
		v1.GET("/templates", api.TemplateHandler(engine, logger))
		
		// Metrics and monitoring
		v1.GET("/metrics", api.MetricsHandler(engine, logger))
		
		// Template management - TODO
		// v1.POST("/templates", api.RegisterTemplate(engine))
		// v1.GET("/templates", api.ListTemplates(engine))
		// v1.GET("/templates/:id", api.GetTemplate(engine))
		// v1.PUT("/templates/:id", api.UpdateTemplate(engine))
		// v1.DELETE("/templates/:id", api.DeleteTemplate(engine))
		
		// Template execution - TODO
		// v1.POST("/templates/:id/execute", api.ExecuteTemplate(engine))
		
		// Chain management - TODO
		// v1.POST("/chains", api.CreateChain(engine))
		// v1.GET("/chains", api.ListChains(engine))
		// v1.POST("/chains/:id/execute", api.ExecuteChain(engine))
		
		// A/B testing - TODO
		// v1.POST("/ab-tests", api.StartABTest(engine))
		// v1.GET("/ab-tests/:id", api.GetABTestResults(engine))
		// v1.PUT("/ab-tests/:id/stop", api.StopABTest(engine))
		
		// Feedback - TODO
		// v1.POST("/executions/:id/feedback", api.RecordFeedback(engine))
		
		// Recommendations - TODO
		// v1.GET("/recommendations", api.GetRecommendations(engine))
	}

	return router
}

// MockLLMClient is a temporary mock implementation
type MockLLMClient struct {
	logger *logrus.Logger
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string, model string) (string, int, error) {
	// Simulate LLM response
	m.logger.WithFields(logrus.Fields{
		"prompt_length": len(prompt),
		"model":         model,
	}).Debug("Mock LLM completion called")
	
	// Return mock response based on prompt content
	if strings.Contains(prompt, "code") {
		return "```python\ndef example_function():\n    return 'Hello, World!'\n```", 50, nil
	}
	
	return "This is a mock response from the LLM.", 10, nil
}