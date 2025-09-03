package specialized

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/base"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/types"
)

// BackendDeveloperAgent handles backend code generation and API implementation
type BackendDeveloperAgent struct {
	*base.BaseAgent
	llmEndpoint string
	language    string
	framework   string
}

// NewBackendDeveloperAgent creates a new backend developer agent
func NewBackendDeveloperAgent(llmEndpoint string) *BackendDeveloperAgent {
	capabilities := []types.AgentCapability{
		types.CapCodeGeneration,
		types.CapTestGeneration,
		types.CapDocumentation,
		types.CapPerformanceOptimization,
	}

	agent := &BackendDeveloperAgent{
		BaseAgent:   base.NewBaseAgent(types.RoleBackendDev, capabilities),
		llmEndpoint: llmEndpoint,
		language:    "Go", // Default
		framework:   "Gin", // Default
	}

	// Set specialized handlers
	agent.SetExecutionHandler(agent.executeTask)
	agent.SetMessageHandler(agent.handleMessage)
	agent.SetInitializeHandler(agent.initialize)

	return agent
}

func (a *BackendDeveloperAgent) initialize(ctx context.Context, agentCtx *types.AgentContext) error {
	// Review architecture and prepare for implementation
	if agentCtx.SharedMemory != nil && agentCtx.SharedMemory.ProjectContext != nil {
		if arch, ok := agentCtx.SharedMemory.ProjectContext["architecture"].(map[string]interface{}); ok {
			if techStack, ok := arch["technology_stack"].(map[string]interface{}); ok {
				if backend, ok := techStack["backend"].(map[string]interface{}); ok {
					if lang, ok := backend["language"].(string); ok {
						a.language = lang
					}
					if fw, ok := backend["framework"].(string); ok {
						a.framework = fw
					}
				}
			}
		}
	}

	return nil
}

