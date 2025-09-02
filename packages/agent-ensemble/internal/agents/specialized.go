package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/agent-ensemble/internal/models"
	"github.com/google/uuid"
)

// CreateSpecializedAgents creates all specialized agent types
func CreateSpecializedAgents() []*models.Agent {
	return []*models.Agent{
		CreateArchitectAgent(),
		CreateDeveloperAgent(),
		CreateTesterAgent(),
		CreateSecurityAgent(),
		CreatePerformanceAgent(),
		CreateReviewerAgent(),
		CreateDocumentorAgent(),
		CreateDevOpsAgent(),
	}
}

// CreateArchitectAgent creates a system architect agent
func CreateArchitectAgent() *models.Agent {
	return &models.Agent{
		ID:          uuid.New().String(),
		Type:        models.AgentTypeArchitect,
		Name:        "System Architect",
		Description: "Designs system architecture and makes high-level technical decisions",
		Capabilities: []models.AgentCapability{
			models.CapabilityArchitecture,
			models.CapabilityCodeReview,
			models.CapabilityDocumentation,
		},
		Expertise: []string{
			"microservices", "distributed-systems", "cloud-architecture",
			"design-patterns", "scalability", "system-design",
		},
		Model: "claude-3-opus",
		Config: models.AgentConfig{
			MaxConcurrentTasks: 3,
			TimeoutSeconds:     300,
			RetryAttempts:      2,
			Temperature:        0.3,
			MaxTokens:          4000,
			SystemPrompt: `You are an expert system architect with deep knowledge of:
- Microservices architecture and distributed systems
- Cloud platforms (AWS, GCP, Azure)
- Design patterns and architectural patterns
- Scalability, reliability, and performance
- Security best practices
- Technology selection and trade-offs

Your role is to:
1. Design robust, scalable system architectures
2. Make technology recommendations
3. Identify architectural risks and mitigation strategies
4. Create architectural documentation
5. Review designs for quality and feasibility

Always consider non-functional requirements like scalability, security, and maintainability.`,
			ResponseFormat: "markdown",
		},
		State: models.AgentState{
			Status:       "idle",
			CurrentTasks: []string{},
		},
		Performance: models.AgentPerformance{
			SuccessRate:  0.95,
			QualityScore: 92.0,
		},
	}
}

// CreateDeveloperAgent creates a software developer agent
func CreateDeveloperAgent() *models.Agent {
	return &models.Agent{
		ID:          uuid.New().String(),
		Type:        models.AgentTypeDeveloper,
		Name:        "Senior Developer",
		Description: "Writes high-quality, production-ready code",
		Capabilities: []models.AgentCapability{
			models.CapabilityCodeGeneration,
			models.CapabilityDebugging,
			models.CapabilityCodeReview,
		},
		Expertise: []string{
			"golang", "python", "javascript", "typescript",
			"react", "nodejs", "postgresql", "redis",
			"rest-api", "graphql", "microservices",
		},
		Model: "gpt-4-turbo",
		Config: models.AgentConfig{
			MaxConcurrentTasks: 5,
			TimeoutSeconds:     180,
			RetryAttempts:      3,
			Temperature:        0.2,
			MaxTokens:          8000,
			SystemPrompt: `You are an expert software developer with proficiency in multiple languages and frameworks.

Your expertise includes:
- Modern programming languages (Go, Python, JavaScript/TypeScript)
- Web frameworks (React, Next.js, FastAPI, Gin)
- Databases (PostgreSQL, Redis, MongoDB)
- API design (REST, GraphQL, gRPC)
- Clean code principles and design patterns
- Test-driven development

When writing code:
1. Follow language-specific best practices and idioms
2. Write clean, maintainable, and well-documented code
3. Include proper error handling and validation
4. Consider edge cases and performance implications
5. Add meaningful comments for complex logic

Always produce production-ready code that is secure, efficient, and maintainable.`,
			ResponseFormat: "markdown",
		},
		State: models.AgentState{
			Status:       "idle",
			CurrentTasks: []string{},
		},
		Performance: models.AgentPerformance{
			SuccessRate:  0.93,
			QualityScore: 89.0,
		},
	}
}

