package base

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/types"
)

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	id           string
	role         types.AgentRole
	capabilities []types.AgentCapability
	status       types.AgentStatus
	context      *types.AgentContext
	metrics      types.AgentMetrics
	
	messageChan  chan *types.Message
	stopChan     chan struct{}
	mu           sync.RWMutex
	
	// Hooks for specialized behavior
	onInitialize func(context.Context, *types.AgentContext) error
	onExecute    func(context.Context, *types.Task) error
	onMessage    func(context.Context, *types.Message) error
	onShutdown   func(context.Context) error
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(role types.AgentRole, capabilities []types.AgentCapability) *BaseAgent {
	return &BaseAgent{
		id:           fmt.Sprintf("%s-%s", role, uuid.New().String()[:8]),
		role:         role,
		capabilities: capabilities,
		status:       types.StatusIdle,
		messageChan:  make(chan *types.Message, 100),
		stopChan:     make(chan struct{}),
		metrics: types.AgentMetrics{
			LastActive: time.Now(),
		},
	}
}

// ID returns the agent's unique identifier
func (a *BaseAgent) ID() string {
	return a.id
}

// Role returns the agent's role
func (a *BaseAgent) Role() types.AgentRole {
	return a.role
}

// Capabilities returns the agent's capabilities
func (a *BaseAgent) Capabilities() []types.AgentCapability {
	return a.capabilities
}

// Status returns the current agent status
func (a *BaseAgent) Status() types.AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// SetStatus updates the agent's status
func (a *BaseAgent) SetStatus(status types.AgentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = status
	a.metrics.LastActive = time.Now()
}

// Initialize prepares the agent for execution
func (a *BaseAgent) Initialize(ctx context.Context, agentCtx *types.AgentContext) error {
	a.mu.Lock()
	a.context = agentCtx
	a.status = types.StatusIdle
	a.mu.Unlock()

	// Start message processing goroutine
	go a.processMessages()

	// Subscribe to relevant topics
	if agentCtx.MessageBus != nil {
		topics := a.getSubscriptionTopics()
		for _, topic := range topics {
			if err := agentCtx.MessageBus.Subscribe(ctx, topic, a.handleBusMessage); err != nil {
				return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
			}
		}
	}

	// Call specialized initialization if provided
	if a.onInitialize != nil {
		return a.onInitialize(ctx, agentCtx)
	}

	return nil
}

// Execute performs a task
func (a *BaseAgent) Execute(ctx context.Context, task *types.Task) error {
	a.SetStatus(types.StatusExecuting)
	defer a.SetStatus(types.StatusIdle)

	startTime := time.Now()
	
	// Update task status
	task.Status = types.TaskInProgress
	now := time.Now()
	task.StartedAt = &now

	var err error
	if a.onExecute != nil {
		err = a.onExecute(ctx, task)
	} else {
		err = fmt.Errorf("no execution handler defined for agent %s", a.id)
	}

	// Update metrics
	duration := time.Since(startTime)
	a.updateMetrics(err == nil, duration)

	// Update task status
	completedTime := time.Now()
	task.CompletedAt = &completedTime
	if err != nil {
		task.Status = types.TaskFailed
		task.Error = err.Error()
		return err
	}

	task.Status = types.TaskCompleted
	return nil
}

// Shutdown cleanly stops the agent
func (a *BaseAgent) Shutdown(ctx context.Context) error {
	a.SetStatus(types.StatusCompleted)
	
	// Stop message processing
	close(a.stopChan)
	
	// Unsubscribe from topics
	if a.context != nil && a.context.MessageBus != nil {
		topics := a.getSubscriptionTopics()
		for _, topic := range topics {
			a.context.MessageBus.Unsubscribe(ctx, topic)
		}
	}

	if a.onShutdown != nil {
		return a.onShutdown(ctx)
	}

	return nil
}

// SendMessage sends a message to another agent or broadcast
func (a *BaseAgent) SendMessage(ctx context.Context, msg *types.Message) error {
	if msg.From == "" {
		msg.From = a.id
	}
	msg.Timestamp = time.Now()

	if a.context != nil && a.context.MessageBus != nil {
		topic := a.getMessageTopic(msg)
		return a.context.MessageBus.Publish(ctx, topic, msg)
	}

	return fmt.Errorf("message bus not available")
}

// ReceiveMessage handles incoming messages
func (a *BaseAgent) ReceiveMessage(ctx context.Context, msg *types.Message) error {
	select {
	case a.messageChan <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("message queue full for agent %s", a.id)
	}
}

