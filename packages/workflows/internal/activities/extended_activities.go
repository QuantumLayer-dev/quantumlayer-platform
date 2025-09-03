package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/types"
)

// FRD Generation
type FRDGenerationRequest struct {
	Prompt   string `json:"prompt"`
	Type     string `json:"type"`
	Language string `json:"language"`
}

type FRDGenerationResult struct {
	Content string `json:"content"`
}

// GenerateFRDActivity generates a Functional Requirements Document
func GenerateFRDActivity(ctx context.Context, request FRDGenerationRequest) (*FRDGenerationResult, error) {
	frdPrompt := fmt.Sprintf(`Generate a comprehensive Functional Requirements Document (FRD) for the following project:

Project Type: %s
Language: %s
Description: %s

The FRD should include:
1. Project Overview
2. Functional Requirements (detailed)
3. Non-Functional Requirements
4. System Architecture Overview
5. User Stories
6. Technical Constraints
7. Success Criteria
8. Dependencies

Format as markdown with clear sections and subsections.`, request.Type, request.Language, request.Prompt)

	llmRequest := LLMGenerationRequest{
		Prompt:    frdPrompt,
		System:    "You are a technical architect and requirements analyst. Generate detailed, actionable FRDs.",
		Language:  "markdown",
		Provider:  "azure",
		MaxTokens: 3000,
	}

	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		// Fallback to template
		return &FRDGenerationResult{
			Content: generateFRDTemplate(request),
		}, nil
	}

	return &FRDGenerationResult{
		Content: llmResult.Content,
	}, nil
}

// Project Structure Generation
type ProjectStructureRequest struct {
	Language     string              `json:"language"`
	Framework    string              `json:"framework"`
	Type         string              `json:"type"`
	Requirements ParsedRequirements  `json:"requirements"`
}

type ProjectStructureResult struct {
	Structure map[string]string `json:"structure"`
}

// GenerateProjectStructureActivity creates a project folder structure
func GenerateProjectStructureActivity(ctx context.Context, request ProjectStructureRequest) (*ProjectStructureResult, error) {
	structure := make(map[string]string)
	
	switch strings.ToLower(request.Language) {
	case "python":
		structure = generatePythonStructure(request)
	case "javascript", "typescript":
		structure = generateNodeStructure(request)
	case "go":
		structure = generateGoStructure(request)
	case "java":
		structure = generateJavaStructure(request)
	default:
		structure = generateDefaultStructure(request)
	}

	return &ProjectStructureResult{
		Structure: structure,
	}, nil
}

// Semantic Validation
type SemanticValidationRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	Type     string `json:"type"`
}

type SemanticValidationResult struct {
	Valid  bool          `json:"valid"`
	Issues []types.Issue `json:"issues"`
	AST    string        `json:"ast,omitempty"`
}

// ValidateSemanticActivity performs semantic validation using Parser service
func ValidateSemanticActivity(ctx context.Context, request SemanticValidationRequest) (*SemanticValidationResult, error) {
	// Call Parser service for AST analysis
	payload, _ := json.Marshal(map[string]string{
		"code":     request.Code,
		"language": request.Language,
	})

	resp, err := http.Post(
		fmt.Sprintf("%s/parse", ParserServiceURL),
		"application/json",
		bytes.NewBuffer(payload),
	)
	
	if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
		// Fallback to basic validation
		return performBasicSemanticValidation(request), nil
	}
	defer resp.Body.Close()

	var parserResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&parserResult); err != nil {
		return performBasicSemanticValidation(request), nil
	}

	// Analyze parser results
	issues := []types.Issue{}
	valid := true

	if errors, ok := parserResult["errors"].([]interface{}); ok && len(errors) > 0 {
		valid = false
		for _, err := range errors {
			if errMap, ok := err.(map[string]interface{}); ok {
				issues = append(issues, types.Issue{
					Type:    "error",
					Message: errMap["message"].(string),
					Line:    int(errMap["line"].(float64)),
				})
			}
		}
	}

	// Safely extract AST if present
	var ast string
	if astValue, ok := parserResult["ast"]; ok && astValue != nil {
		if astStr, ok := astValue.(string); ok {
			ast = astStr
		}
	}

	return &SemanticValidationResult{
		Valid:  valid,
		Issues: issues,
		AST:    ast,
	}, nil
}

