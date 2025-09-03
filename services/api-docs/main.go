package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/quantumlayer-dev/quantumlayer-platform/services/api-docs/docs"
)

// @title QuantumLayer Platform API
// @version 2.0.0
// @description Enterprise AI Software Factory - Universal Platform for Code Generation, Testing, Infrastructure, SRE, and Security
// @termsOfService https://quantumlayer.dev/terms/

// @contact.name API Support
// @contact.url https://quantumlayer.dev/support
// @contact.email support@quantumlayer.dev

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8090
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

// @tag.name Workflows
// @tag.description Code generation workflow management

// @tag.name Agents
// @tag.description AI Agent orchestration and management

// @tag.name Security
// @tag.description QSecure - Security analysis and compliance

// @tag.name LLM
// @tag.description Multi-LLM routing and management

// @tag.name MetaPrompt
// @tag.description Meta-prompt engineering and optimization

func main() {
	r := gin.Default()

	// CORS middleware
	r.Use(corsMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "api-docs"})
	})

	// Swagger endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Redirect root to Swagger UI
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	// API endpoints documentation (these are documentation only - actual endpoints are in other services)
	v1 := r.Group("/api/v1")
	{
		// Workflow endpoints
		workflows := v1.Group("/workflows")
		{
			workflows.POST("/generate", generateCode)
			workflows.GET("/status/:id", getWorkflowStatus)
			workflows.GET("/list", listWorkflows)
		}

		// Agent endpoints
		agents := v1.Group("/agents")
		{
			agents.POST("/spawn", spawnAgent)
			agents.POST("/task", createTask)
			agents.GET("/list", listAgents)
			agents.GET("/metrics", getAgentMetrics)
			agents.POST("/consensus", requestConsensus)
		}

		// Security endpoints
		security := v1.Group("/security")
		{
			security.POST("/analyze", analyzeSecurityHandler)
			security.POST("/threat-model", generateThreatModel)
			security.POST("/compliance", validateCompliance)
			security.POST("/remediate", suggestRemediations)
			security.GET("/audit-log", getAuditLog)
		}

		// LLM endpoints
		llm := v1.Group("/llm")
		{
			llm.POST("/generate", generateLLM)
			llm.GET("/providers", listProviders)
			llm.POST("/route", routeRequest)
		}

		// Meta-prompt endpoints
		metaprompt := v1.Group("/meta-prompt")
		{
			metaprompt.POST("/enhance", enhancePrompt)
			metaprompt.POST("/optimize", optimizePrompt)
			metaprompt.GET("/templates", listTemplates)
		}

		// AI Decision Engine endpoints
		decisions := v1.Group("/decisions")
		{
			decisions.POST("/decide", makeDecision)
			decisions.POST("/language", selectLanguage)
			decisions.POST("/framework", selectFramework)
			decisions.POST("/agent", selectAgent)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	log.Printf("API Documentation server starting on port %s", port)
	log.Printf("Swagger UI available at: http://localhost:%s/swagger/index.html", port)
	
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Handler functions (these return documentation examples)

// generateCode godoc
// @Summary Generate code using AI
// @Description Generate production-ready code from natural language requirements
// @Tags Workflows
// @Accept json
// @Produce json
// @Param request body CodeGenerationRequest true "Code generation request"
// @Success 200 {object} WorkflowResponse
// @Router /workflows/generate [post]
func generateCode(c *gin.Context) {
	c.JSON(200, WorkflowResponse{
		WorkflowID: "code-gen-123",
		Status:     "started",
		Message:    "Code generation workflow started",
	})
}

// spawnAgent godoc
// @Summary Spawn a new AI agent
// @Description Create a specialized agent for a specific role using AI decision making
// @Tags Agents
// @Accept json
// @Produce json
// @Param request body SpawnAgentRequest true "Agent spawn request"
// @Success 200 {object} AgentResponse
// @Router /agents/spawn [post]
func spawnAgent(c *gin.Context) {
	c.JSON(200, AgentResponse{
		AgentID: "agent-backend-dev-123",
		Role:    "backend-developer",
		Status:  "active",
	})
}

// analyzeSecurityHandler godoc
// @Summary Analyze code security
// @Description Perform comprehensive security analysis using QSecure engine
// @Tags Security
// @Accept json
// @Produce json
// @Param request body SecurityAnalysisRequest true "Security analysis request"
// @Success 200 {object} SecurityAnalysisResponse
// @Router /security/analyze [post]
func analyzeSecurityHandler(c *gin.Context) {
	c.JSON(200, SecurityAnalysisResponse{
		OverallRisk: "medium",
		Score:       75.5,
		Vulnerabilities: []Vulnerability{
			{
				Type:     "SQL Injection",
				Severity: "high",
				CWE:      "CWE-89",
			},
		},
	})
}

// Request/Response types for documentation

type CodeGenerationRequest struct {
	Prompt      string            `json:"prompt" example:"Create a REST API with user authentication"`
	Language    string            `json:"language" example:"python"`
	Type        string            `json:"type" example:"api"`
	Framework   string            `json:"framework,omitempty" example:"fastapi"`
	Requirements map[string]string `json:"requirements,omitempty"`
}

type WorkflowResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

type SpawnAgentRequest struct {
	Role         string                 `json:"role" example:"security-architect"`
	Requirements string                 `json:"requirements,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

type AgentResponse struct {
	AgentID      string   `json:"agent_id"`
	Role         string   `json:"role"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type SecurityAnalysisRequest struct {
	Code     string   `json:"code"`
	Language string   `json:"language" example:"python"`
	Standards []string `json:"standards,omitempty" example:"OWASP,GDPR"`
}

type SecurityAnalysisResponse struct {
	ID              string          `json:"id,omitempty"`
	OverallRisk     string          `json:"overall_risk"`
	Score           float64         `json:"score"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Compliance      map[string]bool `json:"compliance,omitempty"`
}

type Vulnerability struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	CWE      string `json:"cwe,omitempty"`
	Location string `json:"location,omitempty"`
}

// Stub handlers for other endpoints
func getWorkflowStatus(c *gin.Context)  { c.JSON(200, gin.H{"status": "completed"}) }
func listWorkflows(c *gin.Context)       { c.JSON(200, []WorkflowResponse{}) }
func createTask(c *gin.Context)          { c.JSON(200, gin.H{"task_id": "task-123"}) }
func listAgents(c *gin.Context)          { c.JSON(200, []AgentResponse{}) }
func getAgentMetrics(c *gin.Context)     { c.JSON(200, gin.H{"metrics": map[string]interface{}{}}) }
func requestConsensus(c *gin.Context)    { c.JSON(200, gin.H{"consensus": true}) }
func generateThreatModel(c *gin.Context) { c.JSON(200, gin.H{"model": "threat-model"}) }
func validateCompliance(c *gin.Context)  { c.JSON(200, gin.H{"compliant": true}) }
func suggestRemediations(c *gin.Context) { c.JSON(200, []gin.H{}) }
func getAuditLog(c *gin.Context)         { c.JSON(200, []gin.H{}) }
func generateLLM(c *gin.Context)         { c.JSON(200, gin.H{"content": "generated"}) }
func listProviders(c *gin.Context)       { c.JSON(200, []string{"openai", "anthropic", "bedrock"}) }
func routeRequest(c *gin.Context)        { c.JSON(200, gin.H{"routed_to": "provider"}) }
func enhancePrompt(c *gin.Context)       { c.JSON(200, gin.H{"enhanced": "prompt"}) }
func optimizePrompt(c *gin.Context)      { c.JSON(200, gin.H{"optimized": "prompt"}) }
func listTemplates(c *gin.Context)       { c.JSON(200, []gin.H{}) }
func makeDecision(c *gin.Context)        { c.JSON(200, gin.H{"decision": "result"}) }
func selectLanguage(c *gin.Context)      { c.JSON(200, gin.H{"language": "python"}) }
func selectFramework(c *gin.Context)     { c.JSON(200, gin.H{"framework": "fastapi"}) }
func selectAgent(c *gin.Context)         { c.JSON(200, gin.H{"agent": "backend-developer"}) }