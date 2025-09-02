package models

import (
	"time"
)

// PromptTemplate represents a reusable prompt template
type PromptTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Category    string                 `json:"category"` // code_generation, testing, documentation, etc.
	Template    string                 `json:"template"`
	Variables   []TemplateVariable     `json:"variables"`
	Metadata    map[string]interface{} `json:"metadata"`
	Performance PerformanceMetrics     `json:"performance"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TemplateVariable represents a variable in a prompt template
type TemplateVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // string, number, boolean, array, object
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Description  string      `json:"description"`
	Validation   string      `json:"validation,omitempty"` // regex or validation rule
}

// PerformanceMetrics tracks template performance
type PerformanceMetrics struct {
	SuccessRate      float64 `json:"success_rate"`
	AverageTokens    int     `json:"average_tokens"`
	AverageLatency   float64 `json:"average_latency_ms"`
	TotalExecutions  int64   `json:"total_executions"`
	LastExecuted     time.Time `json:"last_executed"`
	QualityScore     float64 `json:"quality_score"` // 0-100
}

// PromptExecution represents a single prompt execution
type PromptExecution struct {
	ID           string                 `json:"id"`
	TemplateID   string                 `json:"template_id"`
	Variables    map[string]interface{} `json:"variables"`
	RenderedPrompt string               `json:"rendered_prompt"`
	Model        string                 `json:"model"`
	Response     string                 `json:"response"`
	TokensUsed   int                    `json:"tokens_used"`
	LatencyMs    float64                `json:"latency_ms"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	Feedback     *ExecutionFeedback     `json:"feedback,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ExecutionFeedback represents user feedback on prompt execution
type ExecutionFeedback struct {
	Rating       int       `json:"rating"` // 1-5
	Comments     string    `json:"comments"`
	Improvements string    `json:"improvements"`
	CreatedAt    time.Time `json:"created_at"`
}

// PromptChain represents a chain of prompts for complex reasoning
type PromptChain struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Steps       []ChainStep    `json:"steps"`
	Variables   map[string]interface{} `json:"variables"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// ChainStep represents a single step in a prompt chain
type ChainStep struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	TemplateID     string                 `json:"template_id"`
	InputMapping   map[string]string      `json:"input_mapping"`  // maps previous outputs to inputs
	OutputVariable string                 `json:"output_variable"` // variable name to store output
	Condition      string                 `json:"condition,omitempty"` // conditional execution
	RetryPolicy    *RetryPolicy           `json:"retry_policy,omitempty"`
}

// RetryPolicy defines retry behavior for prompt execution
type RetryPolicy struct {
	MaxAttempts     int           `json:"max_attempts"`
	BackoffMs       int           `json:"backoff_ms"`
	BackoffMultiplier float64     `json:"backoff_multiplier"`
}

// ABTestConfig represents A/B testing configuration
type ABTestConfig struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Variants    []TestVariant `json:"variants"`
	TrafficSplit map[string]float64 `json:"traffic_split"` // variant_id -> percentage
	Metrics     []string      `json:"metrics"` // metrics to track
	Status      string        `json:"status"` // active, paused, completed
	StartedAt   time.Time     `json:"started_at"`
	EndsAt      time.Time     `json:"ends_at"`
}

// TestVariant represents a variant in A/B testing
type TestVariant struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	TemplateID string                 `json:"template_id"`
	Variables  map[string]interface{} `json:"variables"`
	Results    *VariantResults        `json:"results,omitempty"`
}

// VariantResults stores A/B test results for a variant
type VariantResults struct {
	Executions   int64   `json:"executions"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatency   float64 `json:"avg_latency_ms"`
	AvgTokens    int     `json:"avg_tokens"`
	QualityScore float64 `json:"quality_score"`
	Conversions  int64   `json:"conversions"`
}