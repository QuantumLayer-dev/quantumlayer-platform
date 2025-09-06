package activities

import (
	"context"
	"fmt"
	"time"
	"encoding/json"
	"strings"
	"math/rand"

	"go.temporal.io/sdk/activity"
)

// ErrorRecoveryManager handles intelligent error recovery and resilience
type ErrorRecoveryManager struct {
	maxRetries          int
	retryStrategies     map[ErrorType]RetryStrategy
	circuitBreakers     map[string]*CircuitBreaker
	fallbackHandlers    map[ErrorType]FallbackHandler
	healthCheckers      map[string]HealthChecker
	alertManager        AlertManager
	metricsCollector    MetricsCollector
}

// ErrorType represents different categories of errors
type ErrorType string

const (
	NetworkError        ErrorType = "network"
	ServiceUnavailable  ErrorType = "service_unavailable"
	AuthenticationError ErrorType = "authentication"
	AuthorizationError  ErrorType = "authorization"
	ResourceError       ErrorType = "resource"
	ConfigurationError  ErrorType = "configuration"
	ValidationError     ErrorType = "validation"
	TimeoutError        ErrorType = "timeout"
	ConcurrencyError    ErrorType = "concurrency"
	DependencyError     ErrorType = "dependency"
	UnknownError        ErrorType = "unknown"
)

