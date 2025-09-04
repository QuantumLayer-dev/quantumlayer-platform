package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ExecutionRequest represents a code execution request
type ExecutionRequest struct {
	ID           string                 `json:"id"`
	Language     string                 `json:"language" binding:"required"`
	Code         string                 `json:"code" binding:"required"`
	Files        map[string]string      `json:"files,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Command      string                 `json:"command,omitempty"`
	Timeout      int                    `json:"timeout,omitempty"` // seconds, default 30
	Environment  map[string]string      `json:"environment,omitempty"`
	Resources    ResourceLimits         `json:"resources,omitempty"`
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	CPULimit    string `json:"cpu_limit,omitempty"`    // e.g., "0.5" for half CPU
	MemoryLimit string `json:"memory_limit,omitempty"` // e.g., "256m"
	DiskLimit   string `json:"disk_limit,omitempty"`   // e.g., "100m"
}

// ExecutionResult represents the execution output
type ExecutionResult struct {
	ID         string           `json:"id"`
	Status     string           `json:"status"` // running, success, error, timeout
	Output     string           `json:"output"`
	Error      string           `json:"error,omitempty"`
	ExitCode   int              `json:"exit_code"`
	Duration   float64          `json:"duration_seconds"`
	Metrics    ExecutionMetrics `json:"metrics"`
	StartedAt  time.Time        `json:"started_at"`
	FinishedAt time.Time        `json:"finished_at"`
}

// ExecutionMetrics contains performance metrics
type ExecutionMetrics struct {
	CPUUsage    float64 `json:"cpu_usage_percent"`
	MemoryUsage int64   `json:"memory_usage_bytes"`
	DiskUsage   int64   `json:"disk_usage_bytes"`
}

// RuntimeContainer represents a language runtime
type RuntimeContainer struct {
	Language    string
	Image       string
	BuildCmd    string
	RunCmd      string
	Extension   string
	Dockerfile  string
}

var (
	// Runtime configurations
	runtimes = map[string]RuntimeContainer{
		"python": {
			Language:  "python",
			Image:     "python:3.11-slim",
			RunCmd:    "python",
			Extension: ".py",
		},
		"javascript": {
			Language:  "javascript",
			Image:     "node:18-alpine",
			RunCmd:    "node",
			Extension: ".js",
		},
		"typescript": {
			Language:  "typescript",
			Image:     "node:18-alpine",
			BuildCmd:  "npx tsc",
			RunCmd:    "node",
			Extension: ".ts",
		},
		"go": {
			Language:  "go",
			Image:     "golang:1.21-alpine",
			BuildCmd:  "go build -o main",
			RunCmd:    "./main",
			Extension: ".go",
		},
		"java": {
			Language:  "java",
			Image:     "openjdk:17-alpine",
			BuildCmd:  "javac",
			RunCmd:    "java",
			Extension: ".java",
		},
		"rust": {
			Language:  "rust",
			Image:     "rust:1.75-alpine",
			BuildCmd:  "rustc -o main",
			RunCmd:    "./main",
			Extension: ".rs",
		},
		"ruby": {
			Language:  "ruby",
			Image:     "ruby:3.2-alpine",
			RunCmd:    "ruby",
			Extension: ".rb",
		},
		"php": {
			Language:  "php",
			Image:     "php:8.2-cli-alpine",
			RunCmd:    "php",
			Extension: ".php",
		},
	}

	// Active executions
	executions = sync.Map{}
	
	// WebSocket upgrader
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}
	
	// WebSocket connections
	wsConnections = sync.Map{}
)

func main() {
	r := gin.Default()

	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API endpoints
	v1 := r.Group("/api/v1")
	{
		// Execute code
		v1.POST("/execute", handleExecute)
		
		// Get execution status
		v1.GET("/executions/:id", handleGetExecution)
		
		// Stream execution output via WebSocket
		v1.GET("/executions/:id/stream", handleStreamExecution)
		
		// Stop execution
		v1.DELETE("/executions/:id", handleStopExecution)
		
		// List supported runtimes
		v1.GET("/runtimes", handleListRuntimes)
		
		// Validate code without executing
		v1.POST("/validate", handleValidate)
		
		// Execute with file system (multiple files)
		v1.POST("/execute-project", handleExecuteProject)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8091"
	}

	log.Printf("Starting Sandbox Executor on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func handleExecute(c *gin.Context) {
	var req ExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate execution ID
	req.ID = uuid.New().String()

	// Set default timeout
	if req.Timeout == 0 {
		req.Timeout = 30
	}

	// Get runtime configuration
	runtime, exists := runtimes[strings.ToLower(req.Language)]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported language: " + req.Language})
		return
	}

	// Create execution result
	result := &ExecutionResult{
		ID:        req.ID,
		Status:    "running",
		StartedAt: time.Now(),
	}

	// Store execution
	executions.Store(req.ID, result)

	// Execute in background
	go executeCode(req, runtime, result)

	c.JSON(http.StatusAccepted, gin.H{
		"id":      req.ID,
		"status":  "running",
		"message": "Execution started",
	})
}

func executeCode(req ExecutionRequest, runtime RuntimeContainer, result *ExecutionResult) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
	defer cancel()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "sandbox-"+req.ID)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("Failed to create temp directory: %v", err)
		result.FinishedAt = time.Now()
		return
	}
	defer os.RemoveAll(tempDir)

	// Write code to file
	filename := filepath.Join(tempDir, "main"+runtime.Extension)
	if err := os.WriteFile(filename, []byte(req.Code), 0644); err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("Failed to write code file: %v", err)
		result.FinishedAt = time.Now()
		return
	}

	// Write additional files if provided
	for path, content := range req.Files {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("Failed to create directory %s: %v", dir, err)
			result.FinishedAt = time.Now()
			return
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("Failed to write file %s: %v", path, err)
			result.FinishedAt = time.Now()
			return
		}
	}

	// Install dependencies if needed
	if len(req.Dependencies) > 0 {
		if err := installDependencies(ctx, tempDir, req.Language, req.Dependencies); err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("Failed to install dependencies: %v", err)
			result.FinishedAt = time.Now()
			return
		}
	}

	// Build Docker command
	dockerCmd := buildDockerCommand(req, runtime, tempDir, filename)

	// Execute with streaming
	executeWithStreaming(ctx, dockerCmd, req.ID, result)

	// Update metrics
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt).Seconds()
	
	if result.Error == "" && result.Status != "timeout" {
		result.Status = "success"
	}
}

func buildDockerCommand(req ExecutionRequest, runtime RuntimeContainer, tempDir, filename string) []string {
	cmd := []string{"docker", "run", "--rm"}
	
	// Add resource limits
	if req.Resources.CPULimit != "" {
		cmd = append(cmd, "--cpus", req.Resources.CPULimit)
	}
	if req.Resources.MemoryLimit != "" {
		cmd = append(cmd, "-m", req.Resources.MemoryLimit)
	}
	
	// Add environment variables
	for key, value := range req.Environment {
		cmd = append(cmd, "-e", fmt.Sprintf("%s=%s", key, value))
	}
	
	// Mount volume
	cmd = append(cmd, "-v", fmt.Sprintf("%s:/app", tempDir))
	cmd = append(cmd, "-w", "/app")
	
	// Add network isolation
	cmd = append(cmd, "--network", "none")
	
	// Add security options
	cmd = append(cmd, "--security-opt", "no-new-privileges")
	cmd = append(cmd, "--cap-drop", "ALL")
	
	// Add image
	cmd = append(cmd, runtime.Image)
	
	// Add command
	if req.Command != "" {
		cmd = append(cmd, "sh", "-c", req.Command)
	} else if runtime.BuildCmd != "" {
		// Languages that need compilation
		buildAndRun := fmt.Sprintf("%s main%s && %s", runtime.BuildCmd, runtime.Extension, runtime.RunCmd)
		cmd = append(cmd, "sh", "-c", buildAndRun)
	} else {
		// Interpreted languages
		cmd = append(cmd, runtime.RunCmd, filepath.Base(filename))
	}
	
	return cmd
}

func executeWithStreaming(ctx context.Context, dockerCmd []string, execID string, result *ExecutionResult) {
	cmd := exec.CommandContext(ctx, dockerCmd[0], dockerCmd[1:]...)
	
	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("Failed to create stdout pipe: %v", err)
		return
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("Failed to create stderr pipe: %v", err)
		return
	}
	
	// Start command
	if err := cmd.Start(); err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("Failed to start execution: %v", err)
		return
	}
	
	// Read output
	var outputBuffer, errorBuffer bytes.Buffer
	
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				outputBuffer.Write(buf[:n])
				streamToWebSocket(execID, string(buf[:n]), "stdout")
			}
			if err != nil {
				break
			}
		}
	}()
	
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				errorBuffer.Write(buf[:n])
				streamToWebSocket(execID, string(buf[:n]), "stderr")
			}
			if err != nil {
				break
			}
		}
	}()
	
	// Wait for completion
	err = cmd.Wait()
	
	result.Output = outputBuffer.String()
	if errorBuffer.Len() > 0 {
		result.Error = errorBuffer.String()
	}
	
	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "timeout"
		result.Error = "Execution timed out"
	} else if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Status = "error"
		}
	} else {
		result.ExitCode = 0
	}
}

func streamToWebSocket(execID, data, stream string) {
	if conn, ok := wsConnections.Load(execID); ok {
		wsConn := conn.(*websocket.Conn)
		message := map[string]interface{}{
			"type":   stream,
			"data":   data,
			"time":   time.Now().Unix(),
		}
		wsConn.WriteJSON(message)
	}
}

func installDependencies(ctx context.Context, dir, language string, deps []string) error {
	var cmd *exec.Cmd
	
	switch strings.ToLower(language) {
	case "python":
		// Create requirements.txt
		reqFile := filepath.Join(dir, "requirements.txt")
		if err := os.WriteFile(reqFile, []byte(strings.Join(deps, "\n")), 0644); err != nil {
			return err
		}
		cmd = exec.CommandContext(ctx, "docker", "run", "--rm", "-v", dir+":/app", "-w", "/app",
			"python:3.11-slim", "pip", "install", "-r", "requirements.txt", "--target", ".")
			
	case "javascript", "typescript":
		// Create package.json
		packageJSON := map[string]interface{}{
			"name":         "sandbox-exec",
			"version":      "1.0.0",
			"dependencies": make(map[string]string),
		}
		for _, dep := range deps {
			packageJSON["dependencies"].(map[string]string)[dep] = "latest"
		}
		data, _ := json.MarshalIndent(packageJSON, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0644); err != nil {
			return err
		}
		cmd = exec.CommandContext(ctx, "docker", "run", "--rm", "-v", dir+":/app", "-w", "/app",
			"node:18-alpine", "npm", "install")
			
	case "go":
		// Initialize go.mod
		cmd = exec.CommandContext(ctx, "docker", "run", "--rm", "-v", dir+":/app", "-w", "/app",
			"golang:1.21-alpine", "go", "mod", "init", "sandbox")
		if err := cmd.Run(); err != nil {
			return err
		}
		// Get dependencies
		for _, dep := range deps {
			cmd = exec.CommandContext(ctx, "docker", "run", "--rm", "-v", dir+":/app", "-w", "/app",
				"golang:1.21-alpine", "go", "get", dep)
			if err := cmd.Run(); err != nil {
				return err
			}
		}
		return nil
		
	default:
		return nil // Skip dependency installation for other languages
	}
	
	return cmd.Run()
}

func handleGetExecution(c *gin.Context) {
	id := c.Param("id")
	
	if result, ok := executions.Load(id); ok {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
	}
}

func handleStreamExecution(c *gin.Context) {
	id := c.Param("id")
	
	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to websocket: %v", err)
		return
	}
	defer conn.Close()
	
	// Store connection
	wsConnections.Store(id, conn)
	defer wsConnections.Delete(id)
	
	// Send initial status
	if result, ok := executions.Load(id); ok {
		conn.WriteJSON(map[string]interface{}{
			"type": "status",
			"data": result,
		})
	}
	
	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func handleStopExecution(c *gin.Context) {
	id := c.Param("id")
	
	// TODO: Implement execution cancellation
	c.JSON(http.StatusOK, gin.H{"message": "execution stopped", "id": id})
}

func handleListRuntimes(c *gin.Context) {
	runtimeList := make([]map[string]string, 0, len(runtimes))
	for lang, runtime := range runtimes {
		runtimeList = append(runtimeList, map[string]string{
			"language": lang,
			"image":    runtime.Image,
			"extension": runtime.Extension,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"runtimes": runtimeList,
		"total":    len(runtimeList),
	})
}

func handleValidate(c *gin.Context) {
	var req struct {
		Language string `json:"language" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Basic validation for now
	// TODO: Implement actual syntax validation
	issues := []string{}
	
	switch strings.ToLower(req.Language) {
	case "python":
		if !strings.Contains(req.Code, "def ") && !strings.Contains(req.Code, "class ") &&
		   !strings.Contains(req.Code, "import ") && !strings.Contains(req.Code, "print") {
			issues = append(issues, "No Python code detected")
		}
	case "javascript", "typescript":
		if !strings.Contains(req.Code, "function") && !strings.Contains(req.Code, "const") &&
		   !strings.Contains(req.Code, "let") && !strings.Contains(req.Code, "var") {
			issues = append(issues, "No JavaScript code detected")
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"valid":  len(issues) == 0,
		"issues": issues,
	})
}

