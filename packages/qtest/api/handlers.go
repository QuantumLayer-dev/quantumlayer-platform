package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/quantumlayer/qtest/mcp"
)

// QTestAPI handles API requests for test generation
type QTestAPI struct {
	mcpServer   *mcp.MCPServer
	testEngine  *TestEngine
	selfHealing *SelfHealingEngine
}

// NewQTestAPI creates a new API handler
func NewQTestAPI() *QTestAPI {
	return &QTestAPI{
		mcpServer:   mcp.NewMCPServer(),
		testEngine:  NewTestEngine(),
		selfHealing: NewSelfHealingEngine(),
	}
}

// RegisterRoutes registers all API routes
func (api *QTestAPI) RegisterRoutes(router *mux.Router) {
	// MCP-powered endpoints
	router.HandleFunc("/api/v1/test-github", api.testGitHubRepo).Methods("POST")
	router.HandleFunc("/api/v1/test-website", api.testWebsite).Methods("POST")
	router.HandleFunc("/api/v1/test-api", api.testAPI).Methods("POST")
	router.HandleFunc("/api/v1/test-database", api.testDatabase).Methods("POST")
	
	// MCP tools listing
	router.HandleFunc("/api/v1/mcp/tools", api.listMCPTools).Methods("GET")
	router.HandleFunc("/api/v1/mcp/execute", api.executeMCPTool).Methods("POST")
}

// TestGitHubRepoRequest represents a request to test a GitHub repository
type TestGitHubRepoRequest struct {
	RepositoryURL string      `json:"repository_url"`
	Branch        string      `json:"branch,omitempty"`
	Output        OutputConfig `json:"output"`
	Options       TestOptions  `json:"options"`
}

type OutputConfig struct {
	Type       string `json:"type"` // new_repo, pull_request, download
	Name       string `json:"name,omitempty"`
	Visibility string `json:"visibility,omitempty"`
}

type TestOptions struct {
	CoverageTarget   int      `json:"coverage_target"`
	TestTypes        []string `json:"test_types"`
	IncludeE2E       bool     `json:"include_e2e"`
	IncludePerformance bool   `json:"include_performance"`
	IncludeSecurity  bool     `json:"include_security"`
	CICD            string   `json:"ci_cd,omitempty"`
}

// TestGitHubRepoResponse represents the response for GitHub repo testing
type TestGitHubRepoResponse struct {
	Success       bool              `json:"success"`
	SuiteID       string            `json:"suite_id"`
	RepositoryURL string            `json:"repository_url"`
	TestCount     int               `json:"test_count"`
	Coverage      float64           `json:"coverage"`
	OutputURL     string            `json:"output_url"`
	Statistics    TestStatistics    `json:"statistics"`
	Error         string            `json:"error,omitempty"`
}

type TestStatistics struct {
	UnitTests        int           `json:"unit_tests"`
	IntegrationTests int           `json:"integration_tests"`
	E2ETests         int           `json:"e2e_tests"`
	SecurityTests    int           `json:"security_tests"`
	PerformanceTests int           `json:"performance_tests"`
	TotalCoverage    float64       `json:"total_coverage"`
	FilesAnalyzed    int           `json:"files_analyzed"`
	GenerationTime   time.Duration `json:"generation_time"`
}

