package activities

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
	
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// RetryConfig defines retry behavior configuration
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts"`
	InitialInterval time.Duration `json:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval"`
	BackoffFactor   float64       `json:"backoff_factor"`
	MaxJitter       time.Duration `json:"max_jitter"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		BackoffFactor:   2.0,
		MaxJitter:       1 * time.Second,
	}
}

// LLMRetryConfig returns retry configuration optimized for LLM calls
func LLMRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 2 * time.Second,
		MaxInterval:     60 * time.Second,
		BackoffFactor:   1.5, // Less aggressive backoff for LLMs
		MaxJitter:       2 * time.Second,
		RetryableErrors: []string{
			"RATE_LIMIT",
			"TIMEOUT",
			"SERVICE_UNAVAILABLE",
			"NETWORK_ERROR",
			"SERVER_ERROR",
		},
	}
}

// ServiceRetryConfig returns retry configuration for service calls
func ServiceRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:     4,
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     10 * time.Second,
		BackoffFactor:   2.0,
		MaxJitter:       500 * time.Millisecond,
		RetryableErrors: []string{
			"NETWORK_ERROR",
			"SERVICE_UNAVAILABLE",
			"TIMEOUT",
		},
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff[T any](
	ctx context.Context,
	config *RetryConfig,
	operation string,
	fn func() (T, error),
) (T, error) {
	var result T
	var lastErr error
	
	logger := activity.GetLogger(ctx)
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Execute the operation
		result, lastErr = fn()
		
		// Success - return immediately
		if lastErr == nil {
			if attempt > 0 {
				logger.Info("Operation succeeded after retry",
					"operation", operation,
					"attempt", attempt+1,
				)
			}
			return result, nil
		}
		
		// Check if error is retryable
		if !isRetryableError(lastErr, config) {
			logger.Error("Non-retryable error encountered",
				"operation", operation,
				"error", lastErr,
			)
			return result, lastErr
		}
		
		// Calculate backoff duration
		backoffDuration := calculateBackoff(attempt, config)
		
		// Log retry attempt
		logger.Warn("Operation failed, retrying",
			"operation", operation,
			"attempt", attempt+1,
			"max_attempts", config.MaxAttempts,
			"backoff", backoffDuration,
			"error", lastErr,
		)
		
		// Check context cancellation before sleeping
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(backoffDuration):
			// Continue to next attempt
		}
	}
	
	// All attempts exhausted
	logger.Error("All retry attempts exhausted",
		"operation", operation,
		"attempts", config.MaxAttempts,
		"last_error", lastErr,
	)
	
	return result, fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// calculateBackoff calculates the backoff duration for a given attempt
func calculateBackoff(attempt int, config *RetryConfig) time.Duration {
	// Calculate exponential backoff
	backoff := float64(config.InitialInterval) * math.Pow(config.BackoffFactor, float64(attempt))
	
	// Cap at maximum interval
	if backoff > float64(config.MaxInterval) {
		backoff = float64(config.MaxInterval)
	}
	
	// Add jitter to prevent thundering herd
	jitter := time.Duration(rand.Float64() * float64(config.MaxJitter))
	
	return time.Duration(backoff) + jitter
}

// isRetryableError checks if an error should trigger a retry
func isRetryableError(err error, config *RetryConfig) bool {
	if err == nil {
		return false
	}
	
	// Check if it's a classified error
	if classified, ok := err.(*ClassifiedError); ok {
		// Check against configured retryable error codes
		if len(config.RetryableErrors) > 0 {
			for _, code := range config.RetryableErrors {
				if string(classified.Code) == code {
					return classified.Retryable
				}
			}
		}
		return classified.Retryable
	}
	
	// Check Temporal application errors
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return !appErr.NonRetryable()
	}
	
	// Default to retryable for unknown errors
	return true
}

// RetryableOperation wraps an operation with retry logic
type RetryableOperation[T any] struct {
	Name      string
	Config    *RetryConfig
	Operation func(context.Context) (T, error)
	Fallback  func(context.Context, error) (T, error)
}

// Execute runs the operation with retry logic and fallback
func (r *RetryableOperation[T]) Execute(ctx context.Context) (T, error) {
	// Try the main operation with retry
	result, err := RetryWithBackoff(ctx, r.Config, r.Name, func() (T, error) {
		return r.Operation(ctx)
	})
	
	// If successful or no fallback, return
	if err == nil || r.Fallback == nil {
		return result, err
	}
	
	// Check if we should use fallback
	if shouldUseFallback(err) {
		activity.GetLogger(ctx).Info("Using fallback strategy",
			"operation", r.Name,
			"original_error", err,
		)
		return r.Fallback(ctx, err)
	}
	
	return result, err
}

// shouldUseFallback determines if fallback should be used
func shouldUseFallback(err error) bool {
	if err == nil {
		return false
	}
	
	// Check classified errors
	if classified, ok := err.(*ClassifiedError); ok {
		return classified.Fallback
	}
	
	// Default to using fallback for safety
	return true
}

// CircuitBreaker provides circuit breaker pattern for operations
type CircuitBreaker struct {
	Name            string
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenCalls    int
	
	state           CircuitState
	failures        int
	lastFailureTime time.Time
	successCount    int
}

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string) *CircuitBreaker {
	return &CircuitBreaker{
		Name:             name,
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
		HalfOpenCalls:    3,
		state:            CircuitClosed,
	}
}

// Call executes an operation through the circuit breaker
func CallWithCircuitBreaker[T any](cb *CircuitBreaker, ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	
	// Check circuit state
	switch cb.getState() {
	case CircuitOpen:
		return result, fmt.Errorf("circuit breaker is open for %s", cb.Name)
		
	case CircuitHalfOpen:
		// Allow limited calls in half-open state
		result, err := fn()
		if err != nil {
			cb.recordFailure()
			return result, err
		}
		cb.recordSuccess()
		return result, nil
		
	case CircuitClosed:
		// Normal operation
		result, err := fn()
		if err != nil {
			cb.recordFailure()
			return result, err
		}
		cb.recordSuccess()
		return result, nil
	}
	
	return result, fmt.Errorf("unknown circuit state")
}

// getState returns the current state of the circuit breaker
func (cb *CircuitBreaker) getState() CircuitState {
	// Check if we should transition from open to half-open
	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailureTime) > cb.ResetTimeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
		}
	}
	return cb.state
}

// recordFailure records a failure and updates circuit state
func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	cb.successCount = 0
	
	if cb.failures >= cb.FailureThreshold {
		cb.state = CircuitOpen
	}
}

// recordSuccess records a success and updates circuit state
func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == CircuitHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.HalfOpenCalls {
			// Enough successful calls, close the circuit
			cb.state = CircuitClosed
			cb.failures = 0
		}
	} else if cb.state == CircuitClosed {
		// Reset failure count on success in closed state
		cb.failures = 0
	}
}

// BulkheadLimiter provides concurrency limiting
type BulkheadLimiter struct {
	Name       string
	MaxConcurrency int
	semaphore  chan struct{}
}

// NewBulkheadLimiter creates a new bulkhead limiter
func NewBulkheadLimiter(name string, maxConcurrency int) *BulkheadLimiter {
	return &BulkheadLimiter{
		Name:           name,
		MaxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
	}
}

// Execute runs an operation with concurrency limiting
func ExecuteWithBulkhead[T any](b *BulkheadLimiter, ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	
	// Try to acquire semaphore
	select {
	case b.semaphore <- struct{}{}:
		// Acquired, execute operation
		defer func() { <-b.semaphore }()
		return fn()
		
	case <-ctx.Done():
		// Context cancelled while waiting
		return result, ctx.Err()
		
	default:
		// Bulkhead full, reject immediately
		return result, fmt.Errorf("bulkhead limiter full for %s (max: %d)", b.Name, b.MaxConcurrency)
	}
}

// AdaptiveRetryConfig provides adaptive retry configuration
type AdaptiveRetryConfig struct {
	BaseConfig      *RetryConfig
	SuccessRate     float64
	AdjustThreshold int
	adjustCounter   int
	successCounter  int
}

// NewAdaptiveRetryConfig creates a new adaptive retry configuration
func NewAdaptiveRetryConfig() *AdaptiveRetryConfig {
	return &AdaptiveRetryConfig{
		BaseConfig:      DefaultRetryConfig(),
		SuccessRate:     0.0,
		AdjustThreshold: 10,
	}
}

// Adjust adapts the retry configuration based on success rate
func (a *AdaptiveRetryConfig) Adjust(success bool) {
	a.adjustCounter++
	if success {
		a.successCounter++
	}
	
	// Adjust configuration every N calls
	if a.adjustCounter >= a.AdjustThreshold {
		a.SuccessRate = float64(a.successCounter) / float64(a.adjustCounter)
		
		// Adapt based on success rate
		if a.SuccessRate > 0.9 {
			// High success rate, reduce retry aggressiveness
			a.BaseConfig.MaxAttempts = maxInt(2, a.BaseConfig.MaxAttempts-1)
			a.BaseConfig.BackoffFactor = maxFloat(1.2, a.BaseConfig.BackoffFactor-0.2)
		} else if a.SuccessRate < 0.5 {
			// Low success rate, increase retry aggressiveness
			a.BaseConfig.MaxAttempts = minInt(10, a.BaseConfig.MaxAttempts+1)
			a.BaseConfig.BackoffFactor = minFloat(3.0, a.BaseConfig.BackoffFactor+0.2)
		}
		
		// Reset counters
		a.adjustCounter = 0
		a.successCounter = 0
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}