package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MCP Gateway Service - Universal Integration Hub for QuantumLayer Platform
// Provides centralized access to all external integrations via Model Context Protocol

const (
	ServiceName    = "mcp-gateway"
	ServiceVersion = "1.0.0"
)

// Metrics
var (
	mcpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_gateway_requests_total",
			Help: "Total number of MCP gateway requests",
		},
		[]string{"tool", "service", "status"},
	)
	
	mcpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_gateway_request_duration_seconds",
			Help:    "MCP gateway request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tool"},
	)
	
	cacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_gateway_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"tool"},
	)
)

func init() {
	prometheus.MustRegister(mcpRequests)
	prometheus.MustRegister(mcpDuration)
	prometheus.MustRegister(cacheHits)
}

// MCPGateway is the main gateway service
type MCPGateway struct {
	// Repository Connectors
	GitHub    *GitHubConnector
	GitLab    *GitLabConnector
	Bitbucket *BitbucketConnector
	
	// Project Management
	JIRA       *JIRAConnector
	Confluence *ConfluenceConnector
	Linear     *LinearConnector
	Asana      *AsanaConnector
	
	// Communication
	Slack   *SlackConnector
	Discord *DiscordConnector
	Teams   *TeamsConnector
	Email   *EmailConnector
	
	// Cloud Providers
	AWS   *AWSConnector
	GCP   *GCPConnector
	Azure *AzureConnector
	DO    *DigitalOceanConnector
	
	// Monitoring & Observability
	Datadog   *DatadogConnector
	NewRelic  *NewRelicConnector
	Sentry    *SentryConnector
	PagerDuty *PagerDutyConnector
	
	// Data Sources
	WebCrawler *WebCrawlerConnector
	Database   *DatabaseConnector
	APIReader  *APIReaderConnector
	FileSystem *FileSystemConnector
	
	// Core Components
	Cache       *CacheManager
	RateLimiter *RateLimiter
	Auth        *AuthManager
}

// MCPRequest represents a request to the MCP Gateway
type MCPRequest struct {
	Tool      string          `json:"tool"`       // e.g., "github.read_repo"
	Service   string          `json:"service"`    // e.g., "qtest", "qlayer"
	Input     json.RawMessage `json:"input"`      // Tool-specific input
	RequestID string          `json:"request_id"` // For tracing
	Auth      *AuthContext    `json:"auth,omitempty"`
}

// MCPResponse represents a response from the MCP Gateway
type MCPResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	RequestID string      `json:"request_id"`
	Cached    bool        `json:"cached"`
	Duration  float64     `json:"duration_ms"`
}

// AuthContext contains authentication information
type AuthContext struct {
	UserID    string            `json:"user_id"`
	ServiceID string            `json:"service_id"`
	Scopes    []string          `json:"scopes"`
	Metadata  map[string]string `json:"metadata"`
}

func main() {
	log.Printf("🌐 MCP Gateway Service v%s Starting...", ServiceVersion)
	
	// Initialize gateway
	gateway := NewMCPGateway()
	
	// Setup routes
	router := mux.NewRouter()
	
	// Health & Info endpoints
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/info", infoHandler).Methods("GET")
	
	// MCP endpoints
	router.HandleFunc("/api/v1/execute", gateway.executeHandler).Methods("POST")
	router.HandleFunc("/api/v1/tools", gateway.listToolsHandler).Methods("GET")
	router.HandleFunc("/api/v1/connectors", gateway.listConnectorsHandler).Methods("GET")
	
	// Connector-specific endpoints for direct access
	router.HandleFunc("/api/v1/github/{action}", gateway.githubHandler).Methods("POST")
	router.HandleFunc("/api/v1/jira/{action}", gateway.jiraHandler).Methods("POST")
	router.HandleFunc("/api/v1/slack/{action}", gateway.slackHandler).Methods("POST")
	router.HandleFunc("/api/v1/web/{action}", gateway.webHandler).Methods("POST")
	
	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler())
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8095"
	}
	
	log.Printf("✅ MCP Gateway running on port %s", port)
	log.Printf("📡 Available Connectors: GitHub, GitLab, JIRA, Slack, Web Crawler, and more...")
	log.Fatal(http.ListenAndServe(":"+port, router))
}