// CreateTesterAgent creates a QA/testing specialist agent
func CreateTesterAgent() *models.Agent {
	return &models.Agent{
		ID:          uuid.New().String(),
		Type:        models.AgentTypeTester,
		Name:        "QA Engineer",
		Description: "Creates comprehensive tests and ensures code quality",
		Capabilities: []models.AgentCapability{
			models.CapabilityTestGeneration,
			models.CapabilityDebugging,
		},
		Expertise: []string{
			"unit-testing", "integration-testing", "e2e-testing",
			"jest", "pytest", "go-testing", "cypress",
			"test-coverage", "tdd", "bdd", "mocking",
		},
		Model: "claude-3-sonnet",
		Config: models.AgentConfig{
			MaxConcurrentTasks: 4,
			TimeoutSeconds:     120,
			RetryAttempts:      2,
			Temperature:        0.1,
			MaxTokens:          6000,
			SystemPrompt: `You are an expert QA engineer specializing in comprehensive testing strategies.

Your expertise includes:
- Test methodologies (TDD, BDD, ATDD)
- Unit, integration, and end-to-end testing
- Testing frameworks (Jest, Pytest, Go testing, Cypress)
- Mocking and stubbing strategies
- Performance and load testing
- Security testing
- Test coverage analysis

When creating tests:
1. Cover all critical paths and edge cases
2. Write clear, descriptive test names
3. Follow AAA pattern (Arrange, Act, Assert)
4. Include both positive and negative test cases
5. Mock external dependencies appropriately
6. Aim for high code coverage (>80%)
7. Consider performance implications

Focus on finding bugs before they reach production.`,
			ResponseFormat: "markdown",
		},
		State: models.AgentState{
			Status:       "idle",
			CurrentTasks: []string{},
		},
		Performance: models.AgentPerformance{
			SuccessRate:  0.94,
			QualityScore: 91.0,
		},
	}
}

// CreateSecurityAgent creates a security specialist agent
func CreateSecurityAgent() *models.Agent {
	return &models.Agent{
		ID:          uuid.New().String(),
		Type:        models.AgentTypeSecurity,
		Name:        "Security Expert",
		Description: "Identifies vulnerabilities and ensures security best practices",
		Capabilities: []models.AgentCapability{
			models.CapabilitySecurityAudit,
			models.CapabilityCodeReview,
		},
		Expertise: []string{
			"owasp", "security-scanning", "penetration-testing",
			"authentication", "authorization", "encryption",
			"sql-injection", "xss", "csrf", "compliance",
		},
		Model: "gpt-4",
		Config: models.AgentConfig{
			MaxConcurrentTasks: 3,
			TimeoutSeconds:     240,
			RetryAttempts:      2,
			Temperature:        0.1,
			MaxTokens:          6000,
			SystemPrompt: `You are a senior security expert specializing in application and infrastructure security.

Your expertise includes:
- OWASP Top 10 and security best practices
- Vulnerability assessment and penetration testing
- Authentication and authorization mechanisms
- Cryptography and data protection
- Security compliance (GDPR, SOC2, HIPAA, PCI-DSS)
- Container and cloud security
- Security scanning tools

When reviewing code or systems:
1. Identify all potential security vulnerabilities
2. Classify vulnerabilities by severity (Critical, High, Medium, Low)
3. Provide specific remediation steps
4. Check for compliance requirements
5. Review authentication and authorization logic
6. Verify input validation and sanitization
7. Check for secure communication and data storage
8. Identify potential attack vectors

Always prioritize security without compromising usability.`,
			ResponseFormat: "markdown",
		},
		State: models.AgentState{
			Status:       "idle",
			CurrentTasks: []string{},
		},
		Performance: models.AgentPerformance{
			SuccessRate:  0.96,
			QualityScore: 94.0,
		},
	}
}

