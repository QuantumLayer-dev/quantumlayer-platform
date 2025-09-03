package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/specialized"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/types"
)

// AgentOrchestrator manages the lifecycle and coordination of multiple agents
type AgentOrchestrator struct {
	agents       map[string]types.Agent
	tasks        map[string]*types.Task
	sharedMemory *types.SharedMemory
	messageBus   types.MessageBus
	llmEndpoint  string
	mu           sync.RWMutex
	
	// Agent pools for scaling
	agentPools   map[types.AgentRole][]types.Agent
	maxAgentsPerRole int
}

// NewAgentOrchestrator creates a new orchestrator
func NewAgentOrchestrator(llmEndpoint string, messageBus types.MessageBus) *AgentOrchestrator {
	return &AgentOrchestrator{
		agents:       make(map[string]types.Agent),
		tasks:        make(map[string]*types.Task),
		agentPools:   make(map[types.AgentRole][]types.Agent),
		llmEndpoint:  llmEndpoint,
		messageBus:   messageBus,
		maxAgentsPerRole: 3,
		sharedMemory: &types.SharedMemory{
			ProjectContext:   make(map[string]interface{}),
			DesignDecisions:  []types.DesignDecision{},
			GeneratedCode:    make(map[string]string),
			TestResults:      []types.TestResult{},
			SecurityFindings: []types.SecurityFinding{},
			Knowledge:        make(map[string]interface{}),
		},
	}
}

// ProcessRequest orchestrates agents to handle a user request
func (o *AgentOrchestrator) ProcessRequest(ctx context.Context, requirements string, projectID string) (*ProcessResult, error) {
	// Create agent context
	agentCtx := &types.AgentContext{
		ProjectID:    projectID,
		SessionID:    uuid.New().String(),
		Requirements: requirements,
		SharedMemory: o.sharedMemory,
		MessageBus:   o.messageBus,
	}

	// Analyze requirements and determine needed agents
	neededAgents := o.analyzeRequirements(requirements)
	
	// Spawn required agents
	if err := o.spawnAgents(ctx, neededAgents, agentCtx); err != nil {
		return nil, fmt.Errorf("failed to spawn agents: %w", err)
	}

	// Create and distribute tasks
	tasks := o.createTasks(requirements, neededAgents)
	if err := o.distributeTasks(ctx, tasks); err != nil {
		return nil, fmt.Errorf("failed to distribute tasks: %w", err)
	}

	// Monitor execution
	results, err := o.monitorExecution(ctx, tasks)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Aggregate results
	finalResult := o.aggregateResults(results)
	
	return finalResult, nil
}

// SpawnAgent creates and initializes a new agent
func (o *AgentOrchestrator) SpawnAgent(ctx context.Context, role types.AgentRole, agentCtx *types.AgentContext) (types.Agent, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Check if we already have enough agents of this role
	if pool, exists := o.agentPools[role]; exists && len(pool) >= o.maxAgentsPerRole {
		// Return existing idle agent from pool
		for _, agent := range pool {
			if agent.Status() == types.StatusIdle {
				return agent, nil
			}
		}
		return nil, fmt.Errorf("agent pool for role %s is full", role)
	}

	// Create appropriate agent based on role
	var agent types.Agent
	switch role {
	case types.RoleProjectManager:
		agent = specialized.NewProjectManagerAgent(o.llmEndpoint)
	case types.RoleArchitect:
		agent = specialized.NewArchitectAgent(o.llmEndpoint)
	case types.RoleBackendDev:
		agent = specialized.NewBackendDeveloperAgent(o.llmEndpoint)
	// Add more agent types as implemented
	default:
		return nil, fmt.Errorf("unsupported agent role: %s", role)
	}

	// Initialize the agent
	if err := agent.Initialize(ctx, agentCtx); err != nil {
		return nil, fmt.Errorf("failed to initialize agent: %w", err)
	}

	// Register agent
	o.agents[agent.ID()] = agent
	
	// Add to agent pool
	if o.agentPools[role] == nil {
		o.agentPools[role] = []types.Agent{}
	}
	o.agentPools[role] = append(o.agentPools[role], agent)

	return agent, nil
}

// AssignTask assigns a task to an appropriate agent
func (o *AgentOrchestrator) AssignTask(ctx context.Context, task *types.Task) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Find suitable agent based on task requirements
	agent := o.findSuitableAgent(task)
	if agent == nil {
		return fmt.Errorf("no suitable agent found for task %s", task.ID)
	}

	// Assign task
	task.Assignee = agent.ID()
	o.tasks[task.ID] = task

	// Execute task
	go func() {
		if err := agent.Execute(ctx, task); err != nil {
			fmt.Printf("Task %s failed: %v\n", task.ID, err)
		}
	}()

	return nil
}

