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

// ProjectManagerAgent handles requirements analysis and task breakdown
type ProjectManagerAgent struct {
	*base.BaseAgent
	llmEndpoint string
}

// NewProjectManagerAgent creates a new project manager agent
func NewProjectManagerAgent(llmEndpoint string) *ProjectManagerAgent {
	capabilities := []types.AgentCapability{
		types.CapRequirementsAnalysis,
		types.CapDocumentation,
	}

	agent := &ProjectManagerAgent{
		BaseAgent:   base.NewBaseAgent(types.RoleProjectManager, capabilities),
		llmEndpoint: llmEndpoint,
	}

	// Set specialized handlers
	agent.SetExecutionHandler(agent.executeTask)
	agent.SetMessageHandler(agent.handleMessage)
	agent.SetInitializeHandler(agent.initialize)

	return agent
}

func (a *ProjectManagerAgent) initialize(ctx context.Context, agentCtx *types.AgentContext) error {
	// Analyze initial requirements and create project plan
	if agentCtx.Requirements != "" {
		plan, err := a.analyzeRequirements(ctx, agentCtx.Requirements)
		if err != nil {
			return fmt.Errorf("failed to analyze requirements: %w", err)
		}

		// Store plan in shared memory
		if agentCtx.SharedMemory != nil {
			if agentCtx.SharedMemory.ProjectContext == nil {
				agentCtx.SharedMemory.ProjectContext = make(map[string]interface{})
			}
			agentCtx.SharedMemory.ProjectContext["project_plan"] = plan
			agentCtx.SharedMemory.ProjectContext["analyzed_at"] = time.Now()
		}

		// Notify other agents about the plan
		msg := &types.Message{
			From:    a.ID(),
			Type:    types.MsgTypeNotification,
			Content: "Project plan created",
			Metadata: map[string]interface{}{
				"plan": plan,
			},
		}
		a.SendMessage(ctx, msg)
	}

	return nil
}

func (a *ProjectManagerAgent) executeTask(ctx context.Context, task *types.Task) error {
	switch task.Type {
	case "analyze_requirements":
		return a.executeRequirementsAnalysis(ctx, task)
	case "create_project_plan":
		return a.executeProjectPlanning(ctx, task)
	case "breakdown_tasks":
		return a.executeTaskBreakdown(ctx, task)
	case "review_progress":
		return a.executeProgressReview(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
}

func (a *ProjectManagerAgent) handleMessage(ctx context.Context, msg *types.Message) error {
	switch msg.Type {
	case types.MsgTypeRequest:
		return a.handleRequest(ctx, msg)
	case types.MsgTypeCollaboration:
		return a.handleCollaboration(ctx, msg)
	case types.MsgTypeEscalation:
		return a.handleEscalation(ctx, msg)
	default:
		return nil
	}
}

func (a *ProjectManagerAgent) executeRequirementsAnalysis(ctx context.Context, task *types.Task) error {
	requirements, ok := task.Requirements["requirements"].(string)
	if !ok {
		return fmt.Errorf("requirements not provided")
	}

	analysis, err := a.analyzeRequirements(ctx, requirements)
	if err != nil {
		return err
	}

	task.Result = analysis
	return nil
}

func (a *ProjectManagerAgent) analyzeRequirements(ctx context.Context, requirements string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`As a Project Manager, analyze these requirements and provide:
1. Project type and complexity
2. Required components and services
3. Technology stack recommendations
4. Team composition (which agents are needed)
5. Risk assessment
6. Success criteria
7. Estimated timeline

Requirements: %s

Provide the analysis in JSON format.`, requirements)

	response, err := a.callLLM(ctx, prompt, "system")
	if err != nil {
		return nil, fmt.Errorf("failed to analyze requirements: %w", err)
	}

	// Parse the response
	var analysis map[string]interface{}
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		// If JSON parsing fails, create structured response
		analysis = map[string]interface{}{
			"raw_analysis":     response,
			"project_type":     a.detectProjectType(requirements),
			"complexity":       a.assessComplexity(requirements),
			"required_agents":  a.determineRequiredAgents(requirements),
			"estimated_time":   "2-3 minutes",
		}
	}

	return analysis, nil
}

