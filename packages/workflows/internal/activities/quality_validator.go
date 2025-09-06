package activities

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/types"
)

// QualityValidator provides enterprise-grade code validation
type QualityValidator struct {
	minCodeLength   int
	minFunctionCount int
	maxTODOCount    int
	minComplexity   int
}

// NewQualityValidator creates a quality validator with enterprise standards
func NewQualityValidator() *QualityValidator {
	return &QualityValidator{
		minCodeLength:    500,  // Minimum 500 characters for real code
		minFunctionCount: 3,    // At least 3 functions/methods
		maxTODOCount:     0,    // No TODOs in production code
		minComplexity:    10,   // Minimum cyclomatic complexity
	}
}

// ValidateEnterpriseCode performs comprehensive quality validation
func (v *QualityValidator) ValidateEnterpriseCode(ctx context.Context, code string, language string) (*types.ValidationResult, error) {
	result := &types.ValidationResult{
		Valid:  true,
		Score:  100.0,
		Issues: []types.Issue{},
	}

	// Check 1: Minimum code length
	if len(code) < v.minCodeLength {
		result.Valid = false
		result.Score -= 30
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: fmt.Sprintf("Code too minimal: %d chars (minimum %d required)", len(code), v.minCodeLength),
			Line:    0,
		})
	}

	// Check 2: Reject placeholder patterns
	placeholderPatterns := []string{
		`print\s*\(\s*["']Hello`,
		`console\.log\s*\(\s*["']Hello`,
		`fmt\.Println\s*\(\s*["']Hello`,
		`System\.out\.println\s*\(\s*["']Hello`,
		`pass\s*$`,
		`// TODO: Implement`,
		`raise NotImplementedError`,
	}

	for _, pattern := range placeholderPatterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			result.Valid = false
			result.Score -= 50
			result.Issues = append(result.Issues, types.Issue{
				Type:    "error",
				Message: "Placeholder code detected - must provide actual implementation",
				Line:    0,
			})
			break
		}
	}

	// Check 3: Language-specific validation
	switch strings.ToLower(language) {
	case "python":
		v.validatePythonCode(code, result)
	case "javascript", "typescript":
		v.validateJavaScriptCode(code, result)
	case "go":
		v.validateGoCode(code, result)
	case "java":
		v.validateJavaCode(code, result)
	}

	// Check 4: Ensure proper structure
	if !v.hasProperStructure(code, language) {
		result.Score -= 20
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: "Code lacks proper structure (missing classes, functions, or modules)",
			Line:    0,
		})
	}

	// Check 5: Security patterns
	securityIssues := v.checkSecurityPatterns(code)
	if len(securityIssues) > 0 {
		result.Score -= float64(len(securityIssues) * 10)
		result.Issues = append(result.Issues, securityIssues...)
	}

	// Check 6: Test presence
	if !v.hasTests(code, language) && !strings.Contains(code, "test") {
		result.Score -= 15
		result.Issues = append(result.Issues, types.Issue{
			Type:    "warning",
			Message: "No tests found - production code requires tests",
			Line:    0,
		})
	}

	// Final scoring
	if result.Score < 60 {
		result.Valid = false
	}

	return result, nil
}

