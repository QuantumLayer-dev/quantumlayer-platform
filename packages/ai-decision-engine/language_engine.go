package aidecision

import (
	"context"
	"fmt"
	"strings"
)

// AILanguageDecisionEngine makes intelligent language and framework decisions
type AILanguageDecisionEngine struct {
	*AIDecisionEngine
	languageRules   map[string]*LanguageProfile
	frameworkRules  map[string]*FrameworkProfile
}

// LanguageProfile defines capabilities and characteristics of a programming language
type LanguageProfile struct {
	Name         string                 `json:"name"`
	Extensions   []string               `json:"extensions"`
	Paradigms    []string               `json:"paradigms"`
	UseCases     []string               `json:"use_cases"`
	Strengths    []string               `json:"strengths"`
	Ecosystem    map[string]interface{} `json:"ecosystem"`
	Performance  PerformanceProfile     `json:"performance"`
	Embedding    []float32              `json:"embedding"`
}

// FrameworkProfile defines characteristics of a framework
type FrameworkProfile struct {
	Name         string                 `json:"name"`
	Language     string                 `json:"language"`
	Type         string                 `json:"type"`
	UseCases     []string               `json:"use_cases"`
	Features     []string               `json:"features"`
	Complexity   string                 `json:"complexity"`
	Popularity   float64                `json:"popularity"`
	Dependencies []string               `json:"dependencies"`
	Embedding    []float32              `json:"embedding"`
}

// PerformanceProfile defines performance characteristics
type PerformanceProfile struct {
	Speed      float64 `json:"speed"`
	Memory     float64 `json:"memory"`
	Startup    float64 `json:"startup"`
	Concurrent bool    `json:"concurrent"`
}

// NewAILanguageDecisionEngine creates an AI-powered language decision engine
func NewAILanguageDecisionEngine(llmClient LLMClient, vectorStore VectorStore) *AILanguageDecisionEngine {
	engine := &AILanguageDecisionEngine{
		AIDecisionEngine: NewAIDecisionEngine(llmClient, vectorStore),
		languageRules:    make(map[string]*LanguageProfile),
		frameworkRules:   make(map[string]*FrameworkProfile),
	}
	
	// Initialize with language profiles
	engine.initializeLanguageProfiles()
	engine.initializeFrameworkProfiles()
	
	return engine
}

// DecideLanguage intelligently selects the best programming language
func (e *AILanguageDecisionEngine) DecideLanguage(ctx context.Context, requirements string) (string, map[string]interface{}, error) {
	// Extract intent and analyze requirements
	intent, _ := e.llmClient.ExtractIntent(ctx, requirements)
	
	// Generate embedding for requirements
	reqEmbedding, err := e.llmClient.GenerateEmbedding(ctx, requirements)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate requirements embedding: %w", err)
	}
	
	// Search for best matching language profiles
	results, err := e.vectorStore.Search(ctx, reqEmbedding, 5, 0.6)
	if err != nil {
		return "", nil, fmt.Errorf("language search failed: %w", err)
	}
	
	// Score and rank languages based on multiple factors
	bestLanguage := ""
	bestScore := 0.0
	metadata := make(map[string]interface{})
	
	for _, result := range results {
		if profile, exists := e.languageRules[result.ID]; exists {
			score := e.scoreLanguageForRequirements(ctx, profile, requirements, result.Score)
			if score > bestScore {
				bestScore = score
				bestLanguage = profile.Name
				metadata = map[string]interface{}{
					"confidence":  score,
					"reasoning":   e.generateLanguageReasoning(profile, requirements),
					"extensions":  profile.Extensions,
					"paradigms":   profile.Paradigms,
					"performance": profile.Performance,
				}
			}
		}
	}
	
	if bestLanguage == "" {
		// Fallback to intelligent guess based on keywords
		bestLanguage = e.intelligentLanguageGuess(requirements)
		metadata["fallback"] = true
		metadata["confidence"] = 0.5
	}
	
	return bestLanguage, metadata, nil
}

// GetLanguageCapabilities returns capabilities of a language
func (e *AILanguageDecisionEngine) GetLanguageCapabilities(language string) map[string]interface{} {
	if profile, exists := e.languageRules[strings.ToLower(language)]; exists {
		return map[string]interface{}{
			"name":        profile.Name,
			"extensions":  profile.Extensions,
			"paradigms":   profile.Paradigms,
			"use_cases":   profile.UseCases,
			"strengths":   profile.Strengths,
			"ecosystem":   profile.Ecosystem,
			"performance": profile.Performance,
		}
	}
	return nil
}

