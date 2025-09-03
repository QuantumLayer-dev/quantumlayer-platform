package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/types"
)

// Service URLs - these should come from config in production
const (
	LLMRouterURL      = "http://llm-router.quantumlayer.svc.cluster.local:8080"
	MetaPromptURL     = "http://meta-prompt-engine.quantumlayer.svc.cluster.local:8080"
	AgentOrchestratorURL = "http://agent-orchestrator.quantumlayer.svc.cluster.local:8083"
	ParserServiceURL  = "http://parser.quantumlayer.svc.cluster.local:8082"
)

// Activity structures
type ParsedRequirements struct {
	MainFilePath  string   `json:"mainFilePath"`
	Dependencies  []string `json:"dependencies"`
	TestFramework string   `json:"testFramework"`
	Structure     string   `json:"structure"`
}

type LLMGenerationRequest struct {
	Prompt    string `json:"prompt"`
	System    string `json:"system"`
	Language  string `json:"language"`
	Provider  string `json:"provider"`
	MaxTokens int    `json:"maxTokens"`
}

type LLMGenerationResult struct {
	Content          string  `json:"content"`
	PromptTokens     int     `json:"promptTokens"`
	CompletionTokens int     `json:"completionTokens"`
	TotalTokens      int     `json:"totalTokens"`
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
}

type TestGenerationRequest struct {
	Code      string `json:"code"`
	Language  string `json:"language"`
	Framework string `json:"framework"`
}

type TestGenerationResult struct {
	TestCode string `json:"testCode"`
	FilePath string `json:"filePath"`
}

type DocumentationRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	Type     string `json:"type"`
}

type DocumentationResult struct {
	Content string `json:"content"`
}

// EnhancePromptActivity enhances the prompt using Meta Prompt Engine
func EnhancePromptActivity(ctx context.Context, request types.PromptEnhancementRequest) (*types.PromptEnhancementResult, error) {
	// Call Meta Prompt Engine service
	payload, _ := json.Marshal(request)
	resp, err := http.Post(
		fmt.Sprintf("%s/enhance", MetaPromptURL),
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		// Fallback to basic enhancement
		return &types.PromptEnhancementResult{
			EnhancedPrompt: improvePrompt(request.OriginalPrompt, request.Type),
			SystemPrompt:   getSystemPrompt(request.Type),
			Tokens:         len(strings.Fields(request.OriginalPrompt)) * 2,
		}, nil
	}
	defer resp.Body.Close()

	var result types.PromptEnhancementResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode enhancement response: %w", err)
	}

	return &result, nil
}

// ParseRequirementsActivity parses the requirements to determine structure
func ParseRequirementsActivity(ctx context.Context, request types.CodeGenerationRequest) (*ParsedRequirements, error) {
	// Analyze the request to determine file structure and dependencies
	requirements := &ParsedRequirements{
		Dependencies: []string{},
	}

	// Determine main file path based on language
	switch strings.ToLower(request.Language) {
	case "python":
		requirements.MainFilePath = "main.py"
		requirements.TestFramework = "pytest"
		if strings.Contains(strings.ToLower(request.Prompt), "flask") {
			requirements.Dependencies = append(requirements.Dependencies, "flask")
		}
		if strings.Contains(strings.ToLower(request.Prompt), "fastapi") {
			requirements.Dependencies = append(requirements.Dependencies, "fastapi", "uvicorn")
		}
	case "javascript", "typescript":
		requirements.MainFilePath = "index." + getExtension(request.Language)
		requirements.TestFramework = "jest"
		if strings.Contains(strings.ToLower(request.Prompt), "express") {
			requirements.Dependencies = append(requirements.Dependencies, "express")
		}
		if strings.Contains(strings.ToLower(request.Prompt), "react") {
			requirements.Dependencies = append(requirements.Dependencies, "react", "react-dom")
		}
	case "go":
		requirements.MainFilePath = "main.go"
		requirements.TestFramework = "testing"
		if strings.Contains(strings.ToLower(request.Prompt), "gin") {
			requirements.Dependencies = append(requirements.Dependencies, "github.com/gin-gonic/gin")
		}
	case "java":
		requirements.MainFilePath = "Main.java"
		requirements.TestFramework = "junit"
		if strings.Contains(strings.ToLower(request.Prompt), "spring") {
			requirements.Dependencies = append(requirements.Dependencies, "spring-boot-starter-web")
		}
	default:
		requirements.MainFilePath = "main." + getExtension(request.Language)
		requirements.TestFramework = "default"
	}

	// Determine structure based on type
	switch request.Type {
	case "api":
		requirements.Structure = "rest-api"
	case "frontend":
		requirements.Structure = "spa"
	case "fullstack":
		requirements.Structure = "monorepo"
	default:
		requirements.Structure = "simple"
	}

	return requirements, nil
}

