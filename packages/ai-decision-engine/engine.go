package aidecision

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AIDecisionEngine implements intelligent decision making using semantic similarity
type AIDecisionEngine struct {
	mu              sync.RWMutex
	rules           map[string][]*DecisionRule
	embeddingCache  map[string][]float32
	llmClient       LLMClient
	vectorStore     VectorStore
	learningEnabled bool
	feedbackHistory []*Feedback
}

// LLMClient interface for generating embeddings and reasoning
type LLMClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateReasoning(ctx context.Context, input, pattern string) (string, error)
	ExtractIntent(ctx context.Context, text string) (string, error)
}

// VectorStore interface for storing and searching embeddings
type VectorStore interface {
	Store(ctx context.Context, id string, embedding []float32, metadata map[string]interface{}) error
	Search(ctx context.Context, embedding []float32, limit int, threshold float64) ([]SearchResult, error)
	Update(ctx context.Context, id string, embedding []float32) error
	Delete(ctx context.Context, id string) error
}

// SearchResult represents a vector search result
type SearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NewAIDecisionEngine creates a new AI-powered decision engine
func NewAIDecisionEngine(llmClient LLMClient, vectorStore VectorStore) *AIDecisionEngine {
	return &AIDecisionEngine{
		rules:           make(map[string][]*DecisionRule),
		embeddingCache:  make(map[string][]float32),
		llmClient:       llmClient,
		vectorStore:     vectorStore,
		learningEnabled: true,
		feedbackHistory: make([]*Feedback, 0),
	}
}

// RegisterRule registers a new decision rule with the engine
func (e *AIDecisionEngine) RegisterRule(rule *DecisionRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Generate embedding for the rule if not provided
	if len(rule.Embedding) == 0 {
		ctx := context.Background()
		embedding, err := e.generateRuleEmbedding(ctx, rule)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
		rule.Embedding = embedding
	}

	// Store in vector database for similarity search
	metadata := map[string]interface{}{
		"category":    rule.Category,
		"pattern":     rule.Pattern,
		"description": rule.Description,
		"priority":    rule.Priority,
	}
	
	if err := e.vectorStore.Store(context.Background(), rule.ID, rule.Embedding, metadata); err != nil {
		return fmt.Errorf("failed to store in vector database: %w", err)
	}

	// Add to local rules map
	if e.rules[rule.Category] == nil {
		e.rules[rule.Category] = make([]*DecisionRule, 0)
	}
	e.rules[rule.Category] = append(e.rules[rule.Category], rule)

	return nil
}

// Decide makes an intelligent decision based on semantic similarity
func (e *AIDecisionEngine) Decide(ctx context.Context, category string, input string) (*Decision, error) {
	// Extract intent from input
	intent, err := e.llmClient.ExtractIntent(ctx, input)
	if err != nil {
		intent = input // Fallback to raw input
	}

	// Get semantic matches
	matches, err := e.GetMatches(ctx, category, input, 0.7) // 70% similarity threshold
	if err != nil {
		return nil, fmt.Errorf("failed to get matches: %w", err)
	}

	if len(matches) == 0 {
		// No matches found, use fallback or AI generation
		return e.generateFallbackDecision(ctx, category, input, intent)
	}

	// Select best match considering confidence and priority
	bestMatch := e.selectBestMatch(matches)

	// Execute the action
	result, err := bestMatch.Rule.Action(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("action execution failed: %w", err)
	}

	decision := &Decision{
		ID:         uuid.New().String(),
		Context:    category,
		Input:      input,
		Intent:     intent,
		Confidence: bestMatch.Confidence,
		Result:     result,
		Metadata: map[string]interface{}{
			"rule_id":   bestMatch.Rule.ID,
			"score":     bestMatch.Score,
			"reasoning": bestMatch.Reasoning,
		},
		Timestamp: time.Now(),
	}

	return decision, nil
}

// GetMatches returns all semantic matches above a threshold
func (e *AIDecisionEngine) GetMatches(ctx context.Context, category string, input string, threshold float64) ([]*SemanticMatch, error) {
	// Generate embedding for input
	embedding, err := e.llmClient.GenerateEmbedding(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to generate input embedding: %w", err)
	}

	// Search vector store
	searchResults, err := e.vectorStore.Search(ctx, embedding, 10, threshold)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Convert to semantic matches
	matches := make([]*SemanticMatch, 0)
	for _, result := range searchResults {
		// Find corresponding rule
		rule := e.findRuleByID(result.ID)
		if rule == nil || (category != "" && rule.Category != category) {
			continue
		}

		// Generate reasoning for the match
		reasoning, _ := e.llmClient.GenerateReasoning(ctx, input, rule.Pattern)

		match := &SemanticMatch{
			Rule:       rule,
			Score:      result.Score,
			Confidence: e.calculateConfidence(result.Score, rule.Priority),
			Reasoning:  reasoning,
		}
		matches = append(matches, match)
	}

	// Sort by confidence
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	return matches, nil
}

