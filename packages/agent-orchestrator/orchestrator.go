package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// Orchestrator manages agent coordination and task distribution
type Orchestrator struct {
	config         *OrchestratorConfig
	agents         map[string]AgentInterface
	tasks          map[string]*Task
	taskQueue      chan *Task
	resultQueue    chan *Task
	logger         *zap.Logger
	redisClient    *redis.Client
	temporalClient client.Client
	metrics        *MetricsCollector
	mu             sync.RWMutex
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(config *OrchestratorConfig, logger *zap.Logger) (*Orchestrator, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize Redis client
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	redisClient := redis.NewClient(opt)
	
	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis connection failed", zap.Error(err))
	}
	
	// Initialize Temporal client (optional for MVP)
	var temporalClient client.Client
	if config.TemporalHost != "" {
		temporalClient, err = client.Dial(client.Options{
			HostPort: config.TemporalHost,
		})
		if err != nil {
			logger.Warn("Temporal connection failed", zap.Error(err))
		}
	}
	
	o := &Orchestrator{
		config:         config,
		agents:         make(map[string]AgentInterface),
		tasks:          make(map[string]*Task),
		taskQueue:      make(chan *Task, 1000),
		resultQueue:    make(chan *Task, 1000),
		logger:         logger,
		redisClient:    redisClient,
		temporalClient: temporalClient,
		metrics:        NewMetricsCollector(),
		ctx:            ctx,
		cancel:         cancel,
	}
	
	// Start background workers
	o.startWorkers()
	
	return o, nil
}

// startWorkers starts background worker goroutines
func (o *Orchestrator) startWorkers() {
	// Task dispatcher
	o.wg.Add(1)
	go o.taskDispatcher()
	
	// Result processor
	o.wg.Add(1)
	go o.resultProcessor()
	
	// Health checker
	o.wg.Add(1)
	go o.healthChecker()
	
	// Metrics collector
	if o.config.MetricsEnabled {
		o.wg.Add(1)
		go o.metricsCollector()
	}
}

// taskDispatcher assigns tasks to available agents
func (o *Orchestrator) taskDispatcher() {
	defer o.wg.Done()
	
	for {
		select {
		case <-o.ctx.Done():
			return
		case task := <-o.taskQueue:
			o.assignTask(task)
		}
	}
}

// assignTask finds an available agent and assigns the task
func (o *Orchestrator) assignTask(task *Task) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	// Find suitable agent
	var selectedAgent AgentInterface
	minWorkload := o.config.MaxTasksPerAgent + 1
	
	for _, agent := range o.agents {
		if agent.CanHandle(task) && agent.GetWorkload() < minWorkload {
			selectedAgent = agent
			minWorkload = agent.GetWorkload()
		}
	}
	
	if selectedAgent == nil {
		// No available agent, spawn new one if possible
		if len(o.agents) < o.config.MaxAgents {
			o.spawnAgent(task)
		} else {
			// Re-queue the task
			go func() {
				time.Sleep(1 * time.Second)
				o.taskQueue <- task
			}()
		}
		return
	}
	
	// Assign task to agent
	task.AgentID = selectedAgent.GetID()
	task.Status = TaskStatusAssigned
	now := time.Now()
	task.AssignedAt = &now
	
	// Execute task asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(o.ctx, o.config.TaskTimeout)
		defer cancel()
		
		task.Status = TaskStatusInProgress
		err := selectedAgent.Execute(ctx, task)
		
		if err != nil {
			task.Status = TaskStatusFailed
			task.Error = err.Error()
			o.metrics.IncrementFailedTasks()
		} else {
			task.Status = TaskStatusCompleted
			o.metrics.IncrementCompletedTasks()
		}
		
		completedAt := time.Now()
		task.CompletedAt = &completedAt
		
		// Send to result queue
		o.resultQueue <- task
	}()
}

// spawnAgent creates a new agent based on task requirements
func (o *Orchestrator) spawnAgent(task *Task) {
	o.logger.Info("Spawning new agent for task",
		zap.String("task_id", task.ID),
		zap.String("task_type", task.Type),
	)
	
	// Determine agent type based on task
	agentType := o.determineAgentType(task)
	
	// Create new agent
	agent := o.createAgent(agentType)
	if agent != nil {
		o.mu.Lock()
		o.agents[agent.GetID()] = agent
		o.mu.Unlock()
		
		// Assign task to new agent
		o.assignTask(task)
	}
}

// determineAgentType determines the appropriate agent type for a task
func (o *Orchestrator) determineAgentType(task *Task) AgentType {
	switch task.Type {
	case "parse":
		return AgentTypeParser
	case "generate":
		return AgentTypeGenerator
	case "validate":
		return AgentTypeValidator
	case "test":
		return AgentTypeTester
	case "document":
		return AgentTypeDocumenter
	default:
		return AgentTypeGenerator
	}
}

