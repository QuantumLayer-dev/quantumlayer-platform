package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// EventType represents the type of audit event
type EventType string

const (
	EventTypeAPICall        EventType = "api_call"
	EventTypeAuthentication EventType = "authentication"
	EventTypeAuthorization  EventType = "authorization"
	EventTypeDataAccess     EventType = "data_access"
	EventTypeDataModification EventType = "data_modification"
	EventTypeConfigChange   EventType = "config_change"
	EventTypeSystemAccess   EventType = "system_access"
	EventTypeSecurityAlert  EventType = "security_alert"
	EventTypeCompliance     EventType = "compliance"
	EventTypePII            EventType = "pii_access"
)

// Severity represents the severity level of an audit event
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// ComplianceStandard represents compliance standards
type ComplianceStandard string

const (
	ComplianceGDPR   ComplianceStandard = "GDPR"
	ComplianceSOC2   ComplianceStandard = "SOC2"
	ComplianceHIPAA  ComplianceStandard = "HIPAA"
	CompliancePCI    ComplianceStandard = "PCI-DSS"
	ComplianceISO27001 ComplianceStandard = "ISO27001"
)

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID               string                 `json:"id"`
	Timestamp        time.Time              `json:"timestamp"`
	EventType        EventType              `json:"event_type"`
	Severity         Severity               `json:"severity"`
	Actor            *Actor                 `json:"actor"`
	Resource         *Resource              `json:"resource"`
	Action           string                 `json:"action"`
	Result           string                 `json:"result"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	RequestID        string                 `json:"request_id,omitempty"`
	SessionID        string                 `json:"session_id,omitempty"`
	SourceIP         string                 `json:"source_ip"`
	UserAgent        string                 `json:"user_agent,omitempty"`
	Method           string                 `json:"method,omitempty"`
	Path             string                 `json:"path,omitempty"`
	StatusCode       int                    `json:"status_code,omitempty"`
	Latency          time.Duration          `json:"latency,omitempty"`
	DataAccessed     []string               `json:"data_accessed,omitempty"`
	DataModified     map[string]interface{} `json:"data_modified,omitempty"`
	ComplianceFlags  []ComplianceStandard   `json:"compliance_flags,omitempty"`
	Tags             map[string]string      `json:"tags,omitempty"`
	Hash             string                 `json:"hash"`
	PreviousHash     string                 `json:"previous_hash,omitempty"`
}

// Actor represents the entity performing the action
type Actor struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // user, service, system
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	OrgID    string `json:"org_id,omitempty"`
	TeamID   string `json:"team_id,omitempty"`
}

// Resource represents the resource being accessed
type Resource struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Owner    string `json:"owner,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// AuditLogger provides audit logging functionality
type AuditLogger struct {
	logger       *zap.Logger
	storage      StorageBackend
	encryptor    Encryptor
	previousHash string
	config       *Config
}

// Config holds audit logger configuration
type Config struct {
	EnableEncryption     bool
	EnableBlockchain     bool
	RetentionDays        int
	ComplianceMode       []ComplianceStandard
	SensitiveDataMasking bool
	RealTimeAlerts       bool
	AlertWebhook         string
}

// StorageBackend interface for audit log storage
type StorageBackend interface {
	Store(ctx context.Context, event *AuditEvent) error
	Query(ctx context.Context, filter QueryFilter) ([]*AuditEvent, error)
	GetLastHash(ctx context.Context) (string, error)
}

// Encryptor interface for audit log encryption
type Encryptor interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

// QueryFilter for querying audit logs
type QueryFilter struct {
	StartTime      time.Time
	EndTime        time.Time
	EventTypes     []EventType
	Severity       []Severity
	ActorID        string
	ResourceID     string
	ComplianceFlag ComplianceStandard
	Limit          int
	Offset         int
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *Config, storage StorageBackend, encryptor Encryptor, logger *zap.Logger) (*AuditLogger, error) {
	lastHash, err := storage.GetLastHash(context.Background())
	if err != nil {
		logger.Warn("Failed to get last hash", zap.Error(err))
		lastHash = ""
	}
	
	return &AuditLogger{
		logger:       logger,
		storage:      storage,
		encryptor:    encryptor,
		previousHash: lastHash,
		config:       config,
	}, nil
}