// SuggestLanguages suggests multiple suitable languages
func (e *AILanguageDecisionEngine) SuggestLanguages(ctx context.Context, requirements string) ([]string, error) {
	// Get top 5 matches
	reqEmbedding, err := e.llmClient.GenerateEmbedding(ctx, requirements)
	if err != nil {
		return nil, err
	}
	
	results, err := e.vectorStore.Search(ctx, reqEmbedding, 5, 0.5)
	if err != nil {
		return nil, err
	}
	
	suggestions := make([]string, 0)
	for _, result := range results {
		if profile, exists := e.languageRules[result.ID]; exists {
			suggestions = append(suggestions, profile.Name)
		}
	}
	
	return suggestions, nil
}

// DecideFramework intelligently selects the best framework
func (e *AILanguageDecisionEngine) DecideFramework(ctx context.Context, language, requirements string) (string, map[string]interface{}, error) {
	// Filter frameworks by language
	compatibleFrameworks := e.getCompatibleFrameworks(language)
	
	if len(compatibleFrameworks) == 0 {
		return "", map[string]interface{}{"error": "no frameworks for language"}, nil
	}
	
	// Generate embedding for requirements
	reqEmbedding, err := e.llmClient.GenerateEmbedding(ctx, requirements)
	if err != nil {
		return "", nil, err
	}
	
	// Find best matching framework
	bestFramework := ""
	bestScore := 0.0
	metadata := make(map[string]interface{})
	
	for _, framework := range compatibleFrameworks {
		score := e.calculateFrameworkScore(framework, reqEmbedding)
		if score > bestScore {
			bestScore = score
			bestFramework = framework.Name
			metadata = map[string]interface{}{
				"confidence":   score,
				"type":         framework.Type,
				"features":     framework.Features,
				"complexity":   framework.Complexity,
				"dependencies": framework.Dependencies,
			}
		}
	}
	
	return bestFramework, metadata, nil
}

// GetFrameworkFeatures returns features of a framework
func (e *AILanguageDecisionEngine) GetFrameworkFeatures(framework string) map[string]interface{} {
	if profile, exists := e.frameworkRules[strings.ToLower(framework)]; exists {
		return map[string]interface{}{
			"name":         profile.Name,
			"language":     profile.Language,
			"type":         profile.Type,
			"features":     profile.Features,
			"complexity":   profile.Complexity,
			"dependencies": profile.Dependencies,
		}
	}
	return nil
}

// IsCompatible checks if a framework is compatible with a language
func (e *AILanguageDecisionEngine) IsCompatible(language, framework string) bool {
	if profile, exists := e.frameworkRules[strings.ToLower(framework)]; exists {
		return strings.EqualFold(profile.Language, language)
	}
	return false
}

// Private helper methods

