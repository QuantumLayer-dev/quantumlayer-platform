package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

type CodeGenerationRequest struct {
	ID           string                 `json:"id,omitempty"`
	Prompt       string                 `json:"prompt" binding:"required"`
	Language     string                 `json:"language" binding:"required"`
	Framework    string                 `json:"framework,omitempty"`
	Type         string                 `json:"type" binding:"required"`
	GenerateTests bool                  `json:"generate_tests,omitempty"`
	GenerateDocs  bool                  `json:"generate_docs,omitempty"`
	Requirements map[string]interface{} `json:"requirements,omitempty"`
}

type WorkflowResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

var temporalClient client.Client

func main() {
	// Initialize Temporal client
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "temporal-frontend.temporal.svc.cluster.local:7233"
	}

	c, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: "quantumlayer",
	})
	if err != nil {
		log.Fatal("Unable to create Temporal client", err)
	}
	defer c.Close()
	temporalClient = c

	// Setup Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Ready check
	r.GET("/ready", func(c *gin.Context) {
		// Test Temporal connection
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
		_, err := temporalClient.CheckHealth(ctx, &client.CheckHealthRequest{})
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Trigger code generation workflow
	r.POST("/api/v1/workflows/generate", handleGenerateCode)
	
	// Trigger extended code generation workflow
	r.POST("/api/v1/workflows/generate-extended", handleGenerateExtendedCode)
	
	// Trigger intelligent code generation workflow (v2)
	r.POST("/api/v1/workflows/generate-intelligent", handleGenerateIntelligentCode)

	// Get workflow status
	r.GET("/api/v1/workflows/:id", handleGetWorkflow)

	// Get workflow result
	r.GET("/api/v1/workflows/:id/result", handleGetWorkflowResult)
	
	// Infrastructure generation endpoints
	r.POST("/api/v1/workflows/generate-infrastructure", handleGenerateInfrastructure)
	r.GET("/api/v1/workflows/infrastructure/:id", handleGetInfrastructureStatus)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Workflow API on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func handleGenerateCode(c *gin.Context) {
	var req CodeGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate request ID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	// Create workflow ID
	workflowID := fmt.Sprintf("code-gen-%s", req.ID)

	// Workflow options
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "code-generation",
		WorkflowExecutionTimeout: 5 * time.Minute,
	}

	// Start workflow
	we, err := temporalClient.ExecuteWorkflow(
		context.Background(),
		options,
		"CodeGenerationWorkflow",
		req,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start workflow",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, WorkflowResponse{
		WorkflowID: we.GetID(),
		RunID:      we.GetRunID(),
		Status:     "started",
		Message:    "Workflow started successfully",
	})
}

func handleGetWorkflow(c *gin.Context) {
	workflowID := c.Param("id")

	// Get workflow execution
	ctx := context.Background()
	resp, err := temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	status := "unknown"
	if resp.WorkflowExecutionInfo.Status != 0 {
		status = resp.WorkflowExecutionInfo.Status.String()
	}

	c.JSON(http.StatusOK, gin.H{
		"workflow_id": workflowID,
		"status":      status,
		"start_time":  resp.WorkflowExecutionInfo.StartTime,
		"close_time":  resp.WorkflowExecutionInfo.CloseTime,
	})
}

func handleGenerateExtendedCode(c *gin.Context) {
	var req CodeGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate request ID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	// Create workflow ID
	workflowID := fmt.Sprintf("extended-code-gen-%s", req.ID)

	// Workflow options
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "code-generation",
		WorkflowExecutionTimeout: 10 * time.Minute, // Extended timeout
	}

	// Start extended workflow
	we, err := temporalClient.ExecuteWorkflow(
		context.Background(),
		options,
		"ExtendedCodeGenerationWorkflow", // Use extended workflow
		req,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start extended workflow",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, WorkflowResponse{
		WorkflowID: we.GetID(),
		RunID:      we.GetRunID(),
		Status:     "started",
		Message:    "Extended workflow started successfully (12 stages)",
	})
}

func handleGenerateIntelligentCode(c *gin.Context) {
	var req CodeGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate request ID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	// Create workflow ID
	workflowID := fmt.Sprintf("intelligent-code-gen-%s", req.ID)

	// Workflow options
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "code-generation",
		WorkflowExecutionTimeout: 10 * time.Minute, // Extended timeout
	}

	// Start intelligent workflow
	we, err := temporalClient.ExecuteWorkflow(
		context.Background(),
		options,
		"IntelligentCodeGenerationWorkflow", // Use intelligent workflow
		req,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start intelligent workflow",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, WorkflowResponse{
		WorkflowID: we.GetID(),
		RunID:      we.GetRunID(),
		Status:     "started",
		Message:    "Intelligent workflow started successfully (3 stages + multi-file generation)",
	})
}

func handleGetWorkflowResult(c *gin.Context) {
	workflowID := c.Param("id")

	// Get workflow handle
	we := temporalClient.GetWorkflow(context.Background(), workflowID, "")

	// Get result with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result interface{}
	err := we.GetWithOptions(ctx, &result, client.WorkflowRunGetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get workflow result",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Infrastructure generation request
type InfrastructureRequest struct {
	WorkflowID         string   `json:"workflow_id"`          // Reference to code generation workflow
	Provider           string   `json:"provider" binding:"required"` // aws, gcp, azure, kubernetes
	Environment        string   `json:"environment"`           // dev, staging, production
	Compliance         []string `json:"compliance"`            // SOC2, HIPAA, PCI-DSS, GDPR
	EnableGoldenImages bool     `json:"enable_golden_images"`
	EnableSOP          bool     `json:"enable_sop"`
	AutoDeploy         bool     `json:"auto_deploy"`
	DryRun             bool     `json:"dry_run"`
}

func handleGenerateInfrastructure(c *gin.Context) {
	var req InfrastructureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate workflow ID
	workflowID := fmt.Sprintf("infra-gen-%s", uuid.New().String())
	if req.WorkflowID != "" {
		workflowID = fmt.Sprintf("infra-for-%s", req.WorkflowID)
	}

	// Workflow options
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "infrastructure-generation",
		WorkflowExecutionTimeout: 15 * time.Minute,
	}

	// Start infrastructure workflow
	we, err := temporalClient.ExecuteWorkflow(
		context.Background(),
		options,
		"InfrastructureGenerationWorkflow",
		req,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start infrastructure workflow",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, WorkflowResponse{
		WorkflowID: we.GetID(),
		RunID:      we.GetRunID(),
		Status:     "started",
		Message:    fmt.Sprintf("Infrastructure generation started for %s", req.Provider),
	})
}

func handleGetInfrastructureStatus(c *gin.Context) {
	workflowID := c.Param("id")

	// Get workflow execution
	ctx := context.Background()
	resp, err := temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Infrastructure workflow not found"})
		return
	}

	status := "unknown"
	if resp.WorkflowExecutionInfo != nil {
		if resp.WorkflowExecutionInfo.Status == 1 { // Running
			status = "running"
		} else if resp.WorkflowExecutionInfo.Status == 2 { // Completed
			status = "completed"
		} else if resp.WorkflowExecutionInfo.Status == 3 { // Failed
			status = "failed"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"workflow_id": workflowID,
		"status":      status,
		"start_time":  resp.WorkflowExecutionInfo.StartTime,
		"close_time":  resp.WorkflowExecutionInfo.CloseTime,
	})
}