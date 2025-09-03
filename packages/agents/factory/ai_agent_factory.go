package factory

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/base"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/specialized"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/types"
	aidecision "github.com/quantumlayer-dev/quantumlayer-platform/packages/ai-decision-engine"
)

// AIAgentFactory creates agents dynamically using AI decision making
type AIAgentFactory struct {
	decisionEngine *aidecision.AIDecisionEngine
	llmEndpoint    string
	messageBus     types.MessageBus
	agentRegistry  map[string]types.Agent
	agentProfiles  map[types.AgentRole]*AgentProfile
}

// AgentProfile defines the characteristics of an agent type
type AgentProfile struct {
	Role         types.AgentRole         `json:"role"`
	Description  string                  `json:"description"`
	Capabilities []types.AgentCapability `json:"capabilities"`
	Expertise    []string                `json:"expertise"`
	Tools        []string                `json:"tools"`
	Personality  map[string]interface{}  `json:"personality"`
	Constructor  AgentConstructor        `json:"-"`
}

// AgentConstructor is a function that creates an agent instance
type AgentConstructor func(id, llmEndpoint string, msgBus types.MessageBus) types.Agent

// NewAIAgentFactory creates a new AI-powered agent factory
func NewAIAgentFactory(decisionEngine *aidecision.AIDecisionEngine, llmEndpoint string, msgBus types.MessageBus) *AIAgentFactory {
	factory := &AIAgentFactory{
		decisionEngine: decisionEngine,
		llmEndpoint:    llmEndpoint,
		messageBus:     msgBus,
		agentRegistry:  make(map[string]types.Agent),
		agentProfiles:  make(map[types.AgentRole]*AgentProfile),
	}
	
	// Initialize agent profiles
	factory.initializeAgentProfiles()
	
	// Register decision rules for agent selection
	factory.registerAgentSelectionRules()
	
	return factory
}

// CreateAgent creates an agent based on AI decision making
func (f *AIAgentFactory) CreateAgent(ctx context.Context, requirements string) (types.Agent, error) {
	// Use AI to decide which agent type to create
	decision, err := f.decisionEngine.Decide(ctx, "agent_selection", requirements)
	if err != nil {
		return nil, fmt.Errorf("AI decision failed: %w", err)
	}
	
	// Extract agent role from decision
	var agentRole types.AgentRole
	if roleStr, ok := decision.Result.(string); ok {
		agentRole = types.AgentRole(roleStr)
	} else {
		// Use intent-based role selection
		agentRole = f.selectRoleByIntent(decision.Intent)
	}
	
	// Create agent instance
	return f.SpawnAgent(ctx, agentRole)
}

// SpawnAgent creates a specific type of agent
func (f *AIAgentFactory) SpawnAgent(ctx context.Context, role types.AgentRole) (types.Agent, error) {
	// Check if profile exists
	profile, exists := f.agentProfiles[role]
	if !exists {
		// Use AI to generate a new agent profile dynamically
		profile = f.generateAgentProfile(ctx, role)
		if profile == nil {
			return nil, fmt.Errorf("unable to create profile for role: %s", role)
		}
	}
	
	// Generate unique agent ID
	agentID := fmt.Sprintf("agent-%s-%s", role, uuid.New().String()[:8])
	
	// Create agent using constructor or generic agent
	var agent types.Agent
	if profile.Constructor != nil {
		agent = profile.Constructor(agentID, f.llmEndpoint, f.messageBus)
	} else {
		// Create generic AI-powered agent
		agent = f.createGenericAgent(agentID, role, profile)
	}
	
	// Register agent
	f.agentRegistry[agentID] = agent
	
	return agent, nil
}

// CreateAgentTeam creates a team of agents for a project
func (f *AIAgentFactory) CreateAgentTeam(ctx context.Context, projectType string) ([]types.Agent, error) {
	// Use AI to determine optimal team composition
	teamComposition := f.determineTeamComposition(ctx, projectType)
	
	agents := make([]types.Agent, 0)
	for _, role := range teamComposition {
		agent, err := f.SpawnAgent(ctx, role)
		if err != nil {
			continue // Skip if unable to create
		}
		agents = append(agents, agent)
	}
	
	return agents, nil
}