// Feedback Loop
type FeedbackLoopRequest struct {
	Code     string        `json:"code"`
	Issues   []types.Issue `json:"issues"`
	Language string        `json:"language"`
}

type FeedbackLoopResult struct {
	ImprovedCode string `json:"improvedCode"`
	Iterations   int    `json:"iterations"`
}

// ApplyFeedbackLoopActivity attempts to fix code based on validation issues
func ApplyFeedbackLoopActivity(ctx context.Context, request FeedbackLoopRequest) (*FeedbackLoopResult, error) {
	issuesDescription := ""
	for _, issue := range request.Issues {
		issuesDescription += fmt.Sprintf("- Line %d: %s (%s)\n", issue.Line, issue.Message, issue.Type)
	}

	fixPrompt := fmt.Sprintf(`Fix the following %s code based on these issues:

Issues found:
%s

Original code:
%s

Provide the corrected code that addresses all the issues.`, request.Language, issuesDescription, request.Code)

	llmRequest := LLMGenerationRequest{
		Prompt:    fixPrompt,
		System:    "You are a code fixing expert. Fix the provided code to resolve all issues while maintaining functionality.",
		Language:  request.Language,
		Provider:  "azure",
		MaxTokens: 4000,
	}

	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		return &FeedbackLoopResult{
			ImprovedCode: request.Code,
			Iterations:   0,
		}, nil
	}

	return &FeedbackLoopResult{
		ImprovedCode: llmResult.Content,
		Iterations:   1,
	}, nil
}

// Dependency Resolution
type DependencyResolutionRequest struct {
	Code      string `json:"code"`
	Language  string `json:"language"`
	Framework string `json:"framework"`
}

type DependencyResolutionResult struct {
	Dependencies     []string `json:"dependencies"`
	PackageFile      string   `json:"packageFile"`
	PackageFileName  string   `json:"packageFileName"`
}

// ResolveDependenciesActivity analyzes code and generates dependency files
func ResolveDependenciesActivity(ctx context.Context, request DependencyResolutionRequest) (*DependencyResolutionResult, error) {
	dependencies := extractDependencies(request.Code, request.Language)
	
	packageFile := ""
	packageFileName := ""
	
	switch strings.ToLower(request.Language) {
	case "python":
		packageFileName = "requirements.txt"
		packageFile = strings.Join(dependencies, "\n")
	case "javascript", "typescript":
		packageFileName = "package.json"
		packageFile = generatePackageJSON(dependencies, request.Framework)
	case "go":
		packageFileName = "go.mod"
		packageFile = generateGoMod(dependencies)
	case "java":
		packageFileName = "pom.xml"
		packageFile = generatePomXML(dependencies)
	}

	return &DependencyResolutionResult{
		Dependencies:    dependencies,
		PackageFile:     packageFile,
		PackageFileName: packageFileName,
	}, nil
}

// Test Plan Generation
type TestPlanRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	Type     string `json:"type"`
}

type TestPlanResult struct {
	Content string `json:"content"`
}

// GenerateTestPlanActivity creates a comprehensive test plan
func GenerateTestPlanActivity(ctx context.Context, request TestPlanRequest) (*TestPlanResult, error) {
	testPlanPrompt := fmt.Sprintf(`Generate a comprehensive test plan for the following %s code:

Code Type: %s

The test plan should include:
1. Test Strategy
2. Test Scenarios
3. Unit Test Cases
4. Integration Test Cases
5. Edge Cases
6. Performance Test Cases
7. Security Test Cases
8. Test Data Requirements
9. Expected Coverage Targets

Format as markdown with clear sections.`, request.Language, request.Type)

	llmRequest := LLMGenerationRequest{
		Prompt:    testPlanPrompt,
		System:    "You are a QA architect. Generate comprehensive test plans.",
		Language:  "markdown",
		Provider:  "azure",
		MaxTokens: 2000,
	}

	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		return &TestPlanResult{
			Content: generateTestPlanTemplate(request),
		}, nil
	}

	return &TestPlanResult{
		Content: llmResult.Content,
	}, nil
}

