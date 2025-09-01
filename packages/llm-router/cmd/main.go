package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	llmrouter "github.com/QuantumLayer-dev/quantumlayer-platform/packages/llm-router"
)

func main() {
	var (
		port      = flag.String("port", getEnv("PORT", "8080"), "Server port")
		redisURL  = flag.String("redis", getEnv("REDIS_URL", "redis://localhost:6379"), "Redis URL")
		logLevel  = flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level")
		env       = flag.String("env", getEnv("ENVIRONMENT", "development"), "Environment")
	)
	flag.Parse()

	// Initialize logger
	logger := initLogger(*logLevel, *env)
	defer logger.Sync()

	// Initialize Redis client
	opt, err := redis.ParseURL(*redisURL)
	if err != nil {
		logger.Fatal("Failed to parse Redis URL", zap.Error(err))
	}
	
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	
	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis connection failed, caching disabled", zap.Error(err))
		redisClient = nil // Disable caching if Redis is not available
	} else {
		logger.Info("Connected to Redis")
	}

	// Create and start server
	server := llmrouter.NewServer(*port, logger, redisClient)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		logger.Info("Shutting down LLM Router service...")
		
		// Give ongoing requests 10 seconds to complete
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if redisClient != nil {
			redisClient.Close()
		}
		
		// Wait for context or force exit
		<-ctx.Done()
		os.Exit(0)
	}()

	// Start the server
	logger.Info("Starting LLM Router service", 
		zap.String("port", *port),
		zap.String("environment", *env),
	)
	
	if err := server.Start(); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func initLogger(level, env string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	var config zap.Config
	if env == "production" {
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapLevel)
	} else {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zapLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}