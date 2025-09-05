package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/quantumlayer/infra-workflow-worker/client"
	"github.com/quantumlayer/infra-workflow-worker/types"
)

// AnalyzeCodeActivity analyzes generated code to determine infrastructure requirements
func AnalyzeCodeActivity(ctx context.Context, request types.InfrastructureRequest) (*types.CodeAnalysis, error) {
	log.Printf("Analyzing code for workflow: %s", request.WorkflowID)
	
	// In production, this would fetch code from QuantumDrops and analyze it
	// For now, return example analysis
	analysis := &types.CodeAnalysis{
		Language:      "python",
		Framework:     "fastapi",
		RecommendedOS: "ubuntu-22.04",
		RequiredPackages: []string{
			"python3",
			"nginx",
			"postgresql-client",
			"redis-tools",
		},
		RequiredResources: []types.ResourceRequirement{
			{
				Type: "compute",
				Name: "api-servers",
				Properties: map[string]interface{}{
					"instance_type": "t3.medium",
					"count":        3,
					"auto_scaling": true,
				},
			},
			{
				Type: "database",
				Name: "postgres",
				Properties: map[string]interface{}{
					"engine":         "postgresql",
					"version":        "14",
					"instance_class": "db.t3.medium",
					"storage":        100,
					"multi_az":       true,
				},
			},
			{
				Type: "cache",
				Name: "redis",
				Properties: map[string]interface{}{
					"engine":        "redis",
					"node_type":     "cache.t3.micro",
					"num_nodes":     1,
				},
			},
			{
				Type: "storage",
				Name: "static-assets",
				Properties: map[string]interface{}{
					"type":       "s3",
					"versioning": true,
					"encryption": true,
				},
			},
			{
				Type: "network",
				Name: "vpc",
				Properties: map[string]interface{}{
					"cidr":              "10.0.0.0/16",
					"availability_zones": 3,
					"nat_gateways":      2,
				},
			},
		},
		Requirements: map[string]interface{}{
			"high_availability": true,
			"auto_scaling":      true,
			"monitoring":        true,
			"logging":          true,
			"backup":           true,
		},
	}
	
	return analysis, nil
}

// GenerateInfrastructureActivity calls QInfra API to generate infrastructure code
func GenerateInfrastructureActivity(ctx context.Context, request types.GenerateInfraRequest) (*types.InfrastructureCode, error) {
	log.Printf("Generating infrastructure for provider: %s", request.Provider)
	
	// Call QInfra API
	qinfraClient := client.NewQInfraClient()
	
	infraRequest := map[string]interface{}{
		"type":         "cloud",
		"provider":     request.Provider,
		"requirements": "High-availability infrastructure with auto-scaling",
		"resources":    request.Resources,
		"compliance":   request.Compliance,
	}
	
	response, err := qinfraClient.GenerateInfrastructure(ctx, infraRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate infrastructure: %w", err)
	}
	
	// Parse response
	infraCode := &types.InfrastructureCode{
		Framework:    response.Framework,
		Code:         response.Code,
		DeployScript: response.DeployScript,
		Documentation: generateDocumentation(response),
	}
	
	return infraCode, nil
}

// BuildGoldenImageActivity creates hardened OS images
func BuildGoldenImageActivity(ctx context.Context, request types.GoldenImageRequest) (*types.GoldenImageResult, error) {
	log.Printf("Building golden image with base OS: %s", request.BaseOS)
	
	// Call QInfra golden image API
	qinfraClient := client.NewQInfraClient()
	
	imageSpec := map[string]interface{}{
		"base_os":    request.BaseOS,
		"hardening":  request.Hardening,
		"packages":   request.Packages,
		"compliance": request.Compliance,
	}
	
	response, err := qinfraClient.BuildGoldenImage(ctx, imageSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to build golden image: %w", err)
	}
	
	result := &types.GoldenImageResult{
		ImageID:   response.ImageID,
		ImageName: fmt.Sprintf("%s-golden-%s", request.BaseOS, time.Now().Format("20060102")),
		Registry:  "ghcr.io/quantumlayer-dev",
		BuildTime: time.Now(),
		Size:      1024 * 1024 * 500, // Example: 500MB
	}
	
	return result, nil
}

// GenerateSOPActivity creates operational runbooks
func GenerateSOPActivity(ctx context.Context, request types.SOPRequest) (*types.SOPResult, error) {
	log.Printf("Generating SOPs for infrastructure type: %s", request.InfrastructureType)
	
	// Call QInfra SOP API
	qinfraClient := client.NewQInfraClient()
	
	runbooks := make(map[string]types.Runbook)
	
	for _, operation := range request.Operations {
		sopRequest := map[string]interface{}{
			"name": operation,
			"type": operation,
			"automation": true,
		}
		
		response, err := qinfraClient.GenerateSOP(ctx, sopRequest)
		if err != nil {
			log.Printf("Failed to generate SOP for %s: %v", operation, err)
			continue
		}
		
		runbooks[operation] = types.Runbook{
			ID:          response.ID,
			Name:        response.Name,
			Description: fmt.Sprintf("Automated runbook for %s", operation),
			Steps:       extractSteps(response),
			Automation:  response.Executable,
		}
	}
	
	return &types.SOPResult{
		Runbooks: runbooks,
	}, nil
}

