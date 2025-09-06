package activities

import (
	"context"
	"time"
)

// Core monitoring interfaces and types

// EnterpriseMonitoringRequest defines the monitoring setup request
type EnterpriseMonitoringRequest struct {
	DeploymentID         string                    `json:"deployment_id"`
	ApplicationName      string                    `json:"application_name"`
	ApplicationType      string                    `json:"application_type"`
	Language             string                    `json:"language"`
	Framework            string                    `json:"framework"`
	Environment          string                    `json:"environment"`
	Provider             string                    `json:"provider"`
	Region               string                    `json:"region"`
	Configuration        MonitoringConfiguration   `json:"configuration"`
	MetricsEndpoint      string                    `json:"metrics_endpoint"`
	LoggingEndpoint      string                    `json:"logging_endpoint"`
	TracingEndpoint      string                    `json:"tracing_endpoint"`
	DashboardEndpoint    string                    `json:"dashboard_endpoint"`
	BusinessContext      BusinessContext           `json:"business_context"`
	ComplianceRequirements []string                `json:"compliance_requirements"`
	Features             []string                  `json:"features"`
}

// EnterpriseMonitoringResult contains the monitoring setup results
type EnterpriseMonitoringResult struct {
	Success             bool                          `json:"success"`
	DeploymentID        string                        `json:"deployment_id"`
	Components          map[string]MonitoringComponent `json:"components"`
	Dashboards          []Dashboard                   `json:"dashboards"`
	AlertRules          []AlertRule                   `json:"alert_rules"`
	AutomatedWorkflows  []AutomatedWorkflow           `json:"automated_workflows"`
	SLOs                []SLO                         `json:"slos"`
	HealthEndpoints     []HealthEndpoint              `json:"health_endpoints"`
	MonitoringURLs      MonitoringURLs                `json:"monitoring_urls"`
	SetupDuration       time.Duration                 `json:"setup_duration"`
	StartTime           time.Time                     `json:"start_time"`
	EndTime             time.Time                     `json:"end_time"`
	Errors              []string                      `json:"errors,omitempty"`
}

// Configuration structures for different monitoring aspects
type MetricsConfiguration struct {
	Enabled          bool              `json:"enabled"`
	Provider         string            `json:"provider"`
	RetentionDays    int               `json:"retention_days"`
	ScrapeInterval   string            `json:"scrape_interval"`
	CustomMetrics    []CustomMetric    `json:"custom_metrics"`
	Cardinality      CardinalityConfig `json:"cardinality"`
	Exporters        []MetricExporter  `json:"exporters"`
}

type LoggingConfiguration struct {
	Enabled        bool                `json:"enabled"`
	Level          string              `json:"level"`
	Format         string              `json:"format"`
	Structured     bool                `json:"structured"`
	Retention      LogRetentionConfig  `json:"retention"`
	Aggregation    LogAggregationConfig `json:"aggregation"`
	Parsing        LogParsingConfig    `json:"parsing"`
	Sampling       LogSamplingConfig   `json:"sampling"`
}

type TracingConfiguration struct {
	Enabled        bool              `json:"enabled"`
	Provider       string            `json:"provider"`
	SamplingRate   float64           `json:"sampling_rate"`
	MaxSpans       int               `json:"max_spans"`
	RetentionHours int               `json:"retention_hours"`
	Propagators    []string          `json:"propagators"`
	Processors     []TraceProcessor  `json:"processors"`
}

type APMConfiguration struct {
	Enabled              bool     `json:"enabled"`
	Agents               []string `json:"agents"`
	SamplingRate         float64  `json:"sampling_rate"`
	ErrorTracking        bool     `json:"error_tracking"`
	PerformanceProfiling bool     `json:"performance_profiling"`
	MemoryProfiling      bool     `json:"memory_profiling"`
	CPUProfiling         bool     `json:"cpu_profiling"`
}

type RUMConfiguration struct {
	Enabled           bool     `json:"enabled"`
	SamplingRate      float64  `json:"sampling_rate"`
	SessionReplay     bool     `json:"session_replay"`
	UserTracking      bool     `json:"user_tracking"`
	PerformanceMetrics bool    `json:"performance_metrics"`
}

type SyntheticConfiguration struct {
	Enabled    bool                  `json:"enabled"`
	Checks     []SyntheticCheck      `json:"checks"`
	Frequency  time.Duration         `json:"frequency"`
	Locations  []string              `json:"locations"`
}

type SLOConfiguration struct {
	Enabled    bool    `json:"enabled"`
	Objectives []SLO   `json:"objectives"`
}

