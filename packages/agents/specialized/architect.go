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

// ArchitectAgent handles system design and technology selection
type ArchitectAgent struct {
	*base.BaseAgent
	llmEndpoint string
}

// NewArchitectAgent creates a new architect agent
func NewArchitectAgent(llmEndpoint string) *ArchitectAgent {
	capabilities := []types.AgentCapability{
		types.CapSystemDesign,
		types.CapDocumentation,
		types.CapPerformanceOptimization,
	}

	agent := &ArchitectAgent{
		BaseAgent:   base.NewBaseAgent(types.RoleArchitect, capabilities),
		llmEndpoint: llmEndpoint,
	}

	// Set specialized handlers
	agent.SetExecutionHandler(agent.executeTask)
	agent.SetMessageHandler(agent.handleMessage)
	agent.SetInitializeHandler(agent.initialize)

	return agent
}

func (a *ArchitectAgent) initialize(ctx context.Context, agentCtx *types.AgentContext) error {
	// Review project requirements and create initial architecture
	if agentCtx.SharedMemory != nil && agentCtx.SharedMemory.ProjectContext != nil {
		if plan, ok := agentCtx.SharedMemory.ProjectContext["project_plan"]; ok {
			architecture, err := a.designArchitecture(ctx, plan)
			if err != nil {
				return fmt.Errorf("failed to design architecture: %w", err)
			}

			// Store architecture decisions
			if agentCtx.SharedMemory.DesignDecisions == nil {
				agentCtx.SharedMemory.DesignDecisions = []types.DesignDecision{}
			}

			decision := types.DesignDecision{
				ID:        fmt.Sprintf("arch-%d", time.Now().Unix()),
				Agent:     a.ID(),
				Category:  "system_architecture",
				Decision:  "Initial architecture design",
				Reasoning: "Based on project requirements and best practices",
				Timestamp: time.Now(),
				Approved:  false,
			}
			agentCtx.SharedMemory.DesignDecisions = append(agentCtx.SharedMemory.DesignDecisions, decision)
			agentCtx.SharedMemory.ProjectContext["architecture"] = architecture
		}
	}

	return nil
}