// ValidateComplianceActivity checks infrastructure against compliance frameworks
func ValidateComplianceActivity(ctx context.Context, request types.ComplianceRequest) (*types.ComplianceResult, error) {
	log.Printf("Validating compliance for frameworks: %v", request.Frameworks)
	
	// Call QInfra compliance API
	qinfraClient := client.NewQInfraClient()
	
	complianceRequest := map[string]interface{}{
		"code":       request.Code,
		"frameworks": request.Frameworks,
	}
	
	response, err := qinfraClient.ValidateCompliance(ctx, complianceRequest)
	if err != nil {
		return nil, fmt.Errorf("compliance validation failed: %w", err)
	}
	
	result := &types.ComplianceResult{
		Score:       response.Score,
		Passed:      response.Passed,
		Failed:      response.Failed,
		Findings:    extractFindings(response),
		Remediation: response.Remediation,
		Compliant:   response.Score >= 80.0, // 80% threshold for compliance
	}
	
	return result, nil
}

// EstimateCostActivity calculates infrastructure costs
func EstimateCostActivity(ctx context.Context, request types.CostRequest) (*types.CostEstimate, error) {
	log.Printf("Estimating costs for provider: %s", request.Provider)
	
	// Call QInfra cost API
	qinfraClient := client.NewQInfraClient()
	
	costRequest := map[string]interface{}{
		"provider":  request.Provider,
		"resources": request.Resources,
	}
	
	response, err := qinfraClient.OptimizeCost(ctx, costRequest)
	if err != nil {
		return nil, fmt.Errorf("cost estimation failed: %w", err)
	}
	
	// Calculate costs based on resources
	monthly := response.TotalMonthlySavings + 5000.0 // Base cost estimate
	hourly := monthly / 720
	annual := monthly * 12
	
	estimate := &types.CostEstimate{
		Monthly: monthly,
		Hourly:  hourly,
		Annual:  annual,
		Breakdown: map[string]float64{
			"compute":  monthly * 0.4,
			"database": monthly * 0.25,
			"storage":  monthly * 0.1,
			"network":  monthly * 0.15,
			"other":    monthly * 0.1,
		},
		Optimizations: extractOptimizations(response),
	}
	
	return estimate, nil
}

// StoreInfraDropActivity stores infrastructure artifacts in QuantumDrops
func StoreInfraDropActivity(ctx context.Context, request types.InfraDropRequest) error {
	log.Printf("Storing infrastructure drop for workflow: %s", request.WorkflowID)
	
	// Call QuantumDrops API to store artifact
	dropsClient := client.NewQuantumDropsClient()
	
	artifactJSON, err := json.Marshal(request.Artifact)
	if err != nil {
		return fmt.Errorf("failed to marshal artifact: %w", err)
	}
	
	dropRequest := map[string]interface{}{
		"workflow_id": request.WorkflowID,
		"stage":       request.Stage,
		"type":        request.Type,
		"artifact":    string(artifactJSON),
		"metadata": map[string]interface{}{
			"timestamp": time.Now().UTC(),
			"service":   "workflow-worker",
		},
	}
	
	err = dropsClient.CreateDrop(ctx, dropRequest)
	if err != nil {
		return fmt.Errorf("failed to store drop: %w", err)
	}
	
	return nil
}

// DeployInfrastructureActivity deploys infrastructure to the cloud
func DeployInfrastructureActivity(ctx context.Context, request types.DeployRequest) (*types.DeploymentResult, error) {
	log.Printf("Deploying infrastructure for environment: %s", request.Environment)
	
	if request.DryRun {
		log.Println("DRY RUN MODE - Not actually deploying")
		return &types.DeploymentResult{
			DeploymentID: fmt.Sprintf("dry-run-%s", request.WorkflowID),
			Status:       "dry_run_success",
			URL:          fmt.Sprintf("https://preview.quantumlayer.dev/%s", request.WorkflowID),
			Resources:    []string{"vpc", "ec2-instances", "rds", "s3-bucket"},
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(5 * time.Second),
		}, nil
	}
	
	// In production, this would:
	// 1. Create temporary workspace
	// 2. Write infrastructure code to files
	// 3. Run terraform/pulumi commands
	// 4. Track deployment status
	// 5. Return deployment URL
	
	result := &types.DeploymentResult{
		DeploymentID: fmt.Sprintf("deploy-%s", request.WorkflowID),
		Status:       "deployed",
		URL:          fmt.Sprintf("https://%s.quantumlayer.dev", request.Environment),
		Resources: []string{
			"vpc-10.0.0.0/16",
			"ec2-t3.medium-3x",
			"rds-postgresql-14",
			"elasticache-redis",
			"s3-static-assets",
		},
		StartTime: time.Now(),
		EndTime:   time.Now().Add(3 * time.Minute),
	}
	
	return result, nil
}

// Helper functions
func generateDocumentation(response *client.QInfraResponse) string {
	var doc strings.Builder
	doc.WriteString("# Infrastructure Documentation\n\n")
	doc.WriteString(fmt.Sprintf("## Framework: %s\n\n", response.Framework))
	doc.WriteString("## Files Generated:\n")
	for filename := range response.Code {
		doc.WriteString(fmt.Sprintf("- %s\n", filename))
	}
	doc.WriteString("\n## Deployment Instructions:\n")
	doc.WriteString("```bash\n")
	doc.WriteString(response.DeployScript)
	doc.WriteString("\n```\n")
	return doc.String()
}

func extractSteps(response *client.SOPResponse) []string {
	steps := []string{}
	for _, step := range response.Steps {
		steps = append(steps, step.Name)
	}
	return steps
}

func extractFindings(response *client.ComplianceResponse) []string {
	findings := []string{}
	for _, finding := range response.Findings {
		findings = append(findings, fmt.Sprintf("%s: %s", finding.Rule, finding.Description))
	}
	return findings
}

func extractOptimizations(response *client.CostResponse) []string {
	optimizations := []string{}
	for _, opt := range response.Optimizations {
		optimizations = append(optimizations, opt.Description)
	}
	return optimizations
}