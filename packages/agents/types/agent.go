package types

import (
	"context"
	"time"
)

// AgentRole defines the specialized role an agent plays
type AgentRole string

const (
	RoleProjectManager AgentRole = "project-manager"
	RoleArchitect      AgentRole = "architect"
	RoleBackendDev     AgentRole = "backend-developer"
	RoleFrontendDev    AgentRole = "frontend-developer"
	RoleDatabaseAdmin  AgentRole = "database-admin"
	RoleDevOps         AgentRole = "devops"
	RoleQA             AgentRole = "qa-engineer"
	RoleSecurity       AgentRole = "security"
	RoleDataEngineer   AgentRole = "data-engineer"
	RoleSRE            AgentRole = "sre"
)

// AgentStatus represents the current state of an agent
type AgentStatus string

const (
	StatusIdle       AgentStatus = "idle"
	StatusAnalyzing  AgentStatus = "analyzing"
	StatusExecuting  AgentStatus = "executing"
	StatusCollaborating AgentStatus = "collaborating"
	StatusCompleted  AgentStatus = "completed"
	StatusFailed     AgentStatus = "failed"
)

// AgentCapability defines what an agent can do
type AgentCapability string

const (
	CapRequirementsAnalysis AgentCapability = "requirements-analysis"
	CapSystemDesign         AgentCapability = "system-design"
	CapCodeGeneration       AgentCapability = "code-generation"
	CapTestGeneration       AgentCapability = "test-generation"
	CapInfrastructureSetup  AgentCapability = "infrastructure-setup"
	CapSecurityAudit        AgentCapability = "security-audit"
	CapPerformanceOptimization AgentCapability = "performance-optimization"
	CapDocumentation        AgentCapability = "documentation"
	CapDataModeling         AgentCapability = "data-modeling"
	CapMonitoringSetup      AgentCapability = "monitoring-setup"
)

// Message represents inter-agent communication
type Message struct {
	ID          string                 `json:"id"`
	From        string                 `json:"from"`
	To          string                 `json:"to"`
	Type        MessageType            `json:"type"`
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	ReplyTo     string                 `json:"reply_to,omitempty"`
}

// MessageType defines the type of inter-agent message
type MessageType string

const (
	MsgTypeRequest      MessageType = "request"
	MsgTypeResponse     MessageType = "response"
	MsgTypeNotification MessageType = "notification"
	MsgTypeCollaboration MessageType = "collaboration"
	MsgTypeConsensus    MessageType = "consensus"
	MsgTypeEscalation   MessageType = "escalation"
)

// Task represents a unit of work for an agent
type Task struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Priority     int                    `json:"priority"`
	Requirements map[string]interface{} `json:"requirements"`
	Dependencies []string               `json:"dependencies"`
	Assignee     string                 `json:"assignee"`
	Status       TaskStatus             `json:"status"`
	Result       interface{}            `json:"result,omitempty"`
	Error        string                 `json:"error,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
	TaskFailed     TaskStatus = "failed"
	TaskBlocked    TaskStatus = "blocked"
)

// AgentContext contains the context for agent execution
type AgentContext struct {
	ProjectID    string                 `json:"project_id"`
	SessionID    string                 `json:"session_id"`
	UserID       string                 `json:"user_id"`
	Requirements string                 `json:"requirements"`
	Constraints  map[string]interface{} `json:"constraints"`
	SharedMemory *SharedMemory          `json:"-"`
	MessageBus   MessageBus             `json:"-"`
}

// SharedMemory provides shared state between agents
type SharedMemory struct {
	ProjectContext   map[string]interface{} `json:"project_context"`
	DesignDecisions  []DesignDecision       `json:"design_decisions"`
	GeneratedCode    map[string]string      `json:"generated_code"`
	TestResults      []TestResult           `json:"test_results"`
	SecurityFindings []SecurityFinding      `json:"security_findings"`
	Knowledge        map[string]interface{} `json:"knowledge"`
}

// DesignDecision represents an architectural or design decision
type DesignDecision struct {
	ID          string    `json:"id"`
	Agent       string    `json:"agent"`
	Category    string    `json:"category"`
	Decision    string    `json:"decision"`
	Reasoning   string    `json:"reasoning"`
	Timestamp   time.Time `json:"timestamp"`
	Approved    bool      `json:"approved"`
	ApprovedBy  []string  `json:"approved_by"`
}

// TestResult represents the result of a test execution
type TestResult struct {
	ID         string    `json:"id"`
	TestType   string    `json:"test_type"`
	Target     string    `json:"target"`
	Passed     bool      `json:"passed"`
	Coverage   float64   `json:"coverage,omitempty"`
	Details    string    `json:"details"`
	Timestamp  time.Time `json:"timestamp"`
}

// SecurityFinding represents a security issue or recommendation
type SecurityFinding struct {
	ID          string    `json:"id"`
	Severity    string    `json:"severity"`
	Type        string    `json:"type"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
	Remediation string    `json:"remediation"`
	Timestamp   time.Time `json:"timestamp"`
}

// Agent defines the interface all agents must implement
type Agent interface {
	// Core identification
	ID() string
	Role() AgentRole
	Capabilities() []AgentCapability
	Status() AgentStatus

	// Lifecycle management
	Initialize(ctx context.Context, agentCtx *AgentContext) error
	Execute(ctx context.Context, task *Task) error
	Shutdown(ctx context.Context) error

	// Communication
	SendMessage(ctx context.Context, msg *Message) error
	ReceiveMessage(ctx context.Context, msg *Message) error
	
	// Collaboration
	RequestCollaboration(ctx context.Context, targetAgent string, request interface{}) (interface{}, error)
	ParticipateInConsensus(ctx context.Context, topic string, proposal interface{}) (bool, error)

	// Self-improvement
	LearnFromFeedback(ctx context.Context, feedback interface{}) error
	GetMetrics() AgentMetrics
}

// AgentMetrics tracks agent performance
type AgentMetrics struct {
	TasksCompleted   int           `json:"tasks_completed"`
	TasksFailed      int           `json:"tasks_failed"`
	AverageTaskTime  time.Duration `json:"average_task_time"`
	SuccessRate      float64       `json:"success_rate"`
	CollaborationCount int         `json:"collaboration_count"`
	LastActive       time.Time     `json:"last_active"`
}

// MessageBus defines the interface for inter-agent communication
type MessageBus interface {
	Publish(ctx context.Context, topic string, msg *Message) error
	Subscribe(ctx context.Context, topic string, handler func(*Message)) error
	Unsubscribe(ctx context.Context, topic string) error
}

// AgentFactory creates agents based on requirements
type AgentFactory interface {
	CreateAgent(role AgentRole, config map[string]interface{}) (Agent, error)
	GetAvailableRoles() []AgentRole
}

// ConsensusRequest represents a request for multi-agent consensus
type ConsensusRequest struct {
	ID           string                 `json:"id"`
	Topic        string                 `json:"topic"`
	Proposal     interface{}            `json:"proposal"`
	RequiredVotes int                   `json:"required_votes"`
	Deadline     time.Time              `json:"deadline"`
	Participants []string               `json:"participants"`
	Votes        map[string]Vote        `json:"votes"`
}

// Vote represents an agent's vote in a consensus
type Vote struct {
	AgentID   string    `json:"agent_id"`
	Decision  bool      `json:"decision"`
	Reasoning string    `json:"reasoning"`
	Timestamp time.Time `json:"timestamp"`
}