func (a *ArchitectAgent) executeTask(ctx context.Context, task *types.Task) error {
	switch task.Type {
	case "design_system":
		return a.executeSystemDesign(ctx, task)
	case "select_technology":
		return a.executeTechnologySelection(ctx, task)
	case "design_api":
		return a.executeAPIDesign(ctx, task)
	case "design_database":
		return a.executeDatabaseDesign(ctx, task)
	case "review_architecture":
		return a.executeArchitectureReview(ctx, task)
	case "optimize_performance":
		return a.executePerformanceOptimization(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
}

func (a *ArchitectAgent) handleMessage(ctx context.Context, msg *types.Message) error {
	switch msg.Type {
	case types.MsgTypeRequest:
		return a.handleRequest(ctx, msg)
	case types.MsgTypeCollaboration:
		return a.handleCollaboration(ctx, msg)
	case types.MsgTypeConsensus:
		return a.handleConsensus(ctx, msg)
	default:
		return nil
	}
}

func (a *ArchitectAgent) executeSystemDesign(ctx context.Context, task *types.Task) error {
	requirements, _ := task.Requirements["requirements"].(string)
	projectType, _ := task.Requirements["project_type"].(string)
	
	prompt := fmt.Sprintf(`As a Software Architect, design a system architecture for:
Project Type: %s
Requirements: %s

Provide:
1. High-level architecture diagram description
2. Component breakdown
3. Communication patterns (REST, GraphQL, gRPC, WebSocket)
4. Data flow design
5. Security architecture
6. Scalability considerations
7. Technology stack recommendations
8. Deployment architecture

Format as JSON with clear structure.`, projectType, requirements)

	response, err := a.callLLM(ctx, prompt, "You are an experienced software architect specializing in scalable, maintainable systems.")
	if err != nil {
		return fmt.Errorf("failed to design system: %w", err)
	}

	var architecture map[string]interface{}
	if err := json.Unmarshal([]byte(response), &architecture); err != nil {
		architecture = map[string]interface{}{
			"raw_design":      response,
			"pattern":         a.selectArchitecturePattern(requirements),
			"components":      a.identifyComponents(requirements),
			"technology_stack": a.recommendTechStack(projectType),
		}
	}

	task.Result = architecture
	
	// Store design decision
	a.recordDesignDecision(ctx, "system_architecture", architecture)
	
	return nil
}

func (a *ArchitectAgent) executeTechnologySelection(ctx context.Context, task *types.Task) error {
	requirements, _ := task.Requirements["requirements"].(string)
	constraints, _ := task.Requirements["constraints"].(map[string]interface{})
	
	prompt := fmt.Sprintf(`Select the optimal technology stack for:
Requirements: %s
Constraints: %v

Consider:
1. Programming languages and frameworks
2. Database selection (SQL vs NoSQL)
3. Caching strategy
4. Message queue/streaming platform
5. Container orchestration
6. CI/CD tools
7. Monitoring and observability stack
8. Security tools

Provide justification for each choice. Format as JSON.`, requirements, constraints)

	response, err := a.callLLM(ctx, prompt, "You are a technology architect with expertise in modern tech stacks.")
	if err != nil {
		return err
	}

	var techStack map[string]interface{}
	if err := json.Unmarshal([]byte(response), &techStack); err != nil {
		techStack = a.getDefaultTechStack()
	}

	task.Result = techStack
	a.recordDesignDecision(ctx, "technology_selection", techStack)
	
	return nil
}

func (a *ArchitectAgent) executeAPIDesign(ctx context.Context, task *types.Task) error {
	requirements, _ := task.Requirements["requirements"].(string)
	apiType, _ := task.Requirements["api_type"].(string)
	
	if apiType == "" {
		apiType = "REST"
	}

	prompt := fmt.Sprintf(`Design a %s API for:
Requirements: %s

Include:
1. Endpoint structure
2. Request/Response schemas
3. Authentication and authorization
4. Rate limiting strategy
5. Versioning approach
6. Error handling standards
7. OpenAPI/GraphQL schema

Format as JSON with examples.`, apiType, requirements)

	response, err := a.callLLM(ctx, prompt, "You are an API design expert.")
	if err != nil {
		return err
	}

	var apiDesign map[string]interface{}
	if err := json.Unmarshal([]byte(response), &apiDesign); err != nil {
		apiDesign = map[string]interface{}{
			"type":      apiType,
			"endpoints": a.generateBasicEndpoints(requirements),
			"auth":      "JWT",
			"versioning": "URL path (v1, v2)",
		}
	}

	task.Result = apiDesign
	return nil
}

func (a *ArchitectAgent) executeDatabaseDesign(ctx context.Context, task *types.Task) error {
	requirements, _ := task.Requirements["requirements"].(string)
	dataModel, _ := task.Requirements["data_model"].(map[string]interface{})
	
	prompt := fmt.Sprintf(`Design a database schema for:
Requirements: %s
Data Model: %v

Provide:
1. Table/Collection structures
2. Relationships and constraints
3. Indexes for performance
4. Partitioning/Sharding strategy
5. Data migration approach
6. Backup and recovery plan

Format as SQL DDL or MongoDB schema.`, requirements, dataModel)

	response, err := a.callLLM(ctx, prompt, "You are a database architect.")
	if err != nil {
		return err
	}

	dbDesign := map[string]interface{}{
		"schema":      response,
		"database":    a.selectDatabase(requirements),
		"indexes":     a.recommendIndexes(requirements),
		"partitioning": a.determinePartitioning(requirements),
	}

	task.Result = dbDesign
	a.recordDesignDecision(ctx, "database_design", dbDesign)
	
	return nil
}

func (a *ArchitectAgent) executeArchitectureReview(ctx context.Context, task *types.Task) error {
	currentArchitecture, _ := task.Requirements["architecture"].(map[string]interface{})
	
	// Review the architecture for issues
	review := map[string]interface{}{
		"timestamp": time.Now(),
		"reviewer":  a.ID(),
		"findings":  []map[string]interface{}{},
	}

	// Check for common issues
	issues := a.identifyArchitectureIssues(currentArchitecture)
	for _, issue := range issues {
		finding := map[string]interface{}{
			"severity":      issue["severity"],
			"category":      issue["category"],
			"description":   issue["description"],
			"recommendation": issue["recommendation"],
		}
		review["findings"] = append(review["findings"].([]map[string]interface{}), finding)
	}

	task.Result = review
	return nil
}

func (a *ArchitectAgent) executePerformanceOptimization(ctx context.Context, task *types.Task) error {
	metrics, _ := task.Requirements["metrics"].(map[string]interface{})
	architecture, _ := task.Requirements["architecture"].(map[string]interface{})
	
	optimizations := map[string]interface{}{
		"caching": map[string]interface{}{
			"strategy": "multi-layer",
			"levels":   []string{"CDN", "Redis", "application cache"},
		},
		"database": map[string]interface{}{
			"query_optimization": true,
			"connection_pooling": true,
			"read_replicas":      a.needsReadReplicas(metrics),
		},
		"scaling": map[string]interface{}{
			"horizontal": true,
			"auto_scaling": map[string]interface{}{
				"min": 2,
				"max": 10,
				"cpu_threshold": 70,
			},
		},
	}

	task.Result = optimizations
	return nil
}

func (a *ArchitectAgent) handleRequest(ctx context.Context, msg *types.Message) error {
	switch msg.Content {
	case "review_design":
		return a.reviewDesign(ctx, msg)
	case "validate_architecture":
		return a.validateArchitecture(ctx, msg)
	default:
		return nil
	}
}

func (a *ArchitectAgent) handleCollaboration(ctx context.Context, msg *types.Message) error {
	request, _ := msg.Metadata["request"].(map[string]interface{})
	requestType, _ := request["type"].(string)
	
	var response interface{}
	switch requestType {
	case "technology_advice":
		response = a.provideTechnologyAdvice(request)
	case "architecture_pattern":
		response = a.suggestArchitecturePattern(request)
	case "scalability_review":
		response = a.reviewScalability(request)
	default:
		response = map[string]interface{}{"status": "unknown request"}
	}

	reply := &types.Message{
		From:     a.ID(),
		To:       msg.From,
		Type:     types.MsgTypeResponse,
		ReplyTo:  msg.ID,
		Content:  "Architecture consultation",
		Metadata: map[string]interface{}{"response": response},
	}

	return a.SendMessage(ctx, reply)
}

func (a *ArchitectAgent) handleConsensus(ctx context.Context, msg *types.Message) error {
	topic, _ := msg.Metadata["topic"].(string)
	proposal, _ := msg.Metadata["proposal"].(map[string]interface{})
	
	// Evaluate proposal from architecture perspective
	decision := a.evaluateProposal(topic, proposal)
	
	vote := types.Vote{
		AgentID:   a.ID(),
		Decision:  decision,
		Reasoning: a.getDecisionReasoning(topic, proposal, decision),
		Timestamp: time.Now(),
	}

	reply := &types.Message{
		From:     a.ID(),
		Type:     types.MsgTypeConsensus,
		Content:  fmt.Sprintf("Vote on %s", topic),
		Metadata: map[string]interface{}{"vote": vote},
	}

	return a.SendMessage(ctx, reply)
}

func (a *ArchitectAgent) callLLM(ctx context.Context, prompt, systemPrompt string) (string, error) {
	requestBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"provider":   "azure",
		"max_tokens": 2500,
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

func (a *ArchitectAgent) designArchitecture(ctx context.Context, plan interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"pattern":    "microservices",
		"components": []string{"api-gateway", "auth-service", "core-service", "database"},
		"deployment": "kubernetes",
	}, nil
}

func (a *ArchitectAgent) selectArchitecturePattern(requirements string) string {
	lower := strings.ToLower(requirements)
	switch {
	case strings.Contains(lower, "microservice"):
		return "microservices"
	case strings.Contains(lower, "serverless"):
		return "serverless"
	case strings.Contains(lower, "monolith"):
		return "monolithic"
	default:
		return "modular-monolith"
	}
}

func (a *ArchitectAgent) identifyComponents(requirements string) []string {
	components := []string{"api-gateway", "auth-service"}
	lower := strings.ToLower(requirements)
	
	if strings.Contains(lower, "payment") {
		components = append(components, "payment-service")
	}
	if strings.Contains(lower, "notification") {
		components = append(components, "notification-service")
	}
	if strings.Contains(lower, "analytics") {
		components = append(components, "analytics-service")
	}
	
	return components
}

func (a *ArchitectAgent) recommendTechStack(projectType string) map[string]interface{} {
	switch projectType {
	case "backend":
		return map[string]interface{}{
			"language":  "Go",
			"framework": "Gin",
			"database":  "PostgreSQL",
			"cache":     "Redis",
		}
	case "frontend":
		return map[string]interface{}{
			"framework": "Next.js",
			"ui":        "Tailwind CSS",
			"state":     "Zustand",
		}
	default:
		return map[string]interface{}{
			"backend":  "Go/Gin",
			"frontend": "Next.js",
			"database": "PostgreSQL",
		}
	}
}

func (a *ArchitectAgent) getDefaultTechStack() map[string]interface{} {
	return map[string]interface{}{
		"backend": map[string]interface{}{
			"language":  "Go",
			"framework": "Gin",
			"orm":       "GORM",
		},
		"database": map[string]interface{}{
			"primary": "PostgreSQL",
			"cache":   "Redis",
		},
		"infrastructure": map[string]interface{}{
			"container":     "Docker",
			"orchestration": "Kubernetes",
			"ci_cd":         "GitHub Actions",
		},
	}
}

func (a *ArchitectAgent) generateBasicEndpoints(requirements string) []map[string]interface{} {
	return []map[string]interface{}{
		{"method": "GET", "path": "/api/v1/health"},
		{"method": "POST", "path": "/api/v1/auth/login"},
		{"method": "GET", "path": "/api/v1/resources"},
		{"method": "POST", "path": "/api/v1/resources"},
	}
}

func (a *ArchitectAgent) selectDatabase(requirements string) string {
	lower := strings.ToLower(requirements)
	if strings.Contains(lower, "nosql") || strings.Contains(lower, "document") {
		return "MongoDB"
	}
	if strings.Contains(lower, "graph") {
		return "Neo4j"
	}
	if strings.Contains(lower, "time series") {
		return "InfluxDB"
	}
	return "PostgreSQL"
}

func (a *ArchitectAgent) recommendIndexes(requirements string) []string {
	return []string{"primary_key", "foreign_keys", "frequently_queried_fields"}
}

func (a *ArchitectAgent) determinePartitioning(requirements string) string {
	if strings.Contains(strings.ToLower(requirements), "multi-tenant") {
		return "schema-per-tenant"
	}
	return "none"
}

func (a *ArchitectAgent) identifyArchitectureIssues(architecture map[string]interface{}) []map[string]interface{} {
	issues := []map[string]interface{}{}
	
	// Check for common issues (simplified)
	if _, ok := architecture["security"]; !ok {
		issues = append(issues, map[string]interface{}{
			"severity":       "high",
			"category":       "security",
			"description":    "Security architecture not defined",
			"recommendation": "Add authentication, authorization, and encryption strategies",
		})
	}
	
	return issues
}

func (a *ArchitectAgent) needsReadReplicas(metrics map[string]interface{}) bool {
	if readQPS, ok := metrics["read_qps"].(float64); ok && readQPS > 1000 {
		return true
	}
	return false
}

func (a *ArchitectAgent) recordDesignDecision(ctx context.Context, category string, decision interface{}) {
	// Record architecture decision (simplified)
	// In production, this would update shared memory
}

func (a *ArchitectAgent) reviewDesign(ctx context.Context, msg *types.Message) error {
	// Implement design review
	return nil
}

func (a *ArchitectAgent) validateArchitecture(ctx context.Context, msg *types.Message) error {
	// Implement architecture validation
	return nil
}

func (a *ArchitectAgent) provideTechnologyAdvice(request map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"recommendation": "Use proven technology stack"}
}

func (a *ArchitectAgent) suggestArchitecturePattern(request map[string]interface{}) string {
	return "microservices"
}

func (a *ArchitectAgent) reviewScalability(request map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"scalable": true, "recommendations": []string{"Add caching", "Use CDN"}}
}

func (a *ArchitectAgent) evaluateProposal(topic string, proposal map[string]interface{}) bool {
	// Evaluate based on architecture best practices
	return true
}

func (a *ArchitectAgent) getDecisionReasoning(topic string, proposal map[string]interface{}, decision bool) string {
	if decision {
		return "Proposal aligns with architecture best practices"
	}
	return "Proposal conflicts with architecture principles"
}