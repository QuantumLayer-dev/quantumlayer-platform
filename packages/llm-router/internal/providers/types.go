package providers

import (
	"context"
	"sync/atomic"
	"time"
)

// CodeGenerationRequest represents a request to generate code
type CodeGenerationRequest struct {
	Prompt      string            `json:"prompt"`
	Language    string            `json:"language"`
	Framework   string            `json:"framework,omitempty"`
	Type        string            `json:"type,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CodeGenerationResponse represents the generated code response
type CodeGenerationResponse struct {
	Code     string        `json:"code"`
	Language string        `json:"language"`
	Provider string        `json:"provider"`
	Model    string        `json:"model"`
	Usage    TokenUsage    `json:"usage"`
	Latency  time.Duration `json:"latency"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TokenUsage represents token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Provider defines the interface for LLM providers
type Provider interface {
	GenerateCode(ctx context.Context, request CodeGenerationRequest) (*CodeGenerationResponse, error)
	HealthCheck(ctx context.Context) error
	GetMetrics() ProviderMetrics
}

// ProviderMetrics tracks provider performance metrics
type ProviderMetrics struct {
	provider       string
	totalRequests  int64
	successCount   int64
	errorCount     int64
	totalTokens    int64
	totalLatency   int64
	errorTypes     map[string]int64
}

// NewProviderMetrics creates new metrics tracker
func NewProviderMetrics(provider string) *ProviderMetrics {
	return &ProviderMetrics{
		provider:   provider,
		errorTypes: make(map[string]int64),
	}
}

// RecordSuccess records a successful request
func (m *ProviderMetrics) RecordSuccess() {
	atomic.AddInt64(&m.totalRequests, 1)
	atomic.AddInt64(&m.successCount, 1)
}

// RecordError records an error
func (m *ProviderMetrics) RecordError(errorType string) {
	atomic.AddInt64(&m.totalRequests, 1)
	atomic.AddInt64(&m.errorCount, 1)
	if m.errorTypes != nil {
		m.errorTypes[errorType]++
	}
}

// RecordLatency records request latency
func (m *ProviderMetrics) RecordLatency(latency time.Duration) {
	atomic.AddInt64(&m.totalLatency, int64(latency))
}

// RecordTokens records token usage
func (m *ProviderMetrics) RecordTokens(tokens int) {
	atomic.AddInt64(&m.totalTokens, int64(tokens))
}

// GetSuccessRate returns the success rate
func (m *ProviderMetrics) GetSuccessRate() float64 {
	total := atomic.LoadInt64(&m.totalRequests)
	if total == 0 {
		return 0
	}
	success := atomic.LoadInt64(&m.successCount)
	return float64(success) / float64(total)
}

// GetAverageLatency returns average latency
func (m *ProviderMetrics) GetAverageLatency() time.Duration {
	total := atomic.LoadInt64(&m.totalRequests)
	if total == 0 {
		return 0
	}
	latency := atomic.LoadInt64(&m.totalLatency)
	return time.Duration(latency / total)
}