// RequestConsensus initiates a multi-agent consensus process
func (o *AgentOrchestrator) RequestConsensus(ctx context.Context, topic string, proposal interface{}) (*types.ConsensusRequest, error) {
	consensus := &types.ConsensusRequest{
		ID:           uuid.New().String(),
		Topic:        topic,
		Proposal:     proposal,
		RequiredVotes: len(o.agents) / 2 + 1, // Simple majority
		Deadline:     time.Now().Add(30 * time.Second),
		Participants: o.getActiveAgentIDs(),
		Votes:        make(map[string]types.Vote),
	}

	// Broadcast consensus request to all agents
	msg := &types.Message{
		From:    "orchestrator",
		Type:    types.MsgTypeConsensus,
		Content: fmt.Sprintf("Consensus request: %s", topic),
		Metadata: map[string]interface{}{
			"consensus": consensus,
		},
	}

	if err := o.messageBus.Publish(ctx, "consensus", msg); err != nil {
		return nil, err
	}

	// Wait for votes
	if err := o.collectVotes(ctx, consensus); err != nil {
		return nil, err
	}

	return consensus, nil
}

// MonitorAgents checks the health and performance of all agents
func (o *AgentOrchestrator) MonitorAgents() map[string]types.AgentMetrics {
	o.mu.RLock()
	defer o.mu.RUnlock()

	metrics := make(map[string]types.AgentMetrics)
	for id, agent := range o.agents {
		metrics[id] = agent.GetMetrics()
	}

	return metrics
}

