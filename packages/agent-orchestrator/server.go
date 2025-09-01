package orchestrator

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Server represents the Agent Orchestrator HTTP server
type Server struct {
	orchestrator *Orchestrator
	engine       *gin.Engine
	logger       *zap.Logger
	port         string
}

// NewServer creates a new Agent Orchestrator server
func NewServer(port string, config *OrchestratorConfig, logger *zap.Logger) (*Server, error) {
	// Create orchestrator
	orchestrator, err := NewOrchestrator(config, logger)
	if err != nil {
		return nil, err
	}
	
	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(LoggerMiddleware(logger))
	engine.Use(CORSMiddleware())
	
	s := &Server{
		orchestrator: orchestrator,
		engine:       engine,
		logger:       logger,
		port:         port,
	}
	
	s.setupRoutes()
	
	return s, nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health and metrics
	s.engine.GET("/health", s.handleHealth)
	s.engine.GET("/ready", s.handleReadiness)
	s.engine.GET("/metrics", s.handleMetrics)
	
	// API v1 routes
	v1 := s.engine.Group("/api/v1")
	{
		// Generation endpoints
		v1.POST("/generate", s.handleGenerate)
		v1.GET("/generation/:id", s.handleGetGeneration)
		
		// Task management
		v1.POST("/tasks", s.handleSubmitTask)
		v1.GET("/tasks/:id", s.handleGetTask)
		v1.GET("/tasks", s.handleListTasks)
		
		// Agent management
		v1.GET("/agents", s.handleListAgents)
		v1.GET("/agents/:id", s.handleGetAgent)
		
		// Workflow endpoints
		v1.POST("/workflows", s.handleCreateWorkflow)
		v1.GET("/workflows/:id", s.handleGetWorkflow)
	}
}

// handleGenerate handles code generation requests
func (s *Server) handleGenerate(c *gin.Context) {
	var req GenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Create request ID
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	
	// Create generation workflow
	workflow := s.createGenerationWorkflow(&req)
	
	// Submit tasks to orchestrator
	for _, task := range workflow.Tasks {
		if err := s.orchestrator.SubmitTask(task); err != nil {
			s.logger.Error("Failed to submit task",
				zap.String("task_id", task.ID),
				zap.Error(err),
			)
		}
	}
	
	// Return immediate response
	response := GenerationResponse{
		ID:        req.ID,
		RequestID: req.ID,
		Status:    "processing",
		CreatedAt: time.Now(),
		Metadata: map[string]string{
			"workflow_id": workflow.ID,
		},
	}
	
	c.JSON(http.StatusAccepted, response)
}

// createGenerationWorkflow creates a workflow for code generation
func (s *Server) createGenerationWorkflow(req *GenerationRequest) *WorkflowState {
	workflow := &WorkflowState{
		ID:        uuid.New().String(),
		RequestID: req.ID,
		Status:    "created",
		Phase:     "initialization",
		Tasks:     make([]*Task, 0),
		Agents:    make([]*Agent, 0),
		Results:   make(map[string]interface{}),
		StartedAt: time.Now(),
	}
	
	// Create generation task
	generateTask := &Task{
		ID:       uuid.New().String(),
		Type:     "generate",
		Priority: TaskPriorityHigh,
		Status:   TaskStatusPending,
		Input: map[string]interface{}{
			"prompt":    req.Prompt,
			"language":  req.Language,
			"framework": req.Framework,
			"metadata":  req.Metadata,
		},
		CreatedAt: time.Now(),
	}
	workflow.Tasks = append(workflow.Tasks, generateTask)
	
	// Create validation task
	validateTask := &Task{
		ID:       uuid.New().String(),
		Type:     "validate",
		Priority: TaskPriorityMedium,
		Status:   TaskStatusPending,
		Input: map[string]interface{}{
			"depends_on": generateTask.ID,
		},
		CreatedAt: time.Now(),
	}
	workflow.Tasks = append(workflow.Tasks, validateTask)
	
	// Create test generation task
	testTask := &Task{
		ID:       uuid.New().String(),
		Type:     "test",
		Priority: TaskPriorityMedium,
		Status:   TaskStatusPending,
		Input: map[string]interface{}{
			"depends_on": generateTask.ID,
		},
		CreatedAt: time.Now(),
	}
	workflow.Tasks = append(workflow.Tasks, testTask)
	
	return workflow
}

