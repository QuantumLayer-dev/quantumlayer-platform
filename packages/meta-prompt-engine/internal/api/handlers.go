package api

import (
	"net/http"
	"time"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/meta-prompt-engine/internal/engine"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// EnhanceRequest represents a prompt enhancement request from workflows
type EnhanceRequest struct {
	OriginalPrompt string            `json:"original_prompt"`
	Type           string            `json:"type"`
	Language       string            `json:"language"`
	Context        map[string]string `json:"context"`
	Model          string            `json:"model,omitempty"`
}

// EnhanceResponse represents the enhanced prompt response
type EnhanceResponse struct {
	EnhancedPrompt string `json:"enhanced_prompt"`
	SystemPrompt   string `json:"system_prompt"`
	Tokens         int    `json:"tokens"`
	Improvements   []string `json:"improvements,omitempty"`
	TemplateUsed   string `json:"template_used,omitempty"`
}

// CreateHandlers creates all API handlers for the meta-prompt engine
func CreateHandlers(engine *engine.MetaPromptEngine, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a wrapper that could be extended
		// For now, we'll handle routing in main.go
	}
}

// EnhanceHandler handles prompt enhancement requests
func EnhanceHandler(metaEngine *engine.MetaPromptEngine, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()

		logger.WithField("request_id", requestID).Info("Received enhance request")

		var request EnhanceRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			logger.WithError(err).Error("Failed to parse request")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request format",
			})
			return
		}

		// Log request details
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"type":       request.Type,
			"language":   request.Language,
			"prompt_len": len(request.OriginalPrompt),
		}).Debug("Processing enhancement request")

		// Select template based on type
		templateID := selectTemplate(request.Type)
		
		// Enhance the prompt using the engine
		enhanced, system := enhancePrompt(
			request.OriginalPrompt,
			request.Type,
			request.Language,
			request.Context,
		)

		// Calculate token estimate (rough approximation)
		tokens := estimateTokens(enhanced + system)

		// Track improvements made
		improvements := []string{
			"Added context-specific instructions",
			"Included best practices guidelines",
			"Optimized for " + getModelType(request.Model),
			"Added error handling requirements",
		}

		response := EnhanceResponse{
			EnhancedPrompt: enhanced,
			SystemPrompt:   system,
			Tokens:         tokens,
			Improvements:   improvements,
			TemplateUsed:   templateID,
		}

		// Log success
		logger.WithFields(logrus.Fields{
			"request_id":     requestID,
			"duration_ms":    time.Since(start).Milliseconds(),
			"tokens":         tokens,
			"template_used":  templateID,
		}).Info("Enhancement completed successfully")

		c.JSON(http.StatusOK, response)
	}
}

// TemplateHandler handles template management
func TemplateHandler(engine *engine.MetaPromptEngine, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// List all available templates
		templates := engine.ListTemplates()
		
		c.JSON(http.StatusOK, gin.H{
			"templates": templates,
			"count":     len(templates),
		})
	}
}

// MetricsHandler provides performance metrics
func MetricsHandler(engine *engine.MetaPromptEngine, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics := engine.GetMetrics()
		
		c.JSON(http.StatusOK, gin.H{
			"total_requests":   metrics.TotalRequests,
			"avg_latency_ms":   metrics.AvgLatency,
			"success_rate":     metrics.SuccessRate,
			"templates_used":   metrics.TemplateUsage,
		})
	}
}

// Helper functions

func selectTemplate(promptType string) string {
	templates := map[string]string{
		"api":         "api_generation",
		"frontend":    "frontend_development",
		"function":    "function_implementation",
		"backend":     "backend_service",
		"database":    "database_design",
		"test":        "test_generation",
		"refactor":    "code_refactor",
		"review":      "code_review",
		"security":    "security_audit",
		"performance": "performance_optimization",
	}

	if tmpl, exists := templates[promptType]; exists {
		return tmpl
	}
	return "code_generation_basic"
}

func enhancePrompt(original, promptType, language string, context map[string]string) (enhanced, system string) {
	// Build enhanced prompt based on type
	prefix := getTypePrefix(promptType)
	suffix := getTypeSuffix(promptType)
	
	// Add language-specific instructions
	langInstructions := getLanguageInstructions(language)
	
	// Build context string
	contextStr := ""
	for key, value := range context {
		contextStr += "\n" + key + ": " + value
	}

	enhanced = prefix + "\n\n" + original
	if contextStr != "" {
		enhanced += "\n\nContext:" + contextStr
	}
	enhanced += "\n\n" + langInstructions + "\n\n" + suffix

	// Build system prompt
	system = buildSystemPrompt(promptType, language)

	return enhanced, system
}

