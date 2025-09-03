package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/gin-gonic/gin"
)

type GenerateRequest struct {
	// Direct format
	Prompt    string `json:"prompt,omitempty"`
	System    string `json:"system,omitempty"`
	
	// Messages format (from activities)
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages,omitempty"`
	
	Provider    string  `json:"provider,omitempty"`
	Model       string  `json:"model,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

type GenerateResponse struct {
	Content          string `json:"content"`
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
}

var (
	azureEndpoint   string
	azureKey        string
	azureDeployment string
	bedrockClient   *bedrockruntime.Client
)

func init() {
	// Azure OpenAI config
	azureEndpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	if azureEndpoint == "" {
		azureEndpoint = "https://openai-uk-surya.openai.azure.com"
	}
	azureKey = os.Getenv("AZURE_OPENAI_KEY")
	azureDeployment = os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	if azureDeployment == "" {
		azureDeployment = "gpt-4.1"
	}

	// AWS Bedrock config
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	if err == nil {
		bedrockClient = bedrockruntime.NewFromConfig(cfg)
	}
}

func main() {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	r.POST("/generate", handleGenerate)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting LLM Router on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func handleGenerate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert messages format to prompt/system format
	if len(req.Messages) > 0 {
		for _, msg := range req.Messages {
			if msg.Role == "system" {
				req.System = msg.Content
			} else if msg.Role == "user" {
				if req.Prompt != "" {
					req.Prompt += "\n"
				}
				req.Prompt += msg.Content
			}
		}
	}

	// Validate we have a prompt
	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	// Default provider
	if req.Provider == "" {
		req.Provider = "azure"
	}

	// Default max tokens
	if req.MaxTokens == 0 {
		req.MaxTokens = 4000
	}

	var resp GenerateResponse
	var err error

	switch req.Provider {
	case "azure":
		resp, err = callAzureOpenAI(req)
	case "aws", "bedrock":
		resp, err = callAWSBedrock(req)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider: " + req.Provider})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func callAzureOpenAI(req GenerateRequest) (GenerateResponse, error) {
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-06-01",
		azureEndpoint, azureDeployment)

	messages := []map[string]string{}
	if req.System != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": req.System,
		})
	}
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": req.Prompt,
	})

	payload := map[string]interface{}{
		"messages":    messages,
		"max_tokens":  req.MaxTokens,
		"temperature": 0.7,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return GenerateResponse{}, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return GenerateResponse{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", azureKey)

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return GenerateResponse{}, err
	}

	if httpResp.StatusCode != http.StatusOK {
		return GenerateResponse{}, fmt.Errorf("azure API error: %s", string(body))
	}

	var azureResp map[string]interface{}
	if err := json.Unmarshal(body, &azureResp); err != nil {
		return GenerateResponse{}, err
	}

	// Extract response
	choices := azureResp["choices"].([]interface{})
	if len(choices) == 0 {
		return GenerateResponse{}, fmt.Errorf("no response from Azure OpenAI")
	}

	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content := message["content"].(string)

	usage := azureResp["usage"].(map[string]interface{})
	promptTokens := int(usage["prompt_tokens"].(float64))
	completionTokens := int(usage["completion_tokens"].(float64))

	return GenerateResponse{
		Content:          content,
		Provider:         "azure",
		Model:            azureDeployment,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}, nil
}

func callAWSBedrock(req GenerateRequest) (GenerateResponse, error) {
	if bedrockClient == nil {
		return GenerateResponse{}, fmt.Errorf("AWS Bedrock client not initialized")
	}

	modelID := "anthropic.claude-3-5-sonnet-20241022-v2:0"
	if req.Model != "" {
		modelID = req.Model
	}

	// Build Claude messages
	messages := []map[string]interface{}{}
	if req.System != "" {
		// Claude 3 uses system parameter separately
	}
	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": req.Prompt,
	})

	payload := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        req.MaxTokens,
		"messages":          messages,
		"temperature":       0.7,
	}

	if req.System != "" {
		payload["system"] = req.System
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return GenerateResponse{}, err
	}

	result, err := bedrockClient.InvokeModel(context.TODO(), &bedrockruntime.InvokeModelInput{
		ModelId:     &modelID,
		Body:        jsonData,
		ContentType: stringPtr("application/json"),
	})
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("failed to invoke Bedrock: %w", err)
	}

	var bedrockResp map[string]interface{}
	if err := json.Unmarshal(result.Body, &bedrockResp); err != nil {
		return GenerateResponse{}, err
	}

	content := ""
	if contentArray, ok := bedrockResp["content"].([]interface{}); ok && len(contentArray) > 0 {
		if firstContent, ok := contentArray[0].(map[string]interface{}); ok {
			if text, ok := firstContent["text"].(string); ok {
				content = text
			}
		}
	}

	// Extract usage if available
	usage := bedrockResp["usage"].(map[string]interface{})
	inputTokens := int(usage["input_tokens"].(float64))
	outputTokens := int(usage["output_tokens"].(float64))

	return GenerateResponse{
		Content:          content,
		Provider:         "aws",
		Model:            modelID,
		PromptTokens:     inputTokens,
		CompletionTokens: outputTokens,
		TotalTokens:      inputTokens + outputTokens,
	}, nil
}

func stringPtr(s string) *string {
	return &s
}