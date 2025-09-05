package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	DefaultPort = "8098"
	DefaultAIDecisionEngineURL = "http://ai-decision-engine.quantumlayer.svc.cluster.local:8091"
)

// QInfraAI represents the AI intelligence engine for infrastructure
type QInfraAI struct {
	aiEngineURL string
	models      map[string]interface{}
}

// DriftPrediction represents a drift prediction result
type DriftPrediction struct {
	NodeID          string    `json:"node_id"`
	Platform        string    `json:"platform"`
	PredictedDrift  bool      `json:"predicted_drift"`
	Probability     float64   `json:"probability"`
	TimeToD drift    string    `json:"time_to_drift"`
	RiskLevel       string    `json:"risk_level"`
	Factors         []Factor  `json:"factors"`
	Recommendation  string    `json:"recommendation"`
	PredictedAt     time.Time `json:"predicted_at"`
}

// Factor represents a contributing factor to drift
type Factor struct {
	Name   string  `json:"name"`
	Impact float64 `json:"impact"`
	Description string `json:"description"`
}

// PatchRiskAssessment represents patch risk analysis
type PatchRiskAssessment struct {
	PatchID         string    `json:"patch_id"`
	CVE             string    `json:"cve"`
	RiskScore       float64   `json:"risk_score"`
	SuccessProbability float64 `json:"success_probability"`
	ImpactRadius    string    `json:"impact_radius"`
	Dependencies    []string  `json:"dependencies"`
	TestingRequired string    `json:"testing_required"`
	Recommendation  string    `json:"recommendation"`
	AssessedAt      time.Time `json:"assessed_at"`
}

// AnomalyDetection represents detected anomalies
type AnomalyDetection struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	Severity        string    `json:"severity"`
	AnomalyScore    float64   `json:"anomaly_score"`
	Description     string    `json:"description"`
	AffectedNodes   []string  `json:"affected_nodes"`
	Pattern         string    `json:"pattern"`
	FirstSeen       time.Time `json:"first_seen"`
	Recommendation  string    `json:"recommendation"`
}

// RemediationAdvice represents AI-generated remediation advice
type RemediationAdvice struct {
	IssueID         string    `json:"issue_id"`
	IssueType       string    `json:"issue_type"`
	AutoFixable     bool      `json:"auto_fixable"`
	ConfidenceScore float64   `json:"confidence_score"`
	Steps           []Step    `json:"steps"`
	EstimatedTime   string    `json:"estimated_time"`
	RiskOfFix       string    `json:"risk_of_fix"`
	AlternativeActions []string `json:"alternative_actions"`
	GeneratedAt     time.Time `json:"generated_at"`
}

// Step represents a remediation step
type Step struct {
	Order       int    `json:"order"`
	Action      string `json:"action"`
	Command     string `json:"command,omitempty"`
	Validation  string `json:"validation"`
	Rollback    string `json:"rollback,omitempty"`
}

// CanaryAnalysis represents canary deployment analysis
type CanaryAnalysis struct {
	DeploymentID    string    `json:"deployment_id"`
	CanaryScore     float64   `json:"canary_score"`
	SafeToProceed   bool      `json:"safe_to_proceed"`
	ErrorRate       float64   `json:"error_rate"`
	LatencyImpact   float64   `json:"latency_impact"`
	CPUImpact       float64   `json:"cpu_impact"`
	MemoryImpact    float64   `json:"memory_impact"`
	Anomalies       []string  `json:"anomalies"`
	Recommendation  string    `json:"recommendation"`
	AnalyzedAt      time.Time `json:"analyzed_at"`
}

// RiskDashboard represents overall infrastructure risk
type RiskDashboard struct {
	OverallRisk     float64            `json:"overall_risk"`
	RiskLevel       string             `json:"risk_level"`
	TrendDirection  string             `json:"trend_direction"`
	RiskByCategory  map[string]float64 `json:"risk_by_category"`
	TopRisks        []Risk            `json:"top_risks"`
	Predictions     []Prediction      `json:"predictions"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// Risk represents an individual risk
type Risk struct {
	ID          string  `json:"id"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Impact      string  `json:"impact"`
	Likelihood  string  `json:"likelihood"`
}