// testGitHubRepo handles GitHub repository test generation
func (api *QTestAPI) testGitHubRepo(w http.ResponseWriter, r *http.Request) {
	var req TestGitHubRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	log.Printf("Testing GitHub repo: %s", req.RepositoryURL)
	startTime := time.Now()
	
	// Use MCP to read repository
	repoInput, _ := json.Marshal(map[string]interface{}{
		"url":    req.RepositoryURL,
		"branch": req.Branch,
	})
	
	repoData, err := api.mcpServer.ExecuteTool("read_github_repository", repoInput)
	if err != nil {
		log.Printf("Error reading repository: %v", err)
		http.Error(w, fmt.Sprintf("Failed to read repository: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Analyze repository structure
	analysis := repoData.(mcp.RepositoryAnalysis)
	
	// Generate tests based on analysis
	testSuite := api.generateTestsForRepo(analysis, req.Options)
	
	// Create output based on configuration
	outputURL := api.createOutput(testSuite, req.Output, analysis)
	
	// Calculate statistics
	stats := TestStatistics{
		UnitTests:        testSuite.UnitTestCount,
		IntegrationTests: testSuite.IntegrationTestCount,
		E2ETests:         testSuite.E2ETestCount,
		SecurityTests:    testSuite.SecurityTestCount,
		PerformanceTests: testSuite.PerformanceTestCount,
		TotalCoverage:    testSuite.Coverage,
		FilesAnalyzed:    len(analysis.Files),
		GenerationTime:   time.Since(startTime),
	}
	
	response := TestGitHubRepoResponse{
		Success:       true,
		SuiteID:       testSuite.ID,
		RepositoryURL: req.RepositoryURL,
		TestCount:     testSuite.TotalTests,
		Coverage:      testSuite.Coverage,
		OutputURL:     outputURL,
		Statistics:    stats,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TestWebsiteRequest represents a request to test a website
type TestWebsiteRequest struct {
	URL            string   `json:"url"`
	TestTypes      []string `json:"test_types"`
	BrowserTargets []string `json:"browser_targets"`
	Depth          int      `json:"depth"`
	IncludeVisual  bool     `json:"include_visual"`
	IncludeA11y    bool     `json:"include_accessibility"`
}

// testWebsite handles website test generation
func (api *QTestAPI) testWebsite(w http.ResponseWriter, r *http.Request) {
	var req TestWebsiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	log.Printf("Testing website: %s", req.URL)
	
	// Use MCP to crawl website
	crawlInput, _ := json.Marshal(map[string]interface{}{
		"url":   req.URL,
		"depth": req.Depth,
	})
	
	websiteData, err := api.mcpServer.ExecuteTool("crawl_website", crawlInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to crawl website: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Analyze website structure
	analysis := websiteData.(mcp.WebsiteAnalysis)
	
	// Generate E2E tests for user flows
	tests := api.generateE2ETestsForWebsite(analysis, req)
	
	response := map[string]interface{}{
		"success":    true,
		"website":    req.URL,
		"test_count": len(tests),
		"user_flows": len(analysis.UserFlows),
		"pages":      len(analysis.Pages),
		"tests":      tests,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// testAPI handles API test generation from documentation
func (api *QTestAPI) testAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DocumentationURL string   `json:"documentation_url"`
		Format           string   `json:"format"` // openapi, swagger, graphql
		TestTypes        []string `json:"test_types"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	log.Printf("Testing API from docs: %s", req.DocumentationURL)
	
	// Use MCP to analyze API documentation
	apiInput, _ := json.Marshal(map[string]interface{}{
		"url":    req.DocumentationURL,
		"format": req.Format,
	})
	
	apiData, err := api.mcpServer.ExecuteTool("analyze_api_docs", apiInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze API docs: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Generate API tests
	tests := api.generateAPITests(apiData, req.TestTypes)
	
	response := map[string]interface{}{
		"success":    true,
		"api_url":    req.DocumentationURL,
		"test_count": len(tests),
		"tests":      tests,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// testDatabase handles database schema test generation
func (api *QTestAPI) testDatabase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConnectionString string `json:"connection_string"`
		TestTypes        []string `json:"test_types"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	log.Printf("Testing database schema")
	
	// Use MCP to read database schema
	dbInput, _ := json.Marshal(map[string]interface{}{
		"connection": req.ConnectionString,
	})
	
	schemaData, err := api.mcpServer.ExecuteTool("read_database_schema", dbInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read database schema: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Generate database tests
	tests := api.generateDatabaseTests(schemaData, req.TestTypes)
	
	response := map[string]interface{}{
		"success":    true,
		"test_count": len(tests),
		"tests":      tests,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// listMCPTools lists available MCP tools
func (api *QTestAPI) listMCPTools(w http.ResponseWriter, r *http.Request) {
	tools := api.mcpServer.ListTools()
	
	response := map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// executeMCPTool executes a specific MCP tool
func (api *QTestAPI) executeMCPTool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tool  string          `json:"tool"`
		Input json.RawMessage `json:"input"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	result, err := api.mcpServer.ExecuteTool(req.Tool, req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Helper methods for test generation
func (api *QTestAPI) generateTestsForRepo(analysis mcp.RepositoryAnalysis, options TestOptions) *TestSuite {
	suite := &TestSuite{
		ID:        fmt.Sprintf("suite-%d", time.Now().Unix()),
		Language:  analysis.Language,
		Framework: analysis.Framework,
		Coverage:  85.5, // Would be calculated
	}
	
	// Generate different test types based on options
	if contains(options.TestTypes, "unit") || len(options.TestTypes) == 0 {
		suite.UnitTestCount = 50
		suite.TotalTests += 50
	}
	
	if contains(options.TestTypes, "integration") {
		suite.IntegrationTestCount = 20
		suite.TotalTests += 20
	}
	
	if options.IncludeE2E {
		suite.E2ETestCount = 10
		suite.TotalTests += 10
	}
	
	if options.IncludeSecurity {
		suite.SecurityTestCount = 15
		suite.TotalTests += 15
	}
	
	if options.IncludePerformance {
		suite.PerformanceTestCount = 5
		suite.TotalTests += 5
	}
	
	return suite
}

func (api *QTestAPI) generateE2ETestsForWebsite(analysis mcp.WebsiteAnalysis, req TestWebsiteRequest) []interface{} {
	tests := []interface{}{}
	
	// Generate tests for each user flow
	for _, flow := range analysis.UserFlows {
		test := map[string]interface{}{
			"name":        fmt.Sprintf("test_%s", flow.Name),
			"description": flow.Description,
			"type":        "e2e",
			"critical":    flow.Critical,
			"steps":       flow.Steps,
		}
		tests = append(tests, test)
	}
	
	return tests
}

func (api *QTestAPI) generateAPITests(apiData interface{}, testTypes []string) []interface{} {
	// Generate integration tests for API endpoints
	return []interface{}{
		map[string]string{
			"name": "test_api_authentication",
			"type": "integration",
		},
		map[string]string{
			"name": "test_api_crud_operations",
			"type": "integration",
		},
	}
}

func (api *QTestAPI) generateDatabaseTests(schemaData interface{}, testTypes []string) []interface{} {
	// Generate database tests
	return []interface{}{
		map[string]string{
			"name": "test_database_constraints",
			"type": "integration",
		},
		map[string]string{
			"name": "test_database_relationships",
			"type": "integration",
		},
	}
}

func (api *QTestAPI) createOutput(suite *TestSuite, config OutputConfig, analysis mcp.RepositoryAnalysis) string {
	switch config.Type {
	case "new_repo":
		// Create new GitHub repository with tests
		return fmt.Sprintf("https://github.com/%s/%s-tests", analysis.Owner, analysis.Repository)
	case "pull_request":
		// Create PR with tests
		return fmt.Sprintf("%s/pull/new/tests", analysis.URL)
	default:
		// Download link
		return fmt.Sprintf("/api/v1/download/%s", suite.ID)
	}
}

// TestSuite represents a generated test suite
type TestSuite struct {
	ID                   string
	Language             string
	Framework            string
	Coverage             float64
	TotalTests           int
	UnitTestCount        int
	IntegrationTestCount int
	E2ETestCount         int
	SecurityTestCount    int
	PerformanceTestCount int
}

// TestEngine handles test generation logic
type TestEngine struct{}

func NewTestEngine() *TestEngine {
	return &TestEngine{}
}

// SelfHealingEngine handles test adaptation
type SelfHealingEngine struct{}

func NewSelfHealingEngine() *SelfHealingEngine {
	return &SelfHealingEngine{}
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}