// LogEvent logs an audit event
func (a *AuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	// Set defaults
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	
	// Apply compliance rules
	a.applyComplianceRules(event)
	
	// Mask sensitive data if configured
	if a.config.SensitiveDataMasking {
		a.maskSensitiveData(event)
	}
	
	// Calculate hash for blockchain-style integrity
	if a.config.EnableBlockchain {
		event.PreviousHash = a.previousHash
		event.Hash = a.calculateHash(event)
		a.previousHash = event.Hash
	}
	
	// Encrypt if configured
	if a.config.EnableEncryption {
		if err := a.encryptEvent(event); err != nil {
			return fmt.Errorf("failed to encrypt audit event: %w", err)
		}
	}
	
	// Store the event
	if err := a.storage.Store(ctx, event); err != nil {
		a.logger.Error("Failed to store audit event",
			zap.String("event_id", event.ID),
			zap.Error(err),
		)
		return err
	}
	
	// Send real-time alerts for critical events
	if a.config.RealTimeAlerts && event.Severity == SeverityCritical {
		go a.sendAlert(event)
	}
	
	// Log to structured logger as well
	a.logToZap(event)
	
	return nil
}

// LogAPICall logs an API call audit event
func (a *AuditLogger) LogAPICall(ctx context.Context, method, path string, statusCode int, latency time.Duration, actor *Actor, err error) {
	event := &AuditEvent{
		EventType:  EventTypeAPICall,
		Severity:   SeverityInfo,
		Actor:      actor,
		Action:     fmt.Sprintf("%s %s", method, path),
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Latency:    latency,
		Timestamp:  time.Now().UTC(),
	}
	
	if err != nil {
		event.Result = "failure"
		event.ErrorMessage = err.Error()
		if statusCode >= 500 {
			event.Severity = SeverityError
		} else if statusCode >= 400 {
			event.Severity = SeverityWarning
		}
	} else {
		event.Result = "success"
	}
	
	a.LogEvent(ctx, event)
}

// LogDataAccess logs data access events for compliance
func (a *AuditLogger) LogDataAccess(ctx context.Context, actor *Actor, resource *Resource, dataFields []string, purpose string) {
	event := &AuditEvent{
		EventType:    EventTypeDataAccess,
		Severity:     SeverityInfo,
		Actor:        actor,
		Resource:     resource,
		Action:       fmt.Sprintf("accessed %s", purpose),
		Result:       "success",
		DataAccessed: dataFields,
		Timestamp:    time.Now().UTC(),
		ComplianceFlags: []ComplianceStandard{ComplianceGDPR, ComplianceSOC2},
	}
	
	// Check if PII is accessed
	if a.containsPII(dataFields) {
		event.EventType = EventTypePII
		event.Severity = SeverityWarning
		event.ComplianceFlags = append(event.ComplianceFlags, ComplianceGDPR)
	}
	
	a.LogEvent(ctx, event)
}

// LogSecurityEvent logs security-related events
func (a *AuditLogger) LogSecurityEvent(ctx context.Context, eventType EventType, severity Severity, description string, metadata map[string]string) {
	event := &AuditEvent{
		EventType: eventType,
		Severity:  severity,
		Action:    description,
		Tags:      metadata,
		Timestamp: time.Now().UTC(),
		ComplianceFlags: []ComplianceStandard{ComplianceSOC2, ComplianceISO27001},
	}
	
	a.LogEvent(ctx, event)
}

