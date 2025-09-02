package templates

import (
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/meta-prompt-engine/internal/models"
)

// GetBuiltinTemplates returns all built-in prompt templates
func GetBuiltinTemplates() []*models.PromptTemplate {
	return []*models.PromptTemplate{
		// Code Generation Templates
		{
			ID:       "code_generation_basic",
			Name:     "Basic Code Generation",
			Version:  "1.0.0",
			Category: "code_generation",
			Template: `Generate {{.language}} code for the following requirement:

{{.requirement}}

Requirements:
- Follow {{.language}} best practices
- Include proper error handling
- Add meaningful comments
- Ensure code is production-ready

{{if .constraints}}
Additional Constraints:
{{.constraints}}
{{end}}

{{if .examples}}
Examples for reference:
{{.examples}}
{{end}}

Provide the code in a markdown code block.`,
			Variables: []models.TemplateVariable{
				{Name: "language", Type: "string", Required: true, Description: "Programming language"},
				{Name: "requirement", Type: "string", Required: true, Description: "Code requirement"},
				{Name: "constraints", Type: "string", Required: false, Description: "Additional constraints"},
				{Name: "examples", Type: "string", Required: false, Description: "Example code"},
			},
		},
		{
			ID:       "code_refactor",
			Name:     "Code Refactoring",
			Version:  "1.0.0",
			Category: "code_generation",
			Template: `Refactor the following {{.language}} code to improve its quality:

\`\`\`{{.language}}
{{.code}}
\`\`\`

Refactoring Goals:
{{range .goals}}
- {{.}}
{{end}}

Apply these refactoring techniques:
- Extract methods for repeated code
- Improve naming conventions
- Reduce complexity
- Enhance readability
- Optimize performance where applicable

Provide the refactored code with explanations for major changes.`,
			Variables: []models.TemplateVariable{
				{Name: "language", Type: "string", Required: true},
				{Name: "code", Type: "string", Required: true},
				{Name: "goals", Type: "array", Required: true, DefaultValue: []string{"readability", "maintainability"}},
			},
		},
		
		// Testing Templates
		{
			ID:       "test_generation",
			Name:     "Test Generation",
			Version:  "1.0.0",
			Category: "testing",
			Template: `Generate comprehensive {{.test_type}} tests for the following code:

\`\`\`{{.language}}
{{.code}}
\`\`\`

Test Framework: {{.framework}}

Requirements:
- Cover all public methods/functions
- Include edge cases
- Test error conditions
- Aim for {{.coverage}}% code coverage
- Follow {{.framework}} best practices

{{if .mocking_required}}
Include appropriate mocks for external dependencies.
{{end}}

Generate the complete test file.`,
			Variables: []models.TemplateVariable{
				{Name: "language", Type: "string", Required: true},
				{Name: "code", Type: "string", Required: true},
				{Name: "test_type", Type: "string", Required: true, DefaultValue: "unit"},
				{Name: "framework", Type: "string", Required: true},
				{Name: "coverage", Type: "number", Required: false, DefaultValue: 80},
				{Name: "mocking_required", Type: "boolean", Required: false, DefaultValue: false},
			},
		},
		
		// Documentation Templates
		{
			ID:       "api_documentation",
			Name:     "API Documentation",
			Version:  "1.0.0",
			Category: "documentation",
			Template: `Generate comprehensive API documentation for the following endpoint:

Endpoint: {{.method}} {{.path}}
{{if .request_body}}
Request Body:
\`\`\`json
{{.request_body}}
\`\`\`
{{end}}

{{if .response_body}}
Response Body:
\`\`\`json
{{.response_body}}
\`\`\`
{{end}}

Documentation Format: {{.format}}

Include:
- Endpoint description
- Authentication requirements
- Request/Response schemas
- Status codes
- Error responses
- Usage examples
- Rate limiting information`,
			Variables: []models.TemplateVariable{
				{Name: "method", Type: "string", Required: true},
				{Name: "path", Type: "string", Required: true},
				{Name: "request_body", Type: "string", Required: false},
				{Name: "response_body", Type: "string", Required: false},
				{Name: "format", Type: "string", Required: false, DefaultValue: "OpenAPI"},
			},
		},
		
		// Code Review Templates
		{
			ID:       "code_review",
			Name:     "Code Review",
			Version:  "1.0.0",
			Category: "review",
			Template: `Perform a thorough code review of the following {{.language}} code:

\`\`\`{{.language}}
{{.code}}
\`\`\`

Review Criteria:
- Code quality and readability
- Performance considerations
- Security vulnerabilities
- Best practices adherence
- Potential bugs
- Test coverage adequacy

{{if .context}}
Additional Context:
{{.context}}
{{end}}

Provide:
1. Overall assessment (1-10 score)
2. Critical issues that must be fixed
3. Suggestions for improvement
4. Positive aspects worth maintaining`,
			Variables: []models.TemplateVariable{
				{Name: "language", Type: "string", Required: true},
				{Name: "code", Type: "string", Required: true},
				{Name: "context", Type: "string", Required: false},
			},
		},
		
		// Architecture Templates
		{
			ID:       "system_design",
			Name:     "System Design",
			Version:  "1.0.0",
			Category: "architecture",
			Template: `Design a system architecture for the following requirements:

{{.requirements}}

Constraints:
- Scale: {{.scale}}
- Budget: {{.budget}}
- Technology Stack: {{.tech_stack}}
- Timeline: {{.timeline}}

Provide:
1. High-level architecture diagram description
2. Component breakdown
3. Data flow
4. Technology choices with justification
5. Scaling strategy
6. Security considerations
7. Cost estimation`,
			Variables: []models.TemplateVariable{
				{Name: "requirements", Type: "string", Required: true},
				{Name: "scale", Type: "string", Required: true},
				{Name: "budget", Type: "string", Required: false},
				{Name: "tech_stack", Type: "string", Required: false},
				{Name: "timeline", Type: "string", Required: false},
			},
		},
		
		// Security Templates
		{
			ID:       "security_audit",
			Name:     "Security Audit",
			Version:  "1.0.0",
			Category: "security",
			Template: `Perform a security audit on the following code:

\`\`\`{{.language}}
{{.code}}
\`\`\`

Security Focus Areas:
- Input validation
- Authentication/Authorization
- SQL injection
- XSS vulnerabilities
- CSRF protection
- Sensitive data exposure
- Dependency vulnerabilities

Compliance Requirements: {{.compliance}}

Provide:
1. Vulnerability assessment with severity ratings
2. Specific remediation steps
3. Security best practices recommendations`,
			Variables: []models.TemplateVariable{
				{Name: "language", Type: "string", Required: true},
				{Name: "code", Type: "string", Required: true},
				{Name: "compliance", Type: "string", Required: false, DefaultValue: "OWASP Top 10"},
			},
		},
		
		// Performance Templates
		{
			ID:       "performance_optimization",
			Name:     "Performance Optimization",
			Version:  "1.0.0",
			Category: "performance",
			Template: `Analyze and optimize the performance of the following code:

\`\`\`{{.language}}
{{.code}}
\`\`\`

Performance Metrics:
- Current: {{.current_metrics}}
- Target: {{.target_metrics}}

Focus Areas:
- Algorithm complexity
- Memory usage
- Database queries
- Caching opportunities
- Parallelization potential

Provide optimized code with performance improvement estimates.`,
			Variables: []models.TemplateVariable{
				{Name: "language", Type: "string", Required: true},
				{Name: "code", Type: "string", Required: true},
				{Name: "current_metrics", Type: "string", Required: false},
				{Name: "target_metrics", Type: "string", Required: false},
			},
		},
		
		// Database Templates
		{
			ID:       "sql_optimization",
			Name:     "SQL Query Optimization",
			Version:  "1.0.0",
			Category: "database",
			Template: `Optimize the following SQL query:

\`\`\`sql
{{.query}}
\`\`\`

Database: {{.database_type}}
Table Sizes: {{.table_info}}

Optimization Goals:
- Reduce execution time
- Minimize resource usage
- Improve scalability

Provide:
1. Optimized query
2. Explanation of changes
3. Index recommendations
4. Expected performance improvement`,
			Variables: []models.TemplateVariable{
				{Name: "query", Type: "string", Required: true},
				{Name: "database_type", Type: "string", Required: true, DefaultValue: "PostgreSQL"},
				{Name: "table_info", Type: "string", Required: false},
			},
		},
		
		// Debug Templates
		{
			ID:       "bug_diagnosis",
			Name:     "Bug Diagnosis",
			Version:  "1.0.0",
			Category: "debugging",
			Template: `Diagnose the following bug:

Error Message:
{{.error}}

Code:
\`\`\`{{.language}}
{{.code}}
\`\`\`

{{if .stack_trace}}
Stack Trace:
{{.stack_trace}}
{{end}}

{{if .logs}}
Relevant Logs:
{{.logs}}
{{end}}

Provide:
1. Root cause analysis
2. Step-by-step fix
3. Prevention strategies
4. Testing recommendations`,
			Variables: []models.TemplateVariable{
				{Name: "error", Type: "string", Required: true},
				{Name: "code", Type: "string", Required: true},
				{Name: "language", Type: "string", Required: true},
				{Name: "stack_trace", Type: "string", Required: false},
				{Name: "logs", Type: "string", Required: false},
			},
		},
	}
}

