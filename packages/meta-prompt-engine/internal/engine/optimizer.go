package engine

import (
	"strings"
	"regexp"
	"github.com/sirupsen/logrus"
)

// PromptOptimizer optimizes prompts for better performance
type PromptOptimizer struct {
	logger *logrus.Logger
	rules  []OptimizationRule
}

// OptimizationRule represents a prompt optimization rule
type OptimizationRule struct {
	Name        string
	Description string
	Apply       func(string) string
	Models      []string // specific models this applies to, empty means all
}

// NewPromptOptimizer creates a new prompt optimizer
func NewPromptOptimizer(logger *logrus.Logger) *PromptOptimizer {
	optimizer := &PromptOptimizer{
		logger: logger,
		rules:  []OptimizationRule{},
	}
	
	// Register default optimization rules
	optimizer.registerDefaultRules()
	
	return optimizer
}

// Optimize applies optimization rules to a prompt
func (o *PromptOptimizer) Optimize(prompt string, model string) string {
	optimized := prompt
	
	for _, rule := range o.rules {
		// Check if rule applies to this model
		if len(rule.Models) > 0 {
			applies := false
			for _, m := range rule.Models {
				if strings.Contains(model, m) {
					applies = true
					break
				}
			}
			if !applies {
				continue
			}
		}
		
		// Apply the rule
		before := optimized
		optimized = rule.Apply(optimized)
		
		if before != optimized {
			o.logger.WithField("rule", rule.Name).Debug("Applied optimization rule")
		}
	}
	
	return optimized
}