func handleExecuteProject(c *gin.Context) {
	var req struct {
		Language     string            `json:"language" binding:"required"`
		Files        map[string]string `json:"files" binding:"required"`
		EntryPoint   string            `json:"entry_point" binding:"required"`
		Dependencies []string          `json:"dependencies,omitempty"`
		Command      string            `json:"command,omitempty"`
		Timeout      int               `json:"timeout,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get entry point content
	entryContent, exists := req.Files[req.EntryPoint]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entry point file not found"})
		return
	}
	
	// Create execution request
	execReq := ExecutionRequest{
		ID:           uuid.New().String(),
		Language:     req.Language,
		Code:         entryContent,
		Files:        req.Files,
		Dependencies: req.Dependencies,
		Command:      req.Command,
		Timeout:      req.Timeout,
	}
	
	// Get runtime
	runtime, exists := runtimes[strings.ToLower(req.Language)]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported language"})
		return
	}
	
	// Create result
	result := &ExecutionResult{
		ID:        execReq.ID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	
	// Store and execute
	executions.Store(execReq.ID, result)
	go executeCode(execReq, runtime, result)
	
	c.JSON(http.StatusAccepted, gin.H{
		"id":      execReq.ID,
		"status":  "running",
		"message": "Project execution started",
	})
}