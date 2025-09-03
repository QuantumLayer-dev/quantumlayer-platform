package specialized

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/base"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/agents/types"
	"github.com/quantumlayer-dev/quantumlayer-platform/packages/qsecure"
)

// SecurityArchitectAgent specializes in security architecture and design
type SecurityArchitectAgent struct {
	*base.BaseAgent
	securityEngine *qsecure.QSecureEngine
	threatModels   map[string]*qsecure.ThreatModel
	riskProfiles   map[string]*RiskProfile
}

// RiskProfile represents a security risk profile
type RiskProfile struct {
	ID              string                 `json:"id"`
	System          string                 `json:"system"`
	OverallRisk     string                 `json:"overall_risk"`
	TopThreats      []string               `json:"top_threats"`
	Mitigations     []string               `json:"mitigations"`
	ComplianceReqs  []string               `json:"compliance_reqs"`
	LastAssessment  time.Time              `json:"last_assessment"`
}

// NewSecurityArchitectAgent creates a new security architect agent
func NewSecurityArchitectAgent(id string, llmEndpoint string, msgBus types.MessageBus) *SecurityArchitectAgent {
	baseAgent := base.NewBaseAgent(id, types.RoleSecurityArchitect, llmEndpoint, msgBus)
	
	// Add security-specific capabilities
	baseAgent.AddCapability(types.CapabilityThreatModeling)
	baseAgent.AddCapability(types.CapabilitySecurityAnalysis)
	baseAgent.AddCapability(types.CapabilityComplianceValidation)
	baseAgent.AddCapability(types.CapabilityRiskAssessment)
	
	return &SecurityArchitectAgent{
		BaseAgent:     baseAgent,
		threatModels:  make(map[string]*qsecure.ThreatModel),
		riskProfiles:  make(map[string]*RiskProfile),
	}
}

// Execute performs security architecture tasks
func (a *SecurityArchitectAgent) Execute(ctx context.Context, task *types.Task) error {
	a.SetStatus(types.StatusWorking)
	defer a.SetStatus(types.StatusIdle)
	
	switch task.Type {
	case "threat_model":
		return a.generateThreatModel(ctx, task)
	case "security_review":
		return a.performSecurityReview(ctx, task)
	case "compliance_check":
		return a.checkCompliance(ctx, task)
	case "risk_assessment":
		return a.assessRisk(ctx, task)
	case "security_design":
		return a.designSecurityArchitecture(ctx, task)
	default:
		return a.BaseAgent.Execute(ctx, task)
	}
}

func (a *SecurityArchitectAgent) generateThreatModel(ctx context.Context, task *types.Task) error {
	// Extract system description from task
	systemDesc, _ := task.Requirements["system"].(string)
	
	// Generate threat model using AI
	prompt := fmt.Sprintf(`
As a Security Architect, generate a comprehensive threat model for the following system:

%s

Include:
1. Assets and their sensitivity levels
2. Threat actors and their capabilities
3. Attack vectors and entry points
4. STRIDE analysis (Spoofing, Tampering, Repudiation, Information Disclosure, DoS, Elevation of Privilege)
5. Risk assessment matrix
6. Recommended security controls
7. Compliance considerations

Format as structured JSON.
`, systemDesc)
	
	response, err := a.CallLLM(ctx, prompt, "You are an expert security architect specializing in threat modeling.")
	if err != nil {
		return fmt.Errorf("failed to generate threat model: %w", err)
	}
	
	// Parse and store threat model
	var threatModel qsecure.ThreatModel
	if err := json.Unmarshal([]byte(response), &threatModel); err != nil {
		// Create basic threat model from response
		threatModel = qsecure.ThreatModel{
			ID:        uuid.New().String(),
			System:    task.ID,
			Generated: time.Now(),
		}
	}
	
	a.threatModels[task.ID] = &threatModel
	
	// Update task result
	task.Result = map[string]interface{}{
		"threat_model": threatModel,
		"agent":        a.ID(),
		"timestamp":    time.Now(),
	}
	
	// Broadcast threat model to other agents
	msg := &types.Message{
		ID:        uuid.New().String(),
		From:      a.ID(),
		To:        "broadcast",
		Type:      types.MessageTypeAnalysis,
		Content:   fmt.Sprintf("Threat model completed for %s", task.ID),
		Metadata:  map[string]interface{}{"threat_model": threatModel},
		Timestamp: time.Now(),
	}
	
	return a.SendMessage(ctx, msg)
}

func (a *SecurityArchitectAgent) performSecurityReview(ctx context.Context, task *types.Task) error {
	// Extract code from task
	code, _ := task.Requirements["code"].(string)
	language, _ := task.Requirements["language"].(string)
	
	// Perform security analysis
	prompt := fmt.Sprintf(`
Perform a comprehensive security review of the following %s code:

%s

Identify:
1. Security vulnerabilities (with CWE/CVE references)
2. Insecure coding practices
3. Missing security controls
4. Data validation issues
5. Authentication/authorization problems
6. Cryptographic weaknesses
7. Injection vulnerabilities
8. Configuration security issues

Provide specific remediation recommendations.
`, language, code)
	
	response, err := a.CallLLM(ctx, prompt, "You are a security expert performing code review.")
	if err != nil {
		return fmt.Errorf("security review failed: %w", err)
	}
	
	// Update task with review results
	task.Result = map[string]interface{}{
		"review":    response,
		"agent":     a.ID(),
		"timestamp": time.Now(),
	}
	
	return nil
}

