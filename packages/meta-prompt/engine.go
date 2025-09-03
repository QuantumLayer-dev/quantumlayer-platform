package metaprompt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// PromptTemplate represents a reusable prompt template
type PromptTemplate struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Category         string                 `json:"category"`
	Template         string                 `json:"template"`
	Variables        []string               `json:"variables"`
	SystemPrompt     string                 `json:"system_prompt"`
	Examples         []Example              `json:"examples"`
	SuccessRate      float64                `json:"success_rate"`
	UsageCount       int                    `json:"usage_count"`
	AverageTokens    int                    `json:"average_tokens"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
	LastUsed         time.Time              `json:"last_used"`
	Version          int                    `json:"version"`
}

// Example represents an example input/output for a template
type Example struct {
	Input  map[string]string `json:"input"`
	Output string            `json:"output"`
}

// PromptOptimization tracks optimization attempts
type PromptOptimization struct {
	OriginalPrompt   string    `json:"original_prompt"`
	OptimizedPrompt  string    `json:"optimized_prompt"`
	ImprovementScore float64   `json:"improvement_score"`
	Technique        string    `json:"technique"`
	Timestamp        time.Time `json:"timestamp"`
}

// MetaPromptEngine manages dynamic prompt generation and optimization
type MetaPromptEngine struct {
	templates    map[string]*PromptTemplate
	optimizations []PromptOptimization
	templateStore TemplateStore
	mu           sync.RWMutex
	
	// A/B testing
	experiments  map[string]*Experiment
	
	// Learning parameters
	learningRate float64
	minSamples   int
}

// Experiment represents an A/B test for prompt optimization
type Experiment struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	VariantA    *PromptTemplate     `json:"variant_a"`
	VariantB    *PromptTemplate     `json:"variant_b"`
	Metrics     map[string][]float64 `json:"metrics"`
	StartTime   time.Time           `json:"start_time"`
	EndTime     *time.Time          `json:"end_time"`
	Winner      string              `json:"winner"`
}

// TemplateStore interface for persisting templates
type TemplateStore interface {
	Save(ctx context.Context, template *PromptTemplate) error
	Load(ctx context.Context, id string) (*PromptTemplate, error)
	List(ctx context.Context, category string) ([]*PromptTemplate, error)
	Update(ctx context.Context, template *PromptTemplate) error
}

// NewMetaPromptEngine creates a new meta-prompt engine
func NewMetaPromptEngine(store TemplateStore) *MetaPromptEngine {
	engine := &MetaPromptEngine{
		templates:     make(map[string]*PromptTemplate),
		experiments:   make(map[string]*Experiment),
		templateStore: store,
		learningRate:  0.1,
		minSamples:    10,
	}

	// Load default templates
	engine.loadDefaultTemplates()

	return engine
}

// GeneratePrompt dynamically generates an optimized prompt
func (e *MetaPromptEngine) GeneratePrompt(ctx context.Context, request PromptRequest) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Select best template for the task
	template := e.selectBestTemplate(request.Category, request.Task)
	if template == nil {
		// Create new template if none exists
		template = e.createTemplate(request)
	}

	// Apply variables to template
	prompt := e.applyVariables(template.Template, request.Variables)

	// Apply optimization techniques
	optimizedPrompt := e.optimizePrompt(prompt, request)

	// Track usage for learning
	e.trackUsage(template.ID, request)

	return optimizedPrompt, nil
}

// OptimizePrompt applies various optimization techniques
func (e *MetaPromptEngine) optimizePrompt(prompt string, request PromptRequest) string {
	// Chain of Thought (CoT) injection
	if request.RequiresReasoning {
		prompt = e.injectChainOfThought(prompt)
	}

	// Few-shot learning examples
	if len(request.Examples) > 0 {
		prompt = e.addFewShotExamples(prompt, request.Examples)
	}

	// Role-based conditioning
	if request.Role != "" {
		prompt = e.addRoleConditioning(prompt, request.Role)
	}

	// Constraint specification
	if len(request.Constraints) > 0 {
		prompt = e.addConstraints(prompt, request.Constraints)
	}

	// Output format specification
	if request.OutputFormat != "" {
		prompt = e.specifyOutputFormat(prompt, request.OutputFormat)
	}

	return prompt
}

// CreateTemplate creates a new prompt template
func (e *MetaPromptEngine) CreateTemplate(name, category, template string, variables []string) *PromptTemplate {
	e.mu.Lock()
	defer e.mu.Unlock()

	tmpl := &PromptTemplate{
		ID:           generateID(),
		Name:         name,
		Category:     category,
		Template:     template,
		Variables:    variables,
		SuccessRate:  0.5, // Start with neutral success rate
		CreatedAt:    time.Now(),
		Version:      1,
	}

	e.templates[tmpl.ID] = tmpl
	
	// Persist to store
	ctx := context.Background()
	e.templateStore.Save(ctx, tmpl)

	return tmpl
}

// StartExperiment starts an A/B test between two prompt variants
func (e *MetaPromptEngine) StartExperiment(name string, variantA, variantB *PromptTemplate) *Experiment {
	e.mu.Lock()
	defer e.mu.Unlock()

	experiment := &Experiment{
		ID:        generateID(),
		Name:      name,
		VariantA:  variantA,
		VariantB:  variantB,
		Metrics:   make(map[string][]float64),
		StartTime: time.Now(),
	}

	e.experiments[experiment.ID] = experiment
	return experiment
}

// RecordExperimentResult records the result of an experiment trial
func (e *MetaPromptEngine) RecordExperimentResult(experimentID, variant string, metric string, value float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	experiment, exists := e.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment %s not found", experimentID)
	}

	key := fmt.Sprintf("%s_%s", variant, metric)
	experiment.Metrics[key] = append(experiment.Metrics[key], value)

	// Check if we have enough data to determine winner
	if len(experiment.Metrics[key]) >= e.minSamples {
		e.evaluateExperiment(experiment)
	}

	return nil
}

// LearnFromFeedback improves templates based on feedback
func (e *MetaPromptEngine) LearnFromFeedback(templateID string, feedback Feedback) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	template, exists := e.templates[templateID]
	if !exists {
		return fmt.Errorf("template %s not found", templateID)
	}

	// Update success rate using exponential moving average
	template.SuccessRate = (1-e.learningRate)*template.SuccessRate + e.learningRate*feedback.SuccessScore
	template.UsageCount++
	template.LastUsed = time.Now()

	// If performance is poor, create optimization
	if template.SuccessRate < 0.5 && template.UsageCount > e.minSamples {
		e.createOptimization(template, feedback)
	}

	// Update in store
	ctx := context.Background()
	return e.templateStore.Update(ctx, template)
}

// GetBestTemplates returns the best performing templates
func (e *MetaPromptEngine) GetBestTemplates(category string, limit int) []*PromptTemplate {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var templates []*PromptTemplate
	for _, tmpl := range e.templates {
		if category == "" || tmpl.Category == category {
			templates = append(templates, tmpl)
		}
	}

	// Sort by success rate
	// Simplified - in production use proper sorting
	return templates[:min(limit, len(templates))]
}

// Private methods

func (e *MetaPromptEngine) loadDefaultTemplates() {
	// Load pre-defined high-performing templates
	e.templates["code_generation"] = &PromptTemplate{
		ID:       "code_generation",
		Name:     "Code Generation",
		Category: "development",
		Template: `As an expert {{language}} developer, create {{type}} that:
{{requirements}}