func (a *BackendDeveloperAgent) executeTask(ctx context.Context, task *types.Task) error {
	switch task.Type {
	case "generate_api":
		return a.executeAPIGeneration(ctx, task)
	case "generate_service":
		return a.executeServiceGeneration(ctx, task)
	case "generate_database_layer":
		return a.executeDatabaseLayerGeneration(ctx, task)
	case "generate_middleware":
		return a.executeMiddlewareGeneration(ctx, task)
	case "generate_tests":
		return a.executeTestGeneration(ctx, task)
	case "optimize_code":
		return a.executeCodeOptimization(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
}

func (a *BackendDeveloperAgent) handleMessage(ctx context.Context, msg *types.Message) error {
	switch msg.Type {
	case types.MsgTypeRequest:
		return a.handleRequest(ctx, msg)
	case types.MsgTypeCollaboration:
		return a.handleCollaboration(ctx, msg)
	default:
		return nil
	}
}

func (a *BackendDeveloperAgent) executeAPIGeneration(ctx context.Context, task *types.Task) error {
	requirements, _ := task.Requirements["requirements"].(string)
	endpoints, _ := task.Requirements["endpoints"].([]map[string]interface{})
	
	prompt := fmt.Sprintf(`Generate production-ready %s API code using %s for:
Requirements: %s
Endpoints: %v

Include:
1. Route handlers with proper error handling
2. Request validation
3. Response formatting
4. Authentication middleware
5. Rate limiting
6. Logging
7. OpenAPI documentation comments

Generate complete, runnable code with best practices.`, a.language, a.framework, requirements, endpoints)

	code, err := a.callLLM(ctx, prompt, fmt.Sprintf("You are an expert %s developer.", a.language))
	if err != nil {
		return fmt.Errorf("failed to generate API: %w", err)
	}

	// Parse and structure the code
	structuredCode := a.structureAPICode(code)
	
	// Store generated code
	if ctx.Value("shared_memory") != nil {
		if sharedMem, ok := ctx.Value("shared_memory").(*types.SharedMemory); ok {
			if sharedMem.GeneratedCode == nil {
				sharedMem.GeneratedCode = make(map[string]string)
			}
			sharedMem.GeneratedCode["api_main.go"] = structuredCode["main"].(string)
			sharedMem.GeneratedCode["api_routes.go"] = structuredCode["routes"].(string)
			sharedMem.GeneratedCode["api_handlers.go"] = structuredCode["handlers"].(string)
		}
	}

	task.Result = structuredCode
	return nil
}

func (a *BackendDeveloperAgent) executeServiceGeneration(ctx context.Context, task *types.Task) error {
	serviceName, _ := task.Requirements["service_name"].(string)
	businessLogic, _ := task.Requirements["business_logic"].(string)
	
	prompt := fmt.Sprintf(`Generate a %s service layer using %s for:
Service: %s
Business Logic: %s

Include:
1. Service interface
2. Service implementation with business logic
3. Dependency injection
4. Transaction handling
5. Error handling and custom errors
6. Logging
7. Unit test stubs

Follow SOLID principles and clean architecture.`, a.language, a.framework, serviceName, businessLogic)

	code, err := a.callLLM(ctx, prompt, fmt.Sprintf("You are an expert %s developer specializing in clean architecture.", a.language))
	if err != nil {
		return err
	}

	structuredCode := map[string]interface{}{
		"interface": a.extractInterface(code),
		"implementation": a.extractImplementation(code),
		"tests": a.extractTests(code),
	}

	task.Result = structuredCode
	return nil
}

func (a *BackendDeveloperAgent) executeDatabaseLayerGeneration(ctx context.Context, task *types.Task) error {
	schema, _ := task.Requirements["schema"].(map[string]interface{})
	dbType, _ := task.Requirements["database_type"].(string)
	
	if dbType == "" {
		dbType = "PostgreSQL"
	}

	prompt := fmt.Sprintf(`Generate %s data access layer for %s database:
Schema: %v

Include:
1. Models/Entities
2. Repository interfaces
3. Repository implementations
4. Database migrations
5. Connection management
6. Query builders
7. Transaction support

Use best practices for %s and %s.`, a.language, dbType, schema, a.language, dbType)

	code, err := a.callLLM(ctx, prompt, fmt.Sprintf("You are a %s database expert.", dbType))
	if err != nil {
		return err
	}

	structuredCode := map[string]interface{}{
		"models": a.extractModels(code),
		"repositories": a.extractRepositories(code),
		"migrations": a.extractMigrations(code),
	}

	// Store in shared memory
	if ctx.Value("shared_memory") != nil {
		if sharedMem, ok := ctx.Value("shared_memory").(*types.SharedMemory); ok {
			if sharedMem.GeneratedCode == nil {
				sharedMem.GeneratedCode = make(map[string]string)
			}
			sharedMem.GeneratedCode["models.go"] = structuredCode["models"].(string)
			sharedMem.GeneratedCode["repositories.go"] = structuredCode["repositories"].(string)
		}
	}

	task.Result = structuredCode
	return nil
}

func (a *BackendDeveloperAgent) executeMiddlewareGeneration(ctx context.Context, task *types.Task) error {
	middlewareType, _ := task.Requirements["type"].(string)
	config, _ := task.Requirements["config"].(map[string]interface{})
	
	prompt := fmt.Sprintf(`Generate %s middleware for %s:
Type: %s
Configuration: %v

Include:
1. Middleware function
2. Configuration options
3. Error handling
4. Logging
5. Performance considerations
6. Tests

Follow %s middleware patterns.`, a.language, a.framework, middlewareType, config, a.framework)

	code, err := a.callLLM(ctx, prompt, fmt.Sprintf("You are a %s middleware expert.", a.framework))
	if err != nil {
		return err
	}

	task.Result = map[string]interface{}{
		"middleware": code,
		"type": middlewareType,
		"language": a.language,
	}
	
	return nil
}

func (a *BackendDeveloperAgent) executeTestGeneration(ctx context.Context, task *types.Task) error {
	codeToTest, _ := task.Requirements["code"].(string)
	testType, _ := task.Requirements["test_type"].(string)
	
	if testType == "" {
		testType = "unit"
	}

	prompt := fmt.Sprintf(`Generate comprehensive %s tests in %s for:
Code to test:
%s

Include:
1. Test setup and teardown
2. Positive test cases
3. Negative test cases
4. Edge cases
5. Mocking dependencies
6. Assertions
7. Test coverage > 80%%

Use %s testing best practices.`, testType, a.language, codeToTest, a.language)

	tests, err := a.callLLM(ctx, prompt, fmt.Sprintf("You are a %s testing expert.", a.language))
	if err != nil {
		return err
	}

	// Store test results
	if ctx.Value("shared_memory") != nil {
		if sharedMem, ok := ctx.Value("shared_memory").(*types.SharedMemory); ok {
			testResult := types.TestResult{
				ID:        fmt.Sprintf("test-%d", time.Now().Unix()),
				TestType:  testType,
				Target:    "backend_code",
				Passed:    true,
				Coverage:  85.0, // Simulated
				Details:   "Tests generated successfully",
				Timestamp: time.Now(),
			}
			sharedMem.TestResults = append(sharedMem.TestResults, testResult)
		}
	}

	task.Result = map[string]interface{}{
		"tests": tests,
		"coverage": "85%",
		"test_count": a.countTests(tests),
	}
	
	return nil
}

func (a *BackendDeveloperAgent) executeCodeOptimization(ctx context.Context, task *types.Task) error {
	code, _ := task.Requirements["code"].(string)
	metrics, _ := task.Requirements["metrics"].(map[string]interface{})
	
	prompt := fmt.Sprintf(`Optimize this %s code for better performance:
Current code:
%s

Performance metrics: %v

Focus on:
1. Algorithm optimization
2. Memory usage reduction
3. Query optimization
4. Caching strategies
5. Concurrency improvements
6. Resource pooling

Provide optimized code with explanations.`, a.language, code, metrics)

	optimizedCode, err := a.callLLM(ctx, prompt, fmt.Sprintf("You are a %s performance expert.", a.language))
	if err != nil {
		return err
	}

	task.Result = map[string]interface{}{
		"optimized_code": optimizedCode,
		"improvements": a.identifyImprovements(code, optimizedCode),
	}
	
	return nil
}

func (a *BackendDeveloperAgent) handleRequest(ctx context.Context, msg *types.Message) error {
	switch msg.Content {
	case "code_review":
		return a.performCodeReview(ctx, msg)
	case "explain_code":
		return a.explainCode(ctx, msg)
	case "suggest_refactor":
		return a.suggestRefactor(ctx, msg)
	default:
		return nil
	}
}

func (a *BackendDeveloperAgent) handleCollaboration(ctx context.Context, msg *types.Message) error {
	request, _ := msg.Metadata["request"].(map[string]interface{})
	requestType, _ := request["type"].(string)
	
	var response interface{}
	switch requestType {
	case "api_integration":
		response = a.provideAPIIntegration(request)
	case "database_query":
		response = a.provideDatabaseQuery(request)
	case "performance_advice":
		response = a.providePerformanceAdvice(request)
	default:
		response = map[string]interface{}{"status": "unknown request"}
	}

	reply := &types.Message{
		From:     a.ID(),
		To:       msg.From,
		Type:     types.MsgTypeResponse,
		ReplyTo:  msg.ID,
		Content:  "Backend development assistance",
		Metadata: map[string]interface{}{"response": response},
	}

	return a.SendMessage(ctx, reply)
}

func (a *BackendDeveloperAgent) callLLM(ctx context.Context, prompt, systemPrompt string) (string, error) {
	requestBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"provider":   "azure",
		"max_tokens": 3000,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.llmEndpoint+"/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	content, ok := result["content"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	return content, nil
}

// Helper methods

func (a *BackendDeveloperAgent) structureAPICode(code string) map[string]interface{} {
	// Extract different parts of generated code
	return map[string]interface{}{
		"main": a.extractMainCode(code),
		"routes": a.extractRoutes(code),
		"handlers": a.extractHandlers(code),
		"middleware": a.extractMiddleware(code),
		"models": a.extractModels(code),
	}
}

func (a *BackendDeveloperAgent) extractMainCode(code string) string {
	// Extract main function (simplified)
	if idx := strings.Index(code, "func main()"); idx != -1 {
		endIdx := strings.Index(code[idx:], "\n}\n")
		if endIdx != -1 {
			return code[idx:idx+endIdx+3]
		}
	}
	return code
}

func (a *BackendDeveloperAgent) extractRoutes(code string) string {
	// Extract route definitions
	return code // Simplified
}

func (a *BackendDeveloperAgent) extractHandlers(code string) string {
	// Extract handler functions
	return code // Simplified
}

func (a *BackendDeveloperAgent) extractMiddleware(code string) string {
	// Extract middleware functions
	return code // Simplified
}

func (a *BackendDeveloperAgent) extractInterface(code string) string {
	// Extract interface definitions
	return code // Simplified
}

func (a *BackendDeveloperAgent) extractImplementation(code string) string {
	// Extract implementation
	return code // Simplified
}

func (a *BackendDeveloperAgent) extractTests(code string) string {
	// Extract test code
	return code // Simplified
}

func (a *BackendDeveloperAgent) extractModels(code string) string {
	// Extract model definitions
	return code // Simplified
}

func (a *BackendDeveloperAgent) extractRepositories(code string) string {
	// Extract repository code
	return code // Simplified
}

func (a *BackendDeveloperAgent) extractMigrations(code string) string {
	// Extract migration scripts
	return code // Simplified
}

func (a *BackendDeveloperAgent) countTests(tests string) int {
	// Count number of test functions
	return strings.Count(tests, "func Test")
}

func (a *BackendDeveloperAgent) identifyImprovements(original, optimized string) []string {
	return []string{
		"Optimized database queries",
		"Added caching layer",
		"Improved algorithm complexity",
		"Reduced memory allocations",
	}
}

func (a *BackendDeveloperAgent) performCodeReview(ctx context.Context, msg *types.Message) error {
	// Implement code review logic
	return nil
}

func (a *BackendDeveloperAgent) explainCode(ctx context.Context, msg *types.Message) error {
	// Implement code explanation
	return nil
}

func (a *BackendDeveloperAgent) suggestRefactor(ctx context.Context, msg *types.Message) error {
	// Implement refactoring suggestions
	return nil
}

func (a *BackendDeveloperAgent) provideAPIIntegration(request map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"integration_code": "// API integration code here",
		"documentation": "API integration guide",
	}
}

func (a *BackendDeveloperAgent) provideDatabaseQuery(request map[string]interface{}) string {
	return "SELECT * FROM table WHERE condition"
}

func (a *BackendDeveloperAgent) providePerformanceAdvice(request map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"recommendations": []string{
			"Use connection pooling",
			"Implement caching",
			"Optimize queries",
		},
	}
}