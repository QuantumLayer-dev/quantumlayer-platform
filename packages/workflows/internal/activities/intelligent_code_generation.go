package activities

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/types"
)

// IntelligentCodeGenerationRequest for multi-stage enterprise code generation
type IntelligentCodeGenerationRequest struct {
	ProjectName   string              `json:"project_name"`
	Description   string              `json:"description"`
	Language      string              `json:"language"`
	Type          string              `json:"type"`
	Requirements  ParsedRequirements  `json:"requirements"`
	Architecture  string              `json:"architecture,omitempty"`
}

// IntelligentCodeGenerationResult contains all generated enterprise files
type IntelligentCodeGenerationResult struct {
	Files       []types.GeneratedFile `json:"files"`
	MainFile    string               `json:"main_file"`
	Structure   map[string]string    `json:"structure"`
	Dependencies []string            `json:"dependencies"`
	Instructions string              `json:"instructions"`
}

// GenerateIntelligentCodeActivity performs multi-stage enterprise code generation
func GenerateIntelligentCodeActivity(ctx context.Context, request IntelligentCodeGenerationRequest) (*IntelligentCodeGenerationResult, error) {
	fmt.Printf("[IntelligentCodeGeneration] Starting multi-stage generation for %s\n", request.ProjectName)
	
	result := &IntelligentCodeGenerationResult{
		Files:        []types.GeneratedFile{},
		Dependencies: []string{},
	}
	
	// Stage 1: Generate main application file
	mainFile, err := generateMainApplicationFile(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate main application: %w", err)
	}
	result.Files = append(result.Files, *mainFile)
	result.MainFile = mainFile.Path
	
	// Stage 2: Generate data models
	modelsFile, err := generateModelsFile(ctx, request)
	if err != nil {
		fmt.Printf("[IntelligentCodeGeneration] WARNING: Models generation failed: %v\n", err)
	} else {
		result.Files = append(result.Files, *modelsFile)
	}
	
	// Stage 3: Generate authentication module
	authFile, err := generateAuthenticationFile(ctx, request)
	if err != nil {
		fmt.Printf("[IntelligentCodeGeneration] WARNING: Auth generation failed: %v\n", err)
	} else {
		result.Files = append(result.Files, *authFile)
	}
	
	// Stage 4: Generate configuration
	configFile, err := generateConfigurationFile(ctx, request)
	if err != nil {
		fmt.Printf("[IntelligentCodeGeneration] WARNING: Config generation failed: %v\n", err)
	} else {
		result.Files = append(result.Files, *configFile)
	}
	
	// Stage 5: Generate requirements/dependencies
	reqFile, err := generateDependenciesFile(ctx, request)
	if err != nil {
		fmt.Printf("[IntelligentCodeGeneration] WARNING: Dependencies generation failed: %v\n", err)
	} else {
		result.Files = append(result.Files, *reqFile)
		// Extract dependencies from requirements.txt content
		deps := strings.Split(reqFile.Content, "\n")
		for _, dep := range deps {
			if dep = strings.TrimSpace(dep); dep != "" && !strings.HasPrefix(dep, "#") {
				result.Dependencies = append(result.Dependencies, dep)
			}
		}
	}
	
	// Stage 6: Generate comprehensive tests
	testFile, err := generateTestsFile(ctx, request, result.Files)
	if err != nil {
		fmt.Printf("[IntelligentCodeGeneration] WARNING: Tests generation failed: %v\n", err)
	} else {
		result.Files = append(result.Files, *testFile)
	}
	
	// Generate deployment instructions
	result.Instructions = generateDeploymentInstructions(request, result.Dependencies)
	
	fmt.Printf("[IntelligentCodeGeneration] SUCCESS: Generated %d enterprise files\n", len(result.Files))
	return result, nil
}

