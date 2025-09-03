package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	capsule "github.com/QuantumLayer-dev/quantumlayer-platform/packages/quantum-capsule"
)

// CapsuleRequest for creating a new capsule
type CapsuleRequest struct {
	WorkflowID string                 `json:"workflow_id" binding:"required"`
	Name       string                 `json:"name" binding:"required"`
	Files      []capsule.CapsuleFile  `json:"files" binding:"required"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Storage for capsules (in-memory for now, should use S3/MinIO in production)
var capsuleStorage = make(map[string]*capsule.QuantumCapsule)

func main() {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Ready check
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// API Routes
	v1 := r.Group("/api/v1")
	{
		// Create a new capsule
		v1.POST("/capsules", handleCreateCapsule)
		
		// Get capsule metadata
		v1.GET("/capsules/:id", handleGetCapsule)
		
		// Download capsule as tar.gz
		v1.GET("/capsules/:id/download", handleDownloadCapsule)
		
		// List all capsules
		v1.GET("/capsules", handleListCapsules)
		
		// Create capsule from workflow result
		v1.POST("/capsules/from-workflow", handleCreateFromWorkflow)
		
		// Validate a capsule
		v1.POST("/capsules/validate", handleValidateCapsule)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	if err := r.Run(":" + port); err != nil {
		panic(err)
	}
}

func handleCreateCapsule(c *gin.Context) {
	var req CapsuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Add metadata
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	req.Metadata["project_name"] = req.Name
	
	// Create the capsule
	cap, err := capsule.CreateCapsule(req.WorkflowID, req.Files, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Store in memory
	capsuleStorage[cap.ID] = cap

	c.JSON(http.StatusCreated, cap)
}

func handleGetCapsule(c *gin.Context) {
	id := c.Param("id")
	
	cap, exists := capsuleStorage[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "capsule not found"})
		return
	}

	c.JSON(http.StatusOK, cap)
}

func handleDownloadCapsule(c *gin.Context) {
	id := c.Param("id")
	
	cap, exists := capsuleStorage[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "capsule not found"})
		return
	}

	// Package as tar.gz
	data, err := cap.PackageAsTarGz()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set headers for download
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.tar.gz", cap.ID))
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	
	// Write the file
	c.Data(http.StatusOK, "application/gzip", data)
}

func handleListCapsules(c *gin.Context) {
	capsules := make([]*capsule.QuantumCapsule, 0, len(capsuleStorage))
	for _, cap := range capsuleStorage {
		capsules = append(capsules, cap)
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(capsules),
		"capsules": capsules,
	})
}

func handleCreateFromWorkflow(c *gin.Context) {
	var req struct {
		WorkflowID string `json:"workflow_id" binding:"required"`
		ResultID   string `json:"result_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch workflow result from workflow API
	workflowAPI := os.Getenv("WORKFLOW_API_URL")
	if workflowAPI == "" {
		workflowAPI = "http://workflow-api.temporal.svc.cluster.local:8080"
	}

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/workflows/%s/result", workflowAPI, req.WorkflowID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch workflow result"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow result not found"})
		return
	}

	// Parse workflow result
	var result struct {
		Files []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Type    string `json:"type"`
		} `json:"files"`
		Metadata struct {
			Language  string `json:"language"`
			Framework string `json:"framework"`
		} `json:"metadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse workflow result"})
		return
	}

	// Convert to capsule files
	files := make([]capsule.CapsuleFile, 0, len(result.Files))
	for _, f := range result.Files {
		files = append(files, capsule.CapsuleFile{
			Path:         f.Path,
			Content:      f.Content,
			Type:         f.Type,
			Mode:         0644,
			Size:         int64(len(f.Content)),
			LastModified: time.Now(),
		})
	}

	// Create metadata
	metadata := map[string]interface{}{
		"workflow_id": req.WorkflowID,
		"result_id":   req.ResultID,
		"language":    result.Metadata.Language,
		"framework":   result.Metadata.Framework,
		"created_at":  time.Now(),
	}

	// Create capsule
	cap, err := capsule.CreateCapsule(req.WorkflowID, files, metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Store
	capsuleStorage[cap.ID] = cap

	c.JSON(http.StatusCreated, cap)
}

func handleValidateCapsule(c *gin.Context) {
	// Read uploaded file
	file, header, err := c.Request.FormFile("capsule")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}
	defer file.Close()

	// Read file content
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// Validate capsule
	cap, err := capsule.ValidateCapsule(buf.Bytes())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":    true,
		"capsule":  cap,
		"filename": header.Filename,
		"size":     header.Size,
	})
}