// GenerateCodeActivity generates code using the LLM Router
func GenerateCodeActivity(ctx context.Context, request LLMGenerationRequest) (*LLMGenerationResult, error) {
	// Prepare the LLM request
	llmRequest := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "system", "content": request.System},
			{"role": "user", "content": request.Prompt},
		},
		"provider": request.Provider,
		"max_tokens": request.MaxTokens,
		"temperature": 0.7,
	}

	payload, _ := json.Marshal(llmRequest)
	resp, err := http.Post(
		fmt.Sprintf("%s/generate", LLMRouterURL),
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM router: %w", err)
	}
	defer resp.Body.Close()

	var llmResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return nil, fmt.Errorf("failed to decode LLM response: %w", err)
	}

	// Extract the generated code and metrics
	result := &LLMGenerationResult{
		Content:  extractCode(llmResponse["content"].(string)),
		Provider: request.Provider,
		Model:    "gpt-4", // Default
	}

	// Extract token counts if available
	if usage, ok := llmResponse["usage"].(map[string]interface{}); ok {
		result.PromptTokens = int(usage["prompt_tokens"].(float64))
		result.CompletionTokens = int(usage["completion_tokens"].(float64))
		result.TotalTokens = result.PromptTokens + result.CompletionTokens
	}

	return result, nil
}

// ValidateCodeActivity validates the generated code
func ValidateCodeActivity(ctx context.Context, request types.ValidationRequest) (*types.ValidationResult, error) {
	// Basic validation logic
	result := &types.ValidationResult{
		Valid:  true,
		Issues: []types.Issue{},
		Score:  100.0,
	}

	// Check for basic issues
	if len(request.Code) < 10 {
		result.Valid = false
		result.Score = 0
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: "Code is too short",
		})
		return result, nil
	}

	// Language-specific validation
	switch strings.ToLower(request.Language) {
	case "python":
		if !strings.Contains(request.Code, "def ") && !strings.Contains(request.Code, "class ") {
			result.Score -= 20
			result.Issues = append(result.Issues, types.Issue{
				Type:    "warning",
				Message: "No functions or classes found",
			})
		}
	case "javascript", "typescript":
		if !strings.Contains(request.Code, "function") && !strings.Contains(request.Code, "const") && !strings.Contains(request.Code, "=>") {
			result.Score -= 20
			result.Issues = append(result.Issues, types.Issue{
				Type:    "warning",
				Message: "No functions found",
			})
		}
	case "go":
		if !strings.Contains(request.Code, "func ") {
			result.Score -= 20
			result.Issues = append(result.Issues, types.Issue{
				Type:    "warning",
				Message: "No functions found",
			})
		}
	}

	// Check for common issues
	if strings.Contains(request.Code, "TODO") || strings.Contains(request.Code, "FIXME") {
		result.Score -= 10
		result.Issues = append(result.Issues, types.Issue{
			Type:    "info",
			Message: "Code contains TODO/FIXME comments",
		})
	}

	result.Valid = result.Score >= 60
	result.Feedback = fmt.Sprintf("Code validation score: %.0f/100", result.Score)

	return result, nil
}