func (v *QualityValidator) validatePythonCode(code string, result *types.ValidationResult) {
	// Check for proper Python patterns
	hasClass := strings.Contains(code, "class ")
	hasImports := strings.Contains(code, "import ") || strings.Contains(code, "from ")
	
	functionCount := strings.Count(code, "def ")
	
	if functionCount < v.minFunctionCount {
		result.Score -= 20
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: fmt.Sprintf("Insufficient functions: found %d, minimum %d required", functionCount, v.minFunctionCount),
			Line:    0,
		})
	}

	if !hasImports {
		result.Score -= 10
		result.Issues = append(result.Issues, types.Issue{
			Type:    "warning",
			Message: "No imports found - real applications use libraries",
			Line:    0,
		})
	}

	// Check for error handling
	hasErrorHandling := strings.Contains(code, "try:") || strings.Contains(code, "except") || strings.Contains(code, "raise")
	if !hasErrorHandling {
		result.Score -= 15
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: "No error handling found - production code must handle errors",
			Line:    0,
		})
	}

	// Check for type hints (Python 3.5+)
	hasTypeHints := regexp.MustCompile(`def \w+\([^)]*:[\s\w\[\]]+[,)]`).MatchString(code) ||
	                regexp.MustCompile(`->\s*[\w\[\]]+:`).MatchString(code)
	
	if !hasTypeHints && (hasClass || functionCount > 2) {
		result.Score -= 10
		result.Issues = append(result.Issues, types.Issue{
			Type:    "warning",
			Message: "No type hints found - enterprise Python should use type annotations",
			Line:    0,
		})
	}
}

func (v *QualityValidator) validateJavaScriptCode(code string, result *types.ValidationResult) {
	// Check for proper JavaScript/TypeScript patterns
	functionCount := strings.Count(code, "function") + strings.Count(code, "=>")
	
	if functionCount < v.minFunctionCount {
		result.Score -= 20
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: fmt.Sprintf("Insufficient functions: found %d, minimum %d required", functionCount, v.minFunctionCount),
			Line:    0,
		})
	}

	// Check for error handling
	hasErrorHandling := strings.Contains(code, "try") && strings.Contains(code, "catch")
	hasPromiseHandling := strings.Contains(code, ".catch") || strings.Contains(code, "async") || strings.Contains(code, "await")
	
	if !hasErrorHandling && !hasPromiseHandling {
		result.Score -= 15
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: "No error handling found - must handle errors properly",
			Line:    0,
		})
	}

	// Check for module usage
	hasModules := strings.Contains(code, "import ") || strings.Contains(code, "require(") || strings.Contains(code, "export ")
	if !hasModules {
		result.Score -= 10
		result.Issues = append(result.Issues, types.Issue{
			Type:    "warning",
			Message: "No module imports/exports found",
			Line:    0,
		})
	}
}

func (v *QualityValidator) validateGoCode(code string, result *types.ValidationResult) {
	// Check for proper Go patterns
	functionCount := strings.Count(code, "func ")
	
	if functionCount < v.minFunctionCount {
		result.Score -= 20
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: fmt.Sprintf("Insufficient functions: found %d, minimum %d required", functionCount, v.minFunctionCount),
			Line:    0,
		})
	}

	// Check for error handling (Go's explicit error handling)
	errorHandling := strings.Count(code, "if err != nil")
	if errorHandling < 2 && functionCount > 2 {
		result.Score -= 20
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: "Insufficient error handling - Go requires explicit error checks",
			Line:    0,
		})
	}

	// Check for proper package declaration
	if !strings.Contains(code, "package ") {
		result.Score -= 30
		result.Valid = false
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: "Missing package declaration",
			Line:    1,
		})
	}
}

func (v *QualityValidator) validateJavaCode(code string, result *types.ValidationResult) {
	// Check for proper Java patterns
	hasClass := strings.Contains(code, "class ")
	methodCount := strings.Count(code, "public ") + strings.Count(code, "private ") + strings.Count(code, "protected ")
	
	if !hasClass {
		result.Score -= 30
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: "No class definition found",
			Line:    0,
		})
	}

	if methodCount < v.minFunctionCount {
		result.Score -= 20
		result.Issues = append(result.Issues, types.Issue{
			Type:    "error",
			Message: fmt.Sprintf("Insufficient methods: found %d, minimum %d required", methodCount, v.minFunctionCount),
			Line:    0,
		})
	}

	// Check for exception handling
	hasExceptionHandling := strings.Contains(code, "try") && strings.Contains(code, "catch")
	if !hasExceptionHandling {
		result.Score -= 15
		result.Issues = append(result.Issues, types.Issue{
			Type:    "warning",
			Message: "No exception handling found",
			Line:    0,
		})
	}
}

