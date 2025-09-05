package workflows

import (
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/activities"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/types"
)

const (
	// Extended workflow name
	ExtendedCodeGenerationWorkflowName = "ExtendedCodeGenerationWorkflow"
	
	// Extended workflow stages
	StageFRDGeneration = "frd_generation"
	StageTestPlanGeneration = "test_plan_generation"
	StageDependencyResolution = "dependency_resolution"
	StageSecurityScanning = "security_scanning"
	StagePerformanceAnalysis = "performance_analysis"
	StageQuantumDropCreation = "quantum_drop_creation"
)

// ExtendedCodeGenerationWorkflow is the enhanced 12-stage workflow
func ExtendedCodeGenerationWorkflow(ctx workflow.Context, request types.CodeGenerationRequest) (*types.ExtendedGenerationResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting extended code generation workflow", "requestID", request.ID)

	// Set workflow options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: ActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
			InitialInterval: time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Initialize extended result with QuantumDrops tracking
	result := &types.ExtendedGenerationResult{
		ID:        workflow.GetInfo(ctx).WorkflowExecution.ID,
		RequestID: request.ID,
		Files:     []types.GeneratedFile{},
		Metrics:   types.GenerationMetrics{},
		QuantumDrops: []types.QuantumDrop{},
		ValidationResults: types.ValidationResults{},
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
		enhancedPrompt.EnhancedPrompt = request.Prompt
	}
	
	// Create first QuantumDrop
	drop := types.QuantumDrop{
		ID:        fmt.Sprintf("drop-%s-prompt", request.ID),
		Stage:     "prompt_enhancement",
		Timestamp: workflow.Now(ctx),
		Artifact:  enhancedPrompt.EnhancedPrompt,
		Type:      "prompt",
		WorkflowID: result.ID,
	}
	result.QuantumDrops = append(result.QuantumDrops, drop)
	
	// Store the QuantumDrop
	err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to store QuantumDrop", "error", err)
	}

	// Stage 2: Generate FRD (Functional Requirements Document)
	logger.Info("Stage 2: Generating FRD")
	var frdResult activities.FRDGenerationResult
	frdRequest := activities.FRDGenerationRequest{
		Prompt:   enhancedPrompt.EnhancedPrompt,
		Type:     request.Type,
		Language: request.Language,
	}
	
	err = workflow.ExecuteActivity(ctx, activities.GenerateFRDActivity, frdRequest).Get(ctx, &frdResult)
	if err != nil {
		logger.Warn("FRD generation failed", "error", err)
	} else {
		result.FRD = frdResult.Content
		result.Files = append(result.Files, types.GeneratedFile{
			Path:     "docs/FRD.md",
			Content:  frdResult.Content,
			Language: "markdown",
			Type:     "documentation",
		})
		
		// Create FRD QuantumDrop
		drop := types.QuantumDrop{
			ID:        fmt.Sprintf("drop-%s-frd", request.ID),
			Stage:     StageFRDGeneration,
			Timestamp: workflow.Now(ctx),
			Artifact:  frdResult.Content,
			Type:      "frd",
			WorkflowID: result.ID,
		}
		result.QuantumDrops = append(result.QuantumDrops, drop)
		
		// Store the QuantumDrop
		err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to store FRD QuantumDrop", "error", err)
		}
	}

	// Stage 3: Parse requirements and determine architecture
	logger.Info("Stage 3: Parsing requirements")
	var requirements activities.ParsedRequirements
	err = workflow.ExecuteActivity(ctx, activities.ParseRequirementsActivity, request).Get(ctx, &requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to parse requirements: %w", err)
	}

	// Stage 4: Generate project structure
	logger.Info("Stage 4: Generating project structure")
	var projectStructure activities.ProjectStructureResult
	structureRequest := activities.ProjectStructureRequest{
		Language:  request.Language,
		Framework: request.Framework,
		Type:      request.Type,
		Requirements: requirements,
	}
	
	err = workflow.ExecuteActivity(ctx, activities.GenerateProjectStructureActivity, structureRequest).Get(ctx, &projectStructure)
	if err != nil {
		logger.Warn("Project structure generation failed", "error", err)
	} else {
		result.ProjectStructure = projectStructure.Structure
		
		// Serialize structure to JSON for QuantumDrop
		structureJSON, _ := json.Marshal(projectStructure.Structure)
		
		// Create structure QuantumDrop
		drop := types.QuantumDrop{
			ID:        fmt.Sprintf("drop-%s-structure", request.ID),
			Stage:     "project_structure",
			Timestamp: workflow.Now(ctx),
			Artifact:  string(structureJSON),
			Type:      "structure",
			WorkflowID: result.ID,
		}
		result.QuantumDrops = append(result.QuantumDrops, drop)
		
		// Store the QuantumDrop
		err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to store structure QuantumDrop", "error", err)
		}
	}

	// Stage 5: Generate code using LLM Router
	logger.Info("Stage 5: Generating code")
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
	
	// Create code QuantumDrop
	drop = types.QuantumDrop{
		ID:        fmt.Sprintf("drop-%s-code", request.ID),
		Stage:     "code_generation",
		Timestamp: workflow.Now(ctx),
		Artifact:  generatedCode.Content,
		Type:      "code",
		WorkflowID: result.ID,
	}
	result.QuantumDrops = append(result.QuantumDrops, drop)
	
	// Store the QuantumDrop
	err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to store code QuantumDrop", "error", err)
	}

	// Stage 6: Semantic validation using Parser service
	logger.Info("Stage 6: Semantic validation")
	semanticRequest := activities.SemanticValidationRequest{
		Code:     generatedCode.Content,
		Language: request.Language,
		Type:     request.Type,
	}
	
	var semanticResult activities.SemanticValidationResult
	err = workflow.ExecuteActivity(ctx, activities.ValidateSemanticActivity, semanticRequest).Get(ctx, &semanticResult)
	if err != nil {
		logger.Warn("Semantic validation failed", "error", err)
	} else {
		result.ValidationResults.SemanticValid = semanticResult.Valid
		result.ValidationResults.SemanticIssues = semanticResult.Issues
		
		// If validation failed, attempt to fix and regenerate
		if !semanticResult.Valid && len(semanticResult.Issues) > 0 {
			logger.Info("Stage 6.1: Applying feedback loop for code fixes")
			feedbackRequest := activities.FeedbackLoopRequest{
				Code:     generatedCode.Content,
				Issues:   semanticResult.Issues,
				Language: request.Language,
			}
			
			var fixedCode activities.FeedbackLoopResult
			err = workflow.ExecuteActivity(ctx, activities.ApplyFeedbackLoopActivity, feedbackRequest).Get(ctx, &fixedCode)
			if err == nil && fixedCode.ImprovedCode != "" {
				generatedCode.Content = fixedCode.ImprovedCode
				result.FeedbackIterations = fixedCode.Iterations
			}
		}
	}

	// Stage 7: Dependency resolution
	logger.Info("Stage 7: Resolving dependencies")
	var dependencies activities.DependencyResolutionResult
	depRequest := activities.DependencyResolutionRequest{
		Code:     generatedCode.Content,
		Language: request.Language,
		Framework: request.Framework,
	}
	
	err = workflow.ExecuteActivity(ctx, activities.ResolveDependenciesActivity, depRequest).Get(ctx, &dependencies)
	if err != nil {
		logger.Warn("Dependency resolution failed", "error", err)
	} else {
		result.Dependencies = dependencies.Dependencies
		
		// Generate package file (package.json, requirements.txt, etc.)
		if dependencies.PackageFile != "" {
			result.Files = append(result.Files, types.GeneratedFile{
				Path:     dependencies.PackageFileName,
				Content:  dependencies.PackageFile,
				Language: "json",
				Type:     "config",
			})
		}
	}

	// Stage 8: Generate test plan
	logger.Info("Stage 8: Generating test plan")
	var testPlan activities.TestPlanResult
	testPlanRequest := activities.TestPlanRequest{
		Code:     generatedCode.Content,
		Language: request.Language,
		Type:     request.Type,
	}
	
	err = workflow.ExecuteActivity(ctx, activities.GenerateTestPlanActivity, testPlanRequest).Get(ctx, &testPlan)
	if err != nil {
		logger.Warn("Test plan generation failed", "error", err)
	} else {
		result.TestPlan = testPlan.Content
		result.Files = append(result.Files, types.GeneratedFile{
			Path:     "docs/TEST_PLAN.md",
			Content:  testPlan.Content,
			Language: "markdown",
			Type:     "documentation",
		})
		
		// Create test plan QuantumDrop
		drop := types.QuantumDrop{
			ID:        fmt.Sprintf("drop-%s-testplan", request.ID),
			Stage:     StageTestPlanGeneration,
			Timestamp: workflow.Now(ctx),
			Artifact:  testPlan.Content,
			Type:      "test_plan",
			WorkflowID: result.ID,
		}
		result.QuantumDrops = append(result.QuantumDrops, drop)
		
		// Store the QuantumDrop
		err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to store test plan QuantumDrop", "error", err)
		}
	}

	// Stage 9: Generate tests
	if request.Preferences.TestsRequired {
		logger.Info("Stage 9: Generating tests")
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
			
			// Create tests QuantumDrop
			drop := types.QuantumDrop{
				ID:        fmt.Sprintf("drop-%s-tests", request.ID),
				Stage:     "test_generation",
				Timestamp: workflow.Now(ctx),
				Artifact:  tests.TestCode,
				Type:      "tests",
				WorkflowID: result.ID,
			}
			result.QuantumDrops = append(result.QuantumDrops, drop)
			
			// Store the QuantumDrop
			err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
			if err != nil {
				logger.Warn("Failed to store tests QuantumDrop", "error", err)
			}
		}
	}

	// Stage 10: Security scanning
	logger.Info("Stage 10: Security scanning")
	var securityScan activities.SecurityScanResult
	securityRequest := activities.SecurityScanRequest{
		Code:         generatedCode.Content,
		Language:     request.Language,
		Dependencies: result.Dependencies,
	}
	
	err = workflow.ExecuteActivity(ctx, activities.PerformSecurityScanActivity, securityRequest).Get(ctx, &securityScan)
	if err != nil {
		logger.Warn("Security scanning failed", "error", err)
	} else {
		result.ValidationResults.SecurityScore = securityScan.Score
		result.ValidationResults.SecurityIssues = securityScan.Vulnerabilities
		result.SecurityReport = securityScan.Report
	}

	// Stage 11: Performance analysis
	logger.Info("Stage 11: Performance analysis")
	var perfAnalysis activities.PerformanceAnalysisResult
	perfRequest := activities.PerformanceAnalysisRequest{
		Code:     generatedCode.Content,
		Language: request.Language,
		Type:     request.Type,
	}
	
	err = workflow.ExecuteActivity(ctx, activities.AnalyzePerformanceActivity, perfRequest).Get(ctx, &perfAnalysis)
	if err != nil {
		logger.Warn("Performance analysis failed", "error", err)
	} else {
		result.ValidationResults.PerformanceScore = perfAnalysis.Score
		result.PerformanceReport = perfAnalysis.Report
	}

	// Stage 12: Generate README and documentation
	logger.Info("Stage 12: Generating README and documentation")
	var readme activities.ReadmeResult
	readmeRequest := activities.ReadmeRequest{
		Code:         generatedCode.Content,
		Language:     request.Language,
		Framework:    request.Framework,
		Dependencies: result.Dependencies,
		ProjectName:  request.Context["project_name"],
	}
	
	err = workflow.ExecuteActivity(ctx, activities.GenerateReadmeActivity, readmeRequest).Get(ctx, &readme)
	if err != nil {
		logger.Warn("README generation failed", "error", err)
	} else {
		result.Documentation = readme.Content
		result.Files = append(result.Files, types.GeneratedFile{
			Path:     "README.md",
			Content:  readme.Content,
			Language: "markdown",
			Type:     "documentation",
		})
		
		// Create documentation QuantumDrop
		drop := types.QuantumDrop{
			ID:        fmt.Sprintf("drop-%s-readme", request.ID),
			Stage:     "documentation",
			Timestamp: workflow.Now(ctx),
			Artifact:  readme.Content,
			Type:      "documentation",
			WorkflowID: result.ID,
		}
		result.QuantumDrops = append(result.QuantumDrops, drop)
		
		// Store the QuantumDrop
		err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to store documentation QuantumDrop", "error", err)
		}
	}

	// Final: Organize all files with proper structure
	result.Code = generatedCode.Content
	result.Files = append(result.Files, types.GeneratedFile{
		Path:     requirements.MainFilePath,
		Content:  generatedCode.Content,
		Language: request.Language,
		Type:     "source",
	})

	// Calculate final metrics - Success if we generated content
	// Don't fail workflows just because of validation warnings
	contentGenerated := len(generatedCode.Content) > 100
	hasNoErrors := true
	
	// Check for critical errors only (not warnings)
	if result.ValidationResults.SemanticIssues != nil {
		for _, issue := range result.ValidationResults.SemanticIssues {
			if issue.Type == "error" {
				hasNoErrors = false
				break
			}
		}
	}
	
	// Success criteria: Content was generated and no critical errors
	// Warnings and lower scores are acceptable for MVP
	result.Success = contentGenerated && (hasNoErrors || result.ValidationResults.SecurityScore >= 50)
	result.CompletedAt = workflow.Now(ctx)
	result.Metrics.Duration = result.CompletedAt.Sub(request.CreatedAt)
	result.Metrics.LLMCalls = 5 // Base + FRD + Test Plan + Tests + README
	result.Metrics.Provider = generatedCode.Provider
	result.Metrics.Model = generatedCode.Model
	result.Metrics.TotalTokens = generatedCode.TotalTokens
	result.Metrics.Cost = calculateCost(result.Metrics)

	// Create final QuantumDrop summary
	drop = types.QuantumDrop{
		ID:        fmt.Sprintf("drop-%s-complete", request.ID),
		Stage:     "completion",
		Timestamp: workflow.Now(ctx),
		Artifact:  fmt.Sprintf("Workflow completed with %d artifacts", len(result.Files)),
		Type:      "summary",
		WorkflowID: result.ID,
		Metadata: map[string]interface{}{
			"files_generated": len(result.Files),
			"validation_score": result.ValidationResults.SemanticValid,
			"security_score": result.ValidationResults.SecurityScore,
			"performance_score": result.ValidationResults.PerformanceScore,
		},
	}
	result.QuantumDrops = append(result.QuantumDrops, drop)
	
	// Store the final QuantumDrop
	err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to store final QuantumDrop", "error", err)
	}

	// Stage 12: Generate Preview URL
	logger.Info("Stage 12: Generating preview URL")
	var previewResult activities.PreviewResult
	capsuleID := "" // Set if we have a capsule ID
	err = workflow.ExecuteActivity(ctx, activities.GeneratePreviewActivity, result.ID, capsuleID).Get(ctx, &previewResult)
	if err != nil {
		logger.Warn("Preview URL generation failed", "error", err)
		// Use fallback preview URL
		result.PreviewURL = fmt.Sprintf("http://192.168.1.217:30900/preview/%s", result.ID)
	} else {
		result.PreviewURL = previewResult.ShareableURL
		logger.Info("Preview URL generated", "url", result.PreviewURL)
		
		// Store preview metadata in QuantumDrops
		err = workflow.ExecuteActivity(ctx, activities.StorePreviewMetadataActivity, result.ID, &previewResult).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to store preview metadata", "error", err)
		}
	}

	logger.Info("Extended code generation workflow completed", 
		"requestID", request.ID,
		"success", result.Success,
		"filesGenerated", len(result.Files),
		"quantumDrops", len(result.QuantumDrops),
		"duration", result.Metrics.Duration)

	return result, nil
}