package models

import (
	"math"
	"time"
)

type DriftPredictor struct {
	historicalData []DriftDataPoint
	threshold      float64
}

type DriftDataPoint struct {
	Timestamp    time.Time
	NodeID       string
	DriftScore   float64
	ConfigDelta  map[string]interface{}
	PackageDelta []string
	Features     []float64
}

type DriftPrediction struct {
	NodeID          string  `json:"node_id"`
	PredictedDrift  float64 `json:"predicted_drift"`
	TimeToThreshold string  `json:"time_to_threshold"`
	RiskLevel       string  `json:"risk_level"`
	Contributors    []DriftContributor `json:"contributors"`
}

type DriftContributor struct {
	Factor      string  `json:"factor"`
	Impact      float64 `json:"impact"`
	Description string  `json:"description"`
}

func NewDriftPredictor() *DriftPredictor {
	return &DriftPredictor{
		threshold:      0.75,
		historicalData: make([]DriftDataPoint, 0),
	}
}

func (dp *DriftPredictor) Predict(nodeID string, currentConfig map[string]interface{}) *DriftPrediction {
	// Extract features from configuration
	features := dp.extractFeatures(currentConfig)
	
	// Calculate drift score using weighted features
	driftScore := dp.calculateDriftScore(features)
	
	// Predict time to threshold based on historical trends
	timeToThreshold := dp.predictTimeToThreshold(nodeID, driftScore)
	
	// Identify top contributors to drift
	contributors := dp.identifyContributors(currentConfig)
	
	// Determine risk level
	riskLevel := dp.calculateRiskLevel(driftScore)
	
	return &DriftPrediction{
		NodeID:          nodeID,
		PredictedDrift:  driftScore,
		TimeToThreshold: timeToThreshold,
		RiskLevel:       riskLevel,
		Contributors:    contributors,
	}
}

func (dp *DriftPredictor) extractFeatures(config map[string]interface{}) []float64 {
	features := make([]float64, 0)
	
	// Package count feature
	if packages, ok := config["packages"].([]interface{}); ok {
		features = append(features, float64(len(packages)))
	} else {
		features = append(features, 0)
	}
	
	// Configuration complexity feature
	features = append(features, float64(len(config)))
	
	// Time since last update feature
	if lastUpdate, ok := config["last_update"].(time.Time); ok {
		hours := time.Since(lastUpdate).Hours()
		features = append(features, hours)
	} else {
		features = append(features, 0)
	}
	
	// Security patch level feature
	if patchLevel, ok := config["security_patch_level"].(float64); ok {
		features = append(features, patchLevel)
	} else {
		features = append(features, 0)
	}
	
	// Compliance score feature
	if compliance, ok := config["compliance_score"].(float64); ok {
		features = append(features, compliance)
	} else {
		features = append(features, 1.0)
	}
	
	return features
}

func (dp *DriftPredictor) calculateDriftScore(features []float64) float64 {
	// Weights for different features
	weights := []float64{0.2, 0.15, 0.3, 0.25, 0.1}
	
	// Normalize features
	normalizedFeatures := dp.normalizeFeatures(features)
	
	// Calculate weighted drift score
	var driftScore float64
	for i, feature := range normalizedFeatures {
		if i < len(weights) {
			driftScore += feature * weights[i]
		}
	}
	
	// Apply sigmoid to keep score between 0 and 1
	return sigmoid(driftScore)
}

func (dp *DriftPredictor) normalizeFeatures(features []float64) []float64 {
	normalized := make([]float64, len(features))
	
	// Simple min-max normalization
	maxValues := []float64{1000, 100, 720, 100, 1} // Max expected values
	
	for i, feature := range features {
		if i < len(maxValues) && maxValues[i] > 0 {
			normalized[i] = feature / maxValues[i]
			if normalized[i] > 1 {
				normalized[i] = 1
			}
		}
	}
	
	return normalized
}

