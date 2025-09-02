package engine

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/meta-prompt-engine/internal/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// MetaPromptEngine handles dynamic prompt construction and optimization
type MetaPromptEngine struct {
	templates     map[string]*models.PromptTemplate
	chains        map[string]*models.PromptChain
	abTests       map[string]*models.ABTestConfig
	executions    []models.PromptExecution
	logger        *logrus.Logger
	tracer        trace.Tracer
	llmClient     LLMClient
	optimizer     *PromptOptimizer
}

// LLMClient interface for LLM interactions
type LLMClient interface {
	Complete(ctx context.Context, prompt string, model string) (string, int, error)
}

// NewMetaPromptEngine creates a new meta prompt engine
func NewMetaPromptEngine(llmClient LLMClient, logger *logrus.Logger) *MetaPromptEngine {
	return &MetaPromptEngine{
		templates:  make(map[string]*models.PromptTemplate),
		chains:     make(map[string]*models.PromptChain),
		abTests:    make(map[string]*models.ABTestConfig),
		executions: []models.PromptExecution{},
		logger:     logger,
		tracer:     otel.Tracer("meta-prompt-engine"),
		llmClient:  llmClient,
		optimizer:  NewPromptOptimizer(logger),
	}
}

// RegisterTemplate registers a new prompt template
func (e *MetaPromptEngine) RegisterTemplate(template *models.PromptTemplate) error {
	if template.ID == "" {
		template.ID = uuid.New().String()
	}
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	
	// Validate template syntax
	if err := e.validateTemplate(template); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}
	
	e.templates[template.ID] = template
	e.logger.WithField("template_id", template.ID).Info("Registered prompt template")
	
	return nil
}

// ExecuteTemplate executes a prompt template with variables
func (e *MetaPromptEngine) ExecuteTemplate(ctx context.Context, templateID string, variables map[string]interface{}, model string) (*models.PromptExecution, error) {
	ctx, span := e.tracer.Start(ctx, "ExecuteTemplate")
	defer span.End()
	
	template, exists := e.templates[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}
	
	// Check if this execution is part of an A/B test
	if testVariant := e.getActiveTestVariant(templateID); testVariant != nil {
		templateID = testVariant.TemplateID
		template = e.templates[templateID]
		// Merge test variant variables
		for k, v := range testVariant.Variables {
			if _, exists := variables[k]; !exists {
				variables[k] = v
			}
		}
	}
	
	// Validate required variables
	if err := e.validateVariables(template, variables); err != nil {
		return nil, err
	}
	
	// Render the prompt
	renderedPrompt, err := e.renderTemplate(template, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}
	
	// Optimize prompt if optimizer is available
	if e.optimizer != nil {
		renderedPrompt = e.optimizer.Optimize(renderedPrompt, model)
	}
	
	// Execute with LLM
	startTime := time.Now()
	response, tokens, err := e.llmClient.Complete(ctx, renderedPrompt, model)
	latency := time.Since(startTime).Milliseconds()
	
	// Record execution
	execution := models.PromptExecution{
		ID:             uuid.New().String(),
		TemplateID:     templateID,
		Variables:      variables,
		RenderedPrompt: renderedPrompt,
		Model:          model,
		Response:       response,
		TokensUsed:     tokens,
		LatencyMs:      float64(latency),
		Success:        err == nil,
		CreatedAt:      time.Now(),
	}
	
	if err != nil {
		execution.Error = err.Error()
	}
	
	e.recordExecution(&execution)
	e.updateTemplateMetrics(template, &execution)
	
	return &execution, err
}