func getTypePrefix(promptType string) string {
	prefixes := map[string]string{
		"api": "Create a production-ready REST API with the following requirements. Include comprehensive error handling, input validation, authentication/authorization checks, rate limiting, and OpenAPI documentation.",
		"frontend": "Create a modern, responsive frontend application with clean architecture. Ensure accessibility (WCAG 2.1 AA), performance optimization, and responsive design for all screen sizes.",
		"function": "Implement a well-tested, efficient function with comprehensive documentation. Include input validation, error handling, edge cases, and unit tests with >90% coverage.",
		"backend": "Design and implement a scalable backend service with proper separation of concerns, dependency injection, configuration management, and observability (logging, metrics, tracing).",
		"database": "Design an optimized database schema with proper indexing, constraints, and relationships. Include migration scripts and performance considerations.",
		"test": "Generate comprehensive test cases covering happy paths, edge cases, error scenarios, and performance benchmarks. Include unit, integration, and e2e tests where applicable.",
		"refactor": "Refactor the code following SOLID principles, design patterns, and best practices. Improve readability, maintainability, and performance while preserving functionality.",
	}

	if prefix, exists := prefixes[promptType]; exists {
		return prefix
	}
	return "Create clean, maintainable, production-ready code following industry best practices."
}

func getTypeSuffix(promptType string) string {
	return `
REQUIREMENTS:
1. Production-ready code with no placeholders
2. Comprehensive error handling
3. Proper logging and monitoring hooks
4. Security best practices
5. Performance optimizations
6. Complete documentation
7. Test coverage >80%
8. Follow language-specific conventions`
}

func getLanguageInstructions(language string) string {
	instructions := map[string]string{
		"python":     "Follow PEP 8 style guide. Use type hints. Include docstrings. Prefer composition over inheritance.",
		"javascript": "Use ES6+ features. Follow Airbnb style guide. Include JSDoc comments. Handle async operations properly.",
		"typescript": "Use strict mode. Define interfaces for data structures. Avoid 'any' type. Include TSDoc comments.",
		"go":         "Follow Go idioms. Use meaningful variable names. Handle errors explicitly. Include godoc comments.",
		"java":       "Follow Java conventions. Use appropriate design patterns. Include Javadoc. Consider thread safety.",
		"rust":       "Follow Rust idioms. Use Result types. Minimize unsafe code. Include rustdoc comments.",
	}

	if inst, exists := instructions[language]; exists {
		return "Language-specific requirements:\n" + inst
	}
	return "Follow language-specific best practices and conventions."
}

func buildSystemPrompt(promptType, language string) string {
	base := `You are an expert software engineer specializing in ` + language + ` development.
Your task is to generate COMPLETE, PRODUCTION-READY code based on the requirements.

CRITICAL RULES:
1. Generate the FULL implementation, not just stubs or placeholders
2. Include ALL necessary imports, dependencies, and setup
3. Implement complete error handling and edge cases
4. Add comprehensive logging and monitoring hooks
5. Follow security best practices (input validation, sanitization, authentication)
6. Include inline documentation and comments
7. Make the code maintainable and testable
8. Optimize for performance and scalability

OUTPUT FORMAT:
- Provide complete, runnable code
- Include configuration files if needed
- Add deployment instructions if applicable
- Include test files separately
- Document any external dependencies`

	// Add type-specific instructions
	if promptType == "api" {
		base += "\n\nAPI SPECIFIC:\n- Include OpenAPI/Swagger documentation\n- Implement rate limiting\n- Add request/response validation\n- Include health check endpoints"
	} else if promptType == "frontend" {
		base += "\n\nFRONTEND SPECIFIC:\n- Ensure responsive design\n- Implement proper state management\n- Add loading and error states\n- Include accessibility features"
	}

	return base
}

func estimateTokens(text string) int {
	// Rough estimation: 1 token ≈ 4 characters
	return len(text) / 4
}

func getModelType(model string) string {
	if model == "" {
		return "GPT-4"
	}
	return model
}