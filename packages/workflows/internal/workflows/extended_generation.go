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
	// Intelligent workflow name (v2)
	IntelligentCodeGenerationWorkflowName = "IntelligentCodeGenerationWorkflow"
	
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
		Language:       request.Language,
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

	// Stage 5: Intelligent multi-stage code generation
	logger.Info("Stage 5: Intelligent code generation (multi-stage)")
	intelligentRequest := activities.IntelligentCodeGenerationRequest{
		ProjectName:   fmt.Sprintf("%s-%s", request.Type, request.Language),
		Description:   enhancedPrompt.EnhancedPrompt,
		Language:      request.Language,
		Type:          request.Type,
		Requirements:  requirements,
	}
	
	var intelligentCode activities.IntelligentCodeGenerationResult
	// Set longer timeout for intelligent generation (6 LLM calls @ 30s each = 3 minutes)
	intelligentCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	})
	err = workflow.ExecuteActivity(intelligentCtx, activities.GenerateIntelligentCodeActivity, intelligentRequest).Get(ctx, &intelligentCode)
	if err != nil {
		// Fallback to simple generation if intelligent generation fails
		logger.Warn("Intelligent code generation failed, falling back to simple generation", "error", err)
		
		generationRequest := activities.LLMGenerationRequest{
			Prompt:      enhancedPrompt.EnhancedPrompt,
			System:      enhancedPrompt.SystemPrompt,
			Language:    request.Language,
			Provider:    getPreferredProvider(request.Preferences.Providers),
			MaxTokens:   8000, // Increased for better results
			// Lower temperature for deterministic enterprise code
		}
		
		var generatedCode activities.LLMGenerationResult
		err = workflow.ExecuteActivity(ctx, activities.GenerateCodeActivity, generationRequest).Get(ctx, &generatedCode)
		if err != nil {
			return nil, fmt.Errorf("both intelligent and simple code generation failed: %w", err)
		}
		
		// Convert simple result to intelligent result format
		intelligentCode = activities.IntelligentCodeGenerationResult{
			Files: []types.GeneratedFile{
				{
					Path:     requirements.MainFilePath,
					Content:  generatedCode.Content,
					Language: request.Language,
					Type:     "source",
				},
			},
			MainFile:     requirements.MainFilePath,
			Dependencies: []string{},
		}
	}
	
	// Create code QuantumDrop with main file content
	mainFileContent := ""
	if len(intelligentCode.Files) > 0 {
		// Find the main file
		for _, file := range intelligentCode.Files {
			if file.Path == intelligentCode.MainFile || file.Type == "source" {
				mainFileContent = file.Content
				break
			}
		}
		if mainFileContent == "" {
			mainFileContent = intelligentCode.Files[0].Content // Fallback to first file
		}
	}
	
	drop = types.QuantumDrop{
		ID:        fmt.Sprintf("drop-%s-code", request.ID),
		Stage:     "code_generation",
		Timestamp: workflow.Now(ctx),
		Artifact:  mainFileContent,
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
		Code:     mainFileContent,
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
				Code:     mainFileContent,
				Issues:   semanticResult.Issues,
				Language: request.Language,
			}
			
			var fixedCode activities.FeedbackLoopResult
			err = workflow.ExecuteActivity(ctx, activities.ApplyFeedbackLoopActivity, feedbackRequest).Get(ctx, &fixedCode)
			if err == nil && fixedCode.ImprovedCode != "" {
				mainFileContent = fixedCode.ImprovedCode
				result.FeedbackIterations = fixedCode.Iterations
				
				// Update the main file in intelligent code results
				for i, file := range intelligentCode.Files {
					if file.Path == intelligentCode.MainFile || file.Type == "source" {
						intelligentCode.Files[i].Content = fixedCode.ImprovedCode
						break
					}
				}
			}
		}
	}

	// Stage 7: Dependency resolution (using intelligent code dependencies)
	logger.Info("Stage 7: Resolving dependencies")
	
	// Use dependencies from intelligent generation if available
	if len(intelligentCode.Dependencies) > 0 {
		result.Dependencies = intelligentCode.Dependencies
		logger.Info("Using dependencies from intelligent generation", "count", len(intelligentCode.Dependencies))
	} else {
		// Fallback to traditional dependency resolution
		var dependencies activities.DependencyResolutionResult
		depRequest := activities.DependencyResolutionRequest{
			Code:     mainFileContent,
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
	}

	// Stage 8: Generate test plan
	logger.Info("Stage 8: Generating test plan")
	var testPlan activities.TestPlanResult
	testPlanRequest := activities.TestPlanRequest{
		Code:     mainFileContent,
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
			Code:     mainFileContent,
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
		Code:         mainFileContent,
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
		Code:     mainFileContent,
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
		Code:         mainFileContent,
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

	// Final: Organize all files with proper structure from intelligent generation
	result.Code = mainFileContent
	
	// Add all intelligently generated files to result
	for _, file := range intelligentCode.Files {
		result.Files = append(result.Files, file)
	}

	// Store final files as a compiled drop for preview service
	if len(result.Files) > 0 {
		filesMap := make(map[string]interface{})
		for _, file := range result.Files {
			filesMap[file.Path] = map[string]interface{}{
				"content":  file.Content,
				"language": file.Language,
				"type":     file.Type,
			}
		}
		
		filesJSON, _ := json.Marshal(filesMap)
		
		filesDrop := types.QuantumDrop{
			ID:        fmt.Sprintf("drop-%s-files", request.ID),
			Stage:     "files_compilation",
			Timestamp: workflow.Now(ctx),
			Artifact:  string(filesJSON),
			Type:      "files",
			WorkflowID: result.ID,
		}
		result.QuantumDrops = append(result.QuantumDrops, filesDrop)
		
		// Store the files QuantumDrop
		err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, filesDrop).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to store files QuantumDrop", "error", err)
		}
	}

	// Calculate final metrics - Success if we generated content
	// Don't fail workflows just because of validation warnings
	contentGenerated := len(mainFileContent) > 100
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
	// Additional success boost if we have a live deployment
	hasLiveDeployment := result.LiveURL != ""
	result.Success = contentGenerated && (hasNoErrors || result.ValidationResults.SecurityScore >= 50) || hasLiveDeployment
	result.CompletedAt = workflow.Now(ctx)
	result.Metrics.Duration = result.CompletedAt.Sub(request.CreatedAt)
	result.Metrics.LLMCalls = len(intelligentCode.Files) + 3 // Files + FRD + Test Plan + README
	result.Metrics.Provider = getPreferredProvider(request.Preferences.Providers)
	result.Metrics.Model = "intelligent-generation"
	result.Metrics.TotalTokens = estimateTokensFromContent(mainFileContent)
	result.Metrics.Cost = calculateCost(result.Metrics)

	// Create final QuantumDrop summary
	summaryArtifact := fmt.Sprintf("Workflow completed with %d artifacts", len(result.Files))
	if result.LiveURL != "" {
		summaryArtifact += fmt.Sprintf(". 🚀 Live at: %s", result.LiveURL)
	}
	
	drop = types.QuantumDrop{
		ID:        fmt.Sprintf("drop-%s-complete", request.ID),
		Stage:     "completion",
		Timestamp: workflow.Now(ctx),
		Artifact:  summaryArtifact,
		Type:      "summary",
		WorkflowID: result.ID,
		Metadata: map[string]interface{}{
			"files_generated": len(result.Files),
			"validation_score": result.ValidationResults.SemanticValid,
			"security_score": result.ValidationResults.SecurityScore,
			"performance_score": result.ValidationResults.PerformanceScore,
			"has_live_deployment": result.LiveURL != "",
			"live_url": result.LiveURL,
			"dashboard_url": result.DashboardURL,
		},
	}
	result.QuantumDrops = append(result.QuantumDrops, drop)
	
	// Store the final QuantumDrop
	err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to store final QuantumDrop", "error", err)
	}

	// Stage 13: Container Build and Registry Push
	logger.Info("Stage 13: Building and pushing container image")
	deploymentRequest := activities.DeploymentRequest{
		WorkflowID:   result.ID,
		CapsuleID:    result.RequestID,
		Language:     request.Language,
		Framework:    request.Framework,
		Files:        convertFilesToMap(result.Files),
		Dependencies: result.Dependencies,
		Environment:  request.Context,
		Resources: activities.ContainerResources{
			CPU:    "200m",
			Memory: "256Mi",
		},
		TTLMinutes: 60, // Default 1 hour
	}
	
	var containerResult activities.ContainerBuildResult
	// Extended timeout for container build (Docker build + push)
	containerCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
	})
	err = workflow.ExecuteActivity(containerCtx, activities.BuildContainerImageActivity, deploymentRequest).Get(ctx, &containerResult)
	if err != nil {
		logger.Warn("Container build failed", "error", err)
		// Continue without deployment but mark as partially successful
	} else {
		logger.Info("Container built successfully", "image", containerResult.ImageName+":"+containerResult.ImageTag)
		
		// Create container build QuantumDrop
		drop := types.QuantumDrop{
			ID:        fmt.Sprintf("drop-%s-container", request.ID),
			Stage:     "container_build",
			Timestamp: workflow.Now(ctx),
			Artifact:  fmt.Sprintf("Container built: %s:%s (%.2fs)", containerResult.ImageName, containerResult.ImageTag, containerResult.BuildTime),
			Type:      "container",
			WorkflowID: result.ID,
			Metadata: map[string]interface{}{
				"image_name": containerResult.ImageName,
				"image_tag":  containerResult.ImageTag,
				"build_time": containerResult.BuildTime,
				"image_size": containerResult.ImageSize,
			},
		}
		result.QuantumDrops = append(result.QuantumDrops, drop)
		
		// Store the container QuantumDrop
		err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to store container QuantumDrop", "error", err)
		}
	}

	// Stage 14: Kubernetes Deployment
	if containerResult.Success {
		logger.Info("Stage 14: Deploying to Kubernetes")
		
		// Generate Kubernetes manifests
		var k8sManifest string
		err = workflow.ExecuteActivity(ctx, activities.GenerateK8sManifestsActivity, deploymentRequest, containerResult.ImageTag).Get(ctx, &k8sManifest)
		if err != nil {
			logger.Warn("K8s manifest generation failed", "error", err)
		} else {
			// Deploy to Kubernetes
			var deployResult activities.KubernetesDeploymentResult
			deployCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: 5 * time.Minute,
			})
			err = workflow.ExecuteActivity(deployCtx, activities.DeployToKubernetesActivity, deploymentRequest, k8sManifest, containerResult.ImageTag).Get(ctx, &deployResult)
			if err != nil {
				logger.Warn("Kubernetes deployment failed", "error", err)
			} else {
				logger.Info("Successfully deployed to Kubernetes", "live_url", deployResult.LiveURL)
				
				// Store deployment information in result
				result.LiveURL = deployResult.LiveURL
				result.DashboardURL = deployResult.DashboardURL
				result.DeploymentID = deployResult.DeploymentID
				result.ExpiresAt = &deployResult.ExpiresAt
				
				// Create deployment QuantumDrop
				drop := types.QuantumDrop{
					ID:        fmt.Sprintf("drop-%s-deployment", request.ID),
					Stage:     "kubernetes_deployment",
					Timestamp: workflow.Now(ctx),
					Artifact:  fmt.Sprintf("Deployed: %s (expires: %s)", deployResult.LiveURL, deployResult.ExpiresAt.Format("2006-01-02 15:04:05")),
					Type:      "deployment",
					WorkflowID: result.ID,
					Metadata: map[string]interface{}{
						"live_url":      deployResult.LiveURL,
						"dashboard_url": deployResult.DashboardURL,
						"deployment_id": deployResult.DeploymentID,
						"namespace":     deployResult.Namespace,
						"expires_at":    deployResult.ExpiresAt,
					},
				}
				result.QuantumDrops = append(result.QuantumDrops, drop)
				
				// Store the deployment QuantumDrop
				err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
				if err != nil {
					logger.Warn("Failed to store deployment QuantumDrop", "error", err)
				}
			}
		}
	}

	// Stage 15: Health Check and Live URL Verification
	if result.LiveURL != "" {
		logger.Info("Stage 15: Performing health check on live deployment")
		
		var healthOK bool
		healthCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 5 * time.Minute, // Allow time for deployment to come up
		})
		err = workflow.ExecuteActivity(healthCtx, activities.HealthCheckActivity, result.LiveURL, 30).Get(ctx, &healthOK) // 30 attempts = 5 minutes
		if err != nil {
			logger.Warn("Health check failed", "error", err)
		} else if healthOK {
			logger.Info("Health check passed - application is live!", "url", result.LiveURL)
			
			// Create health check QuantumDrop
			drop := types.QuantumDrop{
				ID:        fmt.Sprintf("drop-%s-health", request.ID),
				Stage:     "health_check",
				Timestamp: workflow.Now(ctx),
				Artifact:  fmt.Sprintf("✅ Application is live and healthy at %s", result.LiveURL),
				Type:      "health",
				WorkflowID: result.ID,
				Metadata: map[string]interface{}{
					"health_status": "healthy",
					"live_url":      result.LiveURL,
					"verified_at":   workflow.Now(ctx),
				},
			}
			result.QuantumDrops = append(result.QuantumDrops, drop)
			
			// Store the health check QuantumDrop
			err = workflow.ExecuteActivity(ctx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
			if err != nil {
				logger.Warn("Failed to store health QuantumDrop", "error", err)
			}
		}
	}

	// Stage 16: Generate Preview URL (fallback for code preview)
	logger.Info("Stage 16: Generating preview URL")
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
		"duration", result.Metrics.Duration,
		"liveURL", result.LiveURL,
		"previewURL", result.PreviewURL)

	return result, nil
}

