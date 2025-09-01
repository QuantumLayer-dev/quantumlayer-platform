package orchestrator

import (
	"context"
	"time"
)

// AgentType represents different types of agents
type AgentType string

const (
	AgentTypeParser     AgentType = "parser"
	AgentTypeGenerator  AgentType = "generator"
	AgentTypeValidator  AgentType = "validator"
	AgentTypeOptimizer  AgentType = "optimizer"
	AgentTypeDocumenter AgentType = "documenter"
	AgentTypeTester     AgentType = "tester"
	AgentTypeDeployer   AgentType = "deployer"
)

// AgentRole represents the role of an agent in the system
type AgentRole string

const (
	AgentRolePrimary   AgentRole = "primary"
	AgentRoleSecondary AgentRole = "secondary"
	AgentRoleReviewer  AgentRole = "reviewer"
	AgentRoleSpecialist AgentRole = "specialist"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// TaskPriority represents task priority levels
type TaskPriority int

const (
	TaskPriorityLow    TaskPriority = 0
	TaskPriorityMedium TaskPriority = 1
	TaskPriorityHigh   TaskPriority = 2
	TaskPriorityCritical TaskPriority = 3
)

// GenerationRequest represents a code generation request
type GenerationRequest struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	OrgID       string            `json:"org_id"`
	Prompt      string            `json:"prompt"`
	Language    string            `json:"language,omitempty"`
	Framework   string            `json:"framework,omitempty"`
	Complexity  string            `json:"complexity,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// GenerationResponse represents the response from code generation
type GenerationResponse struct {
	ID          string            `json:"id"`
	RequestID   string            `json:"request_id"`
	Status      string            `json:"status"`
	Code        string            `json:"code,omitempty"`
	Tests       string            `json:"tests,omitempty"`
	Docs        string            `json:"docs,omitempty"`
	Errors      []string          `json:"errors,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

// Task represents a unit of work for an agent
type Task struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Priority    TaskPriority      `json:"priority"`
	Status      TaskStatus        `json:"status"`
	AgentID     string            `json:"agent_id,omitempty"`
	Input       interface{}       `json:"input"`
	Output      interface{}       `json:"output,omitempty"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	AssignedAt  *time.Time        `json:"assigned_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Deadline    *time.Time        `json:"deadline,omitempty"`
}

// Agent represents an agent in the system
type Agent struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         AgentType         `json:"type"`
	Role         AgentRole         `json:"role"`
	Status       string            `json:"status"`
	Capabilities []string          `json:"capabilities"`
	Workload     int               `json:"workload"`
	MaxWorkload  int               `json:"max_workload"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
	LastActiveAt time.Time         `json:"last_active_at"`
}

// AgentInterface defines the interface for all agents
type AgentInterface interface {
	// GetID returns the agent's unique identifier
	GetID() string
	
	// GetType returns the agent type
	GetType() AgentType
	
	// GetCapabilities returns the agent's capabilities
	GetCapabilities() []string
	
	// CanHandle checks if the agent can handle a specific task
	CanHandle(task *Task) bool
	
	// Execute performs the task
	Execute(ctx context.Context, task *Task) error
	
	// GetStatus returns the current status of the agent
	GetStatus() string
	
	// GetWorkload returns current workload
	GetWorkload() int
	
	// Stop gracefully stops the agent
	Stop() error
}

// OrchestratorConfig represents the orchestrator configuration
type OrchestratorConfig struct {
	MaxAgents          int           `json:"max_agents"`
	MaxTasksPerAgent   int           `json:"max_tasks_per_agent"`
	TaskTimeout        time.Duration `json:"task_timeout"`
	AgentSpawnTimeout  time.Duration `json:"agent_spawn_timeout"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	RedisURL           string        `json:"redis_url"`
	TemporalHost       string        `json:"temporal_host"`
	MetricsEnabled     bool          `json:"metrics_enabled"`
}

// WorkflowState represents the state of a generation workflow
type WorkflowState struct {
	ID          string                 `json:"id"`
	RequestID   string                 `json:"request_id"`
	Status      string                 `json:"status"`
	Phase       string                 `json:"phase"`
	Tasks       []*Task                `json:"tasks"`
	Agents      []*Agent               `json:"agents"`
	Results     map[string]interface{} `json:"results"`
	Errors      []string               `json:"errors"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// MetricsData represents orchestrator metrics
type MetricsData struct {
	TotalAgents      int     `json:"total_agents"`
	ActiveAgents     int     `json:"active_agents"`
	TotalTasks       int64   `json:"total_tasks"`
	CompletedTasks   int64   `json:"completed_tasks"`
	FailedTasks      int64   `json:"failed_tasks"`
	AverageTaskTime  float64 `json:"average_task_time_ms"`
	QueuedTasks      int     `json:"queued_tasks"`
	SystemLoad       float64 `json:"system_load"`
}