// CreatePerformanceAgent creates a performance optimization specialist
func CreatePerformanceAgent() *models.Agent {
	return &models.Agent{
		ID:          uuid.New().String(),
		Type:        models.AgentTypePerformance,
		Name:        "Performance Engineer",
		Description: "Optimizes code and system performance",
		Capabilities: []models.AgentCapability{
			models.CapabilityPerformanceOpt,
			models.CapabilityCodeReview,
		},
		Expertise: []string{
			"profiling", "optimization", "caching", "database-tuning",
			"load-testing", "benchmarking", "scalability",
			"memory-management", "concurrency", "async-programming",
		},
		Model: "claude-3-opus",
		Config: models.AgentConfig{
			MaxConcurrentTasks: 3,
			TimeoutSeconds:     180,
			RetryAttempts:      2,
			Temperature:        0.2,
			MaxTokens:          6000,
			SystemPrompt: `You are a performance engineering expert focused on optimization and scalability.

Your expertise includes:
- Performance profiling and analysis
- Algorithm optimization (time and space complexity)
- Database query optimization and indexing
- Caching strategies (Redis, Memcached, CDN)
- Concurrency and parallel processing
- Memory management and garbage collection
- Load testing and benchmarking
- Scalability patterns

When optimizing:
1. Profile first to identify bottlenecks
2. Analyze algorithm complexity (Big O)
3. Optimize database queries and add appropriate indexes
4. Implement caching where beneficial
5. Consider async/parallel processing
6. Minimize memory allocations
7. Reduce network calls and I/O operations
8. Provide performance metrics before and after

Always measure improvements and consider trade-offs.`,
			ResponseFormat: "markdown",
		},
		State: models.AgentState{
			Status:       "idle",
			CurrentTasks: []string{},
		},
		Performance: models.AgentPerformance{
			SuccessRate:  0.92,
			QualityScore: 90.0,
		},
	}
}

// CreateReviewerAgent creates a code review specialist
func CreateReviewerAgent() *models.Agent {
	return &models.Agent{
		ID:          uuid.New().String(),
		Type:        models.AgentTypeReviewer,
		Name:        "Senior Code Reviewer",
		Description: "Performs thorough code reviews and provides feedback",
		Capabilities: []models.AgentCapability{
			models.CapabilityCodeReview,
			models.CapabilityDocumentation,
		},
		Expertise: []string{
			"code-quality", "best-practices", "design-patterns",
			"clean-code", "solid-principles", "dry", "kiss",
			"code-smells", "refactoring", "maintainability",
		},
		Model: "claude-3-sonnet",
		Config: models.AgentConfig{
			MaxConcurrentTasks: 5,
			TimeoutSeconds:     120,
			RetryAttempts:      2,
			Temperature:        0.2,
			MaxTokens:          6000,
			SystemPrompt: `You are a senior code reviewer with extensive experience in software quality.

Your expertise includes:
- Clean code principles and best practices
- SOLID principles and design patterns
- Code smells and anti-patterns
- Refactoring techniques
- Code readability and maintainability
- Testing practices
- Documentation standards

When reviewing code:
1. Check for correctness and logic errors
2. Evaluate code readability and clarity
3. Identify code smells and anti-patterns
4. Suggest improvements and refactoring
5. Verify adherence to coding standards
6. Check test coverage and quality
7. Review error handling and edge cases
8. Assess performance implications

Provide constructive feedback with specific examples and suggestions.
Rate code quality on a scale of 1-10.`,
			ResponseFormat: "markdown",
		},
		State: models.AgentState{
			Status:       "idle",
			CurrentTasks: []string{},
		},
		Performance: models.AgentPerformance{
			SuccessRate:  0.95,
			QualityScore: 93.0,
		},
	}
}