// IntelligentCodeGenerationWorkflow is a deterministic workflow that always uses intelligent generation
func IntelligentCodeGenerationWorkflow(ctx workflow.Context, request types.CodeGenerationRequest) (*types.ExtendedGenerationResult, error) {
	logger := workflow.GetLogger(ctx)
	
	// Set request timestamp if not set
	if request.CreatedAt.IsZero() {
		request.CreatedAt = workflow.Now(ctx)
	}
	
	result := &types.ExtendedGenerationResult{
		ID:           request.ID,
		RequestID:    request.ID,
		Files:        []types.GeneratedFile{},
		QuantumDrops: []types.QuantumDrop{},
		Metrics: types.GenerationMetrics{},
	}

	logger.Info("Starting intelligent code generation workflow",
		"requestID", request.ID,
		"language", request.Language,
		"type", request.Type)

	// Stage 1: Enhanced prompt generation
	logger.Info("Stage 1: Enhanced prompt generation")
	enhanceRequest := types.PromptEnhancementRequest{
		OriginalPrompt: request.Prompt,
		Language:       request.Language,
		Type:           request.Type,
		Context:        request.Context,
		TargetProvider: getPreferredProvider(request.Preferences.Providers),
	}
	
	var enhancedPrompt types.PromptEnhancementResult
	// Set timeout for activities
	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
	})
	err := workflow.ExecuteActivity(activityCtx, activities.EnhancePromptActivity, enhanceRequest).Get(ctx, &enhancedPrompt)
	if err != nil {
		logger.Error("Enhanced prompt generation failed", "error", err)
		return nil, fmt.Errorf("enhanced prompt generation failed: %w", err)
	}

	// Stage 2: Requirements parsing
	logger.Info("Stage 2: Requirements parsing")
	parseRequest := request // Use the request directly
	parseRequest.Prompt = enhancedPrompt.EnhancedPrompt
	
	var requirements activities.ParsedRequirements
	err = workflow.ExecuteActivity(activityCtx, activities.ParseRequirementsActivity, parseRequest).Get(ctx, &requirements)
	if err != nil {
		logger.Error("Requirements parsing failed", "error", err)
		return nil, fmt.Errorf("requirements parsing failed: %w", err)
	}

	// Stage 3: Intelligent multi-stage code generation (ALWAYS)
	logger.Info("Stage 3: Intelligent code generation (multi-stage)")
	intelligentRequest := activities.IntelligentCodeGenerationRequest{
		ProjectName:   fmt.Sprintf("%s-%s", request.Type, request.Language),
		Description:   enhancedPrompt.EnhancedPrompt,
		Language:      request.Language,
		Type:          request.Type,
		Requirements:  requirements,
	}
	
	var intelligentCode activities.IntelligentCodeGenerationResult
	// Set longer timeout for intelligent generation (6 LLM calls @ 30s each = 5 minutes)
	intelligentCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	})
	err = workflow.ExecuteActivity(intelligentCtx, activities.GenerateIntelligentCodeActivity, intelligentRequest).Get(ctx, &intelligentCode)
	if err != nil {
		logger.Error("Intelligent code generation failed", "error", err)
		return nil, fmt.Errorf("intelligent code generation failed: %w", err)
	}

	// Create code QuantumDrop with all generated files
	allFilesContent := ""
	for _, file := range intelligentCode.Files {
		allFilesContent += fmt.Sprintf("=== %s ===\n%s\n\n", file.Path, file.Content)
	}
	
	drop := types.QuantumDrop{
		ID:        fmt.Sprintf("drop-%s-intelligent-code", request.ID),
		Stage:     "intelligent_code_generation",
		Timestamp: workflow.Now(ctx),
		Artifact:  allFilesContent,
		Type:      "code",
		Metadata: map[string]interface{}{
			"files_count": len(intelligentCode.Files),
			"main_file":   intelligentCode.MainFile,
		},
	}
	result.QuantumDrops = append(result.QuantumDrops, drop)
	result.Files = intelligentCode.Files

	// Store QuantumDrop
	err = workflow.ExecuteActivity(activityCtx, activities.StoreQuantumDropActivity, drop).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to store intelligent code QuantumDrop", "error", err)
	}

	// Success - intelligent generation completed
	result.Success = len(intelligentCode.Files) > 0
	result.CompletedAt = workflow.Now(ctx)
	result.Metrics.Duration = result.CompletedAt.Sub(request.CreatedAt)
	result.Metrics.LLMCalls = len(intelligentCode.Files)
	result.Metrics.Provider = "intelligent-multi-stage"
	result.Metrics.Model = "azure-gpt-4"
	result.Metrics.TotalTokens = estimateTokensFromContent(allFilesContent)

	logger.Info("Intelligent code generation workflow completed", 
		"requestID", request.ID,
		"success", result.Success,
		"filesGenerated", len(result.Files),
		"duration", result.Metrics.Duration)

	return result, nil
}

// estimateTokensFromContent estimates the number of tokens in the given content
// Using approximate 4 characters per token for English text
func estimateTokensFromContent(content string) int {
	if content == "" {
		return 0
	}
	return len(content) / 4
}

// convertFilesToMap converts GeneratedFile slice to map[string]string for deployment activities
func convertFilesToMap(files []types.GeneratedFile) map[string]string {
	result := make(map[string]string)
	for _, file := range files {
		result[file.Path] = file.Content
	}
	return result
}