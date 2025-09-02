package models

import (
	"time"
)

// AgentType defines the type of specialized agent
type AgentType string

const (
	AgentTypeArchitect   AgentType = "architect"
	AgentTypeDeveloper   AgentType = "developer"
	AgentTypeTester      AgentType = "tester"
	AgentTypeSecurity    AgentType = "security"
	AgentTypePerformance AgentType = "performance"
	AgentTypeReviewer    AgentType = "reviewer"
	AgentTypeDocumentor  AgentType = "documentor"
	AgentTypeDevOps      AgentType = "devops"
)

// AgentCapability defines what an agent can do
type AgentCapability string

const (
	CapabilityCodeGeneration    AgentCapability = "code_generation"
	CapabilityCodeReview        AgentCapability = "code_review"
	CapabilityTestGeneration    AgentCapability = "test_generation"
	CapabilitySecurityAudit     AgentCapability = "security_audit"
	CapabilityPerformanceOpt    AgentCapability = "performance_optimization"
	CapabilityDocumentation     AgentCapability = "documentation"
	CapabilityArchitecture      AgentCapability = "architecture"
	CapabilityDeployment        AgentCapability = "deployment"
	CapabilityDebugging         AgentCapability = "debugging"
)

// Agent represents a specialized AI agent
type Agent struct {
	ID           string                 `json:"id"`
	Type         AgentType              `json:"type"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Capabilities []AgentCapability      `json:"capabilities"`
	Expertise    []string               `json:"expertise"` // languages, frameworks, domains
	Model        string                 `json:"model"`     // LLM model to use
	Config       AgentConfig            `json:"config"`
	State        AgentState             `json:"state"`
	Memory       *AgentMemory           `json:"memory,omitempty"`
	Performance  AgentPerformance       `json:"performance"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// AgentConfig contains agent configuration
type AgentConfig struct {
	MaxConcurrentTasks int     `json:"max_concurrent_tasks"`
	TimeoutSeconds     int     `json:"timeout_seconds"`
	RetryAttempts      int     `json:"retry_attempts"`
	Temperature        float64 `json:"temperature"`
	MaxTokens          int     `json:"max_tokens"`
	SystemPrompt       string  `json:"system_prompt"`
	ResponseFormat     string  `json:"response_format"` // json, markdown, plain
}

// AgentState represents the current state of an agent
type AgentState struct {
	Status         string    `json:"status"` // idle, busy, error, offline
	CurrentTasks   []string  `json:"current_tasks"`
	LastActiveAt   time.Time `json:"last_active_at"`
	ErrorCount     int       `json:"error_count"`
	ConsecutiveErrors int    `json:"consecutive_errors"`
}

// AgentMemory stores agent's long-term memory
type AgentMemory struct {
	ShortTerm   []MemoryItem `json:"short_term"`   // Recent interactions
	LongTerm    []MemoryItem `json:"long_term"`    // Important learnings
	VectorStore string       `json:"vector_store"` // Qdrant collection ID
}

// MemoryItem represents a single memory
type MemoryItem struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Type      string                 `json:"type"` // fact, pattern, preference, error
	Context   map[string]interface{} `json:"context"`
	Embedding []float32              `json:"embedding,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	AccessCount int                  `json:"access_count"`
}

// AgentPerformance tracks agent performance metrics
type AgentPerformance struct {
	TasksCompleted   int64   `json:"tasks_completed"`
	TasksFailed      int64   `json:"tasks_failed"`
	AverageLatency   float64 `json:"average_latency_ms"`
	SuccessRate      float64 `json:"success_rate"`
	QualityScore     float64 `json:"quality_score"` // 0-100
	TokensUsed       int64   `json:"tokens_used"`
	Cost             float64 `json:"cost_usd"`
}

// Task represents a task for an agent
type Task struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Input        map[string]interface{} `json:"input"`
	Requirements []string               `json:"requirements"`
	Priority     int                    `json:"priority"` // 1-10
	Deadline     *time.Time             `json:"deadline,omitempty"`
	AssignedTo   string                 `json:"assigned_to,omitempty"` // Agent ID
	Status       TaskStatus             `json:"status"`
	Result       *TaskResult            `json:"result,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// TaskResult contains the result of a completed task
type TaskResult struct {
	Output      interface{}   `json:"output"`
	Artifacts   []Artifact    `json:"artifacts"`
	Metrics     TaskMetrics   `json:"metrics"`
	Feedback    *TaskFeedback `json:"feedback,omitempty"`
	CompletedAt time.Time     `json:"completed_at"`
}

// Artifact represents a generated artifact
type Artifact struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // code, document, diagram, test, etc.
	Name     string `json:"name"`
	Content  string `json:"content"`
	Language string `json:"language,omitempty"`
	Size     int    `json:"size"`
}

// TaskMetrics contains task execution metrics
type TaskMetrics struct {
	ExecutionTime float64 `json:"execution_time_ms"`
	TokensUsed    int     `json:"tokens_used"`
	Cost          float64 `json:"cost_usd"`
	Quality       float64 `json:"quality_score"`
	Iterations    int     `json:"iterations"`
}

// TaskFeedback represents feedback on a task result
type TaskFeedback struct {
	Rating      int       `json:"rating"` // 1-5
	Comments    string    `json:"comments"`
	Approved    bool      `json:"approved"`
	Corrections string    `json:"corrections,omitempty"`
	ReviewedBy  string    `json:"reviewed_by"`
	ReviewedAt  time.Time `json:"reviewed_at"`
}

// AgentCollaboration represents collaboration between agents
type AgentCollaboration struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Participants []string               `json:"participants"` // Agent IDs
	Leader       string                 `json:"leader"`       // Lead agent ID
	Task         *Task                  `json:"task"`
	Strategy     CollaborationStrategy  `json:"strategy"`
	Messages     []CollaborationMessage `json:"messages"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

// CollaborationStrategy defines how agents collaborate
type CollaborationStrategy struct {
	Type           string   `json:"type"` // sequential, parallel, voting, consensus
	VotingRequired bool     `json:"voting_required"`
	MinVotes       int      `json:"min_votes"`
	Phases         []string `json:"phases"`
}

// CollaborationMessage represents a message in agent collaboration
type CollaborationMessage struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"` // Agent ID
	To        []string               `json:"to"`   // Agent IDs or "all"
	Type      string                 `json:"type"` // request, response, vote, decision
	Content   string                 `json:"content"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// VotingResult represents the result of agent voting
type VotingResult struct {
	ID         string             `json:"id"`
	Subject    string             `json:"subject"`
	Options    []VotingOption     `json:"options"`
	Votes      map[string]string  `json:"votes"`      // agent_id -> option_id
	Rationales map[string]string  `json:"rationales"` // agent_id -> reasoning
	Winner     string             `json:"winner"`     // winning option ID
	Confidence float64            `json:"confidence"`
	Timestamp  time.Time          `json:"timestamp"`
}

// VotingOption represents an option in voting
type VotingOption struct {
	ID          string      `json:"id"`
	Description string      `json:"description"`
	ProposedBy  string      `json:"proposed_by"` // Agent ID
	Content     interface{} `json:"content"`
	Score       float64     `json:"score"`
}