// Query queries audit logs
func (a *AuditLogger) Query(ctx context.Context, filter QueryFilter) ([]*AuditEvent, error) {
	events, err := a.storage.Query(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	// Decrypt if needed
	if a.config.EnableEncryption {
		for _, event := range events {
			if err := a.decryptEvent(event); err != nil {
				a.logger.Error("Failed to decrypt audit event", zap.Error(err))
			}
		}
	}
	
	return events, nil
}

// applyComplianceRules applies compliance-specific rules
func (a *AuditLogger) applyComplianceRules(event *AuditEvent) {
	for _, standard := range a.config.ComplianceMode {
		switch standard {
		case ComplianceGDPR:
			// GDPR requires explicit logging of data processing
			if event.EventType == EventTypeDataAccess || event.EventType == EventTypeDataModification {
				event.ComplianceFlags = append(event.ComplianceFlags, ComplianceGDPR)
			}
		case ComplianceSOC2:
			// SOC2 requires security event logging
			if event.EventType == EventTypeSecurityAlert || event.EventType == EventTypeSystemAccess {
				event.ComplianceFlags = append(event.ComplianceFlags, ComplianceSOC2)
			}
		case ComplianceHIPAA:
			// HIPAA requires PHI access logging
			if event.EventType == EventTypePII {
				event.ComplianceFlags = append(event.ComplianceFlags, ComplianceHIPAA)
			}
		}
	}
}

// maskSensitiveData masks sensitive information
func (a *AuditLogger) maskSensitiveData(event *AuditEvent) {
	// Mask email addresses
	if event.Actor != nil && event.Actor.Email != "" {
		parts := strings.Split(event.Actor.Email, "@")
		if len(parts) == 2 {
			masked := strings.Repeat("*", len(parts[0])-2) + parts[0][len(parts[0])-2:]
			event.Actor.Email = masked + "@" + parts[1]
		}
	}
	
	// Mask sensitive fields
	sensitiveFields := []string{"password", "token", "secret", "key", "ssn", "credit_card"}
	for _, field := range event.DataAccessed {
		for _, sensitive := range sensitiveFields {
			if strings.Contains(strings.ToLower(field), sensitive) {
				event.DataAccessed = []string{"[REDACTED]"}
				break
			}
		}
	}
}

// calculateHash calculates hash for blockchain integrity
func (a *AuditLogger) calculateHash(event *AuditEvent) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		event.ID,
		event.Timestamp.Format(time.RFC3339Nano),
		event.EventType,
		event.Action,
		event.Result,
		event.PreviousHash,
	)
	
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// containsPII checks if fields contain PII
func (a *AuditLogger) containsPII(fields []string) bool {
	piiFields := []string{"email", "phone", "ssn", "address", "name", "dob", "credit_card"}
	for _, field := range fields {
		fieldLower := strings.ToLower(field)
		for _, pii := range piiFields {
			if strings.Contains(fieldLower, pii) {
				return true
			}
		}
	}
	return false
}

// encryptEvent encrypts sensitive event data
func (a *AuditLogger) encryptEvent(event *AuditEvent) error {
	// Encrypt sensitive fields
	// Implementation depends on encryptor
	return nil
}

// decryptEvent decrypts event data
func (a *AuditLogger) decryptEvent(event *AuditEvent) error {
	// Decrypt sensitive fields
	// Implementation depends on encryptor
	return nil
}

// sendAlert sends real-time alert for critical events
func (a *AuditLogger) sendAlert(event *AuditEvent) {
	// Send to webhook, Slack, PagerDuty, etc.
	a.logger.Warn("Critical audit event",
		zap.String("event_id", event.ID),
		zap.String("type", string(event.EventType)),
		zap.String("severity", string(event.Severity)),
	)
}

// logToZap logs to structured logger
func (a *AuditLogger) logToZap(event *AuditEvent) {
	fields := []zap.Field{
		zap.String("audit_id", event.ID),
		zap.String("event_type", string(event.EventType)),
		zap.String("severity", string(event.Severity)),
		zap.String("action", event.Action),
		zap.String("result", event.Result),
	}
	
	if event.Actor != nil {
		fields = append(fields, 
			zap.String("actor_id", event.Actor.ID),
			zap.String("actor_type", event.Actor.Type),
		)
	}
	
	if event.Resource != nil {
		fields = append(fields,
			zap.String("resource_id", event.Resource.ID),
			zap.String("resource_type", event.Resource.Type),
		)
	}
	
	switch event.Severity {
	case SeverityCritical:
		a.logger.Error("Audit event", fields...)
	case SeverityError:
		a.logger.Error("Audit event", fields...)
	case SeverityWarning:
		a.logger.Warn("Audit event", fields...)
	default:
		a.logger.Info("Audit event", fields...)
	}
}