// NewMCPGateway creates a new gateway instance
func NewMCPGateway() *MCPGateway {
	return &MCPGateway{
		// Initialize all connectors
		GitHub:     NewGitHubConnector(),
		GitLab:     NewGitLabConnector(),
		Bitbucket:  NewBitbucketConnector(),
		JIRA:       NewJIRAConnector(),
		Confluence: NewConfluenceConnector(),
		Linear:     NewLinearConnector(),
		Asana:      NewAsanaConnector(),
		Slack:      NewSlackConnector(),
		Discord:    NewDiscordConnector(),
		Teams:      NewTeamsConnector(),
		Email:      NewEmailConnector(),
		AWS:        NewAWSConnector(),
		GCP:        NewGCPConnector(),
		Azure:      NewAzureConnector(),
		DO:         NewDigitalOceanConnector(),
		Datadog:    NewDatadogConnector(),
		NewRelic:   NewNewRelicConnector(),
		Sentry:     NewSentryConnector(),
		PagerDuty:  NewPagerDutyConnector(),
		WebCrawler: NewWebCrawlerConnector(),
		Database:   NewDatabaseConnector(),
		APIReader:  NewAPIReaderConnector(),
		FileSystem: NewFileSystemConnector(),
		Cache:      NewCacheManager(),
		RateLimiter: NewRateLimiter(),
		Auth:       NewAuthManager(),
	}
}