// ExecuteChain executes a prompt chain
func (e *MetaPromptEngine) ExecuteChain(ctx context.Context, chainID string, initialVariables map[string]interface{}, model string) (map[string]interface{}, error) {
	ctx, span := e.tracer.Start(ctx, "ExecuteChain")
	defer span.End()
	
	chain, exists := e.chains[chainID]
	if !exists {
		return nil, fmt.Errorf("chain not found: %s", chainID)
	}
	
	// Initialize context with initial variables
	context := make(map[string]interface{})
	for k, v := range initialVariables {
		context[k] = v
	}
	
	// Execute each step
	for i, step := range chain.Steps {
		e.logger.WithFields(logrus.Fields{
			"chain_id": chainID,
			"step":     i,
			"step_name": step.Name,
		}).Debug("Executing chain step")
		
		// Check condition if specified
		if step.Condition != "" && !e.evaluateCondition(step.Condition, context) {
			e.logger.WithField("step", step.Name).Debug("Skipping step due to condition")
			continue
		}
		
		// Map inputs from context
		stepVariables := make(map[string]interface{})
		for target, source := range step.InputMapping {
			if val, exists := context[source]; exists {
				stepVariables[target] = val
			}
		}
		
		// Execute with retry policy
		var execution *models.PromptExecution
		var err error
		
		retryPolicy := step.RetryPolicy
		if retryPolicy == nil {
			retryPolicy = &models.RetryPolicy{MaxAttempts: 1}
		}
		
		for attempt := 0; attempt < retryPolicy.MaxAttempts; attempt++ {
			if attempt > 0 {
				backoff := time.Duration(float64(retryPolicy.BackoffMs) * 
					powFloat(retryPolicy.BackoffMultiplier, float64(attempt-1)))
				time.Sleep(backoff * time.Millisecond)
			}
			
			execution, err = e.ExecuteTemplate(ctx, step.TemplateID, stepVariables, model)
			if err == nil {
				break
			}
			
			e.logger.WithError(err).WithField("attempt", attempt+1).Warn("Chain step failed, retrying")
		}
		
		if err != nil {
			return nil, fmt.Errorf("chain step %s failed: %w", step.Name, err)
		}
		
		// Store output in context
		if step.OutputVariable != "" {
			context[step.OutputVariable] = execution.Response
		}
	}
	
	return context, nil
}

// StartABTest starts a new A/B test
func (e *MetaPromptEngine) StartABTest(config *models.ABTestConfig) error {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	config.Status = "active"
	config.StartedAt = time.Now()
	
	// Validate traffic split
	totalTraffic := 0.0
	for _, traffic := range config.TrafficSplit {
		totalTraffic += traffic
	}
	if totalTraffic != 100.0 {
		return fmt.Errorf("traffic split must sum to 100, got %f", totalTraffic)
	}
	
	e.abTests[config.ID] = config
	e.logger.WithField("test_id", config.ID).Info("Started A/B test")
	
	return nil
}

// RecordFeedback records user feedback for an execution
func (e *MetaPromptEngine) RecordFeedback(executionID string, feedback *models.ExecutionFeedback) error {
	for i, exec := range e.executions {
		if exec.ID == executionID {
			feedback.CreatedAt = time.Now()
			e.executions[i].Feedback = feedback
			
			// Update template quality score based on feedback
			if template, exists := e.templates[exec.TemplateID]; exists {
				e.updateQualityScore(template, feedback.Rating)
			}
			
			return nil
		}
	}
	return fmt.Errorf("execution not found: %s", executionID)
}

// GetTemplateRecommendations recommends templates based on task
func (e *MetaPromptEngine) GetTemplateRecommendations(task string, limit int) []*models.PromptTemplate {
	// Simple implementation - in production, use ML model
	recommendations := []*models.PromptTemplate{}
	
	for _, template := range e.templates {
		if strings.Contains(strings.ToLower(template.Category), strings.ToLower(task)) {
			recommendations = append(recommendations, template)
		}
		if len(recommendations) >= limit {
			break
		}
	}
	
	// Sort by performance
	// In production, implement proper sorting
	
	return recommendations
}

// Private helper methods