func (e *AILanguageDecisionEngine) initializeLanguageProfiles() {
	// Initialize language profiles with embeddings
	languages := []LanguageProfile{
		{
			Name:       "python",
			Extensions: []string{"py", "pyw", "pyi"},
			Paradigms:  []string{"object-oriented", "functional", "procedural"},
			UseCases:   []string{"web", "data-science", "ml", "automation", "scripting"},
			Strengths:  []string{"readability", "libraries", "community", "versatility"},
			Ecosystem: map[string]interface{}{
				"package_manager": "pip",
				"test_framework":  "pytest",
				"web_frameworks":  []string{"flask", "django", "fastapi"},
			},
			Performance: PerformanceProfile{
				Speed:      0.6,
				Memory:     0.7,
				Startup:    0.8,
				Concurrent: true,
			},
		},
		{
			Name:       "javascript",
			Extensions: []string{"js", "mjs", "jsx"},
			Paradigms:  []string{"functional", "object-oriented", "event-driven"},
			UseCases:   []string{"web", "frontend", "backend", "mobile", "desktop"},
			Strengths:  []string{"ubiquity", "ecosystem", "async", "flexibility"},
			Ecosystem: map[string]interface{}{
				"package_manager": "npm",
				"test_framework":  "jest",
				"web_frameworks":  []string{"express", "next", "react"},
			},
			Performance: PerformanceProfile{
				Speed:      0.7,
				Memory:     0.6,
				Startup:    0.9,
				Concurrent: true,
			},
		},
		{
			Name:       "go",
			Extensions: []string{"go"},
			Paradigms:  []string{"procedural", "concurrent", "structured"},
			UseCases:   []string{"backend", "microservices", "cloud", "systems", "cli"},
			Strengths:  []string{"performance", "concurrency", "simplicity", "deployment"},
			Ecosystem: map[string]interface{}{
				"package_manager": "go mod",
				"test_framework":  "testing",
				"web_frameworks":  []string{"gin", "echo", "fiber"},
			},
			Performance: PerformanceProfile{
				Speed:      0.9,
				Memory:     0.8,
				Startup:    0.95,
				Concurrent: true,
			},
		},
		{
			Name:       "rust",
			Extensions: []string{"rs"},
			Paradigms:  []string{"systems", "functional", "concurrent"},
			UseCases:   []string{"systems", "embedded", "wasm", "performance", "blockchain"},
			Strengths:  []string{"memory-safety", "performance", "concurrency", "reliability"},
			Ecosystem: map[string]interface{}{
				"package_manager": "cargo",
				"test_framework":  "built-in",
				"web_frameworks":  []string{"actix", "rocket", "warp"},
			},
			Performance: PerformanceProfile{
				Speed:      0.95,
				Memory:     0.9,
				Startup:    0.85,
				Concurrent: true,
			},
		},
		{
			Name:       "java",
			Extensions: []string{"java"},
			Paradigms:  []string{"object-oriented", "functional"},
			UseCases:   []string{"enterprise", "android", "web", "big-data"},
			Strengths:  []string{"ecosystem", "tooling", "performance", "stability"},
			Ecosystem: map[string]interface{}{
				"package_manager": "maven/gradle",
				"test_framework":  "junit",
				"web_frameworks":  []string{"spring", "quarkus", "micronaut"},
			},
			Performance: PerformanceProfile{
				Speed:      0.85,
				Memory:     0.6,
				Startup:    0.5,
				Concurrent: true,
			},
		},
		{
			Name:       "typescript",
			Extensions: []string{"ts", "tsx"},
			Paradigms:  []string{"object-oriented", "functional", "typed"},
			UseCases:   []string{"web", "frontend", "backend", "enterprise"},
			Strengths:  []string{"type-safety", "tooling", "refactoring", "scalability"},
			Ecosystem: map[string]interface{}{
				"package_manager": "npm",
				"test_framework":  "jest",
				"web_frameworks":  []string{"angular", "nestjs", "react"},
			},
			Performance: PerformanceProfile{
				Speed:      0.7,
				Memory:     0.6,
				Startup:    0.8,
				Concurrent: true,
			},
		},
	}
	
	// Store language profiles and generate embeddings
	ctx := context.Background()
	for _, lang := range languages {
		// Generate embedding for language profile
		text := fmt.Sprintf("%s %s %s %s",
			lang.Name,
			strings.Join(lang.UseCases, " "),
			strings.Join(lang.Strengths, " "),
			strings.Join(lang.Paradigms, " "))
		
		if embedding, err := e.llmClient.GenerateEmbedding(ctx, text); err == nil {
			lang.Embedding = embedding
			e.languageRules[lang.Name] = &lang
			
			// Store in vector database
			e.vectorStore.Store(ctx, lang.Name, embedding, map[string]interface{}{
				"type":     "language",
				"name":     lang.Name,
				"useCases": lang.UseCases,
			})
		}
	}
}

func (e *AILanguageDecisionEngine) initializeFrameworkProfiles() {
	// Initialize framework profiles
	frameworks := []FrameworkProfile{
		{
			Name:         "flask",
			Language:     "python",
			Type:         "web",
			UseCases:     []string{"api", "microservice", "prototype"},
			Features:     []string{"lightweight", "flexible", "simple"},
			Complexity:   "low",
			Popularity:   0.8,
			Dependencies: []string{"werkzeug", "jinja2"},
		},
		{
			Name:         "fastapi",
			Language:     "python",
			Type:         "web",
			UseCases:     []string{"api", "async", "modern"},
			Features:     []string{"async", "type-hints", "auto-docs"},
			Complexity:   "medium",
			Popularity:   0.9,
			Dependencies: []string{"pydantic", "starlette"},
		},
		{
			Name:         "express",
			Language:     "javascript",
			Type:         "web",
			UseCases:     []string{"api", "web", "microservice"},
			Features:     []string{"minimal", "flexible", "middleware"},
			Complexity:   "low",
			Popularity:   0.9,
			Dependencies: []string{},
		},
		{
			Name:         "gin",
			Language:     "go",
			Type:         "web",
			UseCases:     []string{"api", "microservice", "high-performance"},
			Features:     []string{"fast", "minimal", "middleware"},
			Complexity:   "low",
			Popularity:   0.85,
			Dependencies: []string{},
		},
		{
			Name:         "spring",
			Language:     "java",
			Type:         "web",
			UseCases:     []string{"enterprise", "microservice", "full-stack"},
			Features:     []string{"comprehensive", "di", "aop"},
			Complexity:   "high",
			Popularity:   0.9,
			Dependencies: []string{"spring-core", "spring-web"},
		},
		{
			Name:         "react",
			Language:     "javascript",
			Type:         "frontend",
			UseCases:     []string{"spa", "ui", "component"},
			Features:     []string{"virtual-dom", "jsx", "hooks"},
			Complexity:   "medium",
			Popularity:   0.95,
			Dependencies: []string{"react-dom"},
		},
	}
	
	// Store framework profiles
	for _, framework := range frameworks {
		e.frameworkRules[framework.Name] = &framework
	}
}