func (a *ProjectManagerAgent) executeProjectPlanning(ctx context.Context, task *types.Task) error {
	requirements, _ := task.Requirements["requirements"].(string)
	constraints, _ := task.Requirements["constraints"].(map[string]interface{})

	prompt := fmt.Sprintf(`Create a detailed project plan for:
Requirements: %s
Constraints: %v

Include:
1. Phases and milestones
2. Task dependencies
3. Resource allocation
4. Risk mitigation strategies
5. Quality gates

Format as actionable tasks for different agent roles.`, requirements, constraints)

	plan, err := a.callLLM(ctx, prompt, "system")
	if err != nil {
		return err
	}

	// Parse and structure the plan
	structuredPlan := a.structurePlan(plan)
	task.Result = structuredPlan

	// Broadcast plan to all agents
	msg := &types.Message{
		From:    a.ID(),
		Type:    types.MsgTypeNotification,
		Content: "Project plan ready for execution",
		Metadata: map[string]interface{}{
			"plan": structuredPlan,
		},
	}
	
	return a.SendMessage(ctx, msg)
}

func (a *ProjectManagerAgent) executeTaskBreakdown(ctx context.Context, task *types.Task) error {
	feature, _ := task.Requirements["feature"].(string)
	
	prompt := fmt.Sprintf(`Break down this feature into specific development tasks:
Feature: %s

For each task specify:
- Task name and description
- Assigned agent role
- Dependencies
- Acceptance criteria
- Priority (1-5)

Format as JSON array.`, feature)

	response, err := a.callLLM(ctx, prompt, "system")
	if err != nil {
		return err
	}

	var tasks []map[string]interface{}
	if err := json.Unmarshal([]byte(response), &tasks); err != nil {
		// Create basic task breakdown
		tasks = a.createBasicTaskBreakdown(feature)
	}

	task.Result = tasks
	return nil
}

func (a *ProjectManagerAgent) executeProgressReview(ctx context.Context, task *types.Task) error {
	// Collect metrics from all agents
	progressReport := map[string]interface{}{
		"timestamp":      time.Now(),
		"overall_status": "in_progress",
		"agents":         make(map[string]interface{}),
	}

	// Request status from all active agents
	statusRequest := &types.Message{
		From:    a.ID(),
		Type:    types.MsgTypeRequest,
		Content: "status_report",
	}
	
	if err := a.SendMessage(ctx, statusRequest); err != nil {
		return err
	}

	task.Result = progressReport
	return nil
}

func (a *ProjectManagerAgent) handleRequest(ctx context.Context, msg *types.Message) error {
	switch msg.Content {
	case "provide_requirements":
		return a.sendRequirements(ctx, msg)
	case "clarify_requirement":
		return a.clarifyRequirement(ctx, msg)
	default:
		return nil
	}
}

func (a *ProjectManagerAgent) handleCollaboration(ctx context.Context, msg *types.Message) error {
	// Provide project management expertise to requesting agent
	request, _ := msg.Metadata["request"].(string)
	
	response := map[string]interface{}{
		"recommendation": a.provideRecommendation(request),
		"priority":       a.assessPriority(request),
	}

	reply := &types.Message{
		From:     a.ID(),
		To:       msg.From,
		Type:     types.MsgTypeResponse,
		ReplyTo:  msg.ID,
		Content:  "Collaboration response",
		Metadata: map[string]interface{}{"response": response},
	}

	return a.SendMessage(ctx, reply)
}