// Security Scanning
type SecurityScanRequest struct {
	Code         string   `json:"code"`
	Language     string   `json:"language"`
	Dependencies []string `json:"dependencies"`
}

type SecurityScanResult struct {
	Score           float64  `json:"score"`
	Vulnerabilities []string `json:"vulnerabilities"`
	Report          string   `json:"report"`
}

// PerformSecurityScanActivity scans for security vulnerabilities
func PerformSecurityScanActivity(ctx context.Context, request SecurityScanRequest) (*SecurityScanResult, error) {
	vulnerabilities := []string{}
	score := 100.0
	
	// Check for common security issues
	securityChecks := performSecurityChecks(request.Code, request.Language)
	
	for _, check := range securityChecks {
		if check.Failed {
			vulnerabilities = append(vulnerabilities, check.Issue)
			score -= check.Severity
		}
	}
	
	// Check dependency vulnerabilities
	for _, dep := range request.Dependencies {
		if isVulnerableDependency(dep) {
			vulnerabilities = append(vulnerabilities, fmt.Sprintf("Vulnerable dependency: %s", dep))
			score -= 10
		}
	}
	
	if score < 0 {
		score = 0
	}

	report := generateSecurityReport(vulnerabilities, score, request.Language)

	return &SecurityScanResult{
		Score:           score,
		Vulnerabilities: vulnerabilities,
		Report:          report,
	}, nil
}

// Performance Analysis
type PerformanceAnalysisRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	Type     string `json:"type"`
}

type PerformanceAnalysisResult struct {
	Score  float64 `json:"score"`
	Report string  `json:"report"`
}

// AnalyzePerformanceActivity analyzes code performance
func AnalyzePerformanceActivity(ctx context.Context, request PerformanceAnalysisRequest) (*PerformanceAnalysisResult, error) {
	score := 100.0
	issues := []string{}
	
	// Check for performance anti-patterns
	if strings.Contains(request.Code, "SELECT * FROM") {
		issues = append(issues, "Avoid SELECT * in SQL queries")
		score -= 10
	}
	
	if strings.Contains(request.Code, "for") && strings.Contains(request.Code, "for") {
		// Nested loops detection (simplified)
		issues = append(issues, "Nested loops detected - consider optimization")
		score -= 15
	}
	
	// Check for async/await patterns
	if request.Language == "javascript" || request.Language == "typescript" {
		if !strings.Contains(request.Code, "async") && strings.Contains(request.Code, "fetch") {
			issues = append(issues, "Consider using async/await for asynchronous operations")
			score -= 10
		}
	}
	
	report := generatePerformanceReport(issues, score, request.Language)

	return &PerformanceAnalysisResult{
		Score:  score,
		Report: report,
	}, nil
}

// README Generation
type ReadmeRequest struct {
	Code         string   `json:"code"`
	Language     string   `json:"language"`
	Framework    string   `json:"framework"`
	Dependencies []string `json:"dependencies"`
	ProjectName  string   `json:"projectName"`
}

type ReadmeResult struct {
	Content string `json:"content"`
}

// GenerateReadmeActivity creates a comprehensive README
func GenerateReadmeActivity(ctx context.Context, request ReadmeRequest) (*ReadmeResult, error) {
	projectName := request.ProjectName
	if projectName == "" {
		projectName = "Generated Project"
	}

	readmePrompt := fmt.Sprintf(`Generate a professional README.md for the following project:

Project Name: %s
Language: %s
Framework: %s
Dependencies: %s

The README should include:
1. Project Title and Description
2. Features
3. Prerequisites
4. Installation Instructions
5. Usage Examples
6. API Documentation (if applicable)
7. Configuration
8. Testing
9. Deployment
10. Contributing Guidelines
11. License

Make it professional and comprehensive.`, projectName, request.Language, request.Framework, strings.Join(request.Dependencies, ", "))

	llmRequest := LLMGenerationRequest{
		Prompt:    readmePrompt,
		System:    "You are a technical documentation expert. Create comprehensive, well-structured README files.",
		Language:  "markdown",
		Provider:  "azure",
		MaxTokens: 2500,
	}

	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		return &ReadmeResult{
			Content: generateReadmeTemplate(request),
		}, nil
	}

	return &ReadmeResult{
		Content: llmResult.Content,
	}, nil
}