// RequestCollaboration requests help from another agent
func (a *BaseAgent) RequestCollaboration(ctx context.Context, targetAgent string, request interface{}) (interface{}, error) {
	msg := &types.Message{
		ID:       uuid.New().String(),
		From:     a.id,
		To:       targetAgent,
		Type:     types.MsgTypeCollaboration,
		Content:  fmt.Sprintf("%v", request),
		Metadata: map[string]interface{}{"request": request},
	}

	if err := a.SendMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to send collaboration request: %w", err)
	}

	// Wait for response (simplified - in production, use proper async handling)
	timeout := time.After(30 * time.Second)
	for {
		select {
		case response := <-a.messageChan:
			if response.Type == types.MsgTypeResponse && response.ReplyTo == msg.ID {
				return response.Metadata["response"], nil
			}
		case <-timeout:
			return nil, fmt.Errorf("collaboration request timeout")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// ParticipateInConsensus participates in a multi-agent consensus
func (a *BaseAgent) ParticipateInConsensus(ctx context.Context, topic string, proposal interface{}) (bool, error) {
	// Analyze the proposal based on agent's expertise
	decision := a.analyzeProposal(proposal)
	
	vote := types.Vote{
		AgentID:   a.id,
		Decision:  decision,
		Reasoning: fmt.Sprintf("Agent %s (%s) votes %v based on analysis", a.id, a.role, decision),
		Timestamp: time.Now(),
	}

	// Send vote
	msg := &types.Message{
		ID:      uuid.New().String(),
		From:    a.id,
		Type:    types.MsgTypeConsensus,
		Content: fmt.Sprintf("Vote for %s", topic),
		Metadata: map[string]interface{}{
			"topic": topic,
			"vote":  vote,
		},
	}

	return decision, a.SendMessage(ctx, msg)
}

// LearnFromFeedback improves agent behavior based on feedback
func (a *BaseAgent) LearnFromFeedback(ctx context.Context, feedback interface{}) error {
	// Store feedback in shared memory for future reference
	if a.context != nil && a.context.SharedMemory != nil {
		if a.context.SharedMemory.Knowledge == nil {
			a.context.SharedMemory.Knowledge = make(map[string]interface{})
		}
		
		key := fmt.Sprintf("feedback_%s_%d", a.id, time.Now().Unix())
		a.context.SharedMemory.Knowledge[key] = feedback
	}

	return nil
}

// GetMetrics returns agent performance metrics
func (a *BaseAgent) GetMetrics() types.AgentMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.metrics
}

// SetExecutionHandler sets the task execution handler
func (a *BaseAgent) SetExecutionHandler(handler func(context.Context, *types.Task) error) {
	a.onExecute = handler
}

// SetMessageHandler sets the message handling function
func (a *BaseAgent) SetMessageHandler(handler func(context.Context, *types.Message) error) {
	a.onMessage = handler
}

// SetInitializeHandler sets the initialization handler
func (a *BaseAgent) SetInitializeHandler(handler func(context.Context, *types.AgentContext) error) {
	a.onInitialize = handler
}

// SetShutdownHandler sets the shutdown handler
func (a *BaseAgent) SetShutdownHandler(handler func(context.Context) error) {
	a.onShutdown = handler
}

// Private methods

func (a *BaseAgent) processMessages() {
	for {
		select {
		case msg := <-a.messageChan:
			if a.onMessage != nil {
				ctx := context.Background()
				if err := a.onMessage(ctx, msg); err != nil {
					// Log error (in production, use proper logging)
					fmt.Printf("Agent %s error processing message: %v\n", a.id, err)
				}
			}
		case <-a.stopChan:
			return
		}
	}
}

func (a *BaseAgent) handleBusMessage(msg *types.Message) {
	// Filter messages intended for this agent or broadcast
	if msg.To == a.id || msg.To == "" {
		a.messageChan <- msg
	}
}

func (a *BaseAgent) getSubscriptionTopics() []string {
	return []string{
		fmt.Sprintf("agent.%s", a.id),        // Direct messages
		fmt.Sprintf("role.%s", a.role),       // Role-based messages
		"agent.broadcast",                     // Broadcast messages
		"consensus",                           // Consensus topics
	}
}

func (a *BaseAgent) getMessageTopic(msg *types.Message) string {
	if msg.To != "" {
		return fmt.Sprintf("agent.%s", msg.To)
	}
	return "agent.broadcast"
}

func (a *BaseAgent) analyzeProposal(proposal interface{}) bool {
	// Simplified decision logic - in production, use sophisticated analysis
	// based on agent's role and expertise
	return true
}

func (a *BaseAgent) updateMetrics(success bool, duration time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if success {
		a.metrics.TasksCompleted++
	} else {
		a.metrics.TasksFailed++
	}

	// Update average task time
	totalTasks := a.metrics.TasksCompleted + a.metrics.TasksFailed
	if totalTasks > 0 {
		currentAvg := a.metrics.AverageTaskTime
		a.metrics.AverageTaskTime = (currentAvg*time.Duration(totalTasks-1) + duration) / time.Duration(totalTasks)
		a.metrics.SuccessRate = float64(a.metrics.TasksCompleted) / float64(totalTasks)
	}

	a.metrics.LastActive = time.Now()
}