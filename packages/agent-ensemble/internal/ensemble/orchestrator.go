package ensemble

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/agent-ensemble/internal/models"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/shared/telemetry"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// Orchestrator manages the agent ensemble
type Orchestrator struct {
	agents         map[string]*models.Agent
	tasks          map[string]*models.Task
	collaborations map[string]*models.AgentCollaboration
	nc             *nats.Conn
	js             nats.JetStreamContext
	logger         *logrus.Logger
	tracer         trace.Tracer
	taskQueue      chan *models.Task
	mu             sync.RWMutex
}

// NewOrchestrator creates a new agent orchestrator
func NewOrchestrator(natsURL string, logger *logrus.Logger) (*Orchestrator, error) {
	// Connect to NATS for agent communication
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	o := &Orchestrator{
		agents:         make(map[string]*models.Agent),
		tasks:          make(map[string]*models.Task),
		collaborations: make(map[string]*models.AgentCollaboration),
		nc:             nc,
		js:             js,
		logger:         logger,
		tracer:         telemetry.GetTracer("agent-orchestrator"),
		taskQueue:      make(chan *models.Task, 1000),
	}

	// Start background workers
	go o.taskScheduler()
	go o.healthMonitor()

	return o, nil
}

// RegisterAgent registers a new agent in the ensemble
func (o *Orchestrator) RegisterAgent(agent *models.Agent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	agent.State.Status = "idle"

	o.agents[agent.ID] = agent
	
	// Subscribe agent to its message channel
	subject := fmt.Sprintf("agents.%s.tasks", agent.ID)
	if _, err := o.nc.Subscribe(subject, o.handleAgentTask); err != nil {
		return fmt.Errorf("failed to subscribe agent: %w", err)
	}

	o.logger.WithFields(logrus.Fields{
		"agent_id":   agent.ID,
		"agent_type": agent.Type,
	}).Info("Registered new agent")

	return nil
}

// SubmitTask submits a new task to the ensemble
func (o *Orchestrator) SubmitTask(ctx context.Context, task *models.Task) error {
	ctx, span := o.tracer.Start(ctx, "SubmitTask")
	defer span.End()

	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	task.Status = models.TaskStatusPending
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	o.mu.Lock()
	o.tasks[task.ID] = task
	o.mu.Unlock()

	// Add to task queue
	select {
	case o.taskQueue <- task:
		o.logger.WithField("task_id", task.ID).Info("Task submitted")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("task queue full")
	}

	return nil
}

// CreateCollaboration creates a multi-agent collaboration
func (o *Orchestrator) CreateCollaboration(ctx context.Context, task *models.Task, strategy models.CollaborationStrategy) (*models.AgentCollaboration, error) {
	ctx, span := o.tracer.Start(ctx, "CreateCollaboration")
	defer span.End()

	// Select agents for collaboration
	agents := o.selectAgentsForTask(task)
	if len(agents) < 2 {
		return nil, fmt.Errorf("insufficient agents for collaboration")
	}

	// Create collaboration
	collab := &models.AgentCollaboration{
		ID:          uuid.New().String(),
		Name:        fmt.Sprintf("Collaboration for %s", task.Description),
		Description: task.Description,
		Task:        task,
		Strategy:    strategy,
		Status:      "active",
		CreatedAt:   time.Now(),
		Messages:    []models.CollaborationMessage{},
	}

	// Assign participants
	for _, agent := range agents {
		collab.Participants = append(collab.Participants, agent.ID)
	}

	// Select leader based on expertise match
	collab.Leader = o.selectLeader(agents, task)

	o.mu.Lock()
	o.collaborations[collab.ID] = collab
	o.mu.Unlock()

	// Start collaboration workflow
	go o.runCollaboration(ctx, collab)

	return collab, nil
}

// GetAgentRecommendations recommends agents for a task
func (o *Orchestrator) GetAgentRecommendations(task *models.Task) []*models.Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	recommendations := []*models.Agent{}
	
	for _, agent := range o.agents {
		score := o.calculateAgentScore(agent, task)
		if score > 0.5 {
			recommendations = append(recommendations, agent)
		}
	}

	// Sort by score
	sort.Slice(recommendations, func(i, j int) bool {
		return o.calculateAgentScore(recommendations[i], task) > 
			   o.calculateAgentScore(recommendations[j], task)
	})

	return recommendations
}

// Private methods

func (o *Orchestrator) taskScheduler() {
	for task := range o.taskQueue {
		o.logger.WithField("task_id", task.ID).Debug("Processing task from queue")
		
		// Find best agent for task
		agent := o.findBestAgent(task)
		if agent == nil {
			o.logger.WithField("task_id", task.ID).Warn("No suitable agent found")
			task.Status = models.TaskStatusFailed
			continue
		}

		// Assign task to agent
		if err := o.assignTaskToAgent(task, agent); err != nil {
			o.logger.WithError(err).Error("Failed to assign task")
			task.Status = models.TaskStatusFailed
		}
	}
}