Requirements:
- Follow best practices for {{language}}
- Include comprehensive error handling
- Add inline documentation
- Ensure type safety
- Optimize for performance

Provide production-ready code that is maintainable and scalable.`,
		Variables:    []string{"language", "type", "requirements"},
		SuccessRate:  0.85,
		SystemPrompt: "You are an expert software engineer with deep knowledge of software design patterns and best practices.",
	}

	e.templates["requirements_analysis"] = &PromptTemplate{
		ID:       "requirements_analysis",
		Name:     "Requirements Analysis",
		Category: "analysis",
		Template: `Analyze the following requirements and provide:
1. Functional requirements breakdown
2. Non-functional requirements
3. Technical constraints
4. Success criteria
5. Risk assessment

Requirements: {{requirements}}

Format your response as structured JSON.`,
		Variables:    []string{"requirements"},
		SuccessRate:  0.80,
		SystemPrompt: "You are a senior business analyst and solution architect.",
	}

	e.templates["architecture_design"] = &PromptTemplate{
		ID:       "architecture_design",
		Name:     "Architecture Design",
		Category: "architecture",
		Template: `Design a {{scale}} architecture for:
{{description}}

Consider:
- Scalability requirements: {{scalability}}
- Performance requirements: {{performance}}
- Security requirements: {{security}}
- Budget constraints: {{budget}}

Provide:
1. High-level architecture diagram description
2. Component breakdown
3. Technology stack recommendations
4. Deployment strategy`,
		Variables:    []string{"scale", "description", "scalability", "performance", "security", "budget"},
		SuccessRate:  0.82,
		SystemPrompt: "You are a cloud solutions architect with expertise in distributed systems.",
	}
}

func (e *MetaPromptEngine) selectBestTemplate(category, task string) *PromptTemplate {
	var bestTemplate *PromptTemplate
	highestScore := 0.0

	for _, tmpl := range e.templates {
		if tmpl.Category == category {
			score := tmpl.SuccessRate * float64(tmpl.UsageCount+1) / 100.0
			if score > highestScore {
				highestScore = score
				bestTemplate = tmpl
			}
		}
	}

	return bestTemplate
}

func (e *MetaPromptEngine) createTemplate(request PromptRequest) *PromptTemplate {
	// Dynamically create a new template based on request
	return &PromptTemplate{
		ID:           generateID(),
		Name:         fmt.Sprintf("Dynamic_%s", request.Task),
		Category:     request.Category,
		Template:     e.generateBaseTemplate(request),
		Variables:    extractVariables(request.Variables),
		SuccessRate:  0.5,
		CreatedAt:    time.Now(),
		Version:      1,
	}
}

func (e *MetaPromptEngine) applyVariables(template string, variables map[string]string) string {
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func (e *MetaPromptEngine) injectChainOfThought(prompt string) string {
	return fmt.Sprintf(`%s