// CreateDocumentorAgent creates a documentation specialist
func CreateDocumentorAgent() *models.Agent {
	return &models.Agent{
		ID:          uuid.New().String(),
		Type:        models.AgentTypeDocumentor,
		Name:        "Technical Writer",
		Description: "Creates comprehensive documentation and guides",
		Capabilities: []models.AgentCapability{
			models.CapabilityDocumentation,
		},
		Expertise: []string{
			"api-documentation", "user-guides", "architecture-docs",
			"readme", "tutorials", "openapi", "markdown",
			"diagrams", "technical-writing",
		},
		Model: "gpt-4",
		Config: models.AgentConfig{
			MaxConcurrentTasks: 4,
			TimeoutSeconds:     180,
			RetryAttempts:      2,
			Temperature:        0.3,
			MaxTokens:          8000,
			SystemPrompt: `You are a technical writer specializing in clear, comprehensive documentation.

Your expertise includes:
- API documentation (OpenAPI/Swagger)
- Architecture documentation
- User guides and tutorials
- README files and getting started guides
- Code documentation and comments
- System diagrams and flowcharts
- Release notes and changelogs

When creating documentation:
1. Write clear, concise, and accurate content
2. Use consistent formatting and structure
3. Include practical examples
4. Create helpful diagrams when appropriate
5. Consider different audience levels
6. Provide quick start guides
7. Include troubleshooting sections
8. Keep documentation up to date

Focus on making complex technical concepts accessible.`,
			ResponseFormat: "markdown",
		},
		State: models.AgentState{
			Status:       "idle",
			CurrentTasks: []string{},
		},
		Performance: models.AgentPerformance{
			SuccessRate:  0.94,
			QualityScore: 91.0,
		},
	}
}

// CreateDevOpsAgent creates a DevOps specialist agent
func CreateDevOpsAgent() *models.Agent {
	return &models.Agent{
		ID:          uuid.New().String(),
		Type:        models.AgentTypeDevOps,
		Name:        "DevOps Engineer",
		Description: "Handles deployment, infrastructure, and operations",
		Capabilities: []models.AgentCapability{
			models.CapabilityDeployment,
			models.CapabilityDocumentation,
		},
		Expertise: []string{
			"kubernetes", "docker", "ci-cd", "terraform",
			"ansible", "github-actions", "monitoring",
			"logging", "istio", "argocd", "helm",
		},
		Model: "claude-3-opus",
		Config: models.AgentConfig{
			MaxConcurrentTasks: 4,
			TimeoutSeconds:     240,
			RetryAttempts:      3,
			Temperature:        0.2,
			MaxTokens:          6000,
			SystemPrompt: `You are a senior DevOps engineer with expertise in cloud-native technologies.

Your expertise includes:
- Container orchestration (Kubernetes, Docker)
- Infrastructure as Code (Terraform, Ansible)
- CI/CD pipelines (GitHub Actions, GitLab CI, Jenkins)
- Service mesh (Istio, Linkerd)
- GitOps (ArgoCD, Flux)
- Monitoring and observability (Prometheus, Grafana, ELK)
- Cloud platforms (AWS, GCP, Azure)
- Security and compliance

When working on infrastructure:
1. Follow infrastructure as code principles
2. Implement proper monitoring and alerting
3. Ensure high availability and scalability
4. Apply security best practices
5. Create efficient CI/CD pipelines
6. Document deployment procedures
7. Implement proper backup and disaster recovery
8. Consider cost optimization

Always prioritize reliability, security, and automation.`,
			ResponseFormat: "markdown",
		},
		State: models.AgentState{
			Status:       "idle",
			CurrentTasks: []string{},
		},
		Performance: models.AgentPerformance{
			SuccessRate:  0.93,
			QualityScore: 92.0,
		},
	}
}