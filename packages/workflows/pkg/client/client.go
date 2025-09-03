package client

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"github.com/google/uuid"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/types"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/workflows"
)

// WorkflowClient wraps Temporal client for workflow operations
type WorkflowClient struct {
	client client.Client
}

// NewWorkflowClient creates a new workflow client
func NewWorkflowClient(temporalHost string) (*WorkflowClient, error) {
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	return &WorkflowClient{
		client: c,
	}, nil
}

// Close closes the client connection
func (wc *WorkflowClient) Close() {
	wc.client.Close()
}

// StartCodeGeneration starts a new code generation workflow
func (wc *WorkflowClient) StartCodeGeneration(ctx context.Context, request types.CodeGenerationRequest) (string, error) {
	// Generate workflow ID
	workflowID := fmt.Sprintf("codegen-%s-%s", request.ID, uuid.New().String())

	// Set workflow options
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: workflows.CodeGenerationTaskQueue,
		WorkflowExecutionTimeout: workflows.WorkflowTimeout,
	}

	// Start workflow
	we, err := wc.client.ExecuteWorkflow(ctx, options, workflows.CodeGenerationWorkflow, request)
	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	return we.GetID(), nil
}

// GetCodeGenerationResult gets the result of a code generation workflow
func (wc *WorkflowClient) GetCodeGenerationResult(ctx context.Context, workflowID string) (*types.CodeGenerationResult, error) {
	// Get workflow execution
	we := wc.client.GetWorkflow(ctx, workflowID, "")

	// Get result
	var result types.CodeGenerationResult
	err := we.Get(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow result: %w", err)
	}

	return &result, nil
}

// GetWorkflowStatus gets the status of a workflow
func (wc *WorkflowClient) GetWorkflowStatus(ctx context.Context, workflowID string) (string, error) {
	// Describe workflow execution
	desc, err := wc.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return "", fmt.Errorf("failed to describe workflow: %w", err)
	}

	status := desc.WorkflowExecutionInfo.Status.String()
	return status, nil
}

// CancelWorkflow cancels a running workflow
func (wc *WorkflowClient) CancelWorkflow(ctx context.Context, workflowID string) error {
	err := wc.client.CancelWorkflow(ctx, workflowID, "")
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}
	return nil
}

// ListWorkflows lists recent workflows
func (wc *WorkflowClient) ListWorkflows(ctx context.Context, limit int) ([]WorkflowInfo, error) {
	query := "WorkflowType = 'CodeGenerationWorkflow'"
	
	iter, err := wc.client.ListWorkflow(ctx, &client.ListWorkflowExecutionsRequest{
		PageSize: int32(limit),
		Query:    query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	var workflows []WorkflowInfo
	for iter.HasNext() {
		exec, err := iter.Next()
		if err != nil {
			break
		}
		workflows = append(workflows, WorkflowInfo{
			ID:        exec.Execution.ID,
			RunID:     exec.Execution.RunID,
			Status:    exec.Status.String(),
			StartTime: exec.StartTime,
		})
	}

	return workflows, nil
}

// WorkflowInfo contains basic workflow information
type WorkflowInfo struct {
	ID        string    `json:"id"`
	RunID     string    `json:"runId"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"startTime"`
}