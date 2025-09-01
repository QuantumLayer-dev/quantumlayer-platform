package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// State represents the state of the circuit breaker
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name            string
	maxFailures     int
	resetTimeout    time.Duration
	halfOpenMax     int
	
	state           State
	failures        int
	lastFailureTime time.Time
	halfOpenCount   int
	
	mu              sync.RWMutex
	logger          *zap.Logger
	
	// Callbacks
	onStateChange   func(from, to State)
}

// Config holds circuit breaker configuration
type Config struct {
	Name            string
	MaxFailures     int
	ResetTimeout    time.Duration
	HalfOpenMax     int
	OnStateChange   func(from, to State)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config, logger *zap.Logger) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 60 * time.Second
	}
	if config.HalfOpenMax <= 0 {
		config.HalfOpenMax = 3
	}
	
	return &CircuitBreaker{
		name:          config.Name,
		maxFailures:   config.MaxFailures,
		resetTimeout:  config.ResetTimeout,
		halfOpenMax:   config.HalfOpenMax,
		state:         StateClosed,
		logger:        logger,
		onStateChange: config.OnStateChange,
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	if err := cb.canExecute(); err != nil {
		return nil, err
	}
	
	result, err := fn(ctx)
	cb.recordResult(err)
	return result, err
}

// ExecuteWithFallback runs the function with a fallback
func (cb *CircuitBreaker) ExecuteWithFallback(
	ctx context.Context,
	fn func(context.Context) (interface{}, error),
	fallback func(context.Context, error) (interface{}, error),
) (interface{}, error) {
	if err := cb.canExecute(); err != nil {
		return fallback(ctx, err)
	}
	
	result, err := fn(ctx)
	cb.recordResult(err)
	
	if err != nil && cb.GetState() == StateOpen {
		return fallback(ctx, err)
	}
	
	return result, err
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	now := time.Now()
	
	switch cb.state {
	case StateClosed:
		return nil
		
	case StateOpen:
		if now.Sub(cb.lastFailureTime) > cb.resetTimeout {
			cb.changeState(StateHalfOpen)
			cb.halfOpenCount = 0
			return nil
		}
		return ErrCircuitOpen
		
	case StateHalfOpen:
		if cb.halfOpenCount >= cb.halfOpenMax {
			return ErrTooManyRequests
		}
		cb.halfOpenCount++
		return nil
		
	default:
		return nil
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if err == nil {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess handles successful execution
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0
		
	case StateHalfOpen:
		cb.failures = 0
		cb.changeState(StateClosed)
		
	case StateOpen:
		// Should not happen
		cb.logger.Warn("Success recorded in open state", zap.String("circuit", cb.name))
	}
}

// onFailure handles failed execution
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	
	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.maxFailures {
			cb.changeState(StateOpen)
		}
		
	case StateHalfOpen:
		cb.changeState(StateOpen)
		
	case StateOpen:
		// Already open, nothing to do
	}
}

// changeState changes the circuit breaker state
func (cb *CircuitBreaker) changeState(newState State) {
	if cb.state == newState {
		return
	}
	
	oldState := cb.state
	cb.state = newState
	
	cb.logger.Info("Circuit breaker state changed",
		zap.String("circuit", cb.name),
		zap.String("from", cb.stateName(oldState)),
		zap.String("to", cb.stateName(newState)),
	)
	
	if cb.onStateChange != nil {
		go cb.onStateChange(oldState, newState)
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenCount = 0
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return map[string]interface{}{
		"name":         cb.name,
		"state":        cb.stateName(cb.state),
		"failures":     cb.failures,
		"last_failure": cb.lastFailureTime,
	}
}

// stateName returns the string representation of a state
func (cb *CircuitBreaker) stateName(state State) string {
	switch state {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}