func (v *QualityValidator) hasProperStructure(code string, language string) bool {
	lines := strings.Split(code, "\n")
	if len(lines) < 20 {
		return false // Too few lines for proper structure
	}

	// Check for common structural elements
	hasMainLogic := false
	hasHelperFunctions := false
	
	switch strings.ToLower(language) {
	case "python":
		hasMainLogic = strings.Contains(code, "if __name__") || strings.Contains(code, "def main")
		hasHelperFunctions = strings.Count(code, "def ") > 2
	case "javascript", "typescript":
		hasMainLogic = strings.Contains(code, "export") || strings.Contains(code, "module.exports")
		hasHelperFunctions = strings.Count(code, "function") + strings.Count(code, "=>") > 2
	case "go":
		hasMainLogic = strings.Contains(code, "func main") || strings.Contains(code, "func ")
		hasHelperFunctions = strings.Count(code, "func ") > 2
	case "java":
		hasMainLogic = strings.Contains(code, "public static void main") || strings.Contains(code, "public class")
		hasHelperFunctions = strings.Count(code, "public ") + strings.Count(code, "private ") > 2
	default:
		return true // Be lenient for unknown languages
	}

	return hasMainLogic || hasHelperFunctions
}

func (v *QualityValidator) checkSecurityPatterns(code string) []types.Issue {
	issues := []types.Issue{}
	
	// Check for hardcoded secrets
	secretPatterns := []string{
		`["']api[_-]?key["']\s*[:=]\s*["'][^"']+["']`,
		`["']password["']\s*[:=]\s*["'][^"']+["']`,
		`["']secret["']\s*[:=]\s*["'][^"']+["']`,
		`["']token["']\s*[:=]\s*["'][^"']+["']`,
	}
	
	for _, pattern := range secretPatterns {
		if matched, _ := regexp.MatchString(pattern, strings.ToLower(code)); matched {
			issues = append(issues, types.Issue{
				Type:    "error",
				Message: "Potential hardcoded secret detected",
				Line:    0,
			})
			break
		}
	}
	
	// Check for SQL injection vulnerabilities
	if strings.Contains(code, "SELECT") || strings.Contains(code, "INSERT") || strings.Contains(code, "UPDATE") {
		// Check if using string concatenation with SQL
		if regexp.MustCompile(`["'].*(?:SELECT|INSERT|UPDATE|DELETE).*["']\s*\+`).MatchString(code) {
			issues = append(issues, types.Issue{
				Type:    "error",
				Message: "Potential SQL injection vulnerability - use parameterized queries",
				Line:    0,
			})
		}
	}
	
	return issues
}

func (v *QualityValidator) hasTests(code string, language string) bool {
	// Check for test patterns
	testPatterns := []string{
		`test_\w+`,           // Python test functions
		`Test\w+`,            // Go/Java test functions
		`describe\s*\(`,      // JavaScript test suites
		`it\s*\(`,            // JavaScript test cases
		`@Test`,              // Java annotations
		`unittest`,           // Python unittest
		`pytest`,             // Python pytest
		`testing\.T`,         // Go testing
	}
	
	for _, pattern := range testPatterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			return true
		}
	}
	
	return false
}

// CalculateComplexity calculates cyclomatic complexity of code
func (v *QualityValidator) CalculateComplexity(code string) int {
	complexity := 1 // Base complexity
	
	// Count decision points
	decisionPatterns := []string{
		`\bif\b`,
		`\belse\b`,
		`\belif\b`,
		`\bfor\b`,
		`\bwhile\b`,
		`\bcase\b`,
		`\bcatch\b`,
		`\bexcept\b`,
		`&&`,
		`\|\|`,
		`\?.*:`, // Ternary operator
	}
	
	for _, pattern := range decisionPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(code, -1)
		complexity += len(matches)
	}
	
	return complexity
}