type AlertingConfiguration struct {
	Enabled        bool                `json:"enabled"`
	Providers      []string            `json:"providers"`
	Channels       []AlertChannel      `json:"channels"`
	Escalation     EscalationPolicy    `json:"escalation"`
	Suppression    SuppressionConfig   `json:"suppression"`
	Correlation    CorrelationConfig   `json:"correlation"`
}

type DashboardConfiguration struct {
	Enabled        bool       `json:"enabled"`
	Provider       string     `json:"provider"`
	Templates      []string   `json:"templates"`
	CustomBoards   []Dashboard `json:"custom_boards"`
}

type SecurityMonitoring struct {
	Enabled           bool     `json:"enabled"`
	ThreatDetection   bool     `json:"threat_detection"`
	AnomalyDetection  bool     `json:"anomaly_detection"`
	AccessMonitoring  bool     `json:"access_monitoring"`
	VulnScanning      bool     `json:"vulnerability_scanning"`
}

type ComplianceMonitoring struct {
	Enabled       bool     `json:"enabled"`
	Standards     []string `json:"standards"`
	AuditLogging  bool     `json:"audit_logging"`
	Reporting     bool     `json:"reporting"`
	Automated     bool     `json:"automated"`
}

type PerformanceMonitoring struct {
	Enabled       bool                  `json:"enabled"`
	Benchmarking  bool                  `json:"benchmarking"`
	LoadTesting   bool                  `json:"load_testing"`
	Optimization  bool                  `json:"optimization"`
	Profiling     ProfilingConfig       `json:"profiling"`
}

type CostMonitoring struct {
	Enabled      bool              `json:"enabled"`
	Provider     string            `json:"provider"`
	Budgets      []Budget          `json:"budgets"`
	Optimization bool              `json:"optimization"`
	Forecasting  bool              `json:"forecasting"`
}

type InfrastructureMonitoring struct {
	Enabled      bool     `json:"enabled"`
	Containers   bool     `json:"containers"`
	Kubernetes   bool     `json:"kubernetes"`
	Network      bool     `json:"network"`
	Storage      bool     `json:"storage"`
	Cloud        bool     `json:"cloud"`
}

type ApplicationMonitoring struct {
	Enabled         bool     `json:"enabled"`
	HealthChecks    bool     `json:"health_checks"`
	Dependencies    bool     `json:"dependencies"`
	BusinessMetrics bool     `json:"business_metrics"`
	UserExperience  bool     `json:"user_experience"`
}

type BusinessMonitoring struct {
	Enabled    bool            `json:"enabled"`
	KPIs       []KPI           `json:"kpis"`
	Conversion []Conversion    `json:"conversion"`
	Revenue    RevenueTracking `json:"revenue"`
}

// Component and service definitions
type MonitoringComponent struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Status        string                 `json:"status"`
	Endpoint      string                 `json:"endpoint"`
	Configuration map[string]interface{} `json:"configuration"`
	Health        string                 `json:"health"`
	StartedAt     time.Time              `json:"started_at"`
	Version       string                 `json:"version,omitempty"`
	Dependencies  []string               `json:"dependencies,omitempty"`
}

type Dashboard struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Type         string              `json:"type"`
	URL          string              `json:"url"`
	Description  string              `json:"description"`
	Panels       []DashboardPanel    `json:"panels"`
	Variables    []DashboardVariable `json:"variables"`
	Permissions  []Permission        `json:"permissions"`
	Tags         []string            `json:"tags"`
}

type AlertRule struct {
	Name        string   `json:"name"`
	Expression  string   `json:"expression"`
	Severity    string   `json:"severity"`
	Duration    string   `json:"duration"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type AutomatedWorkflow struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Trigger     WorkflowTrigger   `json:"trigger"`
	Actions     []WorkflowAction  `json:"actions"`
	Schedule    string            `json:"schedule,omitempty"`
	Enabled     bool              `json:"enabled"`
}

type SLO struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Target      float64       `json:"target"`
	Window      time.Duration `json:"window"`
	Metric      SLOMetric     `json:"metric"`
	AlertPolicy AlertPolicy   `json:"alert_policy"`
}

type HealthEndpoint struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Method   string `json:"method"`
	Interval string `json:"interval"`
	Timeout  string `json:"timeout"`
}

type MonitoringURLs struct {
	Metrics    string `json:"metrics"`
	Logs       string `json:"logs"`
	Traces     string `json:"traces"`
	Dashboards string `json:"dashboards"`
	Alerts     string `json:"alerts"`
	Health     string `json:"health"`
}

// Supporting configuration types
type CustomMetric struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
	Unit        string            `json:"unit"`
}

type CardinalityConfig struct {
	MaxSeries   int      `json:"max_series"`
	MaxLabels   int      `json:"max_labels"`
	HighCardinalityLabels []string `json:"high_cardinality_labels"`
}

type MetricExporter struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Endpoint string                 `json:"endpoint"`
	Config   map[string]interface{} `json:"config"`
}

type LogRetentionConfig struct {
	HotTier  string `json:"hot_tier"`
	WarmTier string `json:"warm_tier"`
	ColdTier string `json:"cold_tier"`
}

type LogAggregationConfig struct {
	Enabled    bool   `json:"enabled"`
	BatchSize  int    `json:"batch_size"`
	FlushInterval string `json:"flush_interval"`
}

type LogParsingConfig struct {
	Enabled   bool              `json:"enabled"`
	Parsers   []LogParser       `json:"parsers"`
	Grok      GrokConfig        `json:"grok"`
	JSON      JSONParsingConfig `json:"json"`
}

type LogSamplingConfig struct {
	Enabled bool    `json:"enabled"`
	Rate    float64 `json:"rate"`
	Rules   []SamplingRule `json:"rules"`
}

type TraceProcessor struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

type SyntheticCheck struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Target    string            `json:"target"`
	Interval  time.Duration     `json:"interval"`
	Timeout   time.Duration     `json:"timeout"`
	Assertions []Assertion      `json:"assertions"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type AlertChannel struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Config   map[string]interface{} `json:"config"`
	Enabled  bool                   `json:"enabled"`
}