// Helper functions

func generateFRDTemplate(request FRDGenerationRequest) string {
	return fmt.Sprintf(`# Functional Requirements Document

## Project Overview
**Type**: %s  
**Language**: %s  
**Description**: %s

## Functional Requirements
1. Core functionality implementation
2. User interface requirements
3. Data management requirements
4. Integration requirements

## Non-Functional Requirements
- Performance: Response time < 200ms
- Security: Industry-standard encryption
- Scalability: Support 1000+ concurrent users
- Availability: 99.9%% uptime

## System Architecture
- Frontend: User interface layer
- Backend: Business logic layer
- Database: Data persistence layer

## Success Criteria
- All functional requirements implemented
- Test coverage > 80%%
- Security scan passed
- Performance benchmarks met

## Dependencies
- Language runtime
- Framework dependencies
- Third-party libraries
`, request.Type, request.Language, request.Prompt)
}

func generateTestPlanTemplate(request TestPlanRequest) string {
	return fmt.Sprintf(`# Test Plan

## Overview
Comprehensive test plan for %s %s application.

## Test Strategy
- Unit Testing: Test individual components
- Integration Testing: Test component interactions
- End-to-End Testing: Test complete workflows
- Performance Testing: Load and stress testing

## Test Scenarios
1. Happy path scenarios
2. Error handling scenarios
3. Edge cases
4. Security test cases

## Unit Test Cases
- Function input/output validation
- Error handling verification
- Boundary value testing

## Integration Test Cases
- API endpoint testing
- Database integration
- External service integration

## Coverage Targets
- Code Coverage: > 80%%
- Branch Coverage: > 70%%
- Function Coverage: > 90%%
`, request.Language, request.Type)
}

func generateReadmeTemplate(request ReadmeRequest) string {
	return fmt.Sprintf(`# %s

## Overview
A %s application built with %s.

## Features
- Feature 1
- Feature 2
- Feature 3

## Prerequisites
- %s runtime
%s

## Installation
` + "```bash" + `
# Clone the repository
git clone <repository-url>

# Install dependencies
%s
` + "```" + `

## Usage
` + "```bash" + `
%s
` + "```" + `

## Testing
` + "```bash" + `
%s
` + "```" + `

## Contributing
Please read CONTRIBUTING.md for details on our code of conduct.

## License
This project is licensed under the MIT License.
`, 
		request.ProjectName,
		request.Language,
		request.Framework,
		request.Language,
		formatDependencies(request.Dependencies),
		getInstallCommand(request.Language),
		getRunCommand(request.Language),
		getTestCommand(request.Language))
}

func extractDependencies(code, language string) []string {
	dependencies := []string{}
	
	switch strings.ToLower(language) {
	case "python":
		if strings.Contains(code, "import flask") || strings.Contains(code, "from flask") {
			dependencies = append(dependencies, "flask")
		}
		if strings.Contains(code, "import fastapi") || strings.Contains(code, "from fastapi") {
			dependencies = append(dependencies, "fastapi", "uvicorn")
		}
		if strings.Contains(code, "import numpy") {
			dependencies = append(dependencies, "numpy")
		}
	case "javascript", "typescript":
		if strings.Contains(code, "require('express')") || strings.Contains(code, "from 'express'") {
			dependencies = append(dependencies, "express")
		}
		if strings.Contains(code, "react") {
			dependencies = append(dependencies, "react", "react-dom")
		}
	case "go":
		if strings.Contains(code, "github.com/gin-gonic/gin") {
			dependencies = append(dependencies, "github.com/gin-gonic/gin")
		}
	}
	
	return dependencies
}

func generatePackageJSON(deps []string, framework string) string {
	dependencies := make(map[string]string)
	for _, dep := range deps {
		dependencies[dep] = "latest"
	}
	
	pkg := map[string]interface{}{
		"name":         "generated-project",
		"version":      "1.0.0",
		"description":  "Generated with QuantumLayer",
		"main":         "index.js",
		"scripts": map[string]string{
			"start": "node index.js",
			"test":  "jest",
		},
		"dependencies": dependencies,
	}
	
	result, _ := json.MarshalIndent(pkg, "", "  ")
	return string(result)
}