// generateMainApplicationFile creates the core application file
func generateMainApplicationFile(ctx context.Context, request IntelligentCodeGenerationRequest) (*types.GeneratedFile, error) {
	var mainPath string
	var systemPrompt string
	
	switch strings.ToLower(request.Language) {
	case "python":
		mainPath = "app/main.py"
		systemPrompt = buildPythonSystemPrompt()
	case "javascript", "typescript":
		mainPath = "src/app.js"
		systemPrompt = buildNodeSystemPrompt()
	case "go":
		mainPath = "cmd/server/main.go"
		systemPrompt = buildGoSystemPrompt()
	default:
		mainPath = "main.py"
		systemPrompt = buildPythonSystemPrompt()
	}
	
	userPrompt := fmt.Sprintf(`Generate a complete %s main application file for: %s

Description: %s
Type: %s

CRITICAL REQUIREMENTS:
1. Complete, runnable production code - no placeholders
2. Full imports and proper error handling
3. Authentication middleware and JWT support
4. Request/response validation with proper schemas
5. Rate limiting and security headers
6. Health check endpoints
7. Comprehensive logging
8. OpenAPI documentation
9. Database connection with proper ORM setup
10. CORS and security middleware

Generate ONLY the main application file with ALL necessary code.`, 
		request.Language, request.ProjectName, request.Description, request.Type)
	
	llmRequest := LLMGenerationRequest{
		Prompt:     userPrompt,
		System:     systemPrompt,
		Language:   request.Language,
		Provider:   "azure",
		MaxTokens:  12000, // Increased for complete file
	}
	
	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}
	
	// Validate the generated code
	if len(llmResult.Content) < 500 {
		return nil, fmt.Errorf("generated code too short (%d chars), likely incomplete", len(llmResult.Content))
	}
	
	return &types.GeneratedFile{
		Path:     mainPath,
		Content:  llmResult.Content,
		Language: request.Language,
		Type:     "source",
	}, nil
}

// generateModelsFile creates database models/schemas
func generateModelsFile(ctx context.Context, request IntelligentCodeGenerationRequest) (*types.GeneratedFile, error) {
	var modelPath string
	switch strings.ToLower(request.Language) {
	case "python":
		modelPath = "app/models.py"
	case "javascript", "typescript":
		modelPath = "src/models.js"
	case "go":
		modelPath = "internal/models/models.go"
	default:
		modelPath = "models.py"
	}
	
	userPrompt := fmt.Sprintf(`Generate complete database models for: %s

Description: %s
Type: %s

Generate comprehensive data models with:
1. All necessary entity relationships
2. Proper field validation and constraints
3. Database indexes for performance
4. Model methods and properties
5. Serialization/deserialization
6. Migration-friendly structure
7. Foreign key relationships
8. Proper field types and validation

Generate ONLY the models file with complete ORM definitions.`, 
		request.ProjectName, request.Description, request.Type)
	
	llmRequest := LLMGenerationRequest{
		Prompt:     userPrompt,
		System:     buildModelsSystemPrompt(request.Language),
		Language:   request.Language,
		Provider:   "azure",
		MaxTokens:  8000,
	}
	
	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		return nil, err
	}
	
	return &types.GeneratedFile{
		Path:     modelPath,
		Content:  llmResult.Content,
		Language: request.Language,
		Type:     "source",
	}, nil
}

// generateAuthenticationFile creates authentication/authorization module
func generateAuthenticationFile(ctx context.Context, request IntelligentCodeGenerationRequest) (*types.GeneratedFile, error) {
	var authPath string
	switch strings.ToLower(request.Language) {
	case "python":
		authPath = "app/auth.py"
	case "javascript", "typescript":
		authPath = "src/auth.js"
	case "go":
		authPath = "internal/auth/auth.go"
	default:
		authPath = "auth.py"
	}
	
	userPrompt := fmt.Sprintf(`Generate complete authentication system for: %s

Generate production-ready authentication with:
1. JWT token generation and validation
2. Password hashing with bcrypt
3. User registration and login endpoints
4. Token refresh functionality
5. Role-based access control
6. Session management
7. Security headers and CORS
8. Rate limiting for auth endpoints
9. Password strength validation
10. Account lockout protection

Generate ONLY the authentication module with complete implementation.`)
	
	llmRequest := LLMGenerationRequest{
		Prompt:     userPrompt,
		System:     buildAuthSystemPrompt(request.Language),
		Language:   request.Language,
		Provider:   "azure",
		MaxTokens:  8000,
	}
	
	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		return nil, err
	}
	
	return &types.GeneratedFile{
		Path:     authPath,
		Content:  llmResult.Content,
		Language: request.Language,
		Type:     "source",
	}, nil
}

// generateConfigurationFile creates configuration management
func generateConfigurationFile(ctx context.Context, request IntelligentCodeGenerationRequest) (*types.GeneratedFile, error) {
	var configPath string
	switch strings.ToLower(request.Language) {
	case "python":
		configPath = "app/config.py"
	case "javascript", "typescript":
		configPath = "src/config.js"
	case "go":
		configPath = "internal/config/config.go"
	default:
		configPath = "config.py"
	}
	
	userPrompt := `Generate production configuration management with:
1. Environment variable handling
2. Database connection strings
3. JWT secrets and algorithm settings
4. API rate limiting configuration
5. CORS settings
6. Logging configuration
7. Security settings
8. Feature flags
9. External service configurations
10. Validation and defaults

Generate ONLY the configuration module.`
	
	llmRequest := LLMGenerationRequest{
		Prompt:     userPrompt,
		System:     buildConfigSystemPrompt(request.Language),
		Language:   request.Language,
		Provider:   "azure",
		MaxTokens:  6000,
	}
	
	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		return nil, err
	}
	
	return &types.GeneratedFile{
		Path:     configPath,
		Content:  llmResult.Content,
		Language: request.Language,
		Type:     "config",
	}, nil
}

