package models

import (
	"math"
	"strings"
	"time"
)

type PatchRiskScorer struct {
	historicalFailures map[string][]PatchFailure
	environmentFactors map[string]float64
}

type PatchFailure struct {
	PatchID   string
	Timestamp time.Time
	Severity  string
	Rollback  bool
	Impact    string
}

type PatchRiskAssessment struct {
	PatchID             string                 `json:"patch_id"`
	RiskScore           float64                `json:"risk_score"`
	RiskLevel           string                 `json:"risk_level"`
	RecommendedWindow   string                 `json:"recommended_window"`
	RequiresRollback    bool                   `json:"requires_rollback_plan"`
	TestingRequired     string                 `json:"testing_required"`
	ImpactedServices    []string               `json:"impacted_services"`
	Recommendations     []string               `json:"recommendations"`
	EnvironmentFactors  map[string]interface{} `json:"environment_factors"`
}

func NewPatchRiskScorer() *PatchRiskScorer {
	return &PatchRiskScorer{
		historicalFailures: make(map[string][]PatchFailure),
		environmentFactors: map[string]float64{
			"production":  1.0,
			"staging":     0.7,
			"development": 0.3,
		},
	}
}

func (prs *PatchRiskScorer) AssessRisk(patchID, patchType, severity string, environment string, dependencies []string) *PatchRiskAssessment {
	// Calculate base risk score
	baseRisk := prs.calculateBaseRisk(patchType, severity)
	
	// Adjust for historical failures
	historicalRisk := prs.calculateHistoricalRisk(patchID, patchType)
	
	// Factor in dependencies
	dependencyRisk := prs.calculateDependencyRisk(dependencies)
	
	// Apply environment multiplier
	envMultiplier := prs.environmentFactors[environment]
	if envMultiplier == 0 {
		envMultiplier = 0.5
	}
	
	// Calculate composite risk score
	riskScore := (baseRisk*0.4 + historicalRisk*0.3 + dependencyRisk*0.3) * envMultiplier
	
	// Ensure score is between 0 and 1
	riskScore = math.Min(math.Max(riskScore, 0), 1)
	
	// Determine risk level
	riskLevel := prs.determineRiskLevel(riskScore)
	
	// Generate recommendations
	recommendations := prs.generateRecommendations(riskScore, patchType, severity, environment)
	
	// Determine maintenance window
	window := prs.recommendMaintenanceWindow(riskScore, environment)
	
	// Assess testing requirements
	testingRequired := prs.determineTestingRequirements(riskScore, patchType)
	
	// Identify impacted services
	impactedServices := prs.identifyImpactedServices(patchType, dependencies)
	
	return &PatchRiskAssessment{
		PatchID:           patchID,
		RiskScore:         riskScore,
		RiskLevel:         riskLevel,
		RecommendedWindow: window,
		RequiresRollback:  riskScore > 0.6,
		TestingRequired:   testingRequired,
		ImpactedServices:  impactedServices,
		Recommendations:   recommendations,
		EnvironmentFactors: map[string]interface{}{
			"base_risk":        baseRisk,
			"historical_risk":  historicalRisk,
			"dependency_risk":  dependencyRisk,
			"environment":      environment,
			"env_multiplier":   envMultiplier,
		},
	}
}

func (prs *PatchRiskScorer) calculateBaseRisk(patchType, severity string) float64 {
	var typeRisk, severityRisk float64
	
	// Patch type risk mapping
	switch strings.ToLower(patchType) {
	case "kernel":
		typeRisk = 0.9
	case "security":
		typeRisk = 0.8
	case "system":
		typeRisk = 0.7
	case "library":
		typeRisk = 0.6
	case "application":
		typeRisk = 0.5
	case "configuration":
		typeRisk = 0.3
	default:
		typeRisk = 0.5
	}
	
	// Severity risk mapping
	switch strings.ToLower(severity) {
	case "critical":
		severityRisk = 1.0
	case "high":
		severityRisk = 0.8
	case "medium":
		severityRisk = 0.5
	case "low":
		severityRisk = 0.3
	default:
		severityRisk = 0.5
	}
	
	// Weighted average
	return typeRisk*0.6 + severityRisk*0.4
}

func (prs *PatchRiskScorer) calculateHistoricalRisk(patchID, patchType string) float64 {
	// Check direct patch history
	directFailures := prs.historicalFailures[patchID]
	
	// Check similar patch type history
	typeFailures := 0
	totalPatches := 0
	
	for id, failures := range prs.historicalFailures {
		if strings.Contains(id, patchType) {
			totalPatches++
			if len(failures) > 0 {
				typeFailures += len(failures)
			}
		}
	}
	
	// Calculate failure rate
	directFailureRate := float64(len(directFailures)) * 0.2
	
	typeFailureRate := 0.0
	if totalPatches > 0 {
		typeFailureRate = float64(typeFailures) / float64(totalPatches)
	}
	
	// Combine rates with decay for older failures
	historicalRisk := directFailureRate*0.7 + typeFailureRate*0.3
	
	// Apply time decay for old failures
	if len(directFailures) > 0 {
		lastFailure := directFailures[len(directFailures)-1].Timestamp
		daysSince := time.Since(lastFailure).Hours() / 24
		decay := math.Exp(-daysSince / 90) // 90-day half-life
		historicalRisk *= decay
	}
	
	return math.Min(historicalRisk, 1.0)
}