// GetPromptChains returns built-in prompt chains for complex workflows
func GetPromptChains() []*models.PromptChain {
	return []*models.PromptChain{
		{
			ID:          "full_feature_development",
			Name:        "Full Feature Development",
			Description: "Complete feature development from requirements to deployment",
			Steps: []models.ChainStep{
				{
					ID:             "analyze_requirements",
					Name:           "Analyze Requirements",
					TemplateID:     "requirement_analysis",
					InputMapping:   map[string]string{"requirements": "user_requirements"},
					OutputVariable: "analyzed_requirements",
				},
				{
					ID:             "design_architecture",
					Name:           "Design Architecture",
					TemplateID:     "system_design",
					InputMapping:   map[string]string{"requirements": "analyzed_requirements"},
					OutputVariable: "architecture",
				},
				{
					ID:             "generate_code",
					Name:           "Generate Code",
					TemplateID:     "code_generation_basic",
					InputMapping:   map[string]string{"requirement": "analyzed_requirements"},
					OutputVariable: "generated_code",
				},
				{
					ID:             "generate_tests",
					Name:           "Generate Tests",
					TemplateID:     "test_generation",
					InputMapping:   map[string]string{"code": "generated_code"},
					OutputVariable: "test_code",
				},
				{
					ID:             "security_review",
					Name:           "Security Review",
					TemplateID:     "security_audit",
					InputMapping:   map[string]string{"code": "generated_code"},
					OutputVariable: "security_report",
				},
				{
					ID:             "generate_docs",
					Name:           "Generate Documentation",
					TemplateID:     "api_documentation",
					InputMapping:   map[string]string{"code": "generated_code"},
					OutputVariable: "documentation",
				},
			},
		},
		{
			ID:          "code_review_pipeline",
			Name:        "Comprehensive Code Review",
			Description: "Multi-stage code review with automated fixes",
			Steps: []models.ChainStep{
				{
					ID:             "initial_review",
					Name:           "Initial Code Review",
					TemplateID:     "code_review",
					OutputVariable: "review_results",
				},
				{
					ID:             "security_check",
					Name:           "Security Analysis",
					TemplateID:     "security_audit",
					OutputVariable: "security_issues",
				},
				{
					ID:             "performance_check",
					Name:           "Performance Analysis",
					TemplateID:     "performance_optimization",
					OutputVariable: "performance_suggestions",
				},
				{
					ID:             "apply_fixes",
					Name:           "Apply Automatic Fixes",
					TemplateID:     "code_refactor",
					InputMapping: map[string]string{
						"code":  "original_code",
						"goals": "review_results",
					},
					OutputVariable: "fixed_code",
				},
			},
		},
	}
}