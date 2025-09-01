package llmrouter

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// LoggerMiddleware creates a Gin middleware for structured logging
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		// Log based on status code
		switch {
		case statusCode >= 500:
			logger.Error("Server error",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.String("ip", clientIP),
				zap.Duration("latency", latency),
				zap.String("error", c.Errors.String()),
			)
		case statusCode >= 400:
			logger.Warn("Client error",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.String("ip", clientIP),
				zap.Duration("latency", latency),
			)
		default:
			logger.Info("Request processed",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.String("ip", clientIP),
				zap.Duration("latency", latency),
			)
		}
	}
}

// CORSMiddleware handles CORS headers
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

// AuthMiddleware handles authentication
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		
		// Validate token (simplified for demo)
		// In production, validate JWT or check with auth service
		if !validateToken(token) {
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user context
		c.Set("user_id", "user_123") // Extract from token
		c.Set("org_id", "org_456")   // Extract from token
		
		c.Next()
	}
}

// RateLimitMiddleware implements rate limiting per IP
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	limiters := make(map[string]*rate.Limiter)
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		// Get or create limiter for this IP
		limiter, exists := limiters[clientIP]
		if !exists {
			limiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), requestsPerMinute)
			limiters[clientIP] = limiter
		}
		
		if !limiter.Allow() {
			c.JSON(429, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// MetricsMiddleware collects request metrics
func MetricsMiddleware(collector *MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		duration := time.Since(start)
		collector.RecordHTTPRequest(c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
	}
}

// validateToken validates an authentication token
func validateToken(token string) bool {
	// Simplified validation - in production, verify JWT signature
	// and check expiration, claims, etc.
	return token != "" && len(token) > 10
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int, duration time.Duration) *rate.Limiter {
	return rate.NewLimiter(rate.Every(duration/time.Duration(requestsPerMinute)), requestsPerMinute)
}