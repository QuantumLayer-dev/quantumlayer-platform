package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/orchestrator"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/types"
)

// Request structures
type AgentRequest struct {
	Requirements string                 `json:"requirements" binding:"required"`
	ProjectID    string                 `json:"project_id,omitempty"`
	ProjectType  string                 `json:"project_type,omitempty"`
	Constraints  map[string]interface{} `json:"constraints,omitempty"`
}

type TaskRequest struct {
	Type         string                 `json:"type" binding:"required"`
	Description  string                 `json:"description" binding:"required"`
	Priority     int                    `json:"priority,omitempty"`
	Requirements map[string]interface{} `json:"requirements,omitempty"`
}

type ConsensusRequest struct {
	Topic    string      `json:"topic" binding:"required"`
	Proposal interface{} `json:"proposal" binding:"required"`
}

// Response structures
type AgentResponse struct {
	Success       bool                   `json:"success"`
	SessionID     string                 `json:"session_id"`
	ProjectID     string                 `json:"project_id"`
	GeneratedCode map[string]string      `json:"generated_code,omitempty"`
	Architecture  map[string]interface{} `json:"architecture,omitempty"`
	Tests         []string               `json:"tests,omitempty"`
	Documentation string                 `json:"documentation,omitempty"`
	Metrics       map[string]interface{} `json:"metrics,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

type AgentMetricsResponse struct {
	Agents  map[string]types.AgentMetrics `json:"agents"`
	Summary map[string]interface{}        `json:"summary"`
}

// Simple in-memory message bus implementation
type InMemoryMessageBus struct {
	subscribers map[string][]func(*types.Message)
}

func NewInMemoryMessageBus() *InMemoryMessageBus {
	return &InMemoryMessageBus{
		subscribers: make(map[string][]func(*types.Message)),
	}
}

func (b *InMemoryMessageBus) Publish(ctx context.Context, topic string, msg *types.Message) error {
	if handlers, ok := b.subscribers[topic]; ok {
		for _, handler := range handlers {
			go handler(msg)
		}
	}
	return nil
}

func (b *InMemoryMessageBus) Subscribe(ctx context.Context, topic string, handler func(*types.Message)) error {
	b.subscribers[topic] = append(b.subscribers[topic], handler)
	return nil
}

func (b *InMemoryMessageBus) Unsubscribe(ctx context.Context, topic string) error {
	delete(b.subscribers, topic)
	return nil
}

var (
	agentOrchestrator *orchestrator.AgentOrchestrator
	llmEndpoint       string
)

func main() {
	// Configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	llmEndpoint = os.Getenv("LLM_ENDPOINT")
	if llmEndpoint == "" {
		llmEndpoint = "http://llm-router.quantumlayer.svc.cluster.local:8080"
	}

	// Create message bus
	messageBus := NewInMemoryMessageBus()

	// Initialize orchestrator
	agentOrchestrator = orchestrator.NewAgentOrchestrator(llmEndpoint, messageBus)

	// Setup Gin router
	r := gin.Default()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// Health endpoints
	r.GET("/health", healthCheck)
	r.GET("/ready", readyCheck)

	// Agent management endpoints
	api := r.Group("/api/v1")
	{
		// Main processing endpoint
		api.POST("/process", handleProcess)

		// Task management
		api.POST("/tasks", handleCreateTask)
		api.GET("/tasks/:id", handleGetTask)

		// Agent management
		api.POST("/agents/spawn", handleSpawnAgent)
		api.GET("/agents", handleListAgents)
		api.GET("/agents/metrics", handleGetMetrics)
		api.DELETE("/agents/:id", handleStopAgent)

		// Consensus
		api.POST("/consensus", handleConsensus)
	}

	// Start server
	log.Printf("Agent Orchestrator starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "agent-orchestrator",
		"version": "1.0.0",
	})
}

func readyCheck(c *gin.Context) {
	// Check if orchestrator is ready
	if agentOrchestrator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "orchestrator not initialized",
		})
		return
	}

	// Check LLM router connectivity
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(llmEndpoint + "/health")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "LLM router not reachable",
		})
		return
	}
	resp.Body.Close()

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"agents": len(agentOrchestrator.MonitorAgents()),
	})
}

func handleProcess(c *gin.Context) {
	var req AgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate project ID if not provided
	if req.ProjectID == "" {
		req.ProjectID = uuid.New().String()
	}

	// Process request with agents
	ctx := context.Background()
	result, err := agentOrchestrator.ProcessRequest(ctx, req.Requirements, req.ProjectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AgentResponse{
			Success:   false,
			ProjectID: req.ProjectID,
			Error:     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AgentResponse{
		Success:       result.Success,
		SessionID:     uuid.New().String(),
		ProjectID:     req.ProjectID,
		GeneratedCode: result.GeneratedCode,
		Architecture:  result.Architecture,
		Tests:         result.Tests,
		Documentation: result.Documentation,
		Metrics:       result.Metrics,
	})
}

func handleCreateTask(c *gin.Context) {
	var req TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := &types.Task{
		ID:           uuid.New().String(),
		Type:         req.Type,
		Description:  req.Description,
		Priority:     req.Priority,
		Requirements: req.Requirements,
		Status:       types.TaskPending,
		CreatedAt:    time.Now(),
	}

	ctx := context.Background()
	if err := agentOrchestrator.AssignTask(ctx, task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func handleGetTask(c *gin.Context) {
	taskID := c.Param("id")
	
	// This would need to be implemented in the orchestrator
	c.JSON(http.StatusOK, gin.H{
		"id":     taskID,
		"status": "pending",
	})
}

func handleSpawnAgent(c *gin.Context) {
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert string to AgentRole
	var role types.AgentRole
	switch req.Role {
	case "project-manager":
		role = types.RoleProjectManager
	case "architect":
		role = types.RoleArchitect
	case "backend-developer":
		role = types.RoleBackendDev
	case "frontend-developer":
		role = types.RoleFrontendDev
	case "database-admin":
		role = types.RoleDatabaseAdmin
	case "devops":
		role = types.RoleDevOps
	case "qa-engineer":
		role = types.RoleQA
	case "security":
		role = types.RoleSecurity
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}

	ctx := context.Background()
	agentCtx := &types.AgentContext{
		ProjectID: uuid.New().String(),
		SessionID: uuid.New().String(),
	}

	agent, err := agentOrchestrator.SpawnAgent(ctx, role, agentCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           agent.ID(),
		"role":         agent.Role(),
		"status":       agent.Status(),
		"capabilities": agent.Capabilities(),
	})
}

func handleListAgents(c *gin.Context) {
	metrics := agentOrchestrator.MonitorAgents()
	
	agents := []gin.H{}
	for id, metric := range metrics {
		agents = append(agents, gin.H{
			"id":         id,
			"metrics":    metric,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"total":  len(agents),
	})
}

func handleGetMetrics(c *gin.Context) {
	metrics := agentOrchestrator.MonitorAgents()
	
	// Calculate summary metrics
	totalTasks := 0
	totalFailures := 0
	for _, m := range metrics {
		totalTasks += m.TasksCompleted + m.TasksFailed
		totalFailures += m.TasksFailed
	}

	successRate := 0.0
	if totalTasks > 0 {
		successRate = float64(totalTasks-totalFailures) / float64(totalTasks)
	}

	c.JSON(http.StatusOK, AgentMetricsResponse{
		Agents: metrics,
		Summary: map[string]interface{}{
			"total_agents":  len(metrics),
			"total_tasks":   totalTasks,
			"success_rate":  successRate,
			"total_failures": totalFailures,
		},
	})
}

func handleStopAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	// This would need to be implemented in the orchestrator
	c.JSON(http.StatusOK, gin.H{
		"id":     agentID,
		"status": "stopped",
	})
}

func handleConsensus(c *gin.Context) {
	var req ConsensusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	consensus, err := agentOrchestrator.RequestConsensus(ctx, req.Topic, req.Proposal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, consensus)
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