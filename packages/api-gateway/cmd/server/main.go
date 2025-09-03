package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/QuantumLayer-dev/quantumlayer-platform/packages/api-gateway/internal/proxy"
    "github.com/QuantumLayer-dev/quantumlayer-platform/packages/shared/config"
    "github.com/QuantumLayer-dev/quantumlayer-platform/packages/shared/telemetry"
    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

func main() {
    // Initialize logger
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{})
    logger.SetLevel(logrus.InfoLevel)

    // Load configuration
    cfg, err := config.Load("api-gateway")
    if err != nil {
        logger.WithError(err).Fatal("Failed to load configuration")
    }

    // Initialize telemetry
    tracer, cleanup, err := telemetry.InitTracer(
        "api-gateway",
        cfg.Tracing.Endpoint,
        cfg.Tracing.SamplingRate,
    )
    if err != nil {
        logger.WithError(err).Fatal("Failed to initialize telemetry")
    }
    defer cleanup()
    _ = tracer

    // Initialize proxy handler
    proxyHandler := proxy.NewProxyHandler()

    // Setup Gin router
    router := gin.New()
    router.Use(gin.Recovery())
    router.Use(corsMiddleware())

    // Health endpoints
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "healthy",
            "service": "api-gateway",
            "version": "2.0.0",
        })
    })

    router.GET("/ready", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ready"})
    })

    // GraphQL endpoint - forward to appropriate service
    router.POST("/graphql", func(c *gin.Context) {
        // For now, return service status
        proxyHandler.GetServiceStatus(c)
    })

    // API v1 endpoints for REST compatibility
    v1 := router.Group("/api/v1")
    {
        // Service status endpoint
        v1.GET("/status", proxyHandler.GetServiceStatus)

        // Workflow generation endpoints
        v1.POST("/generate", proxyHandler.ProxyToWorkflow)
        
        // Workflow endpoints - proxy to workflow-api
        workflows := v1.Group("/workflows")
        {
            // Specific routes must come before wildcard routes
            workflows.POST("/generate", proxyHandler.ProxyToWorkflow)
            workflows.POST("/generate-extended", proxyHandler.ProxyToWorkflowExtended)
            // Remove wildcard routes as they conflict with specific routes
            // For additional workflow endpoints, add them explicitly
        }
        
        // LLM Router endpoints
        llm := v1.Group("/llm")
        {
            llm.POST("/generate", proxyHandler.ProxyToLLMRouter)
            llm.POST("/stream", proxyHandler.ProxyToLLMRouter)
            // Remove wildcard route to avoid conflicts
        }
        
        // Agent Orchestrator endpoints
        agents := v1.Group("/agents")
        {
            agents.POST("/create", proxyHandler.ProxyToAgentOrchestrator)
            agents.GET("/list", proxyHandler.ProxyToAgentOrchestrator)
            agents.GET("/status", proxyHandler.ProxyToAgentOrchestrator)
            agents.POST("/execute", proxyHandler.ProxyToAgentOrchestrator)
        }
        
        // Meta Prompt Engine endpoints
        prompts := v1.Group("/prompts")
        {
            prompts.POST("/generate", proxyHandler.ProxyToMetaPromptEngine)
            prompts.POST("/optimize", proxyHandler.ProxyToMetaPromptEngine)
            prompts.GET("/templates", proxyHandler.ProxyToMetaPromptEngine)
        }
        
        // Parser endpoints
        parser := v1.Group("/parser")
        {
            parser.POST("/parse", proxyHandler.ProxyToParser)
            parser.POST("/validate", proxyHandler.ProxyToParser)
            parser.POST("/transform", proxyHandler.ProxyToParser)
        }
    }

    // Create HTTP server
    httpServer := &http.Server{
        Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
        Handler: router,
    }

    // Start server
    go func() {
        logger.WithField("port", cfg.Server.Port).Info("Starting API Gateway")
        if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.WithError(err).Fatal("Failed to start server")
        }
    }()

    // Wait for interrupt
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := httpServer.Shutdown(ctx); err != nil {
        logger.WithError(err).Error("Server forced to shutdown")
    }

    logger.Info("Server exited")
}

func corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}