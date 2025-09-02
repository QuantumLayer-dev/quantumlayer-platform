package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

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

    // GraphQL endpoint (placeholder for now)
    router.POST("/graphql", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "data": gin.H{
                "systemStatus": gin.H{
                    "version": "2.0.0",
                    "uptime": 86400,
                    "activeAgents": 8,
                    "queuedTasks": 3,
                    "completedToday": 127,
                },
            },
        })
    })

    // API v1 endpoints for REST compatibility
    v1 := router.Group("/api/v1")
    {
        v1.GET("/status", func(c *gin.Context) {
            c.JSON(http.StatusOK, gin.H{
                "platform": "QuantumLayer",
                "version": "2.0.0",
                "services": gin.H{
                    "llm-router": "healthy",
                    "agent-orchestrator": "healthy",
                    "meta-prompt-engine": "healthy",
                    "temporal": "healthy",
                },
            })
        })

        v1.POST("/generate", func(c *gin.Context) {
            var req struct {
                Prompt   string `json:"prompt"`
                Language string `json:"language"`
            }
            if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
            }

            c.JSON(http.StatusOK, gin.H{
                "id": "gen-" + fmt.Sprintf("%d", time.Now().Unix()),
                "status": "processing",
                "message": "Code generation started",
            })
        })
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