// generateDependenciesFile creates requirements/package file
func generateDependenciesFile(ctx context.Context, request IntelligentCodeGenerationRequest) (*types.GeneratedFile, error) {
	var depPath string
	switch strings.ToLower(request.Language) {
	case "python":
		depPath = "requirements.txt"
	case "javascript", "typescript":
		depPath = "package.json"
	case "go":
		depPath = "go.mod"
	default:
		depPath = "requirements.txt"
	}
	
	userPrompt := fmt.Sprintf(`Generate production dependencies for %s %s application:

%s

Include all necessary packages for:
1. Web framework (FastAPI/Express/Gin)
2. Database ORM and drivers
3. Authentication (JWT, bcrypt)
4. Validation libraries
5. Testing frameworks
6. Security packages
7. Logging and monitoring
8. Rate limiting
9. CORS handling
10. Development tools

Generate ONLY the dependencies file with specific versions.`, 
		request.Language, request.Type, request.Description)
	
	llmRequest := LLMGenerationRequest{
		Prompt:     userPrompt,
		System:     buildDependenciesSystemPrompt(request.Language),
		Language:   getDependencyLanguage(request.Language),
		Provider:   "azure",
		MaxTokens:  3000,
	}
	
	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		return nil, err
	}
	
	return &types.GeneratedFile{
		Path:     depPath,
		Content:  llmResult.Content,
		Language: getDependencyLanguage(request.Language),
		Type:     "config",
	}, nil
}

// generateTestsFile creates comprehensive test suite
func generateTestsFile(ctx context.Context, request IntelligentCodeGenerationRequest, existingFiles []types.GeneratedFile) (*types.GeneratedFile, error) {
	var testPath string
	switch strings.ToLower(request.Language) {
	case "python":
		testPath = "tests/test_main.py"
	case "javascript", "typescript":
		testPath = "tests/app.test.js"
	case "go":
		testPath = "internal/handlers/handlers_test.go"
	default:
		testPath = "tests/test_main.py"
	}
	
	// Extract main endpoints/functions from existing files for testing
	codeContext := ""
	for _, file := range existingFiles {
		if file.Type == "source" {
			codeContext += fmt.Sprintf("File: %s\n%s\n\n", file.Path, file.Content[:min(len(file.Content), 1000)])
		}
	}
	
	userPrompt := fmt.Sprintf(`Generate comprehensive tests for: %s

Based on this code context:
%s

Generate complete test suite with:
1. Unit tests for all functions/methods
2. Integration tests for API endpoints
3. Authentication flow tests
4. Database operation tests
5. Error handling tests
6. Edge case and boundary tests
7. Performance tests
8. Security tests
9. Mock/fixture setup
10. Test data factories

Target >90%% code coverage. Generate ONLY the main test file.`, 
		request.ProjectName, codeContext)
	
	llmRequest := LLMGenerationRequest{
		Prompt:     userPrompt,
		System:     buildTestSystemPrompt(request.Language),
		Language:   request.Language,
		Provider:   "azure",
		MaxTokens:  10000,
	}
	
	llmResult, err := GenerateCodeActivity(ctx, llmRequest)
	if err != nil {
		return nil, err
	}
	
	return &types.GeneratedFile{
		Path:     testPath,
		Content:  llmResult.Content,
		Language: request.Language,
		Type:     "test",
	}, nil
}

// buildNodeSystemPrompt creates system prompt for Node.js/TypeScript
func buildNodeSystemPrompt() string {
	return `You are an expert Node.js/TypeScript developer specializing in Express and enterprise applications.

CRITICAL RULES:
1. Generate COMPLETE, runnable code - no placeholders or TODO comments
2. Use Express with proper async/await patterns
3. Include all necessary imports at the top
4. Implement proper error handling with status codes
5. Use proper TypeScript types and interfaces
6. Include comprehensive logging with structured logs
7. Implement JWT authentication middleware
8. Add rate limiting and security headers
9. Include health check and metrics endpoints
10. Use proper dependency injection patterns
11. Follow JavaScript/TypeScript best practices
12. Add comprehensive documentation

OUTPUT: Complete, production-ready Node.js/TypeScript code only.`
}

