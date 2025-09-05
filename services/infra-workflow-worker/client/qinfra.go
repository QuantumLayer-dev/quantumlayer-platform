package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// QInfraClient is the client for QInfra API
type QInfraClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewQInfraClient creates a new QInfra API client
func NewQInfraClient() *QInfraClient {
	baseURL := os.Getenv("QINFRA_URL")
	if baseURL == "" {
		baseURL = "http://qinfra.quantumlayer.svc.cluster.local:8095"
	}
	
	return &QInfraClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// QInfraResponse from infrastructure generation
type QInfraResponse struct {
	ID               string                 `json:"id"`
	Status           string                 `json:"status"`
	Framework        string                 `json:"framework"`
	Code             map[string]string      `json:"code"`
	DeployScript     string                 `json:"deploy_script"`
	EstimatedCost    *CostEstimate         `json:"estimated_cost"`
	ComplianceReport *ComplianceReport     `json:"compliance_report"`
	Vulnerabilities  []VulnerabilityReport `json:"vulnerabilities"`
	Optimizations    []Optimization        `json:"optimizations"`
}

// CostEstimate from QInfra
type CostEstimate struct {
	MonthlyUSD float64            `json:"monthly_usd"`
	HourlyUSD  float64            `json:"hourly_usd"`
	Details    map[string]float64 `json:"details"`
}

// ComplianceReport from QInfra
type ComplianceReport struct {
	Framework   string              `json:"framework"`
	Score       float64             `json:"score"`
	Passed      int                 `json:"passed"`
	Failed      int                 `json:"failed"`
	Findings    []ComplianceFinding `json:"findings"`
	Remediation []string           `json:"remediation"`
}

// ComplianceFinding details
type ComplianceFinding struct {
	Rule        string `json:"rule"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Evidence    string `json:"evidence"`
}

// VulnerabilityReport from QInfra
type VulnerabilityReport struct {
	Severity    string `json:"severity"`
	CVE         string `json:"cve"`
	Description string `json:"description"`
	Affected    string `json:"affected"`
	Fix         string `json:"fix"`
}

// Optimization suggestion
type Optimization struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Savings     float64 `json:"savings_usd"`
}

// GenerateInfrastructure calls QInfra to generate infrastructure code
func (c *QInfraClient) GenerateInfrastructure(ctx context.Context, request map[string]interface{}) (*QInfraResponse, error) {
	url := fmt.Sprintf("%s/generate", c.baseURL)
	
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call QInfra API: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("QInfra API returned status %d: %s", resp.StatusCode, body)
	}
	
	var result QInfraResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// GoldenImageResponse from QInfra
type GoldenImageResponse struct {
	ImageID      string `json:"image_id"`
	Status       string `json:"status"`
	EstimatedTime string `json:"estimated_time"`
}

// BuildGoldenImage calls QInfra to build a golden image
func (c *QInfraClient) BuildGoldenImage(ctx context.Context, imageSpec map[string]interface{}) (*GoldenImageResponse, error) {
	url := fmt.Sprintf("%s/golden-image/build", c.baseURL)
	
	body, err := json.Marshal(imageSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call QInfra API: %w", err)
	}
	defer resp.Body.Close()
	
	var result GoldenImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// SOPResponse from QInfra
type SOPResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Steps       []SOPStep `json:"steps"`
	Executable  bool      `json:"executable"`
	EstDuration string    `json:"estimated_duration"`
}

// SOPStep in a runbook
type SOPStep struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Validation  string `json:"validation"`
	Rollback    string `json:"rollback"`
}

// GenerateSOP calls QInfra to generate standard operating procedures
func (c *QInfraClient) GenerateSOP(ctx context.Context, sopRequest map[string]interface{}) (*SOPResponse, error) {
	url := fmt.Sprintf("%s/sop/generate", c.baseURL)
	
	body, err := json.Marshal(sopRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call QInfra API: %w", err)
	}
	defer resp.Body.Close()
	
	var result SOPResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// ComplianceResponse from QInfra
type ComplianceResponse struct {
	Framework   string              `json:"framework"`
	Score       float64             `json:"score"`
	Passed      int                 `json:"passed"`
	Failed      int                 `json:"failed"`
	Findings    []ComplianceFinding `json:"findings"`
	Remediation []string           `json:"remediation"`
}

// ValidateCompliance calls QInfra to validate compliance
func (c *QInfraClient) ValidateCompliance(ctx context.Context, complianceRequest map[string]interface{}) (*ComplianceResponse, error) {
	url := fmt.Sprintf("%s/compliance/validate", c.baseURL)
	
	body, err := json.Marshal(complianceRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call QInfra API: %w", err)
	}
	defer resp.Body.Close()
	
	var result ComplianceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// CostResponse from QInfra
type CostResponse struct {
	Optimizations       []Optimization `json:"optimizations"`
	TotalMonthlySavings float64       `json:"total_monthly_savings"`
	ROIPercentage       float64       `json:"roi_percentage"`
}

// OptimizeCost calls QInfra to get cost optimizations
func (c *QInfraClient) OptimizeCost(ctx context.Context, costRequest map[string]interface{}) (*CostResponse, error) {
	url := fmt.Sprintf("%s/optimize/cost", c.baseURL)
	
	body, err := json.Marshal(costRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call QInfra API: %w", err)
	}
	defer resp.Body.Close()
	
	var result CostResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// QuantumDropsClient for storing artifacts
type QuantumDropsClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewQuantumDropsClient creates a new QuantumDrops API client
func NewQuantumDropsClient() *QuantumDropsClient {
	baseURL := os.Getenv("QUANTUM_DROPS_URL")
	if baseURL == "" {
		baseURL = "http://quantum-drops.quantumlayer.svc.cluster.local:8090"
	}
	
	return &QuantumDropsClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateDrop stores a new drop in QuantumDrops
func (c *QuantumDropsClient) CreateDrop(ctx context.Context, dropRequest map[string]interface{}) error {
	url := fmt.Sprintf("%s/api/v1/drops", c.baseURL)
	
	body, err := json.Marshal(dropRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call QuantumDrops API: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("QuantumDrops API returned status %d: %s", resp.StatusCode, body)
	}
	
	return nil
}