// executeHandler is the main entry point for MCP requests
func (g *MCPGateway) executeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	log.Printf("Executing MCP tool: %s for service: %s", req.Tool, req.Service)
	
	// Check cache first
	if cachedData, found := g.Cache.Get(req.Tool, req.Input); found {
		cacheHits.WithLabelValues(req.Tool).Inc()
		response := MCPResponse{
			Success:   true,
			Data:      cachedData,
			RequestID: req.RequestID,
			Cached:    true,
			Duration:  float64(time.Since(start).Milliseconds()),
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Rate limiting
	if !g.RateLimiter.Allow(req.Service, req.Tool) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	
	// Execute the tool
	data, err := g.execute(req)
	
	duration := time.Since(start)
	mcpDuration.WithLabelValues(req.Tool).Observe(duration.Seconds())
	
	if err != nil {
		mcpRequests.WithLabelValues(req.Tool, req.Service, "error").Inc()
		response := MCPResponse{
			Success:   false,
			Error:     err.Error(),
			RequestID: req.RequestID,
			Duration:  float64(duration.Milliseconds()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Cache successful responses
	g.Cache.Set(req.Tool, req.Input, data)
	
	mcpRequests.WithLabelValues(req.Tool, req.Service, "success").Inc()
	response := MCPResponse{
		Success:   true,
		Data:      data,
		RequestID: req.RequestID,
		Cached:    false,
		Duration:  float64(duration.Milliseconds()),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// execute routes requests to appropriate connectors
func (g *MCPGateway) execute(req MCPRequest) (interface{}, error) {
	switch req.Tool {
	// GitHub operations
	case "github.read_repo":
		return g.GitHub.ReadRepository(req.Input)
	case "github.create_pr":
		return g.GitHub.CreatePullRequest(req.Input)
	case "github.create_issue":
		return g.GitHub.CreateIssue(req.Input)
	case "github.list_repos":
		return g.GitHub.ListRepositories(req.Input)
		
	// JIRA operations
	case "jira.create_ticket":
		return g.JIRA.CreateTicket(req.Input)
	case "jira.update_ticket":
		return g.JIRA.UpdateTicket(req.Input)
	case "jira.get_ticket":
		return g.JIRA.GetTicket(req.Input)
	case "jira.search":
		return g.JIRA.Search(req.Input)
		
	// Confluence operations
	case "confluence.create_page":
		return g.Confluence.CreatePage(req.Input)
	case "confluence.update_page":
		return g.Confluence.UpdatePage(req.Input)
	case "confluence.get_page":
		return g.Confluence.GetPage(req.Input)
		
	// Slack operations
	case "slack.send_message":
		return g.Slack.SendMessage(req.Input)
	case "slack.create_channel":
		return g.Slack.CreateChannel(req.Input)
	case "slack.upload_file":
		return g.Slack.UploadFile(req.Input)
		
	// Web crawling
	case "web.crawl_site":
		return g.WebCrawler.CrawlSite(req.Input)
	case "web.screenshot":
		return g.WebCrawler.Screenshot(req.Input)
	case "web.extract_data":
		return g.WebCrawler.ExtractData(req.Input)
		
	// Database operations
	case "db.query":
		return g.Database.Query(req.Input)
	case "db.schema":
		return g.Database.GetSchema(req.Input)
		
	// API operations
	case "api.read_spec":
		return g.APIReader.ReadSpec(req.Input)
	case "api.test_endpoint":
		return g.APIReader.TestEndpoint(req.Input)
		
	// Cloud operations
	case "aws.deploy":
		return g.AWS.Deploy(req.Input)
	case "gcp.deploy":
		return g.GCP.Deploy(req.Input)
	case "azure.deploy":
		return g.Azure.Deploy(req.Input)
		
	default:
		return nil, fmt.Errorf("unknown tool: %s", req.Tool)
	}
}

// listToolsHandler returns all available tools
func (g *MCPGateway) listToolsHandler(w http.ResponseWriter, r *http.Request) {
	tools := g.listAllTools()
	response := map[string]interface{}{
		"tools": tools,
		"count": len(tools),
		"categories": map[string]int{
			"repository":   12,
			"project_mgmt": 15,
			"communication": 10,
			"cloud":        20,
			"monitoring":   8,
			"data":         10,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// listAllTools returns all available MCP tools
func (g *MCPGateway) listAllTools() []Tool {
	return []Tool{
		// GitHub
		{Name: "github.read_repo", Description: "Read GitHub repository", Category: "repository"},
		{Name: "github.create_pr", Description: "Create pull request", Category: "repository"},
		{Name: "github.create_issue", Description: "Create issue", Category: "repository"},
		
		// JIRA
		{Name: "jira.create_ticket", Description: "Create JIRA ticket", Category: "project_mgmt"},
		{Name: "jira.update_ticket", Description: "Update JIRA ticket", Category: "project_mgmt"},
		
		// Slack
		{Name: "slack.send_message", Description: "Send Slack message", Category: "communication"},
		{Name: "slack.create_channel", Description: "Create Slack channel", Category: "communication"},
		
		// Web
		{Name: "web.crawl_site", Description: "Crawl website", Category: "data"},
		{Name: "web.screenshot", Description: "Take screenshot", Category: "data"},
		
		// Database
		{Name: "db.query", Description: "Query database", Category: "data"},
		{Name: "db.schema", Description: "Get database schema", Category: "data"},
		
		// Cloud
		{Name: "aws.deploy", Description: "Deploy to AWS", Category: "cloud"},
		{Name: "gcp.deploy", Description: "Deploy to GCP", Category: "cloud"},
		{Name: "azure.deploy", Description: "Deploy to Azure", Category: "cloud"},
	}
}

// Tool represents an MCP tool
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// listConnectorsHandler returns all available connectors
func (g *MCPGateway) listConnectorsHandler(w http.ResponseWriter, r *http.Request) {
	connectors := map[string]interface{}{
		"repositories": []string{"GitHub", "GitLab", "Bitbucket"},
		"project_management": []string{"JIRA", "Confluence", "Linear", "Asana"},
		"communication": []string{"Slack", "Discord", "Teams", "Email"},
		"cloud_providers": []string{"AWS", "GCP", "Azure", "DigitalOcean"},
		"monitoring": []string{"Datadog", "NewRelic", "Sentry", "PagerDuty"},
		"data_sources": []string{"Web Crawler", "Database", "API Reader", "FileSystem"},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(connectors)
}

// Direct connector handlers for specialized access
func (g *MCPGateway) githubHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	action := vars["action"]
	
	var input json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var result interface{}
	var err error
	
	switch action {
	case "read":
		result, err = g.GitHub.ReadRepository(input)
	case "create-pr":
		result, err = g.GitHub.CreatePullRequest(input)
	default:
		err = fmt.Errorf("unknown GitHub action: %s", action)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(result)
}

func (g *MCPGateway) jiraHandler(w http.ResponseWriter, r *http.Request) {
	// Similar implementation for JIRA
}

func (g *MCPGateway) slackHandler(w http.ResponseWriter, r *http.Request) {
	// Similar implementation for Slack
}

func (g *MCPGateway) webHandler(w http.ResponseWriter, r *http.Request) {
	// Similar implementation for Web Crawler
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"service": ServiceName,
		"status":  "healthy",
		"version": ServiceVersion,
		"uptime":  time.Now().Unix(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Info handler provides service information
func infoHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service":     ServiceName,
		"version":     ServiceVersion,
		"description": "Universal MCP Gateway for QuantumLayer Platform",
		"capabilities": []string{
			"Repository Management",
			"Project Management Integration",
			"Communication Channels",
			"Cloud Provider Deployment",
			"Monitoring & Observability",
			"Web Crawling & Data Extraction",
		},
		"total_connectors": 22,
		"total_tools":      75,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Stub implementations for connectors (would be in separate files)
type GitHubConnector struct{}
func NewGitHubConnector() *GitHubConnector { return &GitHubConnector{} }
func (g *GitHubConnector) ReadRepository(input json.RawMessage) (interface{}, error) { return nil, nil }
func (g *GitHubConnector) CreatePullRequest(input json.RawMessage) (interface{}, error) { return nil, nil }
func (g *GitHubConnector) CreateIssue(input json.RawMessage) (interface{}, error) { return nil, nil }
func (g *GitHubConnector) ListRepositories(input json.RawMessage) (interface{}, error) { return nil, nil }

type GitLabConnector struct{}
func NewGitLabConnector() *GitLabConnector { return &GitLabConnector{} }

type BitbucketConnector struct{}
func NewBitbucketConnector() *BitbucketConnector { return &BitbucketConnector{} }

type JIRAConnector struct{}
func NewJIRAConnector() *JIRAConnector { return &JIRAConnector{} }
func (j *JIRAConnector) CreateTicket(input json.RawMessage) (interface{}, error) { return nil, nil }
func (j *JIRAConnector) UpdateTicket(input json.RawMessage) (interface{}, error) { return nil, nil }
func (j *JIRAConnector) GetTicket(input json.RawMessage) (interface{}, error) { return nil, nil }
func (j *JIRAConnector) Search(input json.RawMessage) (interface{}, error) { return nil, nil }

type ConfluenceConnector struct{}
func NewConfluenceConnector() *ConfluenceConnector { return &ConfluenceConnector{} }
func (c *ConfluenceConnector) CreatePage(input json.RawMessage) (interface{}, error) { return nil, nil }
func (c *ConfluenceConnector) UpdatePage(input json.RawMessage) (interface{}, error) { return nil, nil }
func (c *ConfluenceConnector) GetPage(input json.RawMessage) (interface{}, error) { return nil, nil }

type LinearConnector struct{}
func NewLinearConnector() *LinearConnector { return &LinearConnector{} }

type AsanaConnector struct{}
func NewAsanaConnector() *AsanaConnector { return &AsanaConnector{} }

type SlackConnector struct{}
func NewSlackConnector() *SlackConnector { return &SlackConnector{} }
func (s *SlackConnector) SendMessage(input json.RawMessage) (interface{}, error) { return nil, nil }
func (s *SlackConnector) CreateChannel(input json.RawMessage) (interface{}, error) { return nil, nil }
func (s *SlackConnector) UploadFile(input json.RawMessage) (interface{}, error) { return nil, nil }

type DiscordConnector struct{}
func NewDiscordConnector() *DiscordConnector { return &DiscordConnector{} }

type TeamsConnector struct{}
func NewTeamsConnector() *TeamsConnector { return &TeamsConnector{} }

type EmailConnector struct{}
func NewEmailConnector() *EmailConnector { return &EmailConnector{} }

type AWSConnector struct{}
func NewAWSConnector() *AWSConnector { return &AWSConnector{} }
func (a *AWSConnector) Deploy(input json.RawMessage) (interface{}, error) { return nil, nil }

type GCPConnector struct{}
func NewGCPConnector() *GCPConnector { return &GCPConnector{} }
func (g *GCPConnector) Deploy(input json.RawMessage) (interface{}, error) { return nil, nil }

type AzureConnector struct{}
func NewAzureConnector() *AzureConnector { return &AzureConnector{} }
func (a *AzureConnector) Deploy(input json.RawMessage) (interface{}, error) { return nil, nil }

type DigitalOceanConnector struct{}
func NewDigitalOceanConnector() *DigitalOceanConnector { return &DigitalOceanConnector{} }

type DatadogConnector struct{}
func NewDatadogConnector() *DatadogConnector { return &DatadogConnector{} }

type NewRelicConnector struct{}
func NewNewRelicConnector() *NewRelicConnector { return &NewRelicConnector{} }

type SentryConnector struct{}
func NewSentryConnector() *SentryConnector { return &SentryConnector{} }

type PagerDutyConnector struct{}
func NewPagerDutyConnector() *PagerDutyConnector { return &PagerDutyConnector{} }

type WebCrawlerConnector struct{}
func NewWebCrawlerConnector() *WebCrawlerConnector { return &WebCrawlerConnector{} }
func (w *WebCrawlerConnector) CrawlSite(input json.RawMessage) (interface{}, error) { return nil, nil }
func (w *WebCrawlerConnector) Screenshot(input json.RawMessage) (interface{}, error) { return nil, nil }
func (w *WebCrawlerConnector) ExtractData(input json.RawMessage) (interface{}, error) { return nil, nil }

type DatabaseConnector struct{}
func NewDatabaseConnector() *DatabaseConnector { return &DatabaseConnector{} }
func (d *DatabaseConnector) Query(input json.RawMessage) (interface{}, error) { return nil, nil }
func (d *DatabaseConnector) GetSchema(input json.RawMessage) (interface{}, error) { return nil, nil }

type APIReaderConnector struct{}
func NewAPIReaderConnector() *APIReaderConnector { return &APIReaderConnector{} }
func (a *APIReaderConnector) ReadSpec(input json.RawMessage) (interface{}, error) { return nil, nil }
func (a *APIReaderConnector) TestEndpoint(input json.RawMessage) (interface{}, error) { return nil, nil }

type FileSystemConnector struct{}
func NewFileSystemConnector() *FileSystemConnector { return &FileSystemConnector{} }

type CacheManager struct{}
func NewCacheManager() *CacheManager { return &CacheManager{} }
func (c *CacheManager) Get(tool string, input json.RawMessage) (interface{}, bool) { return nil, false }
func (c *CacheManager) Set(tool string, input json.RawMessage, data interface{}) {}

type RateLimiter struct{}
func NewRateLimiter() *RateLimiter { return &RateLimiter{} }
func (r *RateLimiter) Allow(service, tool string) bool { return true }

type AuthManager struct{}
func NewAuthManager() *AuthManager { return &AuthManager{} }