// handleGetGeneration retrieves generation status
func (s *Server) handleGetGeneration(c *gin.Context) {
	generationID := c.Param("id")
	
	// For MVP, return a simple status
	// In production, this would query the actual workflow state
	response := GenerationResponse{
		ID:        generationID,
		RequestID: generationID,
		Status:    "completed",
		Code:      "// Generated code will appear here",
		Tests:     "// Generated tests will appear here",
		Docs:      "// Generated documentation will appear here",
		CreatedAt: time.Now(),
	}
	
	c.JSON(http.StatusOK, response)
}

// handleSubmitTask submits a new task
func (s *Server) handleSubmitTask(c *gin.Context) {
	var task Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := s.orchestrator.SubmitTask(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, task)
}

// handleGetTask retrieves a task by ID
func (s *Server) handleGetTask(c *gin.Context) {
	taskID := c.Param("id")
	
	task, err := s.orchestrator.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, task)
}

// handleListTasks lists all tasks
func (s *Server) handleListTasks(c *gin.Context) {
	// For MVP, return empty list
	// In production, implement proper task listing
	tasks := []Task{}
	
	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"total": len(tasks),
	})
}

// handleListAgents lists all active agents
func (s *Server) handleListAgents(c *gin.Context) {
	agents := s.orchestrator.GetAgents()
	
	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"total":  len(agents),
	})
}

// handleGetAgent retrieves an agent by ID
func (s *Server) handleGetAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	agents := s.orchestrator.GetAgents()
	for _, agent := range agents {
		if agent.ID == agentID {
			c.JSON(http.StatusOK, agent)
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
}

// handleCreateWorkflow creates a new workflow
func (s *Server) handleCreateWorkflow(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	workflow := &WorkflowState{
		ID:        uuid.New().String(),
		Status:    "created",
		Phase:     "initialization",
		Tasks:     make([]*Task, 0),
		Agents:    make([]*Agent, 0),
		Results:   make(map[string]interface{}),
		StartedAt: time.Now(),
	}
	
	c.JSON(http.StatusCreated, workflow)
}

// handleGetWorkflow retrieves a workflow by ID
func (s *Server) handleGetWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
	
	// For MVP, return mock workflow
	workflow := &WorkflowState{
		ID:        workflowID,
		Status:    "in_progress",
		Phase:     "generation",
		Tasks:     make([]*Task, 0),
		Agents:    make([]*Agent, 0),
		Results:   make(map[string]interface{}),
		StartedAt: time.Now(),
	}
	
	c.JSON(http.StatusOK, workflow)
}

// handleHealth returns service health status
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "agent-orchestrator",
		"timestamp": time.Now().Unix(),
	})
}

// handleReadiness checks if service is ready
func (s *Server) handleReadiness(c *gin.Context) {
	agents := s.orchestrator.GetAgents()
	
	c.JSON(http.StatusOK, gin.H{
		"status":       "ready",
		"agent_count":  len(agents),
		"max_agents":   s.orchestrator.config.MaxAgents,
	})
}

// handleMetrics returns service metrics
func (s *Server) handleMetrics(c *gin.Context) {
	metrics := s.orchestrator.metrics.GetMetrics()
	agents := s.orchestrator.GetAgents()
	
	metrics["active_agents"] = len(agents)
	metrics["max_agents"] = s.orchestrator.config.MaxAgents
	
	c.JSON(http.StatusOK, metrics)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting Agent Orchestrator server", zap.String("port", s.port))
	return s.engine.Run(":" + s.port)
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	return s.orchestrator.Stop()
}

// Middleware functions

// LoggerMiddleware creates a Gin middleware for logging
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		
		c.Next()
		
		latency := time.Since(start)
		logger.Info("Request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
		)
	}
}

// CORSMiddleware creates a Gin middleware for CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}