func (prs *PatchRiskScorer) calculateDependencyRisk(dependencies []string) float64 {
	if len(dependencies) == 0 {
		return 0.1
	}
	
	// Risk increases with number of dependencies
	baseRisk := math.Min(float64(len(dependencies))*0.1, 0.5)
	
	// Check for critical dependencies
	criticalDeps := []string{"database", "authentication", "network", "storage", "kernel", "systemd"}
	criticalCount := 0
	
	for _, dep := range dependencies {
		depLower := strings.ToLower(dep)
		for _, critical := range criticalDeps {
			if strings.Contains(depLower, critical) {
				criticalCount++
				break
			}
		}
	}
	
	// Add risk for critical dependencies
	criticalRisk := float64(criticalCount) * 0.15
	
	return math.Min(baseRisk+criticalRisk, 1.0)
}

func (prs *PatchRiskScorer) determineRiskLevel(riskScore float64) string {
	switch {
	case riskScore < 0.25:
		return "low"
	case riskScore < 0.5:
		return "medium"
	case riskScore < 0.75:
		return "high"
	default:
		return "critical"
	}
}

func (prs *PatchRiskScorer) generateRecommendations(riskScore float64, patchType, severity, environment string) []string {
	recommendations := make([]string, 0)
	
	// High risk recommendations
	if riskScore > 0.7 {
		recommendations = append(recommendations, "Implement comprehensive rollback plan")
		recommendations = append(recommendations, "Perform canary deployment (5% → 25% → 50% → 100%)")
		recommendations = append(recommendations, "Schedule extended maintenance window")
		recommendations = append(recommendations, "Have incident response team on standby")
	}
	
	// Security patch recommendations
	if strings.ToLower(patchType) == "security" || strings.ToLower(severity) == "critical" {
		recommendations = append(recommendations, "Expedite patch deployment despite risk")
		recommendations = append(recommendations, "Implement additional security monitoring post-patch")
	}
	
	// Environment-specific recommendations
	if environment == "production" {
		recommendations = append(recommendations, "Test thoroughly in staging environment first")
		recommendations = append(recommendations, "Prepare customer communication plan")
		if riskScore > 0.5 {
			recommendations = append(recommendations, "Consider blue-green deployment strategy")
		}
	}
	
	// Kernel/system patches
	if strings.ToLower(patchType) == "kernel" || strings.ToLower(patchType) == "system" {
		recommendations = append(recommendations, "Prepare for potential system reboot")
		recommendations = append(recommendations, "Verify backup systems are operational")
		recommendations = append(recommendations, "Test failover mechanisms before patching")
	}
	
	// General recommendations
	if riskScore > 0.3 {
		recommendations = append(recommendations, "Create snapshot/backup before patching")
		recommendations = append(recommendations, "Monitor system metrics during and after patch")
	}
	
	return recommendations
}

func (prs *PatchRiskScorer) recommendMaintenanceWindow(riskScore float64, environment string) string {
	if environment == "production" {
		if riskScore > 0.7 {
			return "Weekend 2AM-6AM with 4-hour buffer"
		} else if riskScore > 0.4 {
			return "Weekday 2AM-4AM low-traffic window"
		}
		return "Standard maintenance window"
	}
	
	if riskScore > 0.5 {
		return "Off-peak hours recommended"
	}
	
	return "Any time - low risk"
}

func (prs *PatchRiskScorer) determineTestingRequirements(riskScore float64, patchType string) string {
	if riskScore > 0.7 {
		return "Full regression suite + performance testing + security scan"
	} else if riskScore > 0.5 {
		return "Integration testing + smoke tests + monitoring"
	} else if riskScore > 0.3 {
		return "Smoke tests + basic validation"
	}
	
	if strings.ToLower(patchType) == "security" {
		return "Security validation + penetration testing"
	}
	
	return "Basic smoke tests"
}

func (prs *PatchRiskScorer) identifyImpactedServices(patchType string, dependencies []string) []string {
	services := make([]string, 0)
	
	// Map patch types to common services
	typeServices := map[string][]string{
		"kernel":        {"all-system-services", "container-runtime", "network-stack"},
		"database":      {"api-services", "reporting", "analytics", "backup-services"},
		"network":       {"load-balancer", "api-gateway", "service-mesh"},
		"security":      {"authentication", "authorization", "audit-logging"},
		"library":       {"application-services", "background-jobs"},
		"configuration": {"affected-services-only"},
	}
	
	// Add services based on patch type
	if typeServices[strings.ToLower(patchType)] != nil {
		services = append(services, typeServices[strings.ToLower(patchType)]...)
	}
	
	// Add services based on dependencies
	for _, dep := range dependencies {
		services = append(services, dep)
	}
	
	// Remove duplicates
	uniqueServices := make(map[string]bool)
	result := make([]string, 0)
	for _, service := range services {
		if !uniqueServices[service] {
			uniqueServices[service] = true
			result = append(result, service)
		}
	}
	
	return result
}

func (prs *PatchRiskScorer) RecordFailure(failure PatchFailure) {
	prs.historicalFailures[failure.PatchID] = append(
		prs.historicalFailures[failure.PatchID],
		failure,
	)
	
	// Keep only recent failures (last 100 per patch)
	if len(prs.historicalFailures[failure.PatchID]) > 100 {
		prs.historicalFailures[failure.PatchID] = 
			prs.historicalFailures[failure.PatchID][len(prs.historicalFailures[failure.PatchID])-100:]
	}
}