func (dp *DriftPredictor) predictTimeToThreshold(nodeID string, currentDrift float64) string {
	// Calculate drift velocity based on historical data
	velocity := dp.calculateDriftVelocity(nodeID)
	
	if velocity <= 0 {
		return "Stable"
	}
	
	// Calculate time to reach threshold
	remainingDrift := dp.threshold - currentDrift
	if remainingDrift <= 0 {
		return "Already exceeded"
	}
	
	hoursToThreshold := remainingDrift / velocity
	
	if hoursToThreshold < 24 {
		return "< 24 hours"
	} else if hoursToThreshold < 72 {
		return "1-3 days"
	} else if hoursToThreshold < 168 {
		return "< 1 week"
	} else if hoursToThreshold < 720 {
		return "< 1 month"
	}
	
	return "> 1 month"
}

func (dp *DriftPredictor) calculateDriftVelocity(nodeID string) float64 {
	// Filter historical data for this node
	nodeData := make([]DriftDataPoint, 0)
	for _, point := range dp.historicalData {
		if point.NodeID == nodeID {
			nodeData = append(nodeData, point)
		}
	}
	
	if len(nodeData) < 2 {
		return 0.01 // Default velocity
	}
	
	// Calculate average velocity over recent history
	totalVelocity := 0.0
	count := 0
	
	for i := 1; i < len(nodeData) && i < 10; i++ {
		timeDiff := nodeData[i].Timestamp.Sub(nodeData[i-1].Timestamp).Hours()
		if timeDiff > 0 {
			driftDiff := nodeData[i].DriftScore - nodeData[i-1].DriftScore
			velocity := driftDiff / timeDiff
			totalVelocity += velocity
			count++
		}
	}
	
	if count == 0 {
		return 0.01
	}
	
	return totalVelocity / float64(count)
}

func (dp *DriftPredictor) identifyContributors(config map[string]interface{}) []DriftContributor {
	contributors := make([]DriftContributor, 0)
	
	// Check package updates
	if packages, ok := config["outdated_packages"].(int); ok && packages > 0 {
		contributors = append(contributors, DriftContributor{
			Factor:      "outdated_packages",
			Impact:      float64(packages) * 0.05,
			Description: "Outdated packages increasing security risk",
		})
	}
	
	// Check configuration changes
	if changes, ok := config["config_changes"].(int); ok && changes > 0 {
		contributors = append(contributors, DriftContributor{
			Factor:      "configuration_drift",
			Impact:      float64(changes) * 0.1,
			Description: "Unauthorized configuration changes detected",
		})
	}
	
	// Check compliance violations
	if violations, ok := config["compliance_violations"].(int); ok && violations > 0 {
		contributors = append(contributors, DriftContributor{
			Factor:      "compliance_violations",
			Impact:      float64(violations) * 0.15,
			Description: "Compliance policy violations found",
		})
	}
	
	// Check missing patches
	if patches, ok := config["missing_patches"].(int); ok && patches > 0 {
		contributors = append(contributors, DriftContributor{
			Factor:      "missing_patches",
			Impact:      float64(patches) * 0.2,
			Description: "Critical security patches missing",
		})
	}
	
	// Check uptime
	if uptime, ok := config["uptime_days"].(int); ok && uptime > 30 {
		contributors = append(contributors, DriftContributor{
			Factor:      "extended_uptime",
			Impact:      0.1,
			Description: "System running without maintenance window",
		})
	}
	
	return contributors
}

func (dp *DriftPredictor) calculateRiskLevel(driftScore float64) string {
	switch {
	case driftScore < 0.3:
		return "low"
	case driftScore < 0.5:
		return "medium"
	case driftScore < 0.75:
		return "high"
	default:
		return "critical"
	}
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func (dp *DriftPredictor) AddHistoricalData(data DriftDataPoint) {
	dp.historicalData = append(dp.historicalData, data)
	
	// Keep only recent history (last 1000 points)
	if len(dp.historicalData) > 1000 {
		dp.historicalData = dp.historicalData[len(dp.historicalData)-1000:]
	}
}