package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/activities"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/types"
)

const (
	// Workflow names
	CodeGenerationWorkflowName = "CodeGenerationWorkflow"
	
	// Task queue names
	CodeGenerationTaskQueue = "code-generation"
	
	// Workflow timeouts
	WorkflowTimeout = 5 * time.Minute
	ActivityTimeout = 1 * time.Minute
)

// CodeGenerationWorkflow is the main workflow for generating code
func CodeGenerationWorkflow(ctx workflow.Context, request types.CodeGenerationRequest) (*types.CodeGenerationResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting code generation workflow", "requestID", request.ID)

	// Set workflow options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: ActivityTimeout,
		RetryPolicy: &workflow.RetryPolicy{
			MaximumAttempts: 3,
			InitialInterval: time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Initialize result
	result := &types.CodeGenerationResult{
		ID:        workflow.GetInfo(ctx).WorkflowExecution.ID,
		RequestID: request.ID,
		Files:     []types.GeneratedFile{},
		Metrics:   types.GenerationMetrics{},
	}

	// Stage 1: Enhance prompt using Meta Prompt Engine
	logger.Info("Stage 1: Enhancing prompt")
	var enhancedPrompt types.PromptEnhancementResult
	enhanceRequest := types.PromptEnhancementRequest{
		OriginalPrompt: request.Prompt,
		Type:           request.Type,
		Context:        request.Context,
		TargetProvider: getPreferredProvider(request.Preferences.Providers),
	}
	
	err := workflow.ExecuteActivity(ctx, activities.EnhancePromptActivity, enhanceRequest).Get(ctx, &enhancedPrompt)
	if err != nil {
		logger.Error("Failed to enhance prompt", "error", err)
		// Continue with original prompt if enhancement fails
		enhancedPrompt.EnhancedPrompt = request.Prompt
	}

	// Stage 2: Parse requirements and determine architecture
	logger.Info("Stage 2: Parsing requirements")
	var requirements activities.ParsedRequirements
	err = workflow.ExecuteActivity(ctx, activities.ParseRequirementsActivity, request).Get(ctx, &requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to parse requirements: %w", err)
	}

	// Stage 3: Generate code using LLM Router
	logger.Info("Stage 3: Generating code")
	generationRequest := activities.LLMGenerationRequest{
		Prompt:   enhancedPrompt.EnhancedPrompt,
		System:   enhancedPrompt.SystemPrompt,
		Language: request.Language,
		Provider: getPreferredProvider(request.Preferences.Providers),
		MaxTokens: 4000,
	}
	
	var generatedCode activities.LLMGenerationResult
	err = workflow.ExecuteActivity(ctx, activities.GenerateCodeActivity, generationRequest).Get(ctx, &generatedCode)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	// Update metrics
	result.Metrics.PromptTokens = generatedCode.PromptTokens
	result.Metrics.CompletionTokens = generatedCode.CompletionTokens
	result.Metrics.TotalTokens = generatedCode.TotalTokens
	result.Metrics.Provider = generatedCode.Provider
	result.Metrics.Model = generatedCode.Model

	// Stage 4: Validate generated code
	logger.Info("Stage 4: Validating code")
	validationRequest := types.ValidationRequest{
		Code:     generatedCode.Content,
		Language: request.Language,
		Type:     request.Type,
	}
	
	var validationResult types.ValidationResult
	err = workflow.ExecuteActivity(ctx, activities.ValidateCodeActivity, validationRequest).Get(ctx, &validationResult)
	if err != nil {
		logger.Warn("Code validation failed", "error", err)
	}

	// Stage 5: Generate tests if requested
	if request.Preferences.TestsRequired {
		logger.Info("Stage 5: Generating tests")
		testRequest := activities.TestGenerationRequest{
			Code:     generatedCode.Content,
			Language: request.Language,
			Framework: requirements.TestFramework,
		}
		
		var tests activities.TestGenerationResult
		err = workflow.ExecuteActivity(ctx, activities.GenerateTestsActivity, testRequest).Get(ctx, &tests)
		if err != nil {
			logger.Warn("Test generation failed", "error", err)
		} else {
			result.Tests = tests.TestCode
			result.Files = append(result.Files, types.GeneratedFile{
				Path:     tests.FilePath,
				Content:  tests.TestCode,
				Language: request.Language,
				Type:     "test",
			})
		}
	}

	// Stage 6: Generate documentation if requested
	if request.Preferences.Documentation {
		logger.Info("Stage 6: Generating documentation")
		docRequest := activities.DocumentationRequest{
			Code:     generatedCode.Content,
			Language: request.Language,
			Type:     request.Type,
		}
		
		var docs activities.DocumentationResult
		err = workflow.ExecuteActivity(ctx, activities.GenerateDocumentationActivity, docRequest).Get(ctx, &docs)
		if err != nil {
			logger.Warn("Documentation generation failed", "error", err)
		} else {
			result.Documentation = docs.Content
			result.Files = append(result.Files, types.GeneratedFile{
				Path:     "README.md",
				Content:  docs.Content,
				Language: "markdown",
				Type:     "doc",
			})
		}
	}

	// Stage 7: Structure files and organize output
	logger.Info("Stage 7: Organizing output")
	result.Code = generatedCode.Content
	result.Files = append(result.Files, types.GeneratedFile{
		Path:     requirements.MainFilePath,
		Content:  generatedCode.Content,
		Language: request.Language,
		Type:     "source",
	})
	
	// Add dependencies
	result.Dependencies = requirements.Dependencies

	// Calculate final metrics
	result.Success = validationResult.Valid || validationResult.Score > 70
	result.CompletedAt = workflow.Now(ctx)
	result.Metrics.Duration = result.CompletedAt.Sub(request.CreatedAt)
	result.Metrics.LLMCalls = 1 // Base generation
	if request.Preferences.TestsRequired {
		result.Metrics.LLMCalls++
	}
	if request.Preferences.Documentation {
		result.Metrics.LLMCalls++
	}

	// Calculate estimated cost
	result.Metrics.Cost = calculateCost(result.Metrics)

	logger.Info("Code generation workflow completed", 
		"requestID", request.ID,
		"success", result.Success,
		"filesGenerated", len(result.Files),
		"duration", result.Metrics.Duration)

	return result, nil
}

// Helper function to get preferred provider
func getPreferredProvider(providers []string) string {
	if len(providers) == 0 {
		return "azure" // Default
	}
	return providers[0]
}

// Helper function to calculate cost based on metrics
func calculateCost(metrics types.GenerationMetrics) float64 {
	// Rough cost estimation based on tokens
	// Azure GPT-4: ~$0.03 per 1K tokens
	// AWS Claude: ~$0.015 per 1K tokens
	costPerThousand := 0.03
	if metrics.Provider == "aws" {
		costPerThousand = 0.015
	}
	return float64(metrics.TotalTokens) / 1000.0 * costPerThousand
}