func (e *AILanguageDecisionEngine) scoreLanguageForRequirements(ctx context.Context, profile *LanguageProfile, requirements string, similarity float64) float64 {
	score := similarity * 0.5 // Base score from embedding similarity
	
	// Analyze requirements for keywords
	reqLower := strings.ToLower(requirements)
	
	// Check use case matches
	for _, useCase := range profile.UseCases {
		if strings.Contains(reqLower, useCase) {
			score += 0.1
		}
	}
	
	// Check for performance requirements
	if strings.Contains(reqLower, "performance") || strings.Contains(reqLower, "fast") {
		score += profile.Performance.Speed * 0.2
	}
	
	if strings.Contains(reqLower, "concurrent") || strings.Contains(reqLower, "parallel") {
		if profile.Performance.Concurrent {
			score += 0.15
		}
	}
	
	// Consider ecosystem
	if strings.Contains(reqLower, "web") || strings.Contains(reqLower, "api") {
		if frameworks, ok := profile.Ecosystem["web_frameworks"].([]string); ok && len(frameworks) > 0 {
			score += 0.1
		}
	}
	
	return math.Min(1.0, score)
}

func (e *AILanguageDecisionEngine) generateLanguageReasoning(profile *LanguageProfile, requirements string) string {
	reasons := []string{}
	
	reqLower := strings.ToLower(requirements)
	
	// Match use cases
	for _, useCase := range profile.UseCases {
		if strings.Contains(reqLower, useCase) {
			reasons = append(reasons, fmt.Sprintf("excellent for %s development", useCase))
		}
	}
	
	// Performance reasoning
	if strings.Contains(reqLower, "performance") && profile.Performance.Speed > 0.8 {
		reasons = append(reasons, "high performance capabilities")
	}
	
	// Ecosystem reasoning
	if len(profile.Strengths) > 0 {
		reasons = append(reasons, strings.Join(profile.Strengths[:min(2, len(profile.Strengths))], " and "))
	}
	
	if len(reasons) == 0 {
		return "General purpose language suitable for the requirements"
	}
	
	return strings.Join(reasons, ", ")
}

func (e *AILanguageDecisionEngine) intelligentLanguageGuess(requirements string) string {
	reqLower := strings.ToLower(requirements)
	
	// Smart defaults based on domain
	if strings.Contains(reqLower, "machine learning") || strings.Contains(reqLower, "data science") {
		return "python"
	}
	if strings.Contains(reqLower, "web") || strings.Contains(reqLower, "frontend") {
		return "javascript"
	}
	if strings.Contains(reqLower, "microservice") || strings.Contains(reqLower, "api") {
		return "go"
	}
	if strings.Contains(reqLower, "enterprise") || strings.Contains(reqLower, "android") {
		return "java"
	}
	if strings.Contains(reqLower, "systems") || strings.Contains(reqLower, "embedded") {
		return "rust"
	}
	
	// Default fallback
	return "python"
}

func (e *AILanguageDecisionEngine) getCompatibleFrameworks(language string) []*FrameworkProfile {
	compatible := make([]*FrameworkProfile, 0)
	for _, framework := range e.frameworkRules {
		if strings.EqualFold(framework.Language, language) {
			compatible = append(compatible, framework)
		}
	}
	return compatible
}

func (e *AILanguageDecisionEngine) calculateFrameworkScore(framework *FrameworkProfile, reqEmbedding []float32) float64 {
	// Simplified scoring - in production would use actual embedding similarity
	return framework.Popularity * 0.5 + 0.5
}

func (e *AILanguageDecisionEngine) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}