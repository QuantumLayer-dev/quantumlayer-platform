package activities

import (
	"strings"
)

// DetectContentLanguage detects the actual language/format of generated content
func DetectContentLanguage(content string, requestedType string) string {
	// Normalize content for analysis
	trimmedContent := strings.TrimSpace(content)
	lowerContent := strings.ToLower(trimmedContent)
	
	// Check for specific content types based on requestedType
	switch strings.ToLower(requestedType) {
	case "docker", "dockerfile":
		// Check if it's a Dockerfile
		if strings.HasPrefix(lowerContent, "from ") || strings.Contains(lowerContent, "\nfrom ") {
			return "dockerfile"
		}
		// Check if it's docker-compose YAML
		if isDockerCompose(trimmedContent) {
			return "yaml"
		}
		return "dockerfile"
		
	case "docker-compose", "compose", "docker compose":
		return "yaml"
		
	case "kubernetes", "k8s", "helm":
		return "yaml"
		
	case "terraform", "tf":
		return "hcl"
		
	case "configuration", "config":
		// Try to detect the config format
		if isYAML(trimmedContent) {
			return "yaml"
		}
		if isJSON(trimmedContent) {
			return "json"
		}
		if isTOML(trimmedContent) {
			return "toml"
		}
		if isINI(trimmedContent) {
			return "ini"
		}
		return "text"
		
	case "infrastructure", "iac":
		// Check for various IaC formats
		if isYAML(trimmedContent) {
			return "yaml"
		}
		if isTerraform(trimmedContent) {
			return "hcl"
		}
		return "yaml" // Default to YAML for IaC
		
	case "api", "openapi", "swagger":
		if isYAML(trimmedContent) {
			return "yaml"
		}
		if isJSON(trimmedContent) {
			return "json"
		}
		return "yaml"
		
	case "sql", "database", "schema":
		return "sql"
		
	case "shell", "script", "bash":
		return "bash"
		
	case "documentation", "doc", "readme":
		return "markdown"
		
	case "html", "webpage", "web":
		if strings.Contains(lowerContent, "<html") || strings.Contains(lowerContent, "<!doctype html") {
			return "html"
		}
		if strings.Contains(lowerContent, "<react") || strings.Contains(lowerContent, "jsx") {
			return "jsx"
		}
		return "html"
		
	case "test", "tests", "spec":
		// Keep the original language for tests
		return ""
	}
	
	// If no specific type, try to detect from content patterns
	if isYAML(trimmedContent) {
		return "yaml"
	}
	if isJSON(trimmedContent) {
		return "json"
	}
	if isXML(trimmedContent) {
		return "xml"
	}
	if isHTML(trimmedContent) {
		return "html"
	}
	if isSQL(trimmedContent) {
		return "sql"
	}
	if isShellScript(trimmedContent) {
		return "bash"
	}
	if isDockerfile(trimmedContent) {
		return "dockerfile"
	}
	
	// Return empty to keep original language
	return ""
}

// Helper functions to detect content formats
func isYAML(content string) bool {
	// Common YAML patterns
	patterns := []string{
		"---",
		"version:",
		"services:",
		"name:",
		"apiVersion:",
		"kind:",
		"spec:",
		"metadata:",
	}
	
	hasColonSpace := strings.Contains(content, ": ")
	hasDash := strings.HasPrefix(strings.TrimSpace(content), "-") || strings.Contains(content, "\n-")
	
	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(content), pattern) {
			return true
		}
	}
	
	// Check for YAML-like structure (key: value with consistent indentation)
	return hasColonSpace && (hasDash || strings.Count(content, ":") > 2)
}

func isJSON(content string) bool {
	trimmed := strings.TrimSpace(content)
	return (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		   (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))
}

func isXML(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "<?xml") || 
		   (strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">"))
}

func isHTML(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "<html") || 
		   strings.Contains(lower, "<!doctype html") ||
		   (strings.Contains(lower, "<head") && strings.Contains(lower, "<body"))
}

func isTOML(content string) bool {
	return strings.Contains(content, "[") && strings.Contains(content, "]") && 
		   strings.Contains(content, "=") && !strings.Contains(content, "{")
}

func isINI(content string) bool {
	return strings.Contains(content, "[") && strings.Contains(content, "]") && 
		   strings.Contains(content, "=")
}

func isSQL(content string) bool {
	lower := strings.ToLower(content)
	sqlKeywords := []string{"select ", "insert ", "update ", "delete ", "create table", "alter table", "drop "}
	for _, keyword := range sqlKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func isShellScript(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "#!/") ||
		   strings.Contains(content, "#!/bin/bash") ||
		   strings.Contains(content, "#!/bin/sh")
}

func isDockerfile(content string) bool {
	lower := strings.ToLower(content)
	return strings.HasPrefix(lower, "from ") || 
		   strings.Contains(lower, "\nfrom ") ||
		   strings.Contains(lower, "\nrun ") ||
		   strings.Contains(lower, "\nexpose ")
}

func isDockerCompose(content string) bool {
	lower := strings.ToLower(content)
	return (strings.Contains(lower, "version:") || strings.Contains(lower, "version :")) &&
		   (strings.Contains(lower, "services:") || strings.Contains(lower, "services :"))
}

func isTerraform(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "resource \"") || 
		   strings.Contains(lower, "provider \"") ||
		   strings.Contains(lower, "variable \"") ||
		   strings.Contains(lower, "terraform {")
}