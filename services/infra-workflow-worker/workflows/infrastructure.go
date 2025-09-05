package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	
	"github.com/quantumlayer/infra-workflow-worker/activities"
	"github.com/quantumlayer/infra-workflow-worker/types"
)

// InfrastructureGenerationWorkflow orchestrates the complete infrastructure generation process
func InfrastructureGenerationWorkflow(ctx workflow.Context, request types.InfrastructureRequest) (*types.InfrastructureResult, error) {
	// Configure activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Infrastructure Generation Workflow", 
		"workflow_id", request.WorkflowID,
		"provider", request.Provider)

	result := &types.InfrastructureResult{
		WorkflowID: request.WorkflowID,
		Status:     "started",
		StartTime:  workflow.Now(ctx),
		Stages:     make(map[string]types.StageResult),
	}

	// Stage 1: Analyze Code Requirements
	logger.Info("Stage 1: Analyzing code for infrastructure requirements")
	var codeAnalysis types.CodeAnalysis
	err := workflow.ExecuteActivity(ctx, activities.AnalyzeCodeActivity, request).Get(ctx, &codeAnalysis)
	if err != nil {
		return nil, fmt.Errorf("code analysis failed: %w", err)
	}
	result.Stages["code_analysis"] = types.StageResult{
		Name:      "Code Analysis",
		Status:    "completed",
		StartTime: workflow.Now(ctx),
		Output:    codeAnalysis,
	}

	// Stage 2: Generate Infrastructure as Code
	logger.Info("Stage 2: Generating infrastructure code")
	infraRequest := types.GenerateInfraRequest{
		WorkflowID:   request.WorkflowID,
		Provider:     request.Provider,
		Resources:    codeAnalysis.RequiredResources,
		Compliance:   request.Compliance,
		Environment:  request.Environment,
		Requirements: codeAnalysis.Requirements,
	}
	
	var infraCode types.InfrastructureCode
	err = workflow.ExecuteActivity(ctx, activities.GenerateInfrastructureActivity, infraRequest).Get(ctx, &infraCode)
	if err != nil {
		return nil, fmt.Errorf("infrastructure generation failed: %w", err)
	}
	result.Stages["infrastructure_generation"] = types.StageResult{
		Name:      "Infrastructure Generation",
		Status:    "completed",
		StartTime: workflow.Now(ctx),
		Output:    infraCode,
	}

	// Stage 3: Build Golden Images (if enabled)
	if request.EnableGoldenImages {
		logger.Info("Stage 3: Building golden images")
		imageRequest := types.GoldenImageRequest{
			BaseOS:     codeAnalysis.RecommendedOS,
			Packages:   codeAnalysis.RequiredPackages,
			Hardening:  "CIS",
			Compliance: request.Compliance,
		}
		
		var goldenImage types.GoldenImageResult
		err = workflow.ExecuteActivity(ctx, activities.BuildGoldenImageActivity, imageRequest).Get(ctx, &goldenImage)
		if err != nil {
			logger.Warn("Golden image build failed, continuing", "error", err)
		} else {
			result.Stages["golden_images"] = types.StageResult{
				Name:      "Golden Image Building",
				Status:    "completed",
				StartTime: workflow.Now(ctx),
				Output:    goldenImage,
			}
		}
	}

	// Stage 4: Generate SOPs (if enabled)
	if request.EnableSOP {
		logger.Info("Stage 4: Generating Standard Operating Procedures")
		sopRequest := types.SOPRequest{
			InfrastructureType: infraCode.Framework,
			Operations: []string{
				"deployment",
				"scaling",
				"backup",
				"disaster-recovery",
				"incident-response",
			},
		}
		
		var sopResult types.SOPResult
		err = workflow.ExecuteActivity(ctx, activities.GenerateSOPActivity, sopRequest).Get(ctx, &sopResult)
		if err != nil {
			logger.Warn("SOP generation failed, continuing", "error", err)
		} else {
			result.Stages["sop_generation"] = types.StageResult{
				Name:      "SOP Generation",
				Status:    "completed",
				StartTime: workflow.Now(ctx),
				Output:    sopResult,
			}
		}
	}

	// Stage 5: Validate Compliance
	if len(request.Compliance) > 0 {
		logger.Info("Stage 5: Validating compliance requirements")
		complianceRequest := types.ComplianceRequest{
			Code:       infraCode.Code,
			Frameworks: request.Compliance,
		}
		
		var complianceResult types.ComplianceResult
		err = workflow.ExecuteActivity(ctx, activities.ValidateComplianceActivity, complianceRequest).Get(ctx, &complianceResult)
		if err != nil {
			logger.Warn("Compliance validation failed", "error", err)
		} else {
			result.Stages["compliance_validation"] = types.StageResult{
				Name:      "Compliance Validation",
				Status:    "completed",
				StartTime: workflow.Now(ctx),
				Output:    complianceResult,
			}
		}
	}

	// Stage 6: Estimate Costs
	logger.Info("Stage 6: Estimating infrastructure costs")
	costRequest := types.CostRequest{
		Provider:  request.Provider,
		Resources: codeAnalysis.RequiredResources,
	}
	
	var costEstimate types.CostEstimate
	err = workflow.ExecuteActivity(ctx, activities.EstimateCostActivity, costRequest).Get(ctx, &costEstimate)
	if err != nil {
		logger.Warn("Cost estimation failed", "error", err)
	} else {
		result.Stages["cost_estimation"] = types.StageResult{
			Name:      "Cost Estimation",
			Status:    "completed",
			StartTime: workflow.Now(ctx),
			Output:    costEstimate,
		}
		result.EstimatedCost = &costEstimate
	}

	// Stage 7: Store Infrastructure Drop
	logger.Info("Stage 7: Storing infrastructure artifacts")
	dropRequest := types.InfraDropRequest{
		WorkflowID: request.WorkflowID,
		Stage:      "infrastructure",
		Type:       "infrastructure_code",
		Artifact:   infraCode,
	}
	
	err = workflow.ExecuteActivity(ctx, activities.StoreInfraDropActivity, dropRequest).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to store infrastructure drop", "error", err)
	}

	// Stage 8: Deploy Infrastructure (if auto-deploy enabled)
	if request.AutoDeploy {
		logger.Info("Stage 8: Deploying infrastructure")
		deployRequest := types.DeployRequest{
			WorkflowID:   request.WorkflowID,
			Provider:     request.Provider,
			Environment:  request.Environment,
			Code:         infraCode.Code,
			DryRun:       request.DryRun,
		}
		
		var deployResult types.DeploymentResult
		err = workflow.ExecuteActivity(ctx, activities.DeployInfrastructureActivity, deployRequest).Get(ctx, &deployResult)
		if err != nil {
			logger.Error("Deployment failed", "error", err)
			result.Status = "deployment_failed"
		} else {
			result.Stages["deployment"] = types.StageResult{
				Name:      "Infrastructure Deployment",
				Status:    "completed",
				StartTime: workflow.Now(ctx),
				Output:    deployResult,
			}
			result.DeploymentURL = deployResult.URL
		}
	}

	// Complete workflow
	result.Status = "completed"
	result.EndTime = workflow.Now(ctx)
	result.Message = fmt.Sprintf("Infrastructure generated successfully for %s", request.Provider)
	result.InfrastructureCode = infraCode.Code
	
	// Calculate total duration
	duration := result.EndTime.Sub(result.StartTime)
	result.Duration = duration.String()

	logger.Info("Infrastructure Generation Workflow completed",
		"workflow_id", request.WorkflowID,
		"duration", duration,
		"stages_completed", len(result.Stages))

	return result, nil
}