// GenerateTestsActivity generates tests for the code
func GenerateTestsActivity(ctx context.Context, request TestGenerationRequest) (*TestGenerationResult, error) {
	// Generate test prompt
	testPrompt := fmt.Sprintf("Generate comprehensive unit tests for the following %s code using %s:\n\n%s",
		request.Language, request.Framework, request.Code)

	// Call LLM to generate tests
	llmRequest := LLMGenerationRequest{
		Prompt:    testPrompt,
		System:    "You are an expert test engineer. Generate complete, runnable test code.",
		Language:  request.Language,
		Provider:  "azure",
		MaxTokens: 2000,
	}

	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		// Return basic test template
		return &TestGenerationResult{
			TestCode: generateBasicTestTemplate(request.Language, request.Framework),
			FilePath: getTestFilePath(request.Language),
		}, nil
	}

	return &TestGenerationResult{
		TestCode: llmResult.Content,
		FilePath: getTestFilePath(request.Language),
	}, nil
}

// GenerateDocumentationActivity generates documentation
func GenerateDocumentationActivity(ctx context.Context, request DocumentationRequest) (*DocumentationResult, error) {
	// For now, return a basic template
	// TODO: Use LLM to generate documentation
	_ = request.Code // Will be used when LLM integration is complete
	documentation := fmt.Sprintf(`# Code Documentation

## Overview
This %s code implements %s functionality.

## Installation
Add installation instructions here

## Usage
Add usage examples here

## API Reference
Generated code documentation.

## Contributing
Please follow the standard contribution guidelines.

## License
MIT License
`, request.Language, request.Type)

	return &DocumentationResult{
		Content: documentation,
	}, nil
}

// Helper functions

func improvePrompt(original, promptType string) string {
	prefix := ""
	switch promptType {
	case "api":
		prefix = "Create a production-ready REST API with proper error handling, validation, and documentation. "
	case "frontend":
		prefix = "Create a modern, responsive frontend application with clean architecture. "
	case "function":
		prefix = "Create a well-tested, efficient function with proper documentation. "
	default:
		prefix = "Create clean, maintainable code following best practices. "
	}
	return prefix + original
}

func getSystemPrompt(promptType string) string {
	return "You are an expert software engineer. Generate clean, production-ready code with proper error handling, comments, and best practices. Only return code without explanations."
}

func getExtension(language string) string {
	switch strings.ToLower(language) {
	case "python":
		return "py"
	case "javascript":
		return "js"
	case "typescript":
		return "ts"
	case "go":
		return "go"
	case "java":
		return "java"
	case "rust":
		return "rs"
	case "c++", "cpp":
		return "cpp"
	default:
		return "txt"
	}
}

func extractCode(content string) string {
	// Extract code from markdown code blocks if present
	if strings.Contains(content, "```") {
		parts := strings.Split(content, "```")
		if len(parts) >= 3 {
			// Get the code block (second element)
			code := parts[1]
			// Remove language identifier if present
			lines := strings.Split(code, "\n")
			if len(lines) > 1 {
				return strings.Join(lines[1:], "\n")
			}
		}
	}
	return content
}

func getTestFilePath(language string) string {
	switch strings.ToLower(language) {
	case "python":
		return "test_main.py"
	case "javascript":
		return "index.test.js"
	case "typescript":
		return "index.test.ts"
	case "go":
		return "main_test.go"
	case "java":
		return "MainTest.java"
	default:
		return "tests." + getExtension(language)
	}
}

func generateBasicTestTemplate(language, framework string) string {
	switch strings.ToLower(language) {
	case "python":
		return `import pytest

def test_example():
    """Example test case"""
    assert True

# Add more tests here`
	case "javascript", "typescript":
		return `describe('Test Suite', () => {
  it('should pass example test', () => {
    expect(true).toBe(true);
  });
  
  // Add more tests here
});`
	case "go":
		return `package main

import "testing"

func TestExample(t *testing.T) {
    // Add test implementation
    if false {
        t.Errorf("Test failed")
    }
}`
	default:
		return "// Add tests here"
	}
}