func (o *Orchestrator) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		o.mu.Lock()
		for _, agent := range o.agents {
			// Check agent health
			if time.Since(agent.State.LastActiveAt) > 5*time.Minute {
				agent.State.Status = "offline"
			}
			
			// Reset error count if agent is healthy
			if agent.State.ConsecutiveErrors > 0 && agent.State.Status == "idle" {
				agent.State.ConsecutiveErrors = 0
			}
		}
		o.mu.Unlock()
	}
}

func (o *Orchestrator) findBestAgent(task *models.Task) *models.Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var bestAgent *models.Agent
	bestScore := 0.0

	for _, agent := range o.agents {
		if agent.State.Status != "idle" {
			continue
		}

		score := o.calculateAgentScore(agent, task)
		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}

	return bestAgent
}

func (o *Orchestrator) calculateAgentScore(agent *models.Agent, task *models.Task) float64 {
	score := 0.0
	
	// Check capability match
	taskType := task.Type
	for _, cap := range agent.Capabilities {
		if string(cap) == taskType {
			score += 0.5
			break
		}
	}

	// Check expertise match
	if taskLang, ok := task.Input["language"].(string); ok {
		for _, exp := range agent.Expertise {
			if exp == taskLang {
				score += 0.3
				break
			}
		}
	}

	// Factor in performance
	score += agent.Performance.SuccessRate * 0.2

	// Penalize if agent has errors
	if agent.State.ConsecutiveErrors > 0 {
		score *= 0.5
	}

	return score
}

func (o *Orchestrator) selectAgentsForTask(task *models.Task) []*models.Agent {
	agents := o.GetAgentRecommendations(task)
	
	// Limit to top 5 agents
	if len(agents) > 5 {
		agents = agents[:5]
	}
	
	return agents
}

func (o *Orchestrator) selectLeader(agents []*models.Agent, task *models.Task) string {
	if len(agents) == 0 {
		return ""
	}

	// Select agent with highest score as leader
	bestScore := 0.0
	leaderID := agents[0].ID
	
	for _, agent := range agents {
		score := o.calculateAgentScore(agent, task)
		if score > bestScore {
			bestScore = score
			leaderID = agent.ID
		}
	}
	
	return leaderID
}

func (o *Orchestrator) assignTaskToAgent(task *models.Task, agent *models.Agent) error {
	o.mu.Lock()
	task.AssignedTo = agent.ID
	task.Status = models.TaskStatusAssigned
	task.UpdatedAt = time.Now()
	
	agent.State.Status = "busy"
	agent.State.CurrentTasks = append(agent.State.CurrentTasks, task.ID)
	agent.State.LastActiveAt = time.Now()
	o.mu.Unlock()

	// Send task to agent via NATS
	subject := fmt.Sprintf("agents.%s.tasks", agent.ID)
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	if err := o.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to send task to agent: %w", err)
	}

	o.logger.WithFields(logrus.Fields{
		"task_id":  task.ID,
		"agent_id": agent.ID,
	}).Info("Task assigned to agent")

	return nil
}

func (o *Orchestrator) handleAgentTask(msg *nats.Msg) {
	// This would be implemented by the actual agent
	// For now, simulate task processing
	var task models.Task
	if err := json.Unmarshal(msg.Data, &task); err != nil {
		o.logger.WithError(err).Error("Failed to unmarshal task")
		return
	}

	o.logger.WithField("task_id", task.ID).Debug("Agent received task")
	
	// Simulate processing
	go func() {
		time.Sleep(2 * time.Second)
		
		// Mark task as completed
		o.mu.Lock()
		if t, exists := o.tasks[task.ID]; exists {
			t.Status = models.TaskStatusCompleted
			t.UpdatedAt = time.Now()
			t.Result = &models.TaskResult{
				Output: "Task completed successfully",
				Metrics: models.TaskMetrics{
					ExecutionTime: 2000,
					TokensUsed:    100,
					Quality:       95.0,
				},
				CompletedAt: time.Now(),
			}
		}
		
		// Update agent state
		if agent, exists := o.agents[task.AssignedTo]; exists {
			agent.State.Status = "idle"
			// Remove task from current tasks
			newTasks := []string{}
			for _, tid := range agent.State.CurrentTasks {
				if tid != task.ID {
					newTasks = append(newTasks, tid)
				}
			}
			agent.State.CurrentTasks = newTasks
			agent.Performance.TasksCompleted++
		}
		o.mu.Unlock()
	}()
}

func (o *Orchestrator) runCollaboration(ctx context.Context, collab *models.AgentCollaboration) {
	ctx, span := o.tracer.Start(ctx, "RunCollaboration")
	defer span.End()

	o.logger.WithField("collaboration_id", collab.ID).Info("Starting collaboration")

	switch collab.Strategy.Type {
	case "sequential":
		o.runSequentialCollaboration(ctx, collab)
	case "parallel":
		o.runParallelCollaboration(ctx, collab)
	case "voting":
		o.runVotingCollaboration(ctx, collab)
	case "consensus":
		o.runConsensusCollaboration(ctx, collab)
	default:
		o.logger.WithField("strategy", collab.Strategy.Type).Error("Unknown collaboration strategy")
	}

	// Mark collaboration as completed
	o.mu.Lock()
	collab.Status = "completed"
	now := time.Now()
	collab.CompletedAt = &now
	o.mu.Unlock()
}

