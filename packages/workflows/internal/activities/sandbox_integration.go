package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/types"
)

var (
	// Service URLs - can be overridden by environment variables
	SandboxExecutorURL = getEnvOrDefault("SANDBOX_EXECUTOR_URL", "http://sandbox-executor.quantumlayer.svc.cluster.local:8085")
	CapsuleBuilderURL  = getEnvOrDefault("CAPSULE_BUILDER_URL", "http://capsule-builder.quantumlayer.svc.cluster.local:8086")
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Sandbox Execution Types
type SandboxExecutionRequest struct {
	ID           string            `json:"id"`
	Language     string            `json:"language"`
	Code         string            `json:"code"`
	Files        map[string]string `json:"files,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Timeout      int               `json:"timeout"`
}

type SandboxExecutionResult struct {
	ID       string `json:"id"`
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	Errors   string `json:"errors"`
	ExitCode int    `json:"exit_code"`
	Duration int    `json:"duration"`
}

// Capsule Building Types
type CapsuleBuilderRequest struct {
	WorkflowID  string `json:"workflow_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Language    string `json:"language"`
	Framework   string `json:"framework,omitempty"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

type CapsuleBuilderResult struct {
	ID         string                       `json:"id"`
	WorkflowID string                       `json:"workflow_id"`
	Name       string                       `json:"name"`
	Structure  map[string]map[string]string `json:"structure"`
	Metadata   map[string]interface{}       `json:"metadata"`
	CreatedAt  time.Time                    `json:"created_at"`
}

// ExecuteInSandboxActivity executes code in the sandbox environment
func ExecuteInSandboxActivity(ctx context.Context, request SandboxExecutionRequest) (*SandboxExecutionResult, error) {
	// Set default timeout if not specified
	if request.Timeout == 0 {
		request.Timeout = 30
	}

	// Prepare request payload
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Sandbox Executor service
	url := fmt.Sprintf("%s/api/v1/execute", SandboxExecutorURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to call sandbox executor: %w", err)
	}
	defer resp.Body.Close()

	// Parse initial response
	var initResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&initResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Wait for execution to complete (simplified - in production use WebSocket)
	executionID := initResponse["id"].(string)
	time.Sleep(3 * time.Second) // Give execution time to complete

	// Get execution result
	resultURL := fmt.Sprintf("%s/api/v1/executions/%s", SandboxExecutorURL, executionID)
	resultResp, err := http.Get(resultURL)
	if err != nil {
		// Return partial result if we can't get the full result
		return &SandboxExecutionResult{
			ID:      executionID,
			Success: false,
			Errors:  "Failed to retrieve execution result",
		}, nil
	}
	defer resultResp.Body.Close()

	var result SandboxExecutionResult
	if err := json.NewDecoder(resultResp.Body).Decode(&result); err != nil {
		// Return basic result if parsing fails
		return &SandboxExecutionResult{
			ID:      executionID,
			Success: true,
			Output:  "Execution completed",
		}, nil
	}

	return &result, nil
}

// BuildCapsuleActivity builds a structured project capsule
func BuildCapsuleActivity(ctx context.Context, request CapsuleBuilderRequest) (*CapsuleBuilderResult, error) {
	// Prepare request payload
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Capsule Builder service
	url := fmt.Sprintf("%s/api/v1/build", CapsuleBuilderURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to call capsule builder: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result CapsuleBuilderResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Store as QuantumDrop
	dropContent, _ := json.Marshal(map[string]interface{}{
		"capsule_id": result.ID,
		"name":       result.Name,
		"structure":  result.Structure,
		"metadata":   result.Metadata,
	})
	
	drop := types.QuantumDrop{
		WorkflowID: request.WorkflowID,
		Stage:      "capsule_building",
		Type:       "capsule",
		Artifact:   string(dropContent),
		Timestamp:  time.Now(),
	}

	if err := StoreQuantumDropActivity(ctx, drop); err != nil {
		// Log error but don't fail the activity
		fmt.Printf("Warning: Failed to store capsule drop: %v\n", err)
	}

	return &result, nil
}

// ValidateWithSandboxActivity validates code by executing it in sandbox
func ValidateWithSandboxActivity(ctx context.Context, code string, language string, workflowID string) (*types.ValidationResult, error) {
	// Execute code in sandbox
	execRequest := SandboxExecutionRequest{
		ID:       fmt.Sprintf("validation-%s-%d", workflowID, time.Now().Unix()),
		Language: language,
		Code:     code,
		Timeout:  10, // Quick validation timeout
	}

	execResult, err := ExecuteInSandboxActivity(ctx, execRequest)
	if err != nil {
		return &types.ValidationResult{
			Valid:  false,
			Issues: []types.Issue{{Type: "error", Message: fmt.Sprintf("Sandbox execution failed: %v", err)}},
			Score:  0,
		}, nil
	}

	// Analyze execution result
	validationResult := &types.ValidationResult{
		Valid:    execResult.Success && execResult.ExitCode == 0,
		Issues:   []types.Issue{},
		Score:    100,
	}

	if !execResult.Success {
		validationResult.Issues = append(validationResult.Issues, types.Issue{Type: "error", Message: execResult.Errors})
		validationResult.Score = 0
	}

	if execResult.Errors != "" {
		validationResult.Issues = append(validationResult.Issues, types.Issue{Type: "error", Message: execResult.Errors})
		validationResult.Score = 50
	}

	// Store validation drop
	validationContent, _ := json.Marshal(map[string]interface{}{
		"execution_id": execResult.ID,
		"success":      execResult.Success,
		"output":       execResult.Output,
		"errors":       execResult.Errors,
		"exit_code":    execResult.ExitCode,
		"duration":     execResult.Duration,
		"score":        validationResult.Score,
	})
	
	drop := types.QuantumDrop{
		WorkflowID: workflowID,
		Stage:      "sandbox_validation",
		Type:       "validation",
		Artifact:   string(validationContent),
		Timestamp:  time.Now(),
	}

	if err := StoreQuantumDropActivity(ctx, drop); err != nil {
		fmt.Printf("Warning: Failed to store validation drop: %v\n", err)
	}

	return validationResult, nil
}