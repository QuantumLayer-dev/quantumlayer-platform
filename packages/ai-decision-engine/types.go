package aidecision

import (
	"context"
	"time"
)

// Decision represents a decision point in the system
type Decision struct {
	ID          string                 `json:"id"`
	Context     string                 `json:"context"`
	Input       string                 `json:"input"`
	Intent      string                 `json:"intent"`
	Confidence  float64                `json:"confidence"`
	Result      interface{}            `json:"result"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

// DecisionRule represents a rule that can be matched semantically
type DecisionRule struct {
	ID           string                 `json:"id"`
	Category     string                 `json:"category"`
	Pattern      string                 `json:"pattern"`
	Description  string                 `json:"description"`
	Action       ActionFunc             `json:"-"`
	Embedding    []float32              `json:"embedding"`
	Examples     []string               `json:"examples"`
	Priority     int                    `json:"priority"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ActionFunc is the function executed when a rule matches
type ActionFunc func(ctx context.Context, input interface{}) (interface{}, error)

// SemanticMatch represents a semantic similarity match
type SemanticMatch struct {
	Rule       *DecisionRule `json:"rule"`
	Score      float64       `json:"score"`
	Confidence float64       `json:"confidence"`
	Reasoning  string        `json:"reasoning"`
}

// DecisionEngine interface for AI-powered decision making
type DecisionEngine interface {
	// Register a new decision rule
	RegisterRule(rule *DecisionRule) error
	
	// Make a decision based on input
	Decide(ctx context.Context, category string, input string) (*Decision, error)
	
	// Get all matches for an input
	GetMatches(ctx context.Context, category string, input string, threshold float64) ([]*SemanticMatch, error)
	
	// Learn from feedback
	LearnFromFeedback(ctx context.Context, decision *Decision, feedback Feedback) error
	
	// Export rules for inspection
	ExportRules() map[string][]*DecisionRule
}

// Feedback represents feedback on a decision
type Feedback struct {
	DecisionID string    `json:"decision_id"`
	Correct    bool      `json:"correct"`
	Expected   string    `json:"expected,omitempty"`
	Comment    string    `json:"comment,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// LanguageDecisionEngine handles programming language selection
type LanguageDecisionEngine interface {
	DecideLanguage(ctx context.Context, requirements string) (string, map[string]interface{}, error)
	GetLanguageCapabilities(language string) map[string]interface{}
	SuggestLanguages(ctx context.Context, requirements string) ([]string, error)
}

// FrameworkDecisionEngine handles framework selection
type FrameworkDecisionEngine interface {
	DecideFramework(ctx context.Context, language, requirements string) (string, map[string]interface{}, error)
	GetFrameworkFeatures(framework string) map[string]interface{}
	IsCompatible(language, framework string) bool
}

// AgentDecisionEngine handles agent selection and spawning
type AgentDecisionEngine interface {
	DecideAgent(ctx context.Context, task string) (string, map[string]interface{}, error)
	GetAgentCapabilities(agentType string) []string
	SuggestAgentTeam(ctx context.Context, project string) ([]string, error)
}

// SecurityDecisionEngine handles security-related decisions
type SecurityDecisionEngine interface {
	AssessRisk(ctx context.Context, code string) (RiskLevel, []SecurityIssue, error)
	SuggestMitigation(ctx context.Context, issue SecurityIssue) ([]Mitigation, error)
	ValidateCompliance(ctx context.Context, code string, standards []string) (ComplianceReport, error)
}

// RiskLevel represents the security risk level
type RiskLevel string

const (
	RiskCritical RiskLevel = "critical"
	RiskHigh     RiskLevel = "high"
	RiskMedium   RiskLevel = "medium"
	RiskLow      RiskLevel = "low"
	RiskNone     RiskLevel = "none"
)

// SecurityIssue represents a security vulnerability or concern
type SecurityIssue struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    RiskLevel `json:"severity"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	CWE         string    `json:"cwe,omitempty"`
	OWASP       string    `json:"owasp,omitempty"`
}

// Mitigation represents a security mitigation strategy
type Mitigation struct {
	Issue       string `json:"issue"`
	Strategy    string `json:"strategy"`
	Code        string `json:"code,omitempty"`
	Explanation string `json:"explanation"`
	Effort      string `json:"effort"`
}

// ComplianceReport represents compliance validation results
type ComplianceReport struct {
	Standards  []string               `json:"standards"`
	Compliant  bool                   `json:"compliant"`
	Violations []ComplianceViolation  `json:"violations"`
	Score      float64                `json:"score"`
	Timestamp  time.Time              `json:"timestamp"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	Standard    string `json:"standard"`
	Rule        string `json:"rule"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Location    string `json:"location"`
	Remediation string `json:"remediation"`
}