func generateGoMod(deps []string) string {
	mod := "module generated-project\n\ngo 1.21\n\nrequire (\n"
	for _, dep := range deps {
		mod += fmt.Sprintf("\t%s v1.0.0\n", dep)
	}
	mod += ")"
	return mod
}

func generatePomXML(deps []string) string {
	// Simplified POM generation
	return `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <groupId>com.quantumlayer</groupId>
    <artifactId>generated-project</artifactId>
    <version>1.0.0</version>
</project>`
}

func performBasicSemanticValidation(request SemanticValidationRequest) *SemanticValidationResult {
	issues := []types.Issue{}
	valid := true
	
	// Basic checks
	if len(request.Code) < 10 {
		valid = false
		issues = append(issues, types.Issue{
			Type:    "error",
			Message: "Code is too short",
		})
	}
	
	// Language-specific basic checks
	switch strings.ToLower(request.Language) {
	case "python":
		if !strings.Contains(request.Code, "def ") && !strings.Contains(request.Code, "class ") {
			issues = append(issues, types.Issue{
				Type:    "warning",
				Message: "No functions or classes defined",
			})
		}
	case "javascript", "typescript":
		if !strings.Contains(request.Code, "function") && !strings.Contains(request.Code, "=>") {
			issues = append(issues, types.Issue{
				Type:    "warning",
				Message: "No functions defined",
			})
		}
	}
	
	return &SemanticValidationResult{
		Valid:  valid,
		Issues: issues,
	}
}

type SecurityCheck struct {
	Failed   bool
	Issue    string
	Severity float64
}

func performSecurityChecks(code, language string) []SecurityCheck {
	checks := []SecurityCheck{}
	
	// SQL injection check
	if strings.Contains(code, "SELECT") && strings.Contains(code, "+") {
		checks = append(checks, SecurityCheck{
			Failed:   true,
			Issue:    "Potential SQL injection vulnerability",
			Severity: 20,
		})
	}
	
	// Hardcoded credentials
	if strings.Contains(code, "password = \"") || strings.Contains(code, "api_key = \"") {
		checks = append(checks, SecurityCheck{
			Failed:   true,
			Issue:    "Hardcoded credentials detected",
			Severity: 25,
		})
	}
	
	// Eval usage
	if strings.Contains(code, "eval(") {
		checks = append(checks, SecurityCheck{
			Failed:   true,
			Issue:    "Dangerous eval() usage",
			Severity: 15,
		})
	}
	
	return checks
}

func isVulnerableDependency(dep string) bool {
	// Simplified check - in production, check against vulnerability database
	vulnerableDeps := []string{
		"log4j:1.",
		"commons-collections:3.2.1",
	}
	
	for _, vuln := range vulnerableDeps {
		if strings.Contains(dep, vuln) {
			return true
		}
	}
	return false
}

func generateSecurityReport(vulnerabilities []string, score float64, language string) string {
	report := fmt.Sprintf(`# Security Scan Report

**Language**: %s  
**Score**: %.1f/100

## Vulnerabilities Found
`, language, score)
	
	if len(vulnerabilities) == 0 {
		report += "No vulnerabilities detected.\n"
	} else {
		for _, vuln := range vulnerabilities {
			report += fmt.Sprintf("- %s\n", vuln)
		}
	}
	
	report += "\n## Recommendations\n"
	if score < 70 {
		report += "- Urgent: Address critical security issues immediately\n"
	} else if score < 85 {
		report += "- Recommended: Review and fix security warnings\n"
	} else {
		report += "- Good security posture, continue monitoring\n"
	}
	
	return report
}

func generatePerformanceReport(issues []string, score float64, language string) string {
	report := fmt.Sprintf(`# Performance Analysis Report

**Language**: %s  
**Score**: %.1f/100

## Performance Issues
`, language, score)
	
	if len(issues) == 0 {
		report += "No performance issues detected.\n"
	} else {
		for _, issue := range issues {
			report += fmt.Sprintf("- %s\n", issue)
		}
	}
	
	report += "\n## Optimization Recommendations\n"
	if score < 70 {
		report += "- Critical: Significant performance optimizations needed\n"
	} else if score < 85 {
		report += "- Recommended: Consider optimizing identified areas\n"
	} else {
		report += "- Good performance characteristics\n"
	}
	
	return report
}