// RetryStrategy defines how to retry different types of errors
type RetryStrategy struct {
	MaxRetries      int           `json:"max_retries"`
	BaseDelay       time.Duration `json:"base_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffType     BackoffType   `json:"backoff_type"`
	Jitter          bool          `json:"jitter"`
	RetryableErrors []string      `json:"retryable_errors"`
}

type BackoffType string

const (
	ExponentialBackoff BackoffType = "exponential"
	LinearBackoff      BackoffType = "linear"
	FixedDelay         BackoffType = "fixed"
	CustomBackoff      BackoffType = "custom"
)

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	name           string
	state          CircuitState
	failureCount   int
	successCount   int
	lastFailure    time.Time
	timeout        time.Duration
	failureThreshold int
	successThreshold int
	totalRequests    int
	metrics          CircuitBreakerMetrics
}

type CircuitState string

const (
	ClosedState   CircuitState = "closed"
	OpenState     CircuitState = "open"
	HalfOpenState CircuitState = "half_open"
)

// FallbackHandler defines fallback strategies for different error types
type FallbackHandler interface {
	Handle(ctx context.Context, originalError error, request interface{}) (interface{}, error)
	CanHandle(errorType ErrorType) bool
	Priority() int
}

// DeploymentErrorRecoveryActivity handles deployment-specific error recovery
func DeploymentErrorRecoveryActivity(ctx context.Context, request DeploymentRequest, originalError error) (*ErrorRecoveryResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting intelligent error recovery", "error", originalError.Error())

	manager := &ErrorRecoveryManager{
		maxRetries:       5,
		retryStrategies:  createRetryStrategies(),
		circuitBreakers:  make(map[string]*CircuitBreaker),
		fallbackHandlers: createFallbackHandlers(),
		healthCheckers:   createHealthCheckers(),
		alertManager:     NewAlertManager(),
		metricsCollector: NewMetricsCollector(),
	}

	// Step 1: Analyze and classify the error
	errorType, errorDetails := manager.analyzeError(ctx, originalError)
	logger.Info("Error classified", "type", errorType, "details", errorDetails)

	// Step 2: Check circuit breaker status
	if manager.shouldSkipDueToCircuitBreaker(ctx, request, errorType) {
		return manager.handleCircuitBreakerOpen(ctx, request, originalError)
	}

	// Step 3: Attempt intelligent recovery
	recoveryResult, err := manager.attemptRecovery(ctx, request, originalError, errorType)
	if err != nil {
		// Step 4: Execute fallback strategy
		fallbackResult, fallbackErr := manager.executeFallback(ctx, request, originalError, errorType)
		if fallbackErr != nil {
			return &ErrorRecoveryResult{
				Success:         false,
				OriginalError:   originalError.Error(),
				RecoveryAttempts: recoveryResult.RecoveryAttempts,
				FallbackAttempts: fallbackResult.FallbackAttempts,
				FinalError:      fmt.Sprintf("Recovery and fallback failed: %v", fallbackErr),
			}, nil
		}
		return fallbackResult, nil
	}

	return recoveryResult, nil
}

// analyzeError classifies errors and extracts actionable information
func (m *ErrorRecoveryManager) analyzeError(ctx context.Context, err error) (ErrorType, ErrorDetails) {
	errorMsg := strings.ToLower(err.Error())
	
	details := ErrorDetails{
		Message:     err.Error(),
		Timestamp:   time.Now(),
		Severity:    SeverityMedium,
		Recoverable: true,
		Context:     make(map[string]interface{}),
	}

	// Network-related errors
	if strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "network") || 
	   strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "unreachable") {
		if strings.Contains(errorMsg, "timeout") {
			return TimeoutError, details
		}
		return NetworkError, details
	}

	// Service availability errors
	if strings.Contains(errorMsg, "service unavailable") || strings.Contains(errorMsg, "503") ||
	   strings.Contains(errorMsg, "502") || strings.Contains(errorMsg, "504") {
		details.Severity = SeverityHigh
		return ServiceUnavailable, details
	}

	// Authentication/Authorization errors
	if strings.Contains(errorMsg, "unauthorized") || strings.Contains(errorMsg, "401") {
		details.Recoverable = false
		return AuthenticationError, details
	}
	if strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "403") {
		details.Recoverable = false
		return AuthorizationError, details
	}

	// Resource errors (Docker, Kubernetes, etc.)
	if strings.Contains(errorMsg, "docker") || strings.Contains(errorMsg, "image") ||
	   strings.Contains(errorMsg, "container") || strings.Contains(errorMsg, "registry") {
		return ResourceError, details
	}

	// Configuration errors
	if strings.Contains(errorMsg, "config") || strings.Contains(errorMsg, "invalid") ||
	   strings.Contains(errorMsg, "missing") || strings.Contains(errorMsg, "required") {
		details.Severity = SeverityHigh
		return ConfigurationError, details
	}

	// Validation errors
	if strings.Contains(errorMsg, "validation") || strings.Contains(errorMsg, "invalid") {
		details.Recoverable = false
		return ValidationError, details
	}

	// Concurrency errors
	if strings.Contains(errorMsg, "concurrency") || strings.Contains(errorMsg, "deadlock") ||
	   strings.Contains(errorMsg, "race condition") {
		return ConcurrencyError, details
	}

	// Dependency errors
	if strings.Contains(errorMsg, "dependency") || strings.Contains(errorMsg, "service") ||
	   strings.Contains(errorMsg, "external") {
		return DependencyError, details
	}

	// Default to unknown error
	details.Severity = SeverityHigh
	return UnknownError, details
}

// attemptRecovery tries to recover from errors using intelligent strategies
func (m *ErrorRecoveryManager) attemptRecovery(ctx context.Context, request DeploymentRequest, 
	originalError error, errorType ErrorType) (*ErrorRecoveryResult, error) {
	
	logger := activity.GetLogger(ctx)
	
	result := &ErrorRecoveryResult{
		Success:         false,
		OriginalError:   originalError.Error(),
		ErrorType:       string(errorType),
		RecoveryAttempts: []RecoveryAttempt{},
		StartTime:       time.Now(),
	}

	strategy, exists := m.retryStrategies[errorType]
	if !exists {
		strategy = m.retryStrategies[UnknownError]
	}

	for attempt := 1; attempt <= strategy.MaxRetries; attempt++ {
		logger.Info("Attempting recovery", "attempt", attempt, "error_type", errorType)
		
		// Calculate delay with backoff and jitter
		delay := m.calculateDelay(strategy, attempt)
		if delay > 0 {
			logger.Info("Waiting before retry", "delay", delay)
			time.Sleep(delay)
		}

		recoveryAttempt := RecoveryAttempt{
			AttemptNumber: attempt,
			Strategy:      string(errorType),
			StartTime:     time.Now(),
		}

		// Execute recovery strategy based on error type
		success, recoveryErr := m.executeRecoveryStrategy(ctx, request, errorType, attempt)
		
		recoveryAttempt.EndTime = time.Now()
		recoveryAttempt.Duration = recoveryAttempt.EndTime.Sub(recoveryAttempt.StartTime)
		recoveryAttempt.Success = success

		if recoveryErr != nil {
			recoveryAttempt.Error = recoveryErr.Error()
		}

		result.RecoveryAttempts = append(result.RecoveryAttempts, recoveryAttempt)

		if success {
			logger.Info("Recovery successful", "attempt", attempt, "error_type", errorType)
			result.Success = true
			result.EndTime = time.Now()
			result.TotalDuration = result.EndTime.Sub(result.StartTime)
			m.recordSuccessMetrics(ctx, errorType, attempt)
			return result, nil
		}

		logger.Warn("Recovery attempt failed", "attempt", attempt, "error", recoveryErr)
	}

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)
	m.recordFailureMetrics(ctx, errorType, len(result.RecoveryAttempts))
	
	return result, fmt.Errorf("all recovery attempts failed")
}

// executeRecoveryStrategy executes specific recovery logic based on error type
func (m *ErrorRecoveryManager) executeRecoveryStrategy(ctx context.Context, request DeploymentRequest, 
	errorType ErrorType, attempt int) (bool, error) {
	
	switch errorType {
	case NetworkError:
		return m.recoverFromNetworkError(ctx, request, attempt)
	case ServiceUnavailable:
		return m.recoverFromServiceUnavailable(ctx, request, attempt)
	case ResourceError:
		return m.recoverFromResourceError(ctx, request, attempt)
	case ConfigurationError:
		return m.recoverFromConfigurationError(ctx, request, attempt)
	case TimeoutError:
		return m.recoverFromTimeoutError(ctx, request, attempt)
	case ConcurrencyError:
		return m.recoverFromConcurrencyError(ctx, request, attempt)
	case DependencyError:
		return m.recoverFromDependencyError(ctx, request, attempt)
	default:
		return m.recoverFromUnknownError(ctx, request, attempt)
	}
}

// Recovery strategy implementations
func (m *ErrorRecoveryManager) recoverFromNetworkError(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Attempting network error recovery")
	
	// Strategy 1: Health check dependent services
	if healthy, err := m.checkDependencyHealth(ctx, request); err != nil {
		return false, fmt.Errorf("dependency health check failed: %w", err)
	} else if !healthy {
		return false, fmt.Errorf("dependencies are not healthy")
	}

	// Strategy 2: Try alternative endpoints/regions
	if attempt > 1 {
		if success, err := m.switchToAlternativeEndpoint(ctx, request); err != nil {
			return false, err
		} else if success {
			return true, nil
		}
	}

	// Strategy 3: Reconfigure network settings
	if attempt > 2 {
		return m.reconfigureNetworkSettings(ctx, request)
	}

	return false, fmt.Errorf("network recovery strategies exhausted")
}

func (m *ErrorRecoveryManager) recoverFromServiceUnavailable(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Attempting service unavailable recovery")
	
	// Strategy 1: Check service health and wait for recovery
	if healthy, err := m.waitForServiceRecovery(ctx, request, time.Duration(attempt)*30*time.Second); err != nil {
		return false, err
	} else if healthy {
		return true, nil
	}

	// Strategy 2: Use alternative service instance
	if attempt > 1 {
		return m.switchToAlternativeService(ctx, request)
	}

	return false, fmt.Errorf("service unavailable recovery failed")
}

func (m *ErrorRecoveryManager) recoverFromResourceError(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Attempting resource error recovery")
	
	// Strategy 1: Clean up and retry resource allocation
	if err := m.cleanupResources(ctx, request); err != nil {
		logger.Warn("Resource cleanup failed", "error", err)
	}
	
	// Strategy 2: Try alternative resource configuration
	if attempt > 1 {
		return m.adjustResourceConfiguration(ctx, request, attempt)
	}
	
	// Strategy 3: Switch deployment strategy
	if attempt > 2 {
		return m.switchDeploymentStrategy(ctx, request)
	}

	return false, fmt.Errorf("resource error recovery failed")
}

func (m *ErrorRecoveryManager) recoverFromConfigurationError(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Attempting configuration error recovery")
	
	// Strategy 1: Validate and fix configuration
	if fixed, err := m.autoFixConfiguration(ctx, request); err != nil {
		return false, err
	} else if fixed {
		return true, nil
	}

	// Strategy 2: Use default/fallback configuration
	if attempt > 1 {
		return m.useFallbackConfiguration(ctx, request)
	}

	return false, fmt.Errorf("configuration error recovery failed")
}

func (m *ErrorRecoveryManager) recoverFromTimeoutError(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Attempting timeout error recovery")
	
	// Strategy 1: Increase timeout dynamically
	newTimeout := time.Duration(attempt) * 2 * time.Minute
	return m.retryWithIncreasedTimeout(ctx, request, newTimeout)
}

func (m *ErrorRecoveryManager) recoverFromConcurrencyError(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Attempting concurrency error recovery")
	
	// Strategy 1: Add random delay to avoid thundering herd
	randomDelay := time.Duration(rand.Intn(5000)) * time.Millisecond
	time.Sleep(randomDelay)
	
	// Strategy 2: Use locking or queuing mechanism
	return m.retryWithConcurrencyControl(ctx, request, attempt)
}

func (m *ErrorRecoveryManager) recoverFromDependencyError(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Attempting dependency error recovery")
	
	// Strategy 1: Health check dependencies
	if healthy, err := m.checkAllDependencies(ctx, request); err != nil {
		return false, err
	} else if !healthy {
		// Strategy 2: Try to restart or reinitialize dependencies
		if attempt > 1 {
			return m.reinitializeDependencies(ctx, request)
		}
		return false, fmt.Errorf("dependencies are not healthy")
	}

	return true, nil
}

func (m *ErrorRecoveryManager) recoverFromUnknownError(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Attempting unknown error recovery")
	
	// Generic recovery strategies
	// Strategy 1: Full system health check
	if healthy, err := m.performFullHealthCheck(ctx, request); err != nil {
		return false, err
	} else if !healthy {
		return false, fmt.Errorf("system health check failed")
	}

	// Strategy 2: Reset to known good state
	if attempt > 2 {
		return m.resetToKnownGoodState(ctx, request)
	}

	return false, fmt.Errorf("unknown error recovery failed")
}

// calculateDelay calculates the delay for retry attempts
func (m *ErrorRecoveryManager) calculateDelay(strategy RetryStrategy, attempt int) time.Duration {
	var delay time.Duration
	
	switch strategy.BackoffType {
	case ExponentialBackoff:
		delay = time.Duration(int64(strategy.BaseDelay) * (1 << uint(attempt-1)))
	case LinearBackoff:
		delay = strategy.BaseDelay * time.Duration(attempt)
	case FixedDelay:
		delay = strategy.BaseDelay
	default:
		delay = strategy.BaseDelay * time.Duration(attempt)
	}

	// Cap at max delay
	if delay > strategy.MaxDelay {
		delay = strategy.MaxDelay
	}

	// Add jitter if enabled
	if strategy.Jitter {
		jitterAmount := time.Duration(rand.Int63n(int64(delay) / 4))
		delay += jitterAmount
	}

	return delay
}

// Helper functions for recovery strategies
func createRetryStrategies() map[ErrorType]RetryStrategy {
	return map[ErrorType]RetryStrategy{
		NetworkError: {
			MaxRetries:  5,
			BaseDelay:   1 * time.Second,
			MaxDelay:    30 * time.Second,
			BackoffType: ExponentialBackoff,
			Jitter:      true,
		},
		ServiceUnavailable: {
			MaxRetries:  3,
			BaseDelay:   10 * time.Second,
			MaxDelay:    2 * time.Minute,
			BackoffType: LinearBackoff,
			Jitter:      true,
		},
		ResourceError: {
			MaxRetries:  4,
			BaseDelay:   5 * time.Second,
			MaxDelay:    1 * time.Minute,
			BackoffType: ExponentialBackoff,
			Jitter:      false,
		},
		ConfigurationError: {
			MaxRetries:  2,
			BaseDelay:   2 * time.Second,
			MaxDelay:    10 * time.Second,
			BackoffType: FixedDelay,
			Jitter:      false,
		},
		TimeoutError: {
			MaxRetries:  3,
			BaseDelay:   30 * time.Second,
			MaxDelay:    5 * time.Minute,
			BackoffType: LinearBackoff,
			Jitter:      true,
		},
		UnknownError: {
			MaxRetries:  3,
			BaseDelay:   5 * time.Second,
			MaxDelay:    30 * time.Second,
			BackoffType: ExponentialBackoff,
			Jitter:      true,
		},
	}
}

func createFallbackHandlers() map[ErrorType]FallbackHandler {
	return map[ErrorType]FallbackHandler{
		NetworkError:       &NetworkFallbackHandler{},
		ServiceUnavailable: &ServiceFallbackHandler{},
		ResourceError:      &ResourceFallbackHandler{},
		ConfigurationError: &ConfigurationFallbackHandler{},
	}
}

func createHealthCheckers() map[string]HealthChecker {
	return map[string]HealthChecker{
		"docker":     &DockerHealthChecker{},
		"kubernetes": &KubernetesHealthChecker{},
		"network":    &NetworkHealthChecker{},
		"service":    &ServiceHealthChecker{},
	}
}

// Supporting types and structures
type ErrorDetails struct {
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    Severity               `json:"severity"`
	Recoverable bool                   `json:"recoverable"`
	Context     map[string]interface{} `json:"context"`
}

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
	SeverityCritical Severity = "critical"
)

type ErrorRecoveryResult struct {
	Success          bool             `json:"success"`
	OriginalError    string           `json:"original_error"`
	ErrorType        string           `json:"error_type"`
	RecoveryAttempts []RecoveryAttempt `json:"recovery_attempts"`
	FallbackAttempts []FallbackAttempt `json:"fallback_attempts"`
	FinalError       string           `json:"final_error,omitempty"`
	StartTime        time.Time        `json:"start_time"`
	EndTime          time.Time        `json:"end_time"`
	TotalDuration    time.Duration    `json:"total_duration"`
}

type RecoveryAttempt struct {
	AttemptNumber int           `json:"attempt_number"`
	Strategy      string        `json:"strategy"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	Success       bool          `json:"success"`
	Error         string        `json:"error,omitempty"`
}

type FallbackAttempt struct {
	Handler   string        `json:"handler"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
}

type CircuitBreakerMetrics struct {
	TotalRequests   int     `json:"total_requests"`
	FailureRate     float64 `json:"failure_rate"`
	AverageLatency  float64 `json:"average_latency"`
	LastFailureTime time.Time `json:"last_failure_time"`
}

// Interface implementations and stubs for health checkers and fallback handlers
type AlertManager interface {
	SendAlert(ctx context.Context, alert Alert) error
}

type MetricsCollector interface {
	RecordRecoveryAttempt(ctx context.Context, errorType ErrorType, success bool, duration time.Duration)
	RecordCircuitBreakerState(ctx context.Context, name string, state CircuitState)
}

type Alert struct {
	Title    string    `json:"title"`
	Message  string    `json:"message"`
	Severity Severity  `json:"severity"`
	Time     time.Time `json:"time"`
}

// Placeholder implementations - these would be fully implemented in production
func NewAlertManager() AlertManager { return &DefaultAlertManager{} }
func NewMetricsCollector() MetricsCollector { return &DefaultMetricsCollector{} }

type DefaultAlertManager struct{}
func (am *DefaultAlertManager) SendAlert(ctx context.Context, alert Alert) error { return nil }

type DefaultMetricsCollector struct{}
func (mc *DefaultMetricsCollector) RecordRecoveryAttempt(ctx context.Context, errorType ErrorType, success bool, duration time.Duration) {}
func (mc *DefaultMetricsCollector) RecordCircuitBreakerState(ctx context.Context, name string, state CircuitState) {}

// Stub implementations for recovery methods
func (m *ErrorRecoveryManager) checkDependencyHealth(ctx context.Context, request DeploymentRequest) (bool, error) { return true, nil }
func (m *ErrorRecoveryManager) switchToAlternativeEndpoint(ctx context.Context, request DeploymentRequest) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) reconfigureNetworkSettings(ctx context.Context, request DeploymentRequest) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) waitForServiceRecovery(ctx context.Context, request DeploymentRequest, timeout time.Duration) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) switchToAlternativeService(ctx context.Context, request DeploymentRequest) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) cleanupResources(ctx context.Context, request DeploymentRequest) error { return nil }
func (m *ErrorRecoveryManager) adjustResourceConfiguration(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) switchDeploymentStrategy(ctx context.Context, request DeploymentRequest) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) autoFixConfiguration(ctx context.Context, request DeploymentRequest) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) useFallbackConfiguration(ctx context.Context, request DeploymentRequest) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) retryWithIncreasedTimeout(ctx context.Context, request DeploymentRequest, timeout time.Duration) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) retryWithConcurrencyControl(ctx context.Context, request DeploymentRequest, attempt int) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) checkAllDependencies(ctx context.Context, request DeploymentRequest) (bool, error) { return true, nil }
func (m *ErrorRecoveryManager) reinitializeDependencies(ctx context.Context, request DeploymentRequest) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) performFullHealthCheck(ctx context.Context, request DeploymentRequest) (bool, error) { return true, nil }
func (m *ErrorRecoveryManager) resetToKnownGoodState(ctx context.Context, request DeploymentRequest) (bool, error) { return false, nil }
func (m *ErrorRecoveryManager) shouldSkipDueToCircuitBreaker(ctx context.Context, request DeploymentRequest, errorType ErrorType) bool { return false }
func (m *ErrorRecoveryManager) handleCircuitBreakerOpen(ctx context.Context, request DeploymentRequest, originalError error) (*ErrorRecoveryResult, error) { return nil, nil }
func (m *ErrorRecoveryManager) executeFallback(ctx context.Context, request DeploymentRequest, originalError error, errorType ErrorType) (*ErrorRecoveryResult, error) { return nil, fmt.Errorf("fallback not implemented") }
func (m *ErrorRecoveryManager) recordSuccessMetrics(ctx context.Context, errorType ErrorType, attempts int) {}
func (m *ErrorRecoveryManager) recordFailureMetrics(ctx context.Context, errorType ErrorType, attempts int) {}