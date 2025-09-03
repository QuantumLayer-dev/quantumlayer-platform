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

	// Get workflow status
	r.GET("/api/v1/workflows/:id", handleGetWorkflow)

	// Get workflow result
	r.GET("/api/v1/workflows/:id/result", handleGetWorkflowResult)

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