// registerDefaultRules registers default optimization rules
func (o *PromptOptimizer) registerDefaultRules() {
	// Remove excessive whitespace
	o.rules = append(o.rules, OptimizationRule{
		Name:        "remove_excessive_whitespace",
		Description: "Removes excessive whitespace to reduce tokens",
		Apply: func(prompt string) string {
			// Replace multiple spaces with single space
			re := regexp.MustCompile(`\s+`)
			prompt = re.ReplaceAllString(prompt, " ")
			
			// Remove trailing whitespace
			lines := strings.Split(prompt, "\n")
			for i, line := range lines {
				lines[i] = strings.TrimRight(line, " \t")
			}
			
			return strings.Join(lines, "\n")
		},
	})
	
	// Add XML tags for Claude models
	o.rules = append(o.rules, OptimizationRule{
		Name:        "add_claude_xml_tags",
		Description: "Adds XML tags for better Claude performance",
		Models:      []string{"claude", "anthropic"},
		Apply: func(prompt string) string {
			// Check if already has XML tags
			if strings.Contains(prompt, "<instructions>") {
				return prompt
			}
			
			// Add XML structure for better parsing
			sections := []string{}
			
			// Find instruction section
			if idx := strings.Index(prompt, "Instructions:"); idx != -1 {
				before := prompt[:idx]
				after := prompt[idx+13:]
				if endIdx := strings.Index(after, "\n\n"); endIdx != -1 {
					instructions := after[:endIdx]
					remainder := after[endIdx:]
					sections = append(sections, before)
					sections = append(sections, "<instructions>\n"+instructions+"\n</instructions>")
					sections = append(sections, remainder)
					return strings.Join(sections, "")
				}
			}
			
			return prompt
		},
	})
	
	// Add role clarity for GPT models
	o.rules = append(o.rules, OptimizationRule{
		Name:        "add_gpt_role_clarity",
		Description: "Adds role clarity for GPT models",
		Models:      []string{"gpt", "openai"},
		Apply: func(prompt string) string {
			// Check if already has role definition
			if strings.Contains(prompt, "You are") || strings.Contains(prompt, "Act as") {
				return prompt
			}
			
			// Add role prefix based on content
			if strings.Contains(strings.ToLower(prompt), "code") {
				return "You are an expert software engineer. " + prompt
			} else if strings.Contains(strings.ToLower(prompt), "test") {
				return "You are a quality assurance expert. " + prompt
			} else if strings.Contains(strings.ToLower(prompt), "review") {
				return "You are a senior code reviewer. " + prompt
			}
			
			return prompt
		},
	})
	
	// Optimize list formatting
	o.rules = append(o.rules, OptimizationRule{
		Name:        "optimize_list_formatting",
		Description: "Optimizes list formatting for token efficiency",
		Apply: func(prompt string) string {
			// Convert verbose lists to compact format
			re := regexp.MustCompile(`(?m)^- (.+)$`)
			prompt = re.ReplaceAllString(prompt, "• $1")
			
			return prompt
		},
	})
	
	// Add thinking tags for chain-of-thought
	o.rules = append(o.rules, OptimizationRule{
		Name:        "add_thinking_tags",
		Description: "Adds thinking tags for chain-of-thought reasoning",
		Apply: func(prompt string) string {
			// Check if this is a complex reasoning task
			keywords := []string{"analyze", "explain", "reason", "think", "consider", "evaluate"}
			isComplex := false
			for _, keyword := range keywords {
				if strings.Contains(strings.ToLower(prompt), keyword) {
					isComplex = true
					break
				}
			}
			
			if isComplex && !strings.Contains(prompt, "step by step") {
				prompt += "\n\nPlease think through this step by step before providing your answer."
			}
			
			return prompt
		},
	})
	
	// Remove redundant instructions
	o.rules = append(o.rules, OptimizationRule{
		Name:        "remove_redundant_instructions",
		Description: "Removes redundant instructions",
		Apply: func(prompt string) string {
			// Remove duplicate lines
			lines := strings.Split(prompt, "\n")
			seen := make(map[string]bool)
			unique := []string{}
			
			for _, line := range lines {
				normalized := strings.TrimSpace(strings.ToLower(line))
				if normalized == "" {
					unique = append(unique, line)
					continue
				}
				if !seen[normalized] {
					seen[normalized] = true
					unique = append(unique, line)
				}
			}
			
			return strings.Join(unique, "\n")
		},
	})
	
	// Add output format hints
	o.rules = append(o.rules, OptimizationRule{
		Name:        "add_output_format_hints",
		Description: "Adds clear output format instructions",
		Apply: func(prompt string) string {
			// Check for code generation without format specification
			if strings.Contains(strings.ToLower(prompt), "generate") && 
			   strings.Contains(strings.ToLower(prompt), "code") &&
			   !strings.Contains(strings.ToLower(prompt), "```") {
				prompt += "\n\nProvide the code in markdown code blocks with appropriate language tags."
			}
			
			// Check for JSON output
			if strings.Contains(strings.ToLower(prompt), "json") &&
			   !strings.Contains(prompt, "format") {
				prompt += "\n\nReturn the response as valid JSON."
			}
			
			return prompt
		},
	})
	
	// Optimize for specific model token limits
	o.rules = append(o.rules, OptimizationRule{
		Name:        "optimize_for_token_limits",
		Description: "Optimizes prompt for model token limits",
		Apply: func(prompt string) string {
			// Rough token estimation (1 token ≈ 4 characters)
			estimatedTokens := len(prompt) / 4
			
			// If approaching limits, compress
			if estimatedTokens > 3000 {
				// Remove examples if present
				if idx := strings.Index(prompt, "Example"); idx != -1 {
					if endIdx := strings.Index(prompt[idx:], "\n\n"); endIdx != -1 {
						prompt = prompt[:idx] + prompt[idx+endIdx:]
					}
				}
				
				// Shorten verbose sections
				prompt = strings.ReplaceAll(prompt, "Please ", "")
				prompt = strings.ReplaceAll(prompt, "Could you ", "")
				prompt = strings.ReplaceAll(prompt, "I would like you to ", "")
			}
			
			return prompt
		},
	})
}

// AddCustomRule adds a custom optimization rule
func (o *PromptOptimizer) AddCustomRule(rule OptimizationRule) {
	o.rules = append(o.rules, rule)
	o.logger.WithField("rule", rule.Name).Info("Added custom optimization rule")
}

// RemoveRule removes an optimization rule by name
func (o *PromptOptimizer) RemoveRule(name string) {
	filtered := []OptimizationRule{}
	for _, rule := range o.rules {
		if rule.Name != name {
			filtered = append(filtered, rule)
		}
	}
	o.rules = filtered
}