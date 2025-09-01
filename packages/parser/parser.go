package parser

import (
	"context"
	"fmt"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/json"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"
	"go.uber.org/zap"
)

// Language represents a supported programming language
type Language string

const (
	LangGo         Language = "go"
	LangPython     Language = "python"
	LangJavaScript Language = "javascript"
	LangTypeScript Language = "typescript"
	LangTSX        Language = "tsx"
	LangJava       Language = "java"
	LangRust       Language = "rust"
	LangCpp        Language = "cpp"
	LangC          Language = "c"
	LangCSharp     Language = "csharp"
	LangRuby       Language = "ruby"
	LangPHP        Language = "php"
	LangScala      Language = "scala"
	LangSQL        Language = "sql"
	LangBash       Language = "bash"
	LangYAML       Language = "yaml"
	LangJSON       Language = "json"
	LangTOML       Language = "toml"
	LangHTML       Language = "html"
	LangCSS        Language = "css"
	LangDockerfile Language = "dockerfile"
	LangProtobuf   Language = "protobuf"
	LangHCL        Language = "hcl"
)

// Parser provides code parsing capabilities using Tree-sitter
type Parser struct {
	parsers map[Language]*sitter.Parser
	logger  *zap.Logger
	mu      sync.RWMutex
}

// NewParser creates a new code parser instance
func NewParser(logger *zap.Logger) *Parser {
	p := &Parser{
		parsers: make(map[Language]*sitter.Parser),
		logger:  logger,
	}
	p.initializeParsers()
	return p
}

// initializeParsers sets up all language parsers
func (p *Parser) initializeParsers() {
	languages := map[Language]*sitter.Language{
		LangGo:         golang.GetLanguage(),
		LangPython:     python.GetLanguage(),
		LangJavaScript: javascript.GetLanguage(),
		LangTypeScript: typescript.GetLanguage(),
		LangTSX:        tsx.GetLanguage(),
		LangJava:       java.GetLanguage(),
		LangRust:       rust.GetLanguage(),
		LangCpp:        cpp.GetLanguage(),
		LangC:          c.GetLanguage(),
		LangCSharp:     csharp.GetLanguage(),
		LangRuby:       ruby.GetLanguage(),
		LangPHP:        php.GetLanguage(),
		LangScala:      scala.GetLanguage(),
		LangSQL:        sql.GetLanguage(),
		LangBash:       bash.GetLanguage(),
		LangYAML:       yaml.GetLanguage(),
		LangJSON:       json.GetLanguage(),
		LangTOML:       toml.GetLanguage(),
		LangHTML:       html.GetLanguage(),
		LangCSS:        css.GetLanguage(),
		LangDockerfile: dockerfile.GetLanguage(),
		LangProtobuf:   protobuf.GetLanguage(),
		LangHCL:        hcl.GetLanguage(),
	}

	for lang, sitterLang := range languages {
		parser := sitter.NewParser()
		parser.SetLanguage(sitterLang)
		p.parsers[lang] = parser
		p.logger.Info("Initialized parser", zap.String("language", string(lang)))
	}
}

// ParseResult contains the results of parsing code
type ParseResult struct {
	Tree       *sitter.Tree
	RootNode   *sitter.Node
	Language   Language
	HasErrors  bool
	ErrorNodes []*ErrorNode
	Metrics    *ParseMetrics
}

// ErrorNode represents a parsing error
type ErrorNode struct {
	StartPos  sitter.Point
	EndPos    sitter.Point
	Message   string
	NodeType  string
}

// ParseMetrics contains parsing statistics
type ParseMetrics struct {
	NodeCount       int
	MaxDepth        int
	ParseTimeMs     int64
	FileSize        int
	LinesOfCode     int
	CyclomaticComplexity int
}

// Parse analyzes source code and returns a parse tree
func (p *Parser) Parse(ctx context.Context, code []byte, lang Language) (*ParseResult, error) {
	p.mu.RLock()
	parser, ok := p.parsers[lang]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	tree, err := parser.ParseCtx(ctx, nil, code)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	rootNode := tree.RootNode()
	result := &ParseResult{
		Tree:      tree,
		RootNode:  rootNode,
		Language:  lang,
		HasErrors: rootNode.HasError(),
		Metrics:   p.calculateMetrics(rootNode, code),
	}

	// Collect error nodes if any
	if result.HasErrors {
		result.ErrorNodes = p.collectErrors(rootNode)
	}

	return result, nil
}

// ExtractFunctions extracts all function definitions from the code
func (p *Parser) ExtractFunctions(ctx context.Context, code []byte, lang Language) ([]FunctionInfo, error) {
	result, err := p.Parse(ctx, code, lang)
	if err != nil {
		return nil, err
	}

	var functions []FunctionInfo
	p.walkTree(result.RootNode, code, func(node *sitter.Node) bool {
		if p.isFunctionNode(node, lang) {
			info := p.extractFunctionInfo(node, code, lang)
			functions = append(functions, info)
		}
		return true
	})

	return functions, nil
}

// FunctionInfo contains information about a function
type FunctionInfo struct {
	Name       string
	Parameters []string
	ReturnType string
	StartLine  uint32
	EndLine    uint32
	Body       string
	DocComment string
	Complexity int
}