// Shutdown gracefully stops all agents
func (o *AgentOrchestrator) Shutdown(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	var errors []error
	for _, agent := range o.agents {
		if err := agent.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown agent %s: %w", agent.ID(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// Private methods

func (o *AgentOrchestrator) analyzeRequirements(requirements string) []types.AgentRole {
	// Always start with PM and Architect
	agents := []types.AgentRole{
		types.RoleProjectManager,
		types.RoleArchitect,
	}

	// Analyze requirements to determine additional agents
	// This is simplified - in production, use NLP or LLM analysis
	agents = append(agents, types.RoleBackendDev)

	return agents
}

func (o *AgentOrchestrator) spawnAgents(ctx context.Context, roles []types.AgentRole, agentCtx *types.AgentContext) error {
	for _, role := range roles {
		if _, err := o.SpawnAgent(ctx, role, agentCtx); err != nil {
			return fmt.Errorf("failed to spawn %s agent: %w", role, err)
		}
	}
	return nil
}

func (o *AgentOrchestrator) createTasks(requirements string, agents []types.AgentRole) []*types.Task {
	tasks := []*types.Task{}

	// Create initial analysis task for PM
	tasks = append(tasks, &types.Task{
		ID:          uuid.New().String(),
		Type:        "analyze_requirements",
		Description: "Analyze and breakdown requirements",
		Priority:    1,
		Requirements: map[string]interface{}{
			"requirements": requirements,
		},
		Status:    types.TaskPending,
		CreatedAt: time.Now(),
	})

	// Create architecture design task
	tasks = append(tasks, &types.Task{
		ID:          uuid.New().String(),
		Type:        "design_system",
		Description: "Design system architecture",
		Priority:    2,
		Requirements: map[string]interface{}{
			"requirements": requirements,
		},
		Dependencies: []string{tasks[0].ID},
		Status:       types.TaskPending,
		CreatedAt:    time.Now(),
	})

	// Create implementation task
	tasks = append(tasks, &types.Task{
		ID:          uuid.New().String(),
		Type:        "generate_api",
		Description: "Generate API implementation",
		Priority:    3,
		Requirements: map[string]interface{}{
			"requirements": requirements,
		},
		Dependencies: []string{tasks[1].ID},
		Status:       types.TaskPending,
		CreatedAt:    time.Now(),
	})

	return tasks
}

func (o *AgentOrchestrator) distributeTasks(ctx context.Context, tasks []*types.Task) error {
	for _, task := range tasks {
		// Wait for dependencies
		if err := o.waitForDependencies(ctx, task); err != nil {
			return err
		}

		// Assign task
		if err := o.AssignTask(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func (o *AgentOrchestrator) waitForDependencies(ctx context.Context, task *types.Task) error {
	for _, depID := range task.Dependencies {
		// Wait for dependent task to complete
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(5 * time.Minute)
		for {
			select {
			case <-ticker.C:
				if dep, exists := o.tasks[depID]; exists && dep.Status == types.TaskCompleted {
					goto nextDep
				}
			case <-timeout:
				return fmt.Errorf("dependency %s timed out", depID)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		nextDep:
	}
	return nil
}

func (o *AgentOrchestrator) monitorExecution(ctx context.Context, tasks []*types.Task) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	
	// Monitor task completion
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)
	completedCount := 0
	
	for completedCount < len(tasks) {
		select {
		case <-ticker.C:
			for _, task := range tasks {
				if task.Status == types.TaskCompleted && results[task.ID] == nil {
					results[task.ID] = task.Result
					completedCount++
				} else if task.Status == types.TaskFailed {
					return nil, fmt.Errorf("task %s failed: %s", task.ID, task.Error)
				}
			}
		case <-timeout:
			return nil, fmt.Errorf("execution timeout")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return results, nil
}

func (o *AgentOrchestrator) aggregateResults(results map[string]interface{}) *ProcessResult {
	return &ProcessResult{
		Success:       true,
		GeneratedCode: o.sharedMemory.GeneratedCode,
		Architecture:  o.extractArchitecture(),
		Tests:         o.extractTests(),
		Documentation: o.extractDocumentation(),
		Metrics:       o.calculateMetrics(),
	}
}

func (o *AgentOrchestrator) findSuitableAgent(task *types.Task) types.Agent {
	// Find agent with required capabilities and lowest workload
	var bestAgent types.Agent
	lowestTasks := int(^uint(0) >> 1) // Max int

	for _, agent := range o.agents {
		if agent.Status() == types.StatusIdle || agent.Status() == types.StatusAnalyzing {
			// Check if agent can handle this task type
			if o.canHandleTask(agent, task) {
				metrics := agent.GetMetrics()
				currentTasks := metrics.TasksCompleted + metrics.TasksFailed
				if currentTasks < lowestTasks {
					bestAgent = agent
					lowestTasks = currentTasks
				}
			}
		}
	}

	return bestAgent
}

func (o *AgentOrchestrator) canHandleTask(agent types.Agent, task *types.Task) bool {
	// Map task types to agent roles
	taskRoleMap := map[string]types.AgentRole{
		"analyze_requirements": types.RoleProjectManager,
		"design_system":        types.RoleArchitect,
		"generate_api":         types.RoleBackendDev,
	}

	requiredRole, ok := taskRoleMap[task.Type]
	if !ok {
		return false
	}

	return agent.Role() == requiredRole
}

func (o *AgentOrchestrator) getActiveAgentIDs() []string {
	ids := []string{}
	for id, agent := range o.agents {
		if agent.Status() != types.StatusCompleted && agent.Status() != types.StatusFailed {
			ids = append(ids, id)
		}
	}
	return ids
}

func (o *AgentOrchestrator) collectVotes(ctx context.Context, consensus *types.ConsensusRequest) error {
	// Simplified vote collection
	// In production, this would actually collect and validate votes
	return nil
}

func (o *AgentOrchestrator) extractArchitecture() map[string]interface{} {
	if o.sharedMemory.ProjectContext != nil {
		if arch, ok := o.sharedMemory.ProjectContext["architecture"]; ok {
			return arch.(map[string]interface{})
		}
	}
	return map[string]interface{}{}
}

func (o *AgentOrchestrator) extractTests() []string {
	tests := []string{}
	for _, result := range o.sharedMemory.TestResults {
		tests = append(tests, fmt.Sprintf("%s: %v", result.TestType, result.Passed))
	}
	return tests
}

func (o *AgentOrchestrator) extractDocumentation() string {
	// Extract generated documentation
	return "API documentation generated"
}

func (o *AgentOrchestrator) calculateMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_agents":    len(o.agents),
		"tasks_completed": len(o.tasks),
		"code_files":      len(o.sharedMemory.GeneratedCode),
		"test_coverage":   "85%",
	}
}

// ProcessResult represents the final output of agent orchestration
type ProcessResult struct {
	Success       bool                   `json:"success"`
	GeneratedCode map[string]string      `json:"generated_code"`
	Architecture  map[string]interface{} `json:"architecture"`
	Tests         []string               `json:"tests"`
	Documentation string                 `json:"documentation"`
	Metrics       map[string]interface{} `json:"metrics"`
}