type EscalationPolicy struct {
	Name    string             `json:"name"`
	Rules   []EscalationRule   `json:"rules"`
	Enabled bool               `json:"enabled"`
}

type SuppressionConfig struct {
	Enabled   bool                `json:"enabled"`
	Rules     []SuppressionRule   `json:"rules"`
	Duration  time.Duration       `json:"duration"`
}

type CorrelationConfig struct {
	Enabled    bool              `json:"enabled"`
	TimeWindow time.Duration     `json:"time_window"`
	Rules      []CorrelationRule `json:"rules"`
}

type ProfilingConfig struct {
	CPU      bool              `json:"cpu"`
	Memory   bool              `json:"memory"`
	Goroutine bool             `json:"goroutine"`
	Block    bool              `json:"block"`
	Mutex    bool              `json:"mutex"`
	Interval time.Duration     `json:"interval"`
}

type Budget struct {
	Name      string    `json:"name"`
	Limit     float64   `json:"limit"`
	Period    string    `json:"period"`
	Currency  string    `json:"currency"`
	Alerts    []float64 `json:"alerts"`
}

type KPI struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Target      float64 `json:"target"`
	Current     float64 `json:"current"`
	Trend       string  `json:"trend"`
}

type Conversion struct {
	Name   string  `json:"name"`
	Funnel []string `json:"funnel"`
	Rate   float64 `json:"rate"`
}

type RevenueTracking struct {
	Enabled   bool    `json:"enabled"`
	Currency  string  `json:"currency"`
	Target    float64 `json:"target"`
	Current   float64 `json:"current"`
}

// Dashboard components
type DashboardPanel struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Type        string                 `json:"type"`
	Query       string                 `json:"query"`
	Datasource  string                 `json:"datasource"`
	TimeRange   string                 `json:"time_range"`
	Options     map[string]interface{} `json:"options"`
	Position    PanelPosition          `json:"position"`
}

type DashboardVariable struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Values  []string `json:"values"`
	Default string   `json:"default"`
}

type Permission struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

type PanelPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Workflow components
type WorkflowTrigger struct {
	Type      string                 `json:"type"`
	Condition string                 `json:"condition"`
	Config    map[string]interface{} `json:"config"`
}

type WorkflowAction struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

type SLOMetric struct {
	Name      string  `json:"name"`
	Query     string  `json:"query"`
	Good      string  `json:"good"`
	Total     string  `json:"total"`
	Threshold float64 `json:"threshold"`
}

type AlertPolicy struct {
	BurnRateRules []BurnRateRule `json:"burn_rate_rules"`
	ErrorBudget   ErrorBudget    `json:"error_budget"`
}

// Business context
type BusinessContext struct {
	CriticalUserJourneys []UserJourney `json:"critical_user_journeys"`
	RevenueImpact        string        `json:"revenue_impact"`
	CustomerSegments     []string      `json:"customer_segments"`
	BusinessHours        BusinessHours `json:"business_hours"`
	SLARequirements      []SLARequirement `json:"sla_requirements"`
}

// Additional supporting types
type LogParser struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Fields  []string `json:"fields"`
}

type GrokConfig struct {
	Enabled  bool              `json:"enabled"`
	Patterns map[string]string `json:"patterns"`
}

type JSONParsingConfig struct {
	Enabled     bool     `json:"enabled"`
	FlattenKeys bool     `json:"flatten_keys"`
	IgnoreKeys  []string `json:"ignore_keys"`
}