// LearnFromFeedback updates the engine based on feedback
func (e *AIDecisionEngine) LearnFromFeedback(ctx context.Context, decision *Decision, feedback Feedback) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Store feedback
	e.feedbackHistory = append(e.feedbackHistory, &feedback)

	if !e.learningEnabled {
		return nil
	}

	// If feedback is negative, adjust embeddings or create new rules
	if !feedback.Correct && feedback.Expected != "" {
		// Find the rule that was used
		ruleID, _ := decision.Metadata["rule_id"].(string)
		if rule := e.findRuleByID(ruleID); rule != nil {
			// Reduce priority for incorrect matches
			rule.Priority = int(math.Max(0, float64(rule.Priority-1)))
			
			// Consider creating a new rule for the expected outcome
			if e.shouldCreateNewRule(feedback) {
				newRule := &DecisionRule{
					ID:          uuid.New().String(),
					Category:    decision.Context,
					Pattern:     feedback.Expected,
					Description: fmt.Sprintf("Learned from feedback on %s", decision.ID),
					Priority:    5, // Medium priority for learned rules
					Examples:    []string{decision.Input},
				}
				e.RegisterRule(newRule)
			}
		}
	} else if feedback.Correct {
		// Increase priority for correct matches
		ruleID, _ := decision.Metadata["rule_id"].(string)
		if rule := e.findRuleByID(ruleID); rule != nil {
			rule.Priority = int(math.Min(10, float64(rule.Priority+1)))
			
			// Add input as an example
			if !contains(rule.Examples, decision.Input) {
				rule.Examples = append(rule.Examples, decision.Input)
			}
		}
	}

	return nil
}

// ExportRules exports all registered rules
func (e *AIDecisionEngine) ExportRules() map[string][]*DecisionRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	// Deep copy to prevent external modification
	export := make(map[string][]*DecisionRule)
	for category, rules := range e.rules {
		export[category] = make([]*DecisionRule, len(rules))
		copy(export[category], rules)
	}
	return export
}

// Private helper methods

func (e *AIDecisionEngine) generateRuleEmbedding(ctx context.Context, rule *DecisionRule) ([]float32, error) {
	// Combine pattern, description, and examples for richer embedding
	text := fmt.Sprintf("%s %s", rule.Pattern, rule.Description)
	if len(rule.Examples) > 0 {
		text += " Examples: " + strings.Join(rule.Examples, ", ")
	}
	
	return e.llmClient.GenerateEmbedding(ctx, text)
}

func (e *AIDecisionEngine) selectBestMatch(matches []*SemanticMatch) *SemanticMatch {
	if len(matches) == 0 {
		return nil
	}
	
	// Already sorted by confidence
	return matches[0]
}

func (e *AIDecisionEngine) calculateConfidence(score float64, priority int) float64 {
	// Combine similarity score with rule priority
	priorityBoost := float64(priority) / 10.0 * 0.2 // Max 20% boost from priority
	return math.Min(1.0, score + priorityBoost)
}

func (e *AIDecisionEngine) findRuleByID(id string) *DecisionRule {
	for _, rules := range e.rules {
		for _, rule := range rules {
			if rule.ID == id {
				return rule
			}
		}
	}
	return nil
}

func (e *AIDecisionEngine) generateFallbackDecision(ctx context.Context, category, input, intent string) (*Decision, error) {
	// Use AI to generate a decision when no rules match
	return &Decision{
		ID:         uuid.New().String(),
		Context:    category,
		Input:      input,
		Intent:     intent,
		Confidence: 0.5, // Lower confidence for generated decisions
		Result: map[string]interface{}{
			"generated": true,
			"message":   "No matching rule found, using AI generation",
		},
		Metadata: map[string]interface{}{
			"fallback": true,
		},
		Timestamp: time.Now(),
	}, nil
}

func (e *AIDecisionEngine) shouldCreateNewRule(feedback Feedback) bool {
	// Simple heuristic: create new rule if we've seen similar feedback multiple times
	count := 0
	for _, f := range e.feedbackHistory {
		if f.Expected == feedback.Expected {
			count++
		}
	}
	return count >= 3 // Create rule after 3 similar feedbacks
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}