package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	
	orchestrator "github.com/QuantumLayer-dev/quantumlayer-platform/packages/agent-orchestrator"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Get configuration from environment
	port := getEnv("PORT", "8083")
	redisURL := getEnv("REDIS_URL", "redis://redis.quantumlayer.svc.cluster.local:6379")
	temporalHost := getEnv("TEMPORAL_HOST", "")
	maxAgents := getEnvInt("MAX_AGENTS", 10)
	maxTasksPerAgent := getEnvInt("MAX_TASKS_PER_AGENT", 5)
	
	// Create orchestrator config
	config := &orchestrator.OrchestratorConfig{
		MaxAgents:           maxAgents,
		MaxTasksPerAgent:    maxTasksPerAgent,
		TaskTimeout:         30 * time.Second,
		AgentSpawnTimeout:   5 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		RedisURL:            redisURL,
		TemporalHost:        temporalHost,
		MetricsEnabled:      true,
	}
	
	// Create and start server
	server, err := orchestrator.NewServer(port, config, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}
	
	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	
	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()
	
	logger.Info("Agent Orchestrator started", 
		zap.String("port", port),
		zap.Int("max_agents", maxAgents),
	)
	
	// Wait for shutdown signal
	<-shutdown
	
	logger.Info("Shutting down server...")
	
	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := server.Stop(); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}
	
	// Wait for context to expire
	<-ctx.Done()
	
	logger.Info("Server shutdown complete")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		// Simple conversion, ignoring errors for MVP
		switch value {
		case "1": return 1
		case "2": return 2
		case "3": return 3
		case "5": return 5
		case "10": return 10
		case "20": return 20
		case "50": return 50
		case "100": return 100
		default: return defaultValue
		}
	}
	return defaultValue
}