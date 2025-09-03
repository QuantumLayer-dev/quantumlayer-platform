package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
)

var (
	parseRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "code_parse_requests_total",
		Help: "Total number of code parse requests",
	}, []string{"language", "status"})

	parseDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "code_parse_duration_seconds",
		Help: "Duration of code parsing operations",
	}, []string{"language"})
)

// Service provides HTTP API for code parsing
type Service struct {
	parser *Parser
	logger *zap.Logger
	port   string
}

// NewService creates a new parser service
func NewService(port string, logger *zap.Logger) *Service {
	return &Service{
		parser: NewParser(logger),
		logger: logger,
		port:   port,
	}
}

// ParseRequest represents a code parsing request
type ParseRequest struct {
	Code     string   `json:"code"`
	Language Language `json:"language"`
	Filename string   `json:"filename,omitempty"`
}

// ParseResponse represents a code parsing response
type ParseResponse struct {
	Success    bool              `json:"success"`
	HasErrors  bool              `json:"has_errors"`
	Metrics    *ParseMetrics     `json:"metrics"`
	Functions  []FunctionInfo    `json:"functions,omitempty"`
	Errors     []*ErrorNode      `json:"errors,omitempty"`
	ParseTime  int64            `json:"parse_time_ms"`
}

// Start starts the HTTP server
func (s *Service) Start() error {
	mux := http.NewServeMux()
	
	// API endpoints
	mux.HandleFunc("/parse", s.handleParse)
	mux.HandleFunc("/extract/functions", s.handleExtractFunctions)
	mux.HandleFunc("/analyze", s.handleAnalyze)
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	s.logger.Info("Starting parser service", zap.String("port", s.port))
	return http.ListenAndServe(":"+s.port, mux)
}