// buildGoSystemPrompt creates system prompt for Go
func buildGoSystemPrompt() string {
	return `You are an expert Go developer specializing in Gin/Echo and enterprise applications.

CRITICAL RULES:
1. Generate COMPLETE, runnable code - no placeholders or TODO comments
2. Use Gin or Echo with proper error handling
3. Include all necessary imports at the top
4. Implement proper error handling with custom errors
5. Use proper Go struct tags and validation
6. Include comprehensive logging with structured logs
7. Implement JWT authentication middleware
8. Add rate limiting and security headers
9. Include health check and metrics endpoints
10. Use proper Go idioms and patterns
11. Follow Go best practices and formatting
12. Add comprehensive documentation

OUTPUT: Complete, production-ready Go code only.`
}

// Helper functions for system prompts
func buildPythonSystemPrompt() string {
	return `You are an expert Python developer specializing in FastAPI and enterprise applications.

CRITICAL RULES:
1. Generate COMPLETE, runnable code - no placeholders or TODO comments
2. Use FastAPI with proper async/await patterns
3. Include all necessary imports at the top
4. Implement proper error handling with HTTPException
5. Use Pydantic models for request/response validation
6. Include SQLAlchemy ORM with async support
7. Add comprehensive logging with structured logs
8. Implement JWT authentication middleware
9. Add rate limiting and security headers
10. Include health check and metrics endpoints
11. Use type hints throughout
12. Follow PEP 8 style guide
13. Add docstrings for all functions

OUTPUT: Complete, production-ready Python code only.`
}

func buildModelsSystemPrompt(language string) string {
	base := "You are a database modeling expert. Generate complete ORM models with relationships, validation, and proper field types."
	switch strings.ToLower(language) {
	case "python":
		return base + " Use SQLAlchemy with proper async support and Pydantic integration."
	case "javascript", "typescript":
		return base + " Use Prisma or Sequelize with proper validation."
	case "go":
		return base + " Use GORM with proper struct tags and validation."
	default:
		return base
	}
}

func buildAuthSystemPrompt(language string) string {
	return `You are a cybersecurity expert specializing in authentication systems.

Generate production-ready authentication with:
- JWT token generation and validation
- Secure password hashing (bcrypt)
- Rate limiting on auth endpoints
- Session management
- Role-based access control
- Account lockout protection
- Security headers

Generate COMPLETE implementation - no placeholders.`
}

func buildConfigSystemPrompt(language string) string {
	return `You are a DevOps expert specializing in configuration management.

Generate complete configuration module with:
- Environment variable handling with defaults
- Database connection management
- Security settings (JWT secrets, CORS)
- Feature flags and application settings
- Validation and error handling
- Production-ready defaults

Generate COMPLETE code - no placeholders.`
}

func buildDependenciesSystemPrompt(language string) string {
	switch strings.ToLower(language) {
	case "python":
		return `Generate production requirements.txt with specific versions for enterprise Python applications. Include FastAPI, SQLAlchemy, JWT, validation, testing, and security packages.`
	case "javascript", "typescript":
		return `Generate production package.json with proper scripts, dependencies, and devDependencies for enterprise Node.js applications.`
	case "go":
		return `Generate production go.mod file with proper module declarations and dependencies for enterprise Go applications.`
	default:
		return `Generate production dependency file with specific versions for enterprise applications.`
	}
}

func buildTestSystemPrompt(language string) string {
	return `You are a test automation expert. Generate comprehensive test suites with:
- Unit tests for all functions
- Integration tests for API endpoints
- Authentication and authorization tests
- Database operation tests
- Error handling and edge case tests
- Mock setup and test fixtures
- Proper test data factories
- Performance and security tests

Target >90% code coverage. Generate COMPLETE test implementation.`
}

func generateDeploymentInstructions(request IntelligentCodeGenerationRequest, dependencies []string) string {
	return fmt.Sprintf(`# Deployment Instructions for %s

## Prerequisites
- %s runtime environment
- Database (PostgreSQL/MySQL)
- Redis (for caching/sessions)

## Installation
1. Clone the repository
2. Install dependencies: %s
3. Set environment variables (see config file)
4. Run database migrations
5. Start the application

## Production Deployment
- Use Docker container
- Set up reverse proxy (nginx)
- Configure SSL certificates
- Set up monitoring and logging
- Configure backup strategies

## Dependencies
%s
`, request.ProjectName, request.Language, getInstallCommandForIntelligent(request.Language), strings.Join(dependencies, "\n"))
}

// Helper functions
func getDependencyLanguage(lang string) string {
	switch strings.ToLower(lang) {
	case "javascript", "typescript":
		return "json"
	case "go":
		return "go"
	default:
		return "plaintext"
	}
}

func getInstallCommandForIntelligent(lang string) string {
	switch strings.ToLower(lang) {
	case "python":
		return "pip install -r requirements.txt"
	case "javascript", "typescript":
		return "npm install"
	case "go":
		return "go mod download"
	default:
		return "install dependencies"
	}
}

// Using existing min function from activities.go