func (a *SecurityArchitectAgent) checkCompliance(ctx context.Context, task *types.Task) error {
	// Extract compliance standards
	standards, _ := task.Requirements["standards"].([]string)
	code, _ := task.Requirements["code"].(string)
	
	if len(standards) == 0 {
		standards = []string{"OWASP", "GDPR", "SOC2"} // Default standards
	}
	
	complianceResults := make(map[string]interface{})
	
	for _, standard := range standards {
		prompt := fmt.Sprintf(`
Check the following code for %s compliance:

%s

Identify any violations and provide remediation steps.
Format as structured JSON with:
- compliant: boolean
- violations: array of issues
- remediation: array of fixes
- score: compliance percentage
`, standard, code)
		
		response, err := a.CallLLM(ctx, prompt, fmt.Sprintf("You are a %s compliance expert.", standard))
		if err != nil {
			continue
		}
		
		complianceResults[standard] = response
	}
	
	task.Result = map[string]interface{}{
		"compliance": complianceResults,
		"agent":      a.ID(),
		"timestamp":  time.Now(),
	}
	
	return nil
}

func (a *SecurityArchitectAgent) assessRisk(ctx context.Context, task *types.Task) error {
	// Extract system information
	systemInfo, _ := task.Requirements["system"].(map[string]interface{})
	
	// Create risk profile
	riskProfile := &RiskProfile{
		ID:             uuid.New().String(),
		System:         task.ID,
		LastAssessment: time.Now(),
	}
	
	// Assess risks using AI
	prompt := fmt.Sprintf(`
Perform a risk assessment for the following system:

%v

Provide:
1. Overall risk level (Critical/High/Medium/Low)
2. Top 5 security threats
3. Recommended mitigations
4. Compliance requirements
5. Risk score (0-100)

Format as structured JSON.
`, systemInfo)
	
	response, err := a.CallLLM(ctx, prompt, "You are a security risk assessment expert.")
	if err != nil {
		return fmt.Errorf("risk assessment failed: %w", err)
	}
	
	// Parse risk assessment
	var assessment map[string]interface{}
	if err := json.Unmarshal([]byte(response), &assessment); err == nil {
		if risk, ok := assessment["overall_risk"].(string); ok {
			riskProfile.OverallRisk = risk
		}
		if threats, ok := assessment["top_threats"].([]interface{}); ok {
			for _, threat := range threats {
				if t, ok := threat.(string); ok {
					riskProfile.TopThreats = append(riskProfile.TopThreats, t)
				}
			}
		}
		if mitigations, ok := assessment["mitigations"].([]interface{}); ok {
			for _, mitigation := range mitigations {
				if m, ok := mitigation.(string); ok {
					riskProfile.Mitigations = append(riskProfile.Mitigations, m)
				}
			}
		}
	}
	
	// Store risk profile
	a.riskProfiles[task.ID] = riskProfile
	
	task.Result = map[string]interface{}{
		"risk_profile": riskProfile,
		"agent":        a.ID(),
		"timestamp":    time.Now(),
	}
	
	return nil
}

func (a *SecurityArchitectAgent) designSecurityArchitecture(ctx context.Context, task *types.Task) error {
	// Extract requirements
	requirements, _ := task.Requirements["requirements"].(string)
	
	// Design security architecture
	prompt := fmt.Sprintf(`
Design a comprehensive security architecture for:

%s

Include:
1. Security zones and trust boundaries
2. Authentication and authorization architecture
3. Data protection strategy (encryption at rest/in transit)
4. Network security design
5. Identity and access management
6. Security monitoring and logging
7. Incident response plan
8. Disaster recovery strategy
9. Compliance framework alignment
10. Security technology stack

Provide detailed architecture with diagrams descriptions and implementation guidelines.
`, requirements)
	
	response, err := a.CallLLM(ctx, prompt, "You are a senior security architect designing enterprise security solutions.")
	if err != nil {
		return fmt.Errorf("security architecture design failed: %w", err)
	}
	
	task.Result = map[string]interface{}{
		"architecture": response,
		"agent":        a.ID(),
		"timestamp":    time.Now(),
	}
	
	// Notify other agents about the security architecture
	msg := &types.Message{
		ID:        uuid.New().String(),
		From:      a.ID(),
		To:        "broadcast",
		Type:      types.MessageTypeAnalysis,
		Content:   "Security architecture designed",
		Metadata: map[string]interface{}{
			"task_id":      task.ID,
			"architecture": response,
		},
		Timestamp: time.Now(),
	}
	
	return a.SendMessage(ctx, msg)
}