func (e *MetaPromptEngine) validateTemplate(tmpl *models.PromptTemplate) error {
	// Parse template to check syntax
	_, err := template.New("validate").Parse(tmpl.Template)
	if err != nil {
		return fmt.Errorf("template syntax error: %w", err)
	}
	
	// Validate variables
	for _, v := range tmpl.Variables {
		if v.Name == "" {
			return fmt.Errorf("variable must have a name")
		}
		if v.Validation != "" {
			if _, err := regexp.Compile(v.Validation); err != nil {
				return fmt.Errorf("invalid validation regex for %s: %w", v.Name, err)
			}
		}
	}
	
	return nil
}

func (e *MetaPromptEngine) validateVariables(template *models.PromptTemplate, variables map[string]interface{}) error {
	for _, v := range template.Variables {
		val, exists := variables[v.Name]
		
		// Check required
		if v.Required && !exists {
			return fmt.Errorf("required variable missing: %s", v.Name)
		}
		
		// Use default if not provided
		if !exists && v.DefaultValue != nil {
			variables[v.Name] = v.DefaultValue
			continue
		}
		
		// Validate format if regex provided
		if v.Validation != "" && exists {
			regex, _ := regexp.Compile(v.Validation)
			if !regex.MatchString(fmt.Sprintf("%v", val)) {
				return fmt.Errorf("variable %s failed validation", v.Name)
			}
		}
	}
	
	return nil
}

func (e *MetaPromptEngine) renderTemplate(tmpl *models.PromptTemplate, variables map[string]interface{}) (string, error) {
	t, err := template.New("prompt").Parse(tmpl.Template)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	if err := t.Execute(&buf, variables); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

func (e *MetaPromptEngine) recordExecution(execution *models.PromptExecution) {
	e.executions = append(e.executions, *execution)
	
	// Keep only last 1000 executions in memory
	if len(e.executions) > 1000 {
		e.executions = e.executions[len(e.executions)-1000:]
	}
}

func (e *MetaPromptEngine) updateTemplateMetrics(template *models.PromptTemplate, execution *models.PromptExecution) {
	metrics := &template.Performance
	
	// Update metrics
	metrics.TotalExecutions++
	metrics.LastExecuted = execution.CreatedAt
	
	if execution.Success {
		metrics.SuccessRate = (metrics.SuccessRate*float64(metrics.TotalExecutions-1) + 1) / float64(metrics.TotalExecutions)
	} else {
		metrics.SuccessRate = (metrics.SuccessRate * float64(metrics.TotalExecutions-1)) / float64(metrics.TotalExecutions)
	}
	
	// Update average tokens
	metrics.AverageTokens = int((float64(metrics.AverageTokens)*float64(metrics.TotalExecutions-1) + 
		float64(execution.TokensUsed)) / float64(metrics.TotalExecutions))
	
	// Update average latency
	metrics.AverageLatency = (metrics.AverageLatency*float64(metrics.TotalExecutions-1) + 
		execution.LatencyMs) / float64(metrics.TotalExecutions)
}

func (e *MetaPromptEngine) updateQualityScore(template *models.PromptTemplate, rating int) {
	// Simple moving average - in production, use more sophisticated method
	weight := 0.1 // Weight for new rating
	template.Performance.QualityScore = template.Performance.QualityScore*(1-weight) + float64(rating)*20*weight
}

func (e *MetaPromptEngine) getActiveTestVariant(templateID string) *models.TestVariant {
	for _, test := range e.abTests {
		if test.Status != "active" {
			continue
		}
		
		// Check if template is part of this test
		for _, variant := range test.Variants {
			if variant.TemplateID == templateID {
				// Randomly select variant based on traffic split
				r := rand.Float64() * 100
				cumulative := 0.0
				for variantID, traffic := range test.TrafficSplit {
					cumulative += traffic
					if r <= cumulative {
						for _, v := range test.Variants {
							if v.ID == variantID {
								return &v
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (e *MetaPromptEngine) evaluateCondition(condition string, context map[string]interface{}) bool {
	// Simple implementation - in production, use proper expression evaluator
	// For now, just check if a variable exists and is truthy
	if val, exists := context[condition]; exists {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v != ""
		case int, int64, float64:
			return true
		default:
			return val != nil
		}
	}
	return false
}

func powFloat(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}