type SamplingRule struct {
	Name      string  `json:"name"`
	Condition string  `json:"condition"`
	Rate      float64 `json:"rate"`
}

type Assertion struct {
	Type     string      `json:"type"`
	Target   string      `json:"target"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type EscalationRule struct {
	Delay   time.Duration `json:"delay"`
	Target  string        `json:"target"`
	Actions []string      `json:"actions"`
}

type SuppressionRule struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Duration  time.Duration `json:"duration"`
}

type CorrelationRule struct {
	Name      string   `json:"name"`
	Patterns  []string `json:"patterns"`
	Action    string   `json:"action"`
}

type BurnRateRule struct {
	ShortWindow  time.Duration `json:"short_window"`
	LongWindow   time.Duration `json:"long_window"`
	BurnRate     float64       `json:"burn_rate"`
	Severity     string        `json:"severity"`
}

type ErrorBudget struct {
	Remaining  float64 `json:"remaining"`
	Consumption float64 `json:"consumption"`
	Alerting   bool    `json:"alerting"`
}

type UserJourney struct {
	Name        string   `json:"name"`
	Steps       []string `json:"steps"`
	Importance  string   `json:"importance"`
	SLOTarget   float64  `json:"slo_target"`
}

type BusinessHours struct {
	Timezone string `json:"timezone"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Days     []string `json:"days"`
}

type SLARequirement struct {
	Name        string  `json:"name"`
	Target      float64 `json:"target"`
	Penalty     string  `json:"penalty"`
	Measurement string  `json:"measurement"`
}

// Provider interfaces
type MetricsCollector interface {
	Collect(ctx context.Context, metrics []CustomMetric) error
	Query(ctx context.Context, query string) (interface{}, error)
}

type LogAggregator interface {
	Ingest(ctx context.Context, logs []LogEntry) error
	Search(ctx context.Context, query string) ([]LogEntry, error)
}

type TraceProvider interface {
	StartTrace(ctx context.Context, operation string) (context.Context, error)
	FinishTrace(ctx context.Context) error
}

type DashboardProvider interface {
	CreateDashboard(ctx context.Context, dashboard Dashboard) error
	UpdateDashboard(ctx context.Context, dashboard Dashboard) error
}

type SLOManager interface {
	CreateSLO(ctx context.Context, slo SLO) error
	EvaluateSLO(ctx context.Context, sloName string) (SLOResult, error)
}

type ComplianceMonitor interface {
	CheckCompliance(ctx context.Context, standards []string) (ComplianceResult, error)
	GenerateReport(ctx context.Context) (ComplianceReport, error)
}

type CostMonitor interface {
	CalculateCost(ctx context.Context, resources []Resource) (float64, error)
	OptimizeResources(ctx context.Context) ([]Optimization, error)
}

type PerformanceAnalyzer interface {
	AnalyzePerformance(ctx context.Context, metrics []PerformanceMetric) (PerformanceReport, error)
	GetOptimizationSuggestions(ctx context.Context) ([]Suggestion, error)
}

// Supporting data types
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
}

type SLOResult struct {
	Name           string    `json:"name"`
	Current        float64   `json:"current"`
	Target         float64   `json:"target"`
	ErrorBudget    float64   `json:"error_budget"`
	Status         string    `json:"status"`
	LastEvaluated  time.Time `json:"last_evaluated"`
}

type ComplianceReport struct {
	Standards  []StandardResult `json:"standards"`
	Overall    string           `json:"overall"`
	Generated  time.Time        `json:"generated"`
	ValidUntil time.Time        `json:"valid_until"`
}

type StandardResult struct {
	Name       string  `json:"name"`
	Compliant  bool    `json:"compliant"`
	Score      float64 `json:"score"`
	Issues     []Issue `json:"issues"`
}

type Issue struct {
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

type Resource struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Usage    ResourceUsage          `json:"usage"`
	Config   map[string]interface{} `json:"config"`
}

type ResourceUsage struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Storage float64 `json:"storage"`
	Network float64 `json:"network"`
}

type Optimization struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Savings     float64 `json:"savings"`
	Risk        string  `json:"risk"`
}

type PerformanceMetric struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
}

type PerformanceReport struct {
	Overall     string              `json:"overall"`
	Metrics     []PerformanceMetric `json:"metrics"`
	Bottlenecks []Bottleneck        `json:"bottlenecks"`
	Trends      []Trend             `json:"trends"`
}

type Bottleneck struct {
	Component   string  `json:"component"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Severity    float64 `json:"severity"`
}

type Trend struct {
	Metric    string  `json:"metric"`
	Direction string  `json:"direction"`
	Rate      float64 `json:"rate"`
}

type Suggestion struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Effort      string  `json:"effort"`
	Priority    int     `json:"priority"`
}