Let's approach this step-by-step:
1. First, understand the requirements
2. Break down the problem into smaller parts
3. Solve each part systematically
4. Combine the solutions
5. Verify the result

Now, let's begin:`, prompt)
}

func (e *MetaPromptEngine) addFewShotExamples(prompt string, examples []Example) string {
	exampleText := "\n\nExamples:\n"
	for i, example := range examples {
		exampleText += fmt.Sprintf("Example %d:\nInput: %v\nOutput: %s\n\n", i+1, example.Input, example.Output)
	}
	return prompt + exampleText
}

func (e *MetaPromptEngine) addRoleConditioning(prompt string, role string) string {
	return fmt.Sprintf("You are %s. %s", role, prompt)
}

func (e *MetaPromptEngine) addConstraints(prompt string, constraints []string) string {
	constraintText := "\n\nConstraints:\n"
	for _, constraint := range constraints {
		constraintText += fmt.Sprintf("- %s\n", constraint)
	}
	return prompt + constraintText
}

func (e *MetaPromptEngine) specifyOutputFormat(prompt string, format string) string {
	return fmt.Sprintf("%s\n\nProvide your response in %s format.", prompt, format)
}

func (e *MetaPromptEngine) trackUsage(templateID string, request PromptRequest) {
	// Track usage for analytics and learning
	template := e.templates[templateID]
	if template != nil {
		template.UsageCount++
		template.LastUsed = time.Now()
	}
}

func (e *MetaPromptEngine) evaluateExperiment(experiment *Experiment) {
	// Statistical analysis to determine winner
	// Simplified - in production use proper statistical tests
	
	variantAMetric := fmt.Sprintf("A_%s", "success")
	variantBMetric := fmt.Sprintf("B_%s", "success")
	
	avgA := average(experiment.Metrics[variantAMetric])
	avgB := average(experiment.Metrics[variantBMetric])
	
	if avgA > avgB {
		experiment.Winner = "A"
	} else {
		experiment.Winner = "B"
	}
	
	now := time.Now()
	experiment.EndTime = &now
}

func (e *MetaPromptEngine) createOptimization(template *PromptTemplate, feedback Feedback) {
	// Create an optimized version of the template
	optimized := e.applyOptimizationTechniques(template.Template, feedback)
	
	optimization := PromptOptimization{
		OriginalPrompt:   template.Template,
		OptimizedPrompt:  optimized,
		ImprovementScore: 0.0, // Will be calculated after testing
		Technique:        "auto-optimization",
		Timestamp:        time.Now(),
	}
	
	e.optimizations = append(e.optimizations, optimization)
	
	// Create new version of template
	template.Template = optimized
	template.Version++
}

func (e *MetaPromptEngine) applyOptimizationTechniques(prompt string, feedback Feedback) string {
	// Apply various optimization techniques based on feedback
	optimized := prompt
	
	if feedback.TooVerbose {
		optimized = e.makeMoreConcise(optimized)
	}
	
	if feedback.LacksClarity {
		optimized = e.improveClarity(optimized)
	}
	
	if feedback.MissingContext {
		optimized = e.addContext(optimized, feedback.SuggestedContext)
	}
	
	return optimized
}

func (e *MetaPromptEngine) makeMoreConcise(prompt string) string {
	// Simplify prompt language
	return prompt // Simplified implementation
}

func (e *MetaPromptEngine) improveClarity(prompt string) string {
	// Add structure and clarity
	return prompt // Simplified implementation
}

func (e *MetaPromptEngine) addContext(prompt string, context string) string {
	return fmt.Sprintf("%s\n\nAdditional context: %s", prompt, context)
}

func (e *MetaPromptEngine) generateBaseTemplate(request PromptRequest) string {
	return fmt.Sprintf("Perform %s for %s", request.Task, request.Category)
}

// Helper functions

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func extractVariables(vars map[string]string) []string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	return keys
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Request and response types

// PromptRequest represents a request for prompt generation
type PromptRequest struct {
	Category          string            `json:"category"`
	Task              string            `json:"task"`
	Variables         map[string]string `json:"variables"`
	RequiresReasoning bool              `json:"requires_reasoning"`
	Examples          []Example         `json:"examples,omitempty"`
	Role              string            `json:"role,omitempty"`
	Constraints       []string          `json:"constraints,omitempty"`
	OutputFormat      string            `json:"output_format,omitempty"`
}

// Feedback represents feedback on a generated prompt
type Feedback struct {
	SuccessScore     float64 `json:"success_score"`
	TooVerbose       bool    `json:"too_verbose"`
	LacksClarity     bool    `json:"lacks_clarity"`
	MissingContext   bool    `json:"missing_context"`
	SuggestedContext string  `json:"suggested_context,omitempty"`
}