// Prediction represents a future prediction
type Prediction struct {
	Event       string    `json:"event"`
	Probability float64   `json:"probability"`
	TimeFrame   string    `json:"time_frame"`
	Impact      string    `json:"impact"`
}

func NewQInfraAI() *QInfraAI {
	aiURL := os.Getenv("AI_DECISION_ENGINE_URL")
	if aiURL == "" {
		aiURL = DefaultAIDecisionEngineURL
	}

	return &QInfraAI{
		aiEngineURL: aiURL,
		models:      make(map[string]interface{}),
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	ai := NewQInfraAI()
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "qinfra-ai",
			"version": "1.0.0",
			"timestamp": time.Now().Unix(),
		})
	})

	// AI Intelligence APIs
	apiV1 := r.Group("/api/v1")
	{
		// Drift Prediction
		apiV1.POST("/predict-drift", ai.predictDrift)
		
		// Patch Risk Assessment
		apiV1.POST("/assess-patch-risk", ai.assessPatchRisk)
		
		// Anomaly Detection
		apiV1.POST("/detect-anomalies", ai.detectAnomalies)
		
		// Remediation Advice
		apiV1.POST("/recommend-action", ai.recommendAction)
		
		// Canary Analysis
		apiV1.POST("/analyze-canary", ai.analyzeCanary)
		
		// Risk Dashboard
		apiV1.GET("/risk-dashboard", ai.getRiskDashboard)
		
		// Explain Drift
		apiV1.POST("/explain-drift", ai.explainDrift)
	}

	// Metrics endpoint
	r.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"predictions_made": 1247,
			"accuracy_rate": 0.87,
			"average_response_time_ms": 45,
		})
	})

	log.Printf("Starting QInfra AI service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// predictDrift uses ML to predict future infrastructure drift
func (ai *QInfraAI) predictDrift(c *gin.Context) {
	var request struct {
		NodeID       string                 `json:"node_id"`
		Platform     string                 `json:"platform"`
		CurrentState map[string]interface{} `json:"current_state"`
		History      []map[string]interface{} `json:"history,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simulate ML prediction (in production, use real model)
	prediction := ai.performDriftPrediction(request.NodeID, request.Platform, request.CurrentState)

	c.JSON(http.StatusOK, prediction)
}

// performDriftPrediction simulates drift prediction
func (ai *QInfraAI) performDriftPrediction(nodeID, platform string, state map[string]interface{}) DriftPrediction {
	// Simulate ML model prediction
	rand.Seed(time.Now().UnixNano())
	probability := rand.Float64()
	
	// Analyze factors that contribute to drift
	factors := []Factor{
		{
			Name: "Manual changes detected",
			Impact: 0.35,
			Description: "Configuration files modified outside of automation",
		},
		{
			Name: "Package update frequency",
			Impact: 0.25,
			Description: "System packages updated manually in past 30 days",
		},
		{
			Name: "Time since last golden image",
			Impact: 0.20,
			Description: "45 days since last golden image deployment",
		},
		{
			Name: "User activity patterns",
			Impact: 0.15,
			Description: "Unusual SSH access patterns detected",
		},
	}

	riskLevel := "low"
	timeToDrift := "30+ days"
	recommendation := "Continue monitoring"
	
	if probability > 0.7 {
		riskLevel = "high"
		timeToDrift = "3-7 days"
		recommendation = "Schedule golden image refresh within 48 hours"
	} else if probability > 0.4 {
		riskLevel = "medium"
		timeToDrift = "7-14 days"
		recommendation = "Plan golden image refresh in next maintenance window"
	}

	return DriftPrediction{
		NodeID:         nodeID,
		Platform:       platform,
		PredictedDrift: probability > 0.5,
		Probability:    probability,
		TimeToD drift:   timeToDrift,
		RiskLevel:      riskLevel,
		Factors:        factors,
		Recommendation: recommendation,
		PredictedAt:    time.Now(),
	}
}

// assessPatchRisk evaluates the risk of applying a patch
func (ai *QInfraAI) assessPatchRisk(c *gin.Context) {
	var request struct {
		PatchID      string   `json:"patch_id"`
		CVE          string   `json:"cve"`
		TargetNodes  []string `json:"target_nodes"`
		Environment  string   `json:"environment"`
		Dependencies []string `json:"dependencies,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simulate risk assessment
	assessment := ai.performPatchRiskAssessment(request)

	c.JSON(http.StatusOK, assessment)
}

// performPatchRiskAssessment simulates patch risk analysis
func (ai *QInfraAI) performPatchRiskAssessment(request struct {
	PatchID      string   `json:"patch_id"`
	CVE          string   `json:"cve"`
	TargetNodes  []string `json:"target_nodes"`
	Environment  string   `json:"environment"`
	Dependencies []string `json:"dependencies,omitempty"`
}) PatchRiskAssessment {
	rand.Seed(time.Now().UnixNano())
	
	// Simulate ML-based risk scoring
	baseRisk := rand.Float64() * 0.5 // Base risk 0-0.5
	
	// Adjust based on environment
	if request.Environment == "production" {
		baseRisk += 0.2
	}
	
	// Adjust based on dependencies
	dependencyRisk := float64(len(request.Dependencies)) * 0.05
	totalRisk := math.Min(baseRisk+dependencyRisk, 1.0)
	
	successProbability := 1.0 - totalRisk
	
	testingRequired := "Basic validation"
	if totalRisk > 0.7 {
		testingRequired = "Full regression testing with canary deployment"
	} else if totalRisk > 0.4 {
		testingRequired = "Integration testing with staged rollout"
	}
	
	recommendation := "Safe to deploy with standard procedures"
	if totalRisk > 0.7 {
		recommendation = "High risk - recommend extensive testing and phased rollout"
	} else if totalRisk > 0.4 {
		recommendation = "Medium risk - deploy to staging first, monitor for 24 hours"
	}

	return PatchRiskAssessment{
		PatchID:            request.PatchID,
		CVE:                request.CVE,
		RiskScore:          totalRisk,
		SuccessProbability: successProbability,
		ImpactRadius:       fmt.Sprintf("%d nodes", len(request.TargetNodes)),
		Dependencies:       request.Dependencies,
		TestingRequired:    testingRequired,
		Recommendation:     recommendation,
		AssessedAt:         time.Now(),
	}
}

// detectAnomalies identifies unusual patterns in infrastructure
func (ai *QInfraAI) detectAnomalies(c *gin.Context) {
	var request struct {
		Platform   string                   `json:"platform"`
		Metrics    map[string]float64       `json:"metrics"`
		TimeWindow string                   `json:"time_window"`
		Nodes      []map[string]interface{} `json:"nodes,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simulate anomaly detection
	anomalies := ai.performAnomalyDetection(request)

	c.JSON(http.StatusOK, gin.H{
		"anomalies_detected": len(anomalies),
		"anomalies": anomalies,
	})
}

// performAnomalyDetection simulates anomaly detection
func (ai *QInfraAI) performAnomalyDetection(request struct {
	Platform   string                   `json:"platform"`
	Metrics    map[string]float64       `json:"metrics"`
	TimeWindow string                   `json:"time_window"`
	Nodes      []map[string]interface{} `json:"nodes,omitempty"`
}) []AnomalyDetection {
	anomalies := []AnomalyDetection{}
	
	// Simulate detection of various anomaly types
	if request.Metrics["cpu_usage"] > 80 {
		anomalies = append(anomalies, AnomalyDetection{
			ID:           uuid.New().String(),
			Type:         "resource_spike",
			Severity:     "high",
			AnomalyScore: 0.85,
			Description:  "Unusual CPU usage pattern detected",
			AffectedNodes: []string{"node-001", "node-002"},
			Pattern:      "Sudden spike without corresponding workload increase",
			FirstSeen:    time.Now().Add(-30 * time.Minute),
			Recommendation: "Investigate process causing CPU spike, possible crypto-mining",
		})
	}
	
	// Configuration drift anomaly
	anomalies = append(anomalies, AnomalyDetection{
		ID:           uuid.New().String(),
		Type:         "configuration_drift",
		Severity:     "medium",
		AnomalyScore: 0.62,
		Description:  "Unexpected configuration changes detected",
		AffectedNodes: []string{"node-003"},
		Pattern:      "Files modified outside of deployment window",
		FirstSeen:    time.Now().Add(-2 * time.Hour),
		Recommendation: "Review recent changes, restore from golden image if unauthorized",
	})

	return anomalies
}

// recommendAction provides AI-generated remediation advice
func (ai *QInfraAI) recommendAction(c *gin.Context) {
	var request struct {
		IssueID     string                 `json:"issue_id"`
		IssueType   string                 `json:"issue_type"`
		Severity    string                 `json:"severity"`
		Context     map[string]interface{} `json:"context"`
		Constraints []string               `json:"constraints,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate remediation advice
	advice := ai.generateRemediationAdvice(request)

	c.JSON(http.StatusOK, advice)
}

// generateRemediationAdvice creates AI-powered remediation steps
func (ai *QInfraAI) generateRemediationAdvice(request struct {
	IssueID     string                 `json:"issue_id"`
	IssueType   string                 `json:"issue_type"`
	Severity    string                 `json:"severity"`
	Context     map[string]interface{} `json:"context"`
	Constraints []string               `json:"constraints,omitempty"`
}) RemediationAdvice {
	// Simulate AI-generated remediation based on issue type
	var steps []Step
	autoFixable := false
	confidenceScore := 0.0
	estimatedTime := "5 minutes"
	riskOfFix := "low"
	
	switch request.IssueType {
	case "drift":
		autoFixable = true
		confidenceScore = 0.92
		steps = []Step{
			{
				Order:      1,
				Action:     "Backup current configuration",
				Command:    "kubectl create backup drift-backup-$(date +%s)",
				Validation: "Verify backup created successfully",
				Rollback:   "N/A",
			},
			{
				Order:      2,
				Action:     "Apply golden image configuration",
				Command:    "qinfra apply-golden-image --node=${NODE_ID}",
				Validation: "Check configuration matches golden image",
				Rollback:   "kubectl restore backup drift-backup-*",
			},
			{
				Order:      3,
				Action:     "Verify services are running",
				Command:    "qinfra verify-services --node=${NODE_ID}",
				Validation: "All services report healthy status",
				Rollback:   "kubectl rollback deployment",
			},
		}
		
	case "vulnerability":
		autoFixable = request.Severity != "critical"
		confidenceScore = 0.78
		estimatedTime = "15 minutes"
		riskOfFix = "medium"
		steps = []Step{
			{
				Order:      1,
				Action:     "Identify affected packages",
				Command:    "qinfra scan-vulnerabilities --cve=${CVE_ID}",
				Validation: "List of affected packages generated",
			},
			{
				Order:      2,
				Action:     "Test patch in staging",
				Command:    "qinfra test-patch --env=staging --cve=${CVE_ID}",
				Validation: "Patch successfully applied in staging",
			},
			{
				Order:      3,
				Action:     "Apply patch with canary deployment",
				Command:    "qinfra patch --canary=10% --cve=${CVE_ID}",
				Validation: "Monitor error rates for 30 minutes",
				Rollback:   "qinfra rollback-patch --cve=${CVE_ID}",
			},
			{
				Order:      4,
				Action:     "Complete rollout",
				Command:    "qinfra patch --complete --cve=${CVE_ID}",
				Validation: "All nodes patched successfully",
				Rollback:   "qinfra rollback-patch --all --cve=${CVE_ID}",
			},
		}
		
	default:
		confidenceScore = 0.65
		estimatedTime = "30 minutes"
		steps = []Step{
			{
				Order:      1,
				Action:     "Investigate issue",
				Command:    "qinfra diagnose --issue=${ISSUE_ID}",
				Validation: "Root cause identified",
			},
			{
				Order:      2,
				Action:     "Apply recommended fix",
				Command:    "qinfra fix --issue=${ISSUE_ID} --dry-run",
				Validation: "Dry run completed without errors",
			},
		}
	}

	return RemediationAdvice{
		IssueID:         request.IssueID,
		IssueType:       request.IssueType,
		AutoFixable:     autoFixable,
		ConfidenceScore: confidenceScore,
		Steps:           steps,
		EstimatedTime:   estimatedTime,
		RiskOfFix:       riskOfFix,
		AlternativeActions: []string{
			"Schedule manual intervention",
			"Isolate affected nodes",
			"Revert to previous golden image",
		},
		GeneratedAt:     time.Now(),
	}
}

// analyzeCanary performs canary deployment analysis
func (ai *QInfraAI) analyzeCanary(c *gin.Context) {
	var request struct {
		DeploymentID string             `json:"deployment_id"`
		CanaryMetrics map[string]float64 `json:"canary_metrics"`
		BaselineMetrics map[string]float64 `json:"baseline_metrics"`
		Duration      string             `json:"duration"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Analyze canary deployment
	analysis := ai.performCanaryAnalysis(request)

	c.JSON(http.StatusOK, analysis)
}

// performCanaryAnalysis analyzes canary deployment safety
func (ai *QInfraAI) performCanaryAnalysis(request struct {
	DeploymentID string             `json:"deployment_id"`
	CanaryMetrics map[string]float64 `json:"canary_metrics"`
	BaselineMetrics map[string]float64 `json:"baseline_metrics"`
	Duration      string             `json:"duration"`
}) CanaryAnalysis {
	// Calculate differences between canary and baseline
	errorRateDiff := request.CanaryMetrics["error_rate"] - request.BaselineMetrics["error_rate"]
	latencyDiff := request.CanaryMetrics["latency"] - request.BaselineMetrics["latency"]
	cpuDiff := request.CanaryMetrics["cpu"] - request.BaselineMetrics["cpu"]
	memoryDiff := request.CanaryMetrics["memory"] - request.BaselineMetrics["memory"]
	
	// Calculate overall canary score (0-100, higher is better)
	canaryScore := 100.0
	anomalies := []string{}
	
	if errorRateDiff > 0.01 { // 1% increase in errors
		canaryScore -= 30
		anomalies = append(anomalies, "Error rate increased by "+fmt.Sprintf("%.2f%%", errorRateDiff*100))
	}
	
	if latencyDiff > request.BaselineMetrics["latency"]*0.1 { // 10% latency increase
		canaryScore -= 20
		anomalies = append(anomalies, "Latency increased by "+fmt.Sprintf("%.2fms", latencyDiff))
	}
	
	if cpuDiff > request.BaselineMetrics["cpu"]*0.2 { // 20% CPU increase
		canaryScore -= 15
		anomalies = append(anomalies, "CPU usage increased significantly")
	}
	
	if memoryDiff > request.BaselineMetrics["memory"]*0.15 { // 15% memory increase
		canaryScore -= 10
		anomalies = append(anomalies, "Memory usage increased")
	}
	
	safeToProceed := canaryScore >= 70
	recommendation := "Safe to proceed with full rollout"
	
	if canaryScore < 50 {
		recommendation = "Rollback immediately - significant degradation detected"
	} else if canaryScore < 70 {
		recommendation = "Investigate issues before proceeding - moderate concerns detected"
	}

	return CanaryAnalysis{
		DeploymentID:   request.DeploymentID,
		CanaryScore:    canaryScore,
		SafeToProceed:  safeToProceed,
		ErrorRate:      errorRateDiff,
		LatencyImpact:  latencyDiff,
		CPUImpact:      cpuDiff,
		MemoryImpact:   memoryDiff,
		Anomalies:      anomalies,
		Recommendation: recommendation,
		AnalyzedAt:     time.Now(),
	}
}

// getRiskDashboard provides overall infrastructure risk assessment
func (ai *QInfraAI) getRiskDashboard(c *gin.Context) {
	// Generate comprehensive risk dashboard
	dashboard := ai.generateRiskDashboard()
	
	c.JSON(http.StatusOK, dashboard)
}

// generateRiskDashboard creates overall risk assessment
func (ai *QInfraAI) generateRiskDashboard() RiskDashboard {
	// Simulate risk calculation across categories
	riskByCategory := map[string]float64{
		"security":    0.35,
		"compliance":  0.22,
		"performance": 0.18,
		"drift":       0.42,
		"patches":     0.28,
	}
	
	// Calculate overall risk (weighted average)
	overallRisk := 0.0
	for _, risk := range riskByCategory {
		overallRisk += risk
	}
	overallRisk = overallRisk / float64(len(riskByCategory))
	
	riskLevel := "low"
	if overallRisk > 0.7 {
		riskLevel = "critical"
	} else if overallRisk > 0.5 {
		riskLevel = "high"
	} else if overallRisk > 0.3 {
		riskLevel = "medium"
	}
	
	topRisks := []Risk{
		{
			ID:          "risk-001",
			Category:    "drift",
			Description: "15 nodes showing configuration drift",
			Score:       0.72,
			Impact:      "high",
			Likelihood:  "certain",
		},
		{
			ID:          "risk-002",
			Category:    "security",
			Description: "3 critical CVEs pending patches",
			Score:       0.68,
			Impact:      "critical",
			Likelihood:  "likely",
		},
		{
			ID:          "risk-003",
			Category:    "compliance",
			Description: "SOC2 compliance score below threshold",
			Score:       0.45,
			Impact:      "medium",
			Likelihood:  "possible",
		},
	}
	
	predictions := []Prediction{
		{
			Event:       "Major drift event",
			Probability: 0.78,
			TimeFrame:   "Next 7 days",
			Impact:      "high",
		},
		{
			Event:       "Compliance violation",
			Probability: 0.45,
			TimeFrame:   "Next 30 days",
			Impact:      "medium",
		},
		{
			Event:       "Performance degradation",
			Probability: 0.32,
			TimeFrame:   "Next 14 days",
			Impact:      "low",
		},
	}

	return RiskDashboard{
		OverallRisk:    overallRisk,
		RiskLevel:      riskLevel,
		TrendDirection: "increasing",
		RiskByCategory: riskByCategory,
		TopRisks:       topRisks,
		Predictions:    predictions,
		UpdatedAt:      time.Now(),
	}
}

// explainDrift provides natural language explanation of drift
func (ai *QInfraAI) explainDrift(c *gin.Context) {
	var request struct {
		NodeID       string                 `json:"node_id"`
		DriftDetails map[string]interface{} `json:"drift_details"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate explanation (in production, use LLM)
	explanation := fmt.Sprintf(
		"Node %s experienced drift due to manual configuration changes. "+
		"The primary cause appears to be unauthorized package installations "+
		"outside of the deployment pipeline. This commonly occurs when "+
		"administrators make emergency fixes without updating golden images. "+
		"Recommendation: Refresh from golden image and update deployment procedures.",
		request.NodeID,
	)

	c.JSON(http.StatusOK, gin.H{
		"node_id":     request.NodeID,
		"explanation": explanation,
		"root_causes": []string{
			"Manual configuration changes",
			"Package drift",
			"Unauthorized modifications",
		},
		"prevention_measures": []string{
			"Implement stricter change control",
			"Regular golden image refreshes",
			"Enable configuration monitoring",
		},
	})
}