func (o *Orchestrator) runSequentialCollaboration(ctx context.Context, collab *models.AgentCollaboration) {
	// Each agent works on the task sequentially
	for _, agentID := range collab.Participants {
		o.mu.RLock()
		agent, exists := o.agents[agentID]
		o.mu.RUnlock()
		
		if !exists {
			continue
		}

		// Create sub-task for this agent
		subTask := &models.Task{
			ID:          uuid.New().String(),
			Type:        collab.Task.Type,
			Description: fmt.Sprintf("Phase for %s", agent.Name),
			Input:       collab.Task.Input,
			Priority:    collab.Task.Priority,
			CreatedAt:   time.Now(),
		}

		if err := o.assignTaskToAgent(subTask, agent); err != nil {
			o.logger.WithError(err).Error("Failed to assign subtask")
			continue
		}

		// Wait for completion
		// In production, use proper synchronization
		time.Sleep(3 * time.Second)
	}
}

func (o *Orchestrator) runParallelCollaboration(ctx context.Context, collab *models.AgentCollaboration) {
	// All agents work on the task in parallel
	var wg sync.WaitGroup
	
	for _, agentID := range collab.Participants {
		wg.Add(1)
		go func(aid string) {
			defer wg.Done()
			
			o.mu.RLock()
			agent, exists := o.agents[aid]
			o.mu.RUnlock()
			
			if !exists {
				return
			}

			subTask := &models.Task{
				ID:          uuid.New().String(),
				Type:        collab.Task.Type,
				Description: fmt.Sprintf("Parallel task for %s", agent.Name),
				Input:       collab.Task.Input,
				Priority:    collab.Task.Priority,
				CreatedAt:   time.Now(),
			}

			if err := o.assignTaskToAgent(subTask, agent); err != nil {
				o.logger.WithError(err).Error("Failed to assign parallel task")
			}
		}(agentID)
	}
	
	wg.Wait()
}

func (o *Orchestrator) runVotingCollaboration(ctx context.Context, collab *models.AgentCollaboration) {
	// Each agent proposes a solution, then they vote
	proposals := make(map[string]interface{})
	
	// Collect proposals
	for _, agentID := range collab.Participants {
		// In production, actually get proposals from agents
		proposals[agentID] = fmt.Sprintf("Proposal from agent %s", agentID)
	}

	// Create voting options
	options := []models.VotingOption{}
	for agentID, proposal := range proposals {
		options = append(options, models.VotingOption{
			ID:          uuid.New().String(),
			Description: fmt.Sprintf("Proposal from %s", agentID),
			ProposedBy:  agentID,
			Content:     proposal,
		})
	}

	// Conduct voting
	voting := &models.VotingResult{
		ID:         uuid.New().String(),
		Subject:    collab.Task.Description,
		Options:    options,
		Votes:      make(map[string]string),
		Rationales: make(map[string]string),
		Timestamp:  time.Now(),
	}

	// Simulate voting
	for _, agentID := range collab.Participants {
		// In production, agents would actually vote
		voting.Votes[agentID] = options[0].ID
		voting.Rationales[agentID] = "Best solution based on analysis"
	}

	// Determine winner
	voting.Winner = options[0].ID
	voting.Confidence = 0.85

	o.logger.WithField("winner", voting.Winner).Info("Voting completed")
}

func (o *Orchestrator) runConsensusCollaboration(ctx context.Context, collab *models.AgentCollaboration) {
	// Agents discuss until consensus is reached
	maxRounds := 5
	consensusReached := false
	
	for round := 0; round < maxRounds && !consensusReached; round++ {
		o.logger.WithField("round", round).Debug("Consensus round")
		
		// Each agent shares their view
		for _, agentID := range collab.Participants {
			msg := models.CollaborationMessage{
				ID:        uuid.New().String(),
				From:      agentID,
				To:        []string{"all"},
				Type:      "response",
				Content:   fmt.Sprintf("Agent %s view for round %d", agentID, round),
				Timestamp: time.Now(),
			}
			
			o.mu.Lock()
			collab.Messages = append(collab.Messages, msg)
			o.mu.Unlock()
		}
		
		// Check for consensus (simplified)
		if round >= 2 {
			consensusReached = true
		}
		
		time.Sleep(1 * time.Second)
	}
	
	if consensusReached {
		o.logger.Info("Consensus reached")
	} else {
		o.logger.Warn("Failed to reach consensus")
	}
}

// Close gracefully shuts down the orchestrator
func (o *Orchestrator) Close() error {
	close(o.taskQueue)
	return o.nc.Close()
}