// handleParse handles code parsing requests
func (s *Service) handleParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		parseRequests.WithLabelValues(string(req.Language), "error").Inc()
		return
	}

	// Auto-detect language if not provided
	if req.Language == "" && req.Filename != "" {
		lang, err := DetectLanguage(req.Filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			parseRequests.WithLabelValues("unknown", "error").Inc()
			return
		}
		req.Language = lang
	}

	start := time.Now()
	result, err := s.parser.Parse(context.Background(), []byte(req.Code), req.Language)
	duration := time.Since(start)

	parseDuration.WithLabelValues(string(req.Language)).Observe(duration.Seconds())

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		parseRequests.WithLabelValues(string(req.Language), "error").Inc()
		return
	}

	parseRequests.WithLabelValues(string(req.Language), "success").Inc()

	response := ParseResponse{
		Success:   true,
		HasErrors: result.HasErrors,
		Metrics:   result.Metrics,
		Errors:    result.ErrorNodes,
		ParseTime: duration.Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleExtractFunctions handles function extraction requests
func (s *Service) handleExtractFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	start := time.Now()
	functions, err := s.parser.ExtractFunctions(context.Background(), []byte(req.Code), req.Language)
	duration := time.Since(start)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := ParseResponse{
		Success:   true,
		Functions: functions,
		ParseTime: duration.Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CodeAnalysis represents comprehensive code analysis results
type CodeAnalysis struct {
	Language           Language             `json:"language"`
	Metrics           *ParseMetrics        `json:"metrics"`
	Functions         []FunctionInfo       `json:"functions"`
	Imports           []string             `json:"imports"`
	Classes           []ClassInfo          `json:"classes,omitempty"`
	ComplexityScore   int                  `json:"complexity_score"`
	MaintainabilityIndex float64           `json:"maintainability_index"`
	CodeSmells        []CodeSmell          `json:"code_smells,omitempty"`
	SecurityIssues    []SecurityIssue      `json:"security_issues,omitempty"`
}

// ClassInfo represents information about a class
type ClassInfo struct {
	Name       string         `json:"name"`
	Methods    []FunctionInfo `json:"methods"`
	Properties []string       `json:"properties"`
	StartLine  uint32        `json:"start_line"`
	EndLine    uint32        `json:"end_line"`
}

// CodeSmell represents a potential code quality issue
type CodeSmell struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Line        uint32 `json:"line"`
	Column      uint32 `json:"column"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// SecurityIssue represents a potential security vulnerability
type SecurityIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Line        uint32 `json:"line"`
	CWE         string `json:"cwe,omitempty"`
	OWASP       string `json:"owasp,omitempty"`
	Remediation string `json:"remediation"`
}

// handleAnalyze performs comprehensive code analysis
func (s *Service) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse the code
	result, err := s.parser.Parse(context.Background(), []byte(req.Code), req.Language)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract functions
	functions, _ := s.parser.ExtractFunctions(context.Background(), []byte(req.Code), req.Language)

	// Perform analysis
	analysis := CodeAnalysis{
		Language:  req.Language,
		Metrics:   result.Metrics,
		Functions: functions,
		Imports:   s.extractImports(result.RootNode, []byte(req.Code), req.Language),
	}

	// Calculate complexity score
	totalComplexity := 0
	for _, fn := range functions {
		totalComplexity += fn.Complexity
	}
	analysis.ComplexityScore = totalComplexity

	// Calculate maintainability index (simplified version)
	// MI = 171 - 5.2*ln(HV) - 0.23*CC - 16.2*ln(LOC)
	// Where HV = Halstead Volume, CC = Cyclomatic Complexity, LOC = Lines of Code
	if result.Metrics.LinesOfCode > 0 {
		analysis.MaintainabilityIndex = 100.0 // Simplified for now
	}

	// Check for code smells
	analysis.CodeSmells = s.detectCodeSmells(result, []byte(req.Code))

	// Check for security issues
	analysis.SecurityIssues = s.detectSecurityIssues(result, []byte(req.Code), req.Language)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// extractImports extracts import statements from code
func (s *Service) extractImports(node *sitter.Node, code []byte, lang Language) []string {
	var imports []string

	s.parser.walkTree(node, code, func(n *sitter.Node) bool {
		nodeType := n.Type()
		
		switch lang {
		case LangGo:
			if nodeType == "import_declaration" || nodeType == "import_spec" {
				if pathNode := n.ChildByFieldName("path"); pathNode != nil {
					imports = append(imports, pathNode.Content(code))
				}
			}
		case LangPython:
			if nodeType == "import_statement" || nodeType == "import_from_statement" {
				imports = append(imports, n.Content(code))
			}
		case LangJavaScript, LangTypeScript:
			if nodeType == "import_statement" {
				imports = append(imports, n.Content(code))
			}
		case LangJava:
			if nodeType == "import_declaration" {
				imports = append(imports, n.Content(code))
			}
		}
		return true
	})

	return imports
}

// detectCodeSmells identifies potential code quality issues
func (s *Service) detectCodeSmells(result *ParseResult, code []byte) []CodeSmell {
	var smells []CodeSmell

	// Check for long functions
	s.parser.walkTree(result.RootNode, code, func(node *sitter.Node) bool {
		if s.parser.isFunctionNode(node, result.Language) {
			lines := node.EndPoint().Row - node.StartPoint().Row
			if lines > 50 {
				smells = append(smells, CodeSmell{
					Type:     "long_function",
					Severity: "warning",
					Message:  fmt.Sprintf("Function is too long (%d lines)", lines),
					Line:     node.StartPoint().Row + 1,
					Column:   node.StartPoint().Column + 1,
					Suggestion: "Consider breaking this function into smaller, more focused functions",
				})
			}
		}
		return true
	})

	// Check for deeply nested code
	s.checkDeepNesting(result.RootNode, code, 0, &smells)

	return smells
}

// checkDeepNesting checks for excessive nesting depth
func (s *Service) checkDeepNesting(node *sitter.Node, code []byte, depth int, smells *[]CodeSmell) {
	if depth > 4 {
		*smells = append(*smells, CodeSmell{
			Type:     "deep_nesting",
			Severity: "warning",
			Message:  fmt.Sprintf("Code is nested too deeply (depth: %d)", depth),
			Line:     node.StartPoint().Row + 1,
			Column:   node.StartPoint().Column + 1,
			Suggestion: "Consider extracting nested logic into separate functions",
		})
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		newDepth := depth
		if isNestingNode(child.Type()) {
			newDepth++
		}
		s.checkDeepNesting(child, code, newDepth, smells)
	}
}

func isNestingNode(nodeType string) bool {
	nestingTypes := []string{
		"if_statement", "for_statement", "while_statement",
		"switch_statement", "try_statement", "block",
	}
	for _, t := range nestingTypes {
		if nodeType == t {
			return true
		}
	}
	return false
}

// detectSecurityIssues identifies potential security vulnerabilities
func (s *Service) detectSecurityIssues(result *ParseResult, code []byte, lang Language) []SecurityIssue {
	var issues []SecurityIssue

	codeStr := string(code)

	// Check for hardcoded secrets
	patterns := []struct {
		pattern string
		message string
		cwe     string
	}{
		{`(?i)(api[_-]?key|apikey|secret|password|passwd|pwd)\s*[:=]\s*["'][^"']+["']`, "Potential hardcoded secret detected", "CWE-798"},
		{`(?i)jwt\.secret\s*[:=]\s*["'][^"']+["']`, "Hardcoded JWT secret detected", "CWE-798"},
		{`(?i)private[_-]?key\s*[:=]\s*["'][^"']+["']`, "Hardcoded private key detected", "CWE-798"},
	}

	for _, p := range patterns {
		// Simplified check - in production, use proper regex
		if containsPattern(codeStr, p.pattern) {
			issues = append(issues, SecurityIssue{
				Type:        "hardcoded_secret",
				Severity:    "high",
				Message:     p.message,
				Line:        1, // Would need proper line detection
				CWE:         p.cwe,
				OWASP:       "A3:2021",
				Remediation: "Use environment variables or secure secret management systems",
			})
		}
	}

	// Language-specific checks
	switch lang {
	case LangGo:
		// Check for SQL injection vulnerabilities
		if containsPattern(codeStr, `fmt\.Sprintf.*SELECT.*FROM`) {
			issues = append(issues, SecurityIssue{
				Type:        "sql_injection",
				Severity:    "critical",
				Message:     "Potential SQL injection vulnerability",
				Line:        1,
				CWE:         "CWE-89",
				OWASP:       "A1:2021",
				Remediation: "Use parameterized queries or prepared statements",
			})
		}
	}

	return issues
}

func containsPattern(text, pattern string) bool {
	// Simplified - in production, compile and use regex
	return false
}

// handleHealth handles health check requests
func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "parser",
	})
}