package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/types"
)

const (
	// PreviewServiceURL is the URL for the preview service
	PreviewServiceURL = "http://preview-service:3000"
)

// PreviewRequest represents a request to generate a preview URL
type PreviewRequest struct {
	WorkflowID string `json:"workflowId"`
	CapsuleID  string `json:"capsuleId,omitempty"`
	TTLMinutes int    `json:"ttlMinutes"`
}

// PreviewResult represents the result of preview URL generation
type PreviewResult struct {
	Success       bool   `json:"success"`
	PreviewID     string `json:"previewId"`
	PreviewURL    string `json:"previewUrl"`
	ShareableURL  string `json:"shareableUrl"`
	ExpiresAt     string `json:"expiresAt"`
	TTLMinutes    int    `json:"ttlMinutes"`
	Message       string `json:"message"`
}

// GeneratePreviewActivity generates a preview URL for the workflow artifacts
func GeneratePreviewActivity(ctx context.Context, workflowID string, capsuleID string) (*PreviewResult, error) {
	fmt.Printf("[GeneratePreviewActivity] Creating preview for workflow: %s, capsule: %s\n", workflowID, capsuleID)
	
	// Create the request
	request := PreviewRequest{
		WorkflowID: workflowID,
		CapsuleID:  capsuleID,
		TTLMinutes: 60, // Default 1 hour TTL
	}
	
	// Use retry logic for preview generation
	operation := &RetryableOperation[*PreviewResult]{
		Name:   "preview-generation",
		Config: ServiceRetryConfig(),
		Operation: func(ctx context.Context) (*PreviewResult, error) {
			return callPreviewService(request)
		},
		Fallback: func(ctx context.Context, err error) (*PreviewResult, error) {
			// Generate a fallback preview URL
			return generateFallbackPreview(workflowID, capsuleID), nil
		},
	}
	
	return operation.Execute(ctx)
}

// callPreviewService makes the actual HTTP call to the preview service
func callPreviewService(request PreviewRequest) (*PreviewResult, error) {
	// Marshal request
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal preview request: %w", err)
	}
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	// Create request
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/preview", PreviewServiceURL), bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create preview request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, ClassifyError(err, "preview-service")
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ClassifyHTTPError(resp.StatusCode, body, "preview-service")
	}
	
	// Parse response
	var result PreviewResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode preview response: %w", err)
	}
	
	fmt.Printf("[GeneratePreviewActivity] Preview created successfully: %s\n", result.ShareableURL)
	return &result, nil
}

// generateFallbackPreview generates a fallback preview URL when service is unavailable
func generateFallbackPreview(workflowID string, capsuleID string) *PreviewResult {
	// Generate a simple preview ID
	previewID := fmt.Sprintf("preview-%s", workflowID[:8])
	
	// Use the known service URL
	baseURL := "http://192.168.1.217:30900"
	
	return &PreviewResult{
		Success:      true,
		PreviewID:    previewID,
		PreviewURL:   fmt.Sprintf("%s/preview/%s", baseURL, workflowID),
		ShareableURL: fmt.Sprintf("%s/p/%s", baseURL, previewID),
		ExpiresAt:    time.Now().Add(60 * time.Minute).Format(time.RFC3339),
		TTLMinutes:   60,
		Message:      "Preview URL generated (fallback mode)",
	}
}

// StorePreviewMetadataActivity stores preview metadata in QuantumDrops
func StorePreviewMetadataActivity(ctx context.Context, workflowID string, previewResult *PreviewResult) error {
	fmt.Printf("[StorePreviewMetadataActivity] Storing preview metadata for workflow: %s\n", workflowID)
	
	// Create a QuantumDrop for the preview metadata
	drop := types.QuantumDrop{
		ID:         fmt.Sprintf("drop-%s-preview", workflowID),
		Stage:      "preview",
		Timestamp:  time.Now(),
		Type:       "preview",
		WorkflowID: workflowID,
		Artifact:   previewResult.ShareableURL,
		Metadata: map[string]interface{}{
			"preview_id":    previewResult.PreviewID,
			"preview_url":   previewResult.PreviewURL,
			"shareable_url": previewResult.ShareableURL,
			"expires_at":    previewResult.ExpiresAt,
			"ttl_minutes":   previewResult.TTLMinutes,
		},
	}
	
	// Store in QuantumDrops
	return StoreQuantumDropActivity(ctx, drop)
}