func (a *ProjectManagerAgent) handleEscalation(ctx context.Context, msg *types.Message) error {
	issue, _ := msg.Metadata["issue"].(string)
	
	// Analyze escalation and provide resolution
	resolution := a.resolveEscalation(issue)
	
	reply := &types.Message{
		From:     a.ID(),
		To:       msg.From,
		Type:     types.MsgTypeResponse,
		ReplyTo:  msg.ID,
		Content:  "Escalation resolved",
		Metadata: map[string]interface{}{
			"resolution": resolution,
			"action":     "proceed",
		},
	}

	return a.SendMessage(ctx, reply)
}

func (a *ProjectManagerAgent) callLLM(ctx context.Context, prompt, systemPrompt string) (string, error) {
	requestBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"provider":   "azure",
		"max_tokens": 2000,
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

func (a *ProjectManagerAgent) detectProjectType(requirements string) string {
	lower := strings.ToLower(requirements)
	switch {
	case strings.Contains(lower, "api") || strings.Contains(lower, "backend"):
		return "backend"
	case strings.Contains(lower, "frontend") || strings.Contains(lower, "ui"):
		return "frontend"
	case strings.Contains(lower, "mobile") || strings.Contains(lower, "app"):
		return "mobile"
	case strings.Contains(lower, "ml") || strings.Contains(lower, "machine learning"):
		return "ml"
	default:
		return "fullstack"
	}
}

func (a *ProjectManagerAgent) assessComplexity(requirements string) string {
	wordCount := len(strings.Fields(requirements))
	switch {
	case wordCount < 50:
		return "simple"
	case wordCount < 200:
		return "moderate"
	default:
		return "complex"
	}
}

func (a *ProjectManagerAgent) determineRequiredAgents(requirements string) []string {
	agents := []string{"project-manager", "architect"}
	
	lower := strings.ToLower(requirements)
	if strings.Contains(lower, "backend") || strings.Contains(lower, "api") {
		agents = append(agents, "backend-developer")
	}
	if strings.Contains(lower, "frontend") || strings.Contains(lower, "ui") {
		agents = append(agents, "frontend-developer")
	}
	if strings.Contains(lower, "database") || strings.Contains(lower, "data") {
		agents = append(agents, "database-admin")
	}
	if strings.Contains(lower, "deploy") || strings.Contains(lower, "infrastructure") {
		agents = append(agents, "devops")
	}
	if strings.Contains(lower, "test") || strings.Contains(lower, "quality") {
		agents = append(agents, "qa-engineer")
	}
	if strings.Contains(lower, "security") || strings.Contains(lower, "auth") {
		agents = append(agents, "security")
	}
	
	return agents
}

func (a *ProjectManagerAgent) structurePlan(plan string) map[string]interface{} {
	return map[string]interface{}{
		"raw_plan":   plan,
		"phases":     []string{"design", "implementation", "testing", "deployment"},
		"milestones": []string{"requirements_complete", "design_approved", "code_complete", "tests_passing", "deployed"},
		"created_at": time.Now(),
	}
}

func (a *ProjectManagerAgent) createBasicTaskBreakdown(feature string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "Design " + feature,
			"role":        "architect",
			"priority":    1,
			"dependencies": []string{},
		},
		{
			"name":        "Implement " + feature,
			"role":        "backend-developer",
			"priority":    2,
			"dependencies": []string{"Design " + feature},
		},
		{
			"name":        "Test " + feature,
			"role":        "qa-engineer",
			"priority":    3,
			"dependencies": []string{"Implement " + feature},
		},
	}
}

func (a *ProjectManagerAgent) sendRequirements(ctx context.Context, msg *types.Message) error {
	// Implement requirements sharing logic
	return nil
}

func (a *ProjectManagerAgent) clarifyRequirement(ctx context.Context, msg *types.Message) error {
	// Implement requirement clarification logic
	return nil
}

func (a *ProjectManagerAgent) provideRecommendation(request string) string {
	return "Proceed with standard approach"
}

func (a *ProjectManagerAgent) assessPriority(request string) int {
	return 3 // Medium priority
}

func (a *ProjectManagerAgent) resolveEscalation(issue string) string {
	return "Apply standard resolution procedure"
}