// Private methods

func (f *AIAgentFactory) initializeAgentProfiles() {
	// Define agent profiles with their characteristics
	profiles := []AgentProfile{
		{
			Role:        types.RoleProjectManager,
			Description: "Manages project execution and coordinates team efforts",
			Capabilities: []types.AgentCapability{
				types.CapRequirementsAnalysis,
				types.CapDocumentation,
			},
			Expertise: []string{"project planning", "risk management", "stakeholder communication"},
			Tools:     []string{"jira", "confluence", "gantt"},
			Constructor: func(id, llm string, bus types.MessageBus) types.Agent {
				return specialized.NewProjectManagerAgent(id, llm, bus)
			},
		},
		{
			Role:        types.RoleArchitect,
			Description: "Designs system architecture and technical solutions",
			Capabilities: []types.AgentCapability{
				types.CapSystemDesign,
				types.CapDocumentation,
			},
			Expertise: []string{"system design", "patterns", "scalability"},
			Tools:     []string{"draw.io", "c4model", "uml"},
			Constructor: func(id, llm string, bus types.MessageBus) types.Agent {
				return specialized.NewArchitectAgent(id, llm, bus)
			},
		},
		{
			Role:        types.RoleBackendDev,
			Description: "Develops backend services and APIs",
			Capabilities: []types.AgentCapability{
				types.CapCodeGeneration,
				types.CapTestGeneration,
			},
			Expertise: []string{"api development", "databases", "microservices"},
			Tools:     []string{"golang", "python", "nodejs", "postgresql"},
			Constructor: func(id, llm string, bus types.MessageBus) types.Agent {
				return specialized.NewBackendDeveloperAgent(id, llm, bus)
			},
		},
		{
			Role:        types.RoleSecurityArchitect,
			Description: "Ensures security best practices and compliance",
			Capabilities: []types.AgentCapability{
				types.CapabilityThreatModeling,
				types.CapabilitySecurityAnalysis,
				types.CapabilityComplianceValidation,
				types.CapabilityRiskAssessment,
			},
			Expertise: []string{"threat modeling", "security architecture", "compliance"},
			Tools:     []string{"owasp", "nist", "mitre-attack"},
			Constructor: func(id, llm string, bus types.MessageBus) types.Agent {
				return specialized.NewSecurityArchitectAgent(id, llm, bus)
			},
		},
		// Add more agent profiles as needed
	}
	
	// Store profiles
	for _, profile := range profiles {
		f.agentProfiles[profile.Role] = &profile
		
		// Register with decision engine
		f.registerAgentProfile(&profile)
	}
}

func (f *AIAgentFactory) registerAgentSelectionRules() {
	// Register decision rules for agent selection
	rules := []aidecision.DecisionRule{
		{
			ID:          "agent_project_management",
			Category:    "agent_selection",
			Pattern:     "project management planning coordination timeline",
			Description: "Select project manager for coordination tasks",
			Priority:    8,
			Examples:    []string{"manage the project", "coordinate team", "create timeline"},
			Action: func(ctx context.Context, input interface{}) (interface{}, error) {
				return string(types.RoleProjectManager), nil
			},
		},
		{
			ID:          "agent_architecture",
			Category:    "agent_selection",
			Pattern:     "architecture design system structure scalability",
			Description: "Select architect for system design tasks",
			Priority:    8,
			Examples:    []string{"design the system", "create architecture", "plan structure"},
			Action: func(ctx context.Context, input interface{}) (interface{}, error) {
				return string(types.RoleArchitect), nil
			},
		},
		{
			ID:          "agent_backend",
			Category:    "agent_selection",
			Pattern:     "backend api service database server logic",
			Description: "Select backend developer for server-side tasks",
			Priority:    7,
			Examples:    []string{"create API", "implement backend", "database logic"},
			Action: func(ctx context.Context, input interface{}) (interface{}, error) {
				return string(types.RoleBackendDev), nil
			},
		},
		{
			ID:          "agent_security",
			Category:    "agent_selection",
			Pattern:     "security threat vulnerability compliance audit penetration",
			Description: "Select security specialist for security tasks",
			Priority:    9,
			Examples:    []string{"security review", "threat model", "compliance check"},
			Action: func(ctx context.Context, input interface{}) (interface{}, error) {
				return string(types.RoleSecurityArchitect), nil
			},
		},
	}
	
	// Register rules with decision engine
	for _, rule := range rules {
		f.decisionEngine.RegisterRule(&rule)
	}
}