// isFunctionNode checks if a node represents a function definition
func (p *Parser) isFunctionNode(node *sitter.Node, lang Language) bool {
	nodeType := node.Type()
	
	switch lang {
	case LangGo:
		return nodeType == "function_declaration" || nodeType == "method_declaration"
	case LangPython:
		return nodeType == "function_definition"
	case LangJavaScript, LangTypeScript, LangTSX:
		return nodeType == "function_declaration" || 
			   nodeType == "arrow_function" || 
			   nodeType == "function_expression" ||
			   nodeType == "method_definition"
	case LangJava:
		return nodeType == "method_declaration"
	case LangRust:
		return nodeType == "function_item"
	case LangC, LangCpp:
		return nodeType == "function_definition"
	case LangCSharp:
		return nodeType == "method_declaration"
	case LangRuby:
		return nodeType == "method"
	case LangPHP:
		return nodeType == "function_definition" || nodeType == "method_declaration"
	default:
		return false
	}
}

// extractFunctionInfo extracts detailed information about a function
func (p *Parser) extractFunctionInfo(node *sitter.Node, code []byte, lang Language) FunctionInfo {
	info := FunctionInfo{
		StartLine: node.StartPoint().Row + 1,
		EndLine:   node.EndPoint().Row + 1,
		Body:      node.Content(code),
	}

	// Extract function name based on language
	switch lang {
	case LangGo:
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			info.Name = nameNode.Content(code)
		}
		if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
			info.Parameters = p.extractParameters(paramsNode, code)
		}
		if returnNode := node.ChildByFieldName("result"); returnNode != nil {
			info.ReturnType = returnNode.Content(code)
		}
	case LangPython:
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			info.Name = nameNode.Content(code)
		}
		if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
			info.Parameters = p.extractParameters(paramsNode, code)
		}
	// Add other language-specific extraction logic
	}

	// Calculate cyclomatic complexity
	info.Complexity = p.calculateComplexity(node)

	return info
}

// extractParameters extracts parameter names from a parameter list
func (p *Parser) extractParameters(node *sitter.Node, code []byte) []string {
	var params []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "parameter" || child.Type() == "parameter_declaration" {
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				params = append(params, nameNode.Content(code))
			}
		}
	}
	return params
}

// calculateComplexity calculates cyclomatic complexity of a node
func (p *Parser) calculateComplexity(node *sitter.Node) int {
	complexity := 1 // Base complexity

	p.walkTree(node, nil, func(n *sitter.Node) bool {
		switch n.Type() {
		case "if_statement", "if_expression", "conditional_expression",
			 "while_statement", "for_statement", "for_in_statement",
			 "case_statement", "catch_clause":
			complexity++
		}
		return true
	})

	return complexity
}

// walkTree traverses the AST and applies a function to each node
func (p *Parser) walkTree(node *sitter.Node, code []byte, fn func(*sitter.Node) bool) {
	if !fn(node) {
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		p.walkTree(child, code, fn)
	}
}

// collectErrors collects all error nodes in the tree
func (p *Parser) collectErrors(node *sitter.Node) []*ErrorNode {
	var errors []*ErrorNode

	p.walkTree(node, nil, func(n *sitter.Node) bool {
		if n.IsError() || n.IsMissing() {
			errors = append(errors, &ErrorNode{
				StartPos: n.StartPoint(),
				EndPos:   n.EndPoint(),
				NodeType: n.Type(),
				Message:  fmt.Sprintf("Parse error at %d:%d", n.StartPoint().Row, n.StartPoint().Column),
			})
		}
		return true
	})

	return errors
}

// calculateMetrics calculates various metrics for the parsed code
func (p *Parser) calculateMetrics(node *sitter.Node, code []byte) *ParseMetrics {
	metrics := &ParseMetrics{
		FileSize:    len(code),
		LinesOfCode: strings.Count(string(code), "\n") + 1,
	}

	var maxDepth int
	p.calculateDepthAndCount(node, 0, &maxDepth, &metrics.NodeCount)
	metrics.MaxDepth = maxDepth

	return metrics
}

// calculateDepthAndCount recursively calculates tree depth and node count
func (p *Parser) calculateDepthAndCount(node *sitter.Node, currentDepth int, maxDepth *int, nodeCount *int) {
	*nodeCount++
	if currentDepth > *maxDepth {
		*maxDepth = currentDepth
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		p.calculateDepthAndCount(child, currentDepth+1, maxDepth, nodeCount)
	}
}

// DetectLanguage attempts to detect the language from file extension
func DetectLanguage(filename string) (Language, error) {
	ext := strings.ToLower(getFileExtension(filename))
	
	langMap := map[string]Language{
		".go":         LangGo,
		".py":         LangPython,
		".js":         LangJavaScript,
		".mjs":        LangJavaScript,
		".ts":         LangTypeScript,
		".tsx":        LangTSX,
		".jsx":        LangTSX,
		".java":       LangJava,
		".rs":         LangRust,
		".cpp":        LangCpp,
		".cc":         LangCpp,
		".cxx":        LangCpp,
		".c":          LangC,
		".h":          LangC,
		".cs":         LangCSharp,
		".rb":         LangRuby,
		".php":        LangPHP,
		".scala":      LangScala,
		".sql":        LangSQL,
		".sh":         LangBash,
		".bash":       LangBash,
		".yaml":       LangYAML,
		".yml":        LangYAML,
		".json":       LangJSON,
		".toml":       LangTOML,
		".html":       LangHTML,
		".htm":        LangHTML,
		".css":        LangCSS,
		".dockerfile": LangDockerfile,
		".proto":      LangProtobuf,
		".hcl":        LangHCL,
		".tf":         LangHCL,
	}

	if lang, ok := langMap[ext]; ok {
		return lang, nil
	}

	// Check for Dockerfile without extension
	if strings.ToLower(filename) == "dockerfile" {
		return LangDockerfile, nil
	}

	return "", fmt.Errorf("unsupported file type: %s", ext)
}

func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return "." + parts[len(parts)-1]
	}
	return ""
}