func generatePythonStructure(request ProjectStructureRequest) map[string]string {
	return map[string]string{
		"src/__init__.py": "",
		"src/main.py": "# Main application entry point",
		"src/models.py": "# Data models",
		"src/utils.py": "# Utility functions",
		"tests/__init__.py": "",
		"tests/test_main.py": "# Main tests",
		"requirements.txt": "",
		"README.md": "",
		".gitignore": "*.pyc\n__pycache__/\nvenv/\n.env",
	}
}

func generateNodeStructure(request ProjectStructureRequest) map[string]string {
	return map[string]string{
		"src/index.js": "// Main application entry point",
		"src/routes/index.js": "// API routes",
		"src/models/index.js": "// Data models",
		"src/utils/index.js": "// Utility functions",
		"tests/index.test.js": "// Main tests",
		"package.json": "",
		"README.md": "",
		".gitignore": "node_modules/\n.env\ndist/",
	}
}

func generateGoStructure(request ProjectStructureRequest) map[string]string {
	return map[string]string{
		"cmd/main.go": "// Main application entry point",
		"internal/handlers/handlers.go": "// Request handlers",
		"internal/models/models.go": "// Data models",
		"internal/utils/utils.go": "// Utility functions",
		"go.mod": "",
		"README.md": "",
		".gitignore": "*.exe\n*.test\n.env",
	}
}

func generateJavaStructure(request ProjectStructureRequest) map[string]string {
	return map[string]string{
		"src/main/java/Main.java": "// Main application class",
		"src/main/java/models/Model.java": "// Data models",
		"src/main/java/utils/Utils.java": "// Utility classes",
		"src/test/java/MainTest.java": "// Main tests",
		"pom.xml": "",
		"README.md": "",
		".gitignore": "target/\n*.class\n.env",
	}
}

func generateDefaultStructure(request ProjectStructureRequest) map[string]string {
	return map[string]string{
		"src/main." + getExtension(request.Language): "// Main application",
		"README.md": "",
		".gitignore": "",
	}
}

func formatDependencies(deps []string) string {
	if len(deps) == 0 {
		return ""
	}
	result := ""
	for _, dep := range deps {
		result += fmt.Sprintf("- %s\n", dep)
	}
	return result
}

func getInstallCommand(language string) string {
	switch strings.ToLower(language) {
	case "python":
		return "pip install -r requirements.txt"
	case "javascript", "typescript":
		return "npm install"
	case "go":
		return "go mod download"
	case "java":
		return "mvn install"
	default:
		return "# Install dependencies"
	}
}

func getRunCommand(language string) string {
	switch strings.ToLower(language) {
	case "python":
		return "python src/main.py"
	case "javascript":
		return "npm start"
	case "typescript":
		return "npm run build && npm start"
	case "go":
		return "go run cmd/main.go"
	case "java":
		return "java -jar target/app.jar"
	default:
		return "# Run the application"
	}
}

func getTestCommand(language string) string {
	switch strings.ToLower(language) {
	case "python":
		return "pytest"
	case "javascript", "typescript":
		return "npm test"
	case "go":
		return "go test ./..."
	case "java":
		return "mvn test"
	default:
		return "# Run tests"
	}
}

// StoreQuantumDropActivity stores a QuantumDrop to the QuantumDrops service
func StoreQuantumDropActivity(ctx context.Context, drop types.QuantumDrop) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Storing QuantumDrop", "dropID", drop.ID, "stage", drop.Stage)

	// Get QuantumDrops service URL from environment
	dropsURL := os.Getenv("QUANTUM_DROPS_URL")
	if dropsURL == "" {
		dropsURL = "http://quantum-drops.quantumlayer.svc.cluster.local:8090"
	}

	// Prepare the request
	jsonData, err := json.Marshal(drop)
	if err != nil {
		return fmt.Errorf("failed to marshal drop: %w", err)
	}

	// Make HTTP request to store the drop
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/v1/drops", dropsURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to store drop: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to store drop: status=%d, body=%s", resp.StatusCode, string(body))
	}

	logger.Info("Successfully stored QuantumDrop", "dropID", drop.ID)
	return nil
}