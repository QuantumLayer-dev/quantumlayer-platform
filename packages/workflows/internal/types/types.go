package types

import (
	"time"
)

// CodeGenerationRequest represents a request to generate code
type CodeGenerationRequest struct {
	ID          string            `json:"id"`
	UserID      string            `json:"userId"`
	Prompt      string            `json:"prompt"`
	Language    string            `json:"language"`
	Framework   string            `json:"framework,omitempty"`
	Type        string            `json:"type"` // api, frontend, fullstack, function, etc.
	Context     map[string]string `json:"context,omitempty"`
	Preferences GenerationPrefs   `json:"preferences"`
	CreatedAt   time.Time         `json:"createdAt"`
}

// GenerationPrefs represents user preferences for code generation
type GenerationPrefs struct {
	Style           string   `json:"style"` // clean, detailed, minimal
	TestsRequired   bool     `json:"testsRequired"`
	Documentation   bool     `json:"documentation"`
	TypeSafety      bool     `json:"typeSafety"`
	LinterCompliant bool     `json:"linterCompliant"`
	Providers       []string `json:"providers"` // azure, aws, openai
}

// CodeGenerationResult represents the result of code generation
type CodeGenerationResult struct {
	ID            string              `json:"id"`
	RequestID     string              `json:"requestId"`
	Success       bool                `json:"success"`
	Code          string              `json:"code"`
	Tests         string              `json:"tests,omitempty"`
	Documentation string              `json:"documentation,omitempty"`
	Dependencies  []string            `json:"dependencies,omitempty"`
	Files         []GeneratedFile     `json:"files"`
	Metrics       GenerationMetrics   `json:"metrics"`
	Errors        []string            `json:"errors,omitempty"`
	CompletedAt   time.Time           `json:"completedAt"`
}

// GeneratedFile represents a single generated file
type GeneratedFile struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Type     string `json:"type"` // source, test, config, doc
}

// GenerationMetrics contains metrics about the generation process
type GenerationMetrics struct {
	TotalTokens      int           `json:"totalTokens"`
	PromptTokens     int           `json:"promptTokens"`
	CompletionTokens int           `json:"completionTokens"`
	LLMCalls         int           `json:"llmCalls"`
	Duration         time.Duration `json:"duration"`
	Provider         string        `json:"provider"`
	Model            string        `json:"model"`
	Cost             float64       `json:"cost"`
}

// WorkflowState represents the state of a workflow execution
type WorkflowState struct {
	Stage           string            `json:"stage"`
	Progress        int               `json:"progress"` // 0-100
	CurrentActivity string            `json:"currentActivity"`
	Metadata        map[string]string `json:"metadata"`
}

// PromptEnhancementRequest for Meta Prompt Engine
type PromptEnhancementRequest struct {
	OriginalPrompt string            `json:"originalPrompt"`
	Type           string            `json:"type"`
	Context        map[string]string `json:"context"`
	TargetProvider string            `json:"targetProvider"`
}

// PromptEnhancementResult from Meta Prompt Engine
type PromptEnhancementResult struct {
	EnhancedPrompt string   `json:"enhancedPrompt"`
	SystemPrompt   string   `json:"systemPrompt"`
	Examples       []string `json:"examples,omitempty"`
	Tokens         int      `json:"tokens"`
}

// ValidationRequest for code validation
type ValidationRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	Type     string `json:"type"`
	Rules    []string `json:"rules,omitempty"`
}

// ValidationResult from validation activities
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Issues   []Issue  `json:"issues,omitempty"`
	Score    float64  `json:"score"` // 0-100
	Feedback string   `json:"feedback"`
}

// Issue represents a code issue found during validation
type Issue struct {
	Type        string `json:"type"` // error, warning, info
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// AgentTask represents a task for an agent
type AgentTask struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Prompt   string            `json:"prompt"`
	Context  map[string]string `json:"context"`
	Priority int               `json:"priority"`
}

// AgentResult from agent execution
type AgentResult struct {
	TaskID   string `json:"taskId"`
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	Metadata map[string]interface{} `json:"metadata"`
}

// QuantumDrop represents an intermediate generation artifact
type QuantumDrop struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	Stage      string                 `json:"stage"`
	Timestamp  time.Time              `json:"timestamp"`
	Artifact   string                 `json:"artifact"`
	Type       string                 `json:"type"` // prompt, frd, code, tests, etc.
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ExtendedGenerationResult for the 12-stage workflow
type ExtendedGenerationResult struct {
	ID                 string              `json:"id"`
	RequestID          string              `json:"requestId"`
	Success            bool                `json:"success"`
	Code               string              `json:"code"`
	FRD                string              `json:"frd,omitempty"`
	TestPlan           string              `json:"testPlan,omitempty"`
	Tests              string              `json:"tests,omitempty"`
	Documentation      string              `json:"documentation,omitempty"`
	SecurityReport     string              `json:"securityReport,omitempty"`
	PerformanceReport  string              `json:"performanceReport,omitempty"`
	ProjectStructure   map[string]string   `json:"projectStructure,omitempty"`
	Dependencies       []string            `json:"dependencies,omitempty"`
	Files              []GeneratedFile     `json:"files"`
	QuantumDrops       []QuantumDrop       `json:"quantumDrops"`
	ValidationResults  ValidationResults   `json:"validationResults"`
	FeedbackIterations int                 `json:"feedbackIterations"`
	Metrics            GenerationMetrics   `json:"metrics"`
	Errors             []string            `json:"errors,omitempty"`
	CompletedAt        time.Time           `json:"completedAt"`
	PreviewURL         string              `json:"previewUrl,omitempty"`
}

// ValidationResults aggregates all validation scores
type ValidationResults struct {
	SemanticValid    bool     `json:"semanticValid"`
	SemanticIssues   []Issue  `json:"semanticIssues,omitempty"`
	SecurityScore    float64  `json:"securityScore"`
	SecurityIssues   []string `json:"securityIssues,omitempty"`
	PerformanceScore float64  `json:"performanceScore"`
	TestCoverage     float64  `json:"testCoverage"`
}