// createAgent creates a new agent of the specified type
func (o *Orchestrator) createAgent(agentType AgentType) AgentInterface {
	switch agentType {
	case AgentTypeGenerator:
		return NewGeneratorAgent(o.logger)
	case AgentTypeValidator:
		return NewValidatorAgent(o.logger)
	case AgentTypeTester:
		return NewTesterAgent(o.logger)
	default:
		return NewGeneratorAgent(o.logger)
	}
}

// resultProcessor processes completed tasks
func (o *Orchestrator) resultProcessor() {
	defer o.wg.Done()
	
	for {
		select {
		case <-o.ctx.Done():
			return
		case result := <-o.resultQueue:
			o.processResult(result)
		}
	}
}

// processResult handles completed task results
func (o *Orchestrator) processResult(task *Task) {
	o.logger.Info("Processing task result",
		zap.String("task_id", task.ID),
		zap.String("status", string(task.Status)),
	)
	
	// Update task in storage
	o.mu.Lock()
	o.tasks[task.ID] = task
	o.mu.Unlock()
	
	// Store result in Redis
	if o.redisClient != nil {
		ctx, cancel := context.WithTimeout(o.ctx, 5*time.Second)
		defer cancel()
		
		key := fmt.Sprintf("task:result:%s", task.ID)
		o.redisClient.Set(ctx, key, task.Output, 1*time.Hour)
	}
	
	// Update metrics
	if task.AssignedAt != nil && task.CompletedAt != nil {
		duration := task.CompletedAt.Sub(*task.AssignedAt)
		o.metrics.RecordTaskDuration(duration)
	}
}

// healthChecker monitors agent health
func (o *Orchestrator) healthChecker() {
	defer o.wg.Done()
	
	ticker := time.NewTicker(o.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.checkAgentHealth()
		}
	}
}

// checkAgentHealth checks the health of all agents
func (o *Orchestrator) checkAgentHealth() {
	o.mu.RLock()
	agents := make([]AgentInterface, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}
	o.mu.RUnlock()
	
	for _, agent := range agents {
		if agent.GetStatus() == "unhealthy" {
			o.logger.Warn("Removing unhealthy agent",
				zap.String("agent_id", agent.GetID()),
			)
			
			o.mu.Lock()
			delete(o.agents, agent.GetID())
			o.mu.Unlock()
			
			agent.Stop()
		}
	}
}

// metricsCollector collects and publishes metrics
func (o *Orchestrator) metricsCollector() {
	defer o.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.publishMetrics()
		}
	}
}

// publishMetrics publishes current metrics
func (o *Orchestrator) publishMetrics() {
	o.mu.RLock()
	activeAgents := len(o.agents)
	queuedTasks := len(o.taskQueue)
	o.mu.RUnlock()
	
	metrics := &MetricsData{
		TotalAgents:     o.config.MaxAgents,
		ActiveAgents:    activeAgents,
		TotalTasks:      o.metrics.GetTotalTasks(),
		CompletedTasks:  o.metrics.GetCompletedTasks(),
		FailedTasks:     o.metrics.GetFailedTasks(),
		AverageTaskTime: o.metrics.GetAverageTaskTime(),
		QueuedTasks:     queuedTasks,
	}
	
	o.logger.Info("Orchestrator metrics",
		zap.Int("active_agents", metrics.ActiveAgents),
		zap.Int64("total_tasks", metrics.TotalTasks),
		zap.Int("queued_tasks", metrics.QueuedTasks),
	)
}

// SubmitTask submits a new task to the orchestrator
func (o *Orchestrator) SubmitTask(task *Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()
	
	o.mu.Lock()
	o.tasks[task.ID] = task
	o.mu.Unlock()
	
	o.metrics.IncrementTotalTasks()
	
	select {
	case o.taskQueue <- task:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("task queue is full")
	}
}

// GetTask retrieves a task by ID
func (o *Orchestrator) GetTask(taskID string) (*Task, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	task, exists := o.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	return task, nil
}

// GetAgents returns all active agents
func (o *Orchestrator) GetAgents() []*Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	agents := make([]*Agent, 0, len(o.agents))
	for _, agentInterface := range o.agents {
		agent := &Agent{
			ID:          agentInterface.GetID(),
			Type:        agentInterface.GetType(),
			Status:      agentInterface.GetStatus(),
			Workload:    agentInterface.GetWorkload(),
			MaxWorkload: o.config.MaxTasksPerAgent,
		}
		agents = append(agents, agent)
	}
	
	return agents
}

// Stop gracefully stops the orchestrator
func (o *Orchestrator) Stop() error {
	o.logger.Info("Stopping orchestrator")
	
	// Cancel context
	o.cancel()
	
	// Stop all agents
	o.mu.RLock()
	agents := make([]AgentInterface, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}
	o.mu.RUnlock()
	
	for _, agent := range agents {
		agent.Stop()
	}
	
	// Wait for workers to finish
	o.wg.Wait()
	
	// Close connections
	if o.redisClient != nil {
		o.redisClient.Close()
	}
	
	if o.temporalClient != nil {
		o.temporalClient.Close()
	}
	
	o.logger.Info("Orchestrator stopped")
	return nil
}