func (f *AIAgentFactory) registerAgentProfile(profile *AgentProfile) {
	// Create embedding text from profile
	embeddingText := fmt.Sprintf("%s %s %s %s",
		profile.Role,
		profile.Description,
		strings.Join(profile.Expertise, " "),
		strings.Join(profile.Tools, " "))
	
	// Register as decision rule
	rule := aidecision.DecisionRule{
		ID:          fmt.Sprintf("agent_%s", profile.Role),
		Category:    "agent_profile",
		Pattern:     embeddingText,
		Description: profile.Description,
		Priority:    5,
		Action: func(ctx context.Context, input interface{}) (interface{}, error) {
			return profile, nil
		},
	}
	
	f.decisionEngine.RegisterRule(&rule)
}

func (f *AIAgentFactory) selectRoleByIntent(intent string) types.AgentRole {
	// Map intents to roles using semantic understanding
	intentLower := strings.ToLower(intent)
	
	// Use keyword matching as fallback
	roleMap := map[string]types.AgentRole{
		"manage":       types.RoleProjectManager,
		"coordinate":   types.RoleProjectManager,
		"design":       types.RoleArchitect,
		"architecture": types.RoleArchitect,
		"backend":      types.RoleBackendDev,
		"api":          types.RoleBackendDev,
		"frontend":     types.RoleFrontendDev,
		"ui":           types.RoleFrontendDev,
		"security":     types.RoleSecurityArchitect,
		"threat":       types.RoleSecurityArchitect,
		"compliance":   types.RoleComplianceOfficer,
		"database":     types.RoleDatabaseAdmin,
		"devops":       types.RoleDevOps,
		"test":         types.RoleQA,
		"quality":      types.RoleQA,
	}
	
	for keyword, role := range roleMap {
		if strings.Contains(intentLower, keyword) {
			return role
		}
	}
	
	// Default to backend developer
	return types.RoleBackendDev
}

func (f *AIAgentFactory) generateAgentProfile(ctx context.Context, role types.AgentRole) *AgentProfile {
	// Dynamically generate agent profile using AI
	profile := &AgentProfile{
		Role:         role,
		Description:  fmt.Sprintf("AI-generated agent for %s role", role),
		Capabilities: f.inferCapabilities(role),
		Expertise:    f.inferExpertise(role),
		Tools:        f.inferTools(role),
		Personality: map[string]interface{}{
			"analytical":   true,
			"collaborative": true,
			"proactive":    true,
		},
	}
	
	// Store the generated profile
	f.agentProfiles[role] = profile
	
	return profile
}

func (f *AIAgentFactory) createGenericAgent(id string, role types.AgentRole, profile *AgentProfile) types.Agent {
	// Create a generic AI-powered agent
	agent := base.NewBaseAgent(id, role, f.llmEndpoint, f.messageBus)
	
	// Add capabilities from profile
	for _, capability := range profile.Capabilities {
		agent.AddCapability(capability)
	}
	
	return agent
}

func (f *AIAgentFactory) determineTeamComposition(ctx context.Context, projectType string) []types.AgentRole {
	// Use AI to determine optimal team composition
	projectLower := strings.ToLower(projectType)
	
	// Base team
	team := []types.AgentRole{
		types.RoleProjectManager,
		types.RoleArchitect,
	}
	
	// Add specialized roles based on project type
	if strings.Contains(projectLower, "web") || strings.Contains(projectLower, "api") {
		team = append(team, types.RoleBackendDev)
		if strings.Contains(projectLower, "full") {
			team = append(team, types.RoleFrontendDev)
		}
	}
	
	if strings.Contains(projectLower, "mobile") {
		team = append(team, types.RoleFrontendDev)
	}
	
	if strings.Contains(projectLower, "data") || strings.Contains(projectLower, "ml") {
		team = append(team, types.RoleDataEngineer)
	}
	
	// Always include security for enterprise projects
	if strings.Contains(projectLower, "enterprise") || strings.Contains(projectLower, "production") {
		team = append(team, types.RoleSecurityArchitect)
		team = append(team, types.RoleDevOps)
		team = append(team, types.RoleQA)
	}
	
	return team
}

func (f *AIAgentFactory) inferCapabilities(role types.AgentRole) []types.AgentCapability {
	// Infer capabilities based on role
	capabilityMap := map[types.AgentRole][]types.AgentCapability{
		types.RoleProjectManager: {
			types.CapRequirementsAnalysis,
			types.CapDocumentation,
		},
		types.RoleArchitect: {
			types.CapSystemDesign,
			types.CapDocumentation,
		},
		types.RoleBackendDev: {
			types.CapCodeGeneration,
			types.CapTestGeneration,
		},
		types.RoleFrontendDev: {
			types.CapCodeGeneration,
			types.CapTestGeneration,
		},
		types.RoleSecurityArchitect: {
			types.CapabilityThreatModeling,
			types.CapabilitySecurityAnalysis,
			types.CapabilityComplianceValidation,
		},
		types.RoleDevOps: {
			types.CapInfrastructureSetup,
			types.CapMonitoringSetup,
		},
		types.RoleQA: {
			types.CapTestGeneration,
			types.CapSecurityAudit,
		},
	}
	
	if capabilities, exists := capabilityMap[role]; exists {
		return capabilities
	}
	
	// Default capabilities
	return []types.AgentCapability{types.CapCodeGeneration}
}

func (f *AIAgentFactory) inferExpertise(role types.AgentRole) []string {
	// Infer expertise based on role
	expertiseMap := map[types.AgentRole][]string{
		types.RoleProjectManager:     {"project planning", "risk management", "team coordination"},
		types.RoleArchitect:          {"system design", "patterns", "scalability", "best practices"},
		types.RoleBackendDev:         {"api development", "databases", "microservices", "performance"},
		types.RoleFrontendDev:        {"ui/ux", "responsive design", "spa", "accessibility"},
		types.RoleSecurityArchitect:  {"threat modeling", "compliance", "vulnerability assessment"},
		types.RoleDevOps:             {"ci/cd", "kubernetes", "monitoring", "automation"},
		types.RoleQA:                 {"testing strategies", "automation", "quality metrics"},
	}
	
	if expertise, exists := expertiseMap[role]; exists {
		return expertise
	}
	
	return []string{"software development"}
}

func (f *AIAgentFactory) inferTools(role types.AgentRole) []string {
	// Infer tools based on role
	toolsMap := map[types.AgentRole][]string{
		types.RoleProjectManager:     {"jira", "confluence", "gantt", "slack"},
		types.RoleArchitect:          {"draw.io", "c4model", "uml", "swagger"},
		types.RoleBackendDev:         {"golang", "python", "nodejs", "postgresql", "redis"},
		types.RoleFrontendDev:        {"react", "vue", "angular", "css", "webpack"},
		types.RoleSecurityArchitect:  {"owasp", "burp", "metasploit", "nmap"},
		types.RoleDevOps:             {"terraform", "ansible", "docker", "kubernetes", "prometheus"},
		types.RoleQA:                 {"selenium", "cypress", "jest", "postman"},
	}
	
	if tools, exists := toolsMap[role]; exists {
		return tools
	}
	
	return []string{"vscode", "git"}
}

// GetAgent retrieves an agent by ID
func (f *AIAgentFactory) GetAgent(agentID string) types.Agent {
	return f.agentRegistry[agentID]
}

// ListAgents returns all registered agents
func (f *AIAgentFactory) ListAgents() map[string]types.Agent {
	return f.agentRegistry
}