package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/activity"
)

// ErrorType represents the classification of an error
type ErrorType string

const (
	// Critical errors that should stop workflow
	ErrorTypeCritical ErrorType = "critical"
	
	// Recoverable errors that can be retried
	ErrorTypeRecoverable ErrorType = "recoverable"
	
	// Transient errors that should be retried with backoff
	ErrorTypeTransient ErrorType = "transient"
	
	// Non-blocking warnings that don't affect workflow
	ErrorTypeWarning ErrorType = "warning"
	
	// Resource errors (quota, rate limits)
	ErrorTypeResource ErrorType = "resource"
	
	// Validation errors (can continue with fallback)
	ErrorTypeValidation ErrorType = "validation"
)

// ClassifiedError wraps an error with classification and metadata
type ClassifiedError struct {
	Type        ErrorType              `json:"type"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	Retryable   bool                   `json:"retryable"`
	MaxRetries  int                    `json:"max_retries"`
	RetryDelay  time.Duration          `json:"retry_delay"`
	Fallback    bool                   `json:"fallback"`
	Context     map[string]interface{} `json:"context"`
	OriginalErr error                  `json:"-"`
}

// Error implements the error interface
func (e *ClassifiedError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// ClassifyError analyzes an error and returns a classified version
func ClassifyError(err error, service string) *ClassifiedError {
	if err == nil {
		return nil
	}
	
	// Check if already classified
	if classified, ok := err.(*ClassifiedError); ok {
		return classified
	}
	
	errMsg := err.Error()
	lowerMsg := strings.ToLower(errMsg)
	
	// Check for Temporal-specific errors
	if temporal.IsApplicationError(err) {
		return &ClassifiedError{
			Type:        ErrorTypeCritical,
			Code:        "APP_ERROR",
			Message:     errMsg,
			Service:     service,
			Retryable:   false,
			Fallback:    false,
			OriginalErr: err,
		}
	}
	
	// Network/Connection errors
	if strings.Contains(lowerMsg, "connection refused") ||
	   strings.Contains(lowerMsg, "connection reset") ||
	   strings.Contains(lowerMsg, "no such host") ||
	   strings.Contains(lowerMsg, "dial tcp") {
		return &ClassifiedError{
			Type:        ErrorTypeTransient,
			Code:        "NETWORK_ERROR",
			Message:     "Service temporarily unavailable",
			Service:     service,
			Retryable:   true,
			MaxRetries:  5,
			RetryDelay:  2 * time.Second,
			Fallback:    true,
			OriginalErr: err,
			Context: map[string]interface{}{
				"original_error": errMsg,
			},
		}
	}
	
	// Timeout errors
	if strings.Contains(lowerMsg, "timeout") ||
	   strings.Contains(lowerMsg, "deadline exceeded") {
		return &ClassifiedError{
			Type:        ErrorTypeTransient,
			Code:        "TIMEOUT",
			Message:     "Operation timed out",
			Service:     service,
			Retryable:   true,
			MaxRetries:  3,
			RetryDelay:  5 * time.Second,
			Fallback:    true,
			OriginalErr: err,
			Context: map[string]interface{}{
				"suggestion": "Consider increasing timeout or reducing payload size",
			},
		}
	}
	
	// Rate limiting errors
	if strings.Contains(lowerMsg, "rate limit") ||
	   strings.Contains(lowerMsg, "too many requests") ||
	   strings.Contains(lowerMsg, "429") {
		return &ClassifiedError{
			Type:        ErrorTypeResource,
			Code:        "RATE_LIMIT",
			Message:     "Rate limit exceeded",
			Service:     service,
			Retryable:   true,
			MaxRetries:  3,
			RetryDelay:  30 * time.Second,
			Fallback:    true,
			OriginalErr: err,
			Context: map[string]interface{}{
				"suggestion": "Implement exponential backoff",
			},
		}
	}
	
	// Quota/Resource errors
	if strings.Contains(lowerMsg, "quota exceeded") ||
	   strings.Contains(lowerMsg, "insufficient") ||
	   strings.Contains(lowerMsg, "out of memory") {
		return &ClassifiedError{
			Type:        ErrorTypeResource,
			Code:        "RESOURCE_EXHAUSTED",
			Message:     "Resource limit exceeded",
			Service:     service,
			Retryable:   false,
			Fallback:    true,
			OriginalErr: err,
			Context: map[string]interface{}{
				"action": "Check resource quotas and limits",
			},
		}
	}
	
	// Authentication errors
	if strings.Contains(lowerMsg, "unauthorized") ||
	   strings.Contains(lowerMsg, "403") ||
	   strings.Contains(lowerMsg, "401") ||
	   strings.Contains(lowerMsg, "authentication") {
		return &ClassifiedError{
			Type:        ErrorTypeCritical,
			Code:        "AUTH_ERROR",
			Message:     "Authentication failed",
			Service:     service,
			Retryable:   false,
			Fallback:    false,
			OriginalErr: err,
			Context: map[string]interface{}{
				"action": "Check API keys and credentials",
			},
		}
	}
	
	// Validation errors
	if strings.Contains(lowerMsg, "invalid") ||
	   strings.Contains(lowerMsg, "validation") ||
	   strings.Contains(lowerMsg, "malformed") {
		return &ClassifiedError{
			Type:        ErrorTypeValidation,
			Code:        "VALIDATION_ERROR",
			Message:     "Input validation failed",
			Service:     service,
			Retryable:   false,
			Fallback:    true,
			OriginalErr: err,
			Context: map[string]interface{}{
				"suggestion": "Check input format and requirements",
			},
		}
	}
	
	// Parser errors (non-critical for generation)
	if strings.Contains(lowerMsg, "parser") ||
	   strings.Contains(lowerMsg, "syntax") ||
	   strings.Contains(lowerMsg, "parse error") {
		return &ClassifiedError{
			Type:        ErrorTypeValidation,
			Code:        "PARSE_ERROR",
			Message:     "Code parsing failed",
			Service:     service,
			Retryable:   false,
			Fallback:    true,
			OriginalErr: err,
			Context: map[string]interface{}{
				"action": "Using basic validation instead",
			},
		}
	}
	
	// LLM-specific errors
	if strings.Contains(lowerMsg, "context length") ||
	   strings.Contains(lowerMsg, "token limit") {
		return &ClassifiedError{
			Type:        ErrorTypeRecoverable,
			Code:        "TOKEN_LIMIT",
			Message:     "Token limit exceeded",
			Service:     service,
			Retryable:   true,
			MaxRetries:  1,
			RetryDelay:  1 * time.Second,
			Fallback:    true,
			OriginalErr: err,
			Context: map[string]interface{}{
				"suggestion": "Reduce prompt size or use smaller model",
			},
		}
	}
	
	// Service-specific errors
	if strings.Contains(lowerMsg, "service unavailable") ||
	   strings.Contains(lowerMsg, "503") {
		return &ClassifiedError{
			Type:        ErrorTypeTransient,
			Code:        "SERVICE_UNAVAILABLE",
			Message:     fmt.Sprintf("%s service is temporarily unavailable", service),
			Service:     service,
			Retryable:   true,
			MaxRetries:  5,
			RetryDelay:  10 * time.Second,
			Fallback:    true,
			OriginalErr: err,
		}
	}
	
	// Default classification
	return &ClassifiedError{
		Type:        ErrorTypeRecoverable,
		Code:        "UNKNOWN_ERROR",
		Message:     errMsg,
		Service:     service,
		Retryable:   true,
		MaxRetries:  2,
		RetryDelay:  3 * time.Second,
		Fallback:    true,
		OriginalErr: err,
	}
}

// HandleError processes an error with appropriate recovery strategy
func HandleError(ctx context.Context, err error, service string) error {
	classified := ClassifyError(err, service)
	
	// Log the error with classification
	activity.GetLogger(ctx).Error("Classified error",
		"type", classified.Type,
		"code", classified.Code,
		"service", classified.Service,
		"retryable", classified.Retryable,
		"message", classified.Message,
	)
	
	// For critical errors, return immediately
	if classified.Type == ErrorTypeCritical {
		return temporal.NewApplicationError(
			classified.Message,
			string(classified.Code),
			classified,
		)
	}
	
	// For warnings, just log and continue
	if classified.Type == ErrorTypeWarning {
		activity.GetLogger(ctx).Warn("Non-blocking warning",
			"service", service,
			"message", classified.Message,
		)
		return nil
	}
	
	// For retryable errors, return with retry policy
	if classified.Retryable {
		return temporal.NewApplicationErrorWithCause(
			classified.Message,
			string(classified.Code),
			classified.OriginalErr,
		)
	}
	
	return classified
}

// ErrorRecoveryStrategy defines how to recover from specific error types
type ErrorRecoveryStrategy struct {
	UseTemplate     bool   `json:"use_template"`
	UseFallback     bool   `json:"use_fallback"`
	RetryWithBackoff bool  `json:"retry_with_backoff"`
	SwitchProvider  bool   `json:"switch_provider"`
	ReduceScope     bool   `json:"reduce_scope"`
	SkipValidation  bool   `json:"skip_validation"`
	FallbackMessage string `json:"fallback_message"`
}

// GetRecoveryStrategy returns the appropriate recovery strategy for an error
func GetRecoveryStrategy(classified *ClassifiedError) *ErrorRecoveryStrategy {
	if classified == nil {
		return nil
	}
	
	switch classified.Type {
	case ErrorTypeCritical:
		// No recovery for critical errors
		return &ErrorRecoveryStrategy{
			FallbackMessage: "Critical error - manual intervention required",
		}
		
	case ErrorTypeTransient:
		// Retry with exponential backoff
		return &ErrorRecoveryStrategy{
			RetryWithBackoff: true,
			SwitchProvider:   true, // Try alternative provider if available
			FallbackMessage:  "Retrying with alternative provider",
		}
		
	case ErrorTypeResource:
		// Switch provider or reduce scope
		return &ErrorRecoveryStrategy{
			SwitchProvider: true,
			ReduceScope:    true,
			FallbackMessage: "Resource limit reached - switching strategy",
		}
		
	case ErrorTypeValidation:
		// Use fallback validation or skip
		return &ErrorRecoveryStrategy{
			UseFallback:     true,
			SkipValidation:  true,
			FallbackMessage: "Validation failed - using fallback",
		}
		
	case ErrorTypeRecoverable:
		// Use template or fallback
		return &ErrorRecoveryStrategy{
			UseTemplate:     true,
			UseFallback:     true,
			RetryWithBackoff: true,
			FallbackMessage: "Using fallback strategy",
		}
		
	default:
		// Default recovery
		return &ErrorRecoveryStrategy{
			UseFallback:     true,
			FallbackMessage: "Using default recovery",
		}
	}
}

// ClassifyHTTPError classifies HTTP response errors
func ClassifyHTTPError(statusCode int, body []byte, service string) *ClassifiedError {
	var message string
	if len(body) > 0 {
		// Try to parse JSON error response
		var errResp map[string]interface{}
		if err := json.Unmarshal(body, &errResp); err == nil {
			if msg, ok := errResp["error"].(string); ok {
				message = msg
			} else if msg, ok := errResp["message"].(string); ok {
				message = msg
			}
		}
		if message == "" {
			message = string(body)
		}
	}
	
	switch statusCode {
	case http.StatusTooManyRequests:
		return &ClassifiedError{
			Type:        ErrorTypeResource,
			Code:        "RATE_LIMIT",
			Message:     "Rate limit exceeded",
			Service:     service,
			Retryable:   true,
			MaxRetries:  3,
			RetryDelay:  30 * time.Second,
			Fallback:    true,
			Context: map[string]interface{}{
				"status_code": statusCode,
				"response":    message,
			},
		}
		
	case http.StatusUnauthorized, http.StatusForbidden:
		return &ClassifiedError{
			Type:        ErrorTypeCritical,
			Code:        "AUTH_ERROR",
			Message:     fmt.Sprintf("Authentication failed: %s", message),
			Service:     service,
			Retryable:   false,
			Fallback:    false,
			Context: map[string]interface{}{
				"status_code": statusCode,
			},
		}
		
	case http.StatusBadRequest:
		return &ClassifiedError{
			Type:        ErrorTypeValidation,
			Code:        "BAD_REQUEST",
			Message:     fmt.Sprintf("Invalid request: %s", message),
			Service:     service,
			Retryable:   false,
			Fallback:    true,
			Context: map[string]interface{}{
				"status_code": statusCode,
			},
		}
		
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return &ClassifiedError{
			Type:        ErrorTypeTransient,
			Code:        "SERVER_ERROR",
			Message:     fmt.Sprintf("Server error: %s", message),
			Service:     service,
			Retryable:   true,
			MaxRetries:  3,
			RetryDelay:  5 * time.Second,
			Fallback:    true,
			Context: map[string]interface{}{
				"status_code": statusCode,
			},
		}
		
	case http.StatusGatewayTimeout:
		return &ClassifiedError{
			Type:        ErrorTypeTransient,
			Code:        "TIMEOUT",
			Message:     "Gateway timeout",
			Service:     service,
			Retryable:   true,
			MaxRetries:  2,
			RetryDelay:  10 * time.Second,
			Fallback:    true,
			Context: map[string]interface{}{
				"status_code": statusCode,
			},
		}
		
	default:
		if statusCode >= 400 && statusCode < 500 {
			// Client errors
			return &ClassifiedError{
				Type:        ErrorTypeValidation,
				Code:        fmt.Sprintf("CLIENT_ERROR_%d", statusCode),
				Message:     fmt.Sprintf("Client error: %s", message),
				Service:     service,
				Retryable:   false,
				Fallback:    true,
				Context: map[string]interface{}{
					"status_code": statusCode,
				},
			}
		} else if statusCode >= 500 {
			// Server errors
			return &ClassifiedError{
				Type:        ErrorTypeTransient,
				Code:        fmt.Sprintf("SERVER_ERROR_%d", statusCode),
				Message:     fmt.Sprintf("Server error: %s", message),
				Service:     service,
				Retryable:   true,
				MaxRetries:  3,
				RetryDelay:  5 * time.Second,
				Fallback:    true,
				Context: map[string]interface{}{
					"status_code": statusCode,
				},
			}
		}
	}
	
	return &ClassifiedError{
		Type:        ErrorTypeRecoverable,
		Code:        fmt.Sprintf("HTTP_%d", statusCode),
		Message:     fmt.Sprintf("HTTP error %d: %s", statusCode, message),
		Service:     service,
		Retryable:   true,
		MaxRetries:  2,
		RetryDelay:  3 * time.Second,
		Fallback:    true,
		Context: map[string]interface{}{
			"status_code": statusCode,
			"response":    message,
		},
	}
}

// WrapError wraps an error with additional context
func WrapError(err error, service string, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	
	classified := ClassifyError(err, service)
	if classified.Context == nil {
		classified.Context = make(map[string]interface{})
	}
	
	for k, v := range context {
		classified.Context[k] = v
	}
	
	return classified
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	
	if classified, ok := err.(*ClassifiedError); ok {
		return classified.Retryable
	}
	
	// Check for Temporal retryable errors
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return !appErr.NonRetryable()
	}
	
	return false
}

// ShouldFallback checks if we should use fallback for this error
func ShouldFallback(err error) bool {
	if err == nil {
		return false
	}
	
	if classified, ok := err.(*ClassifiedError); ok {
		return classified.Fallback
	}
	
	return true // Default to using fallback
}