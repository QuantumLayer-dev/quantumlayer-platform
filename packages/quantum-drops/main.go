package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// QuantumDrop represents an intermediate generation artifact
type QuantumDrop struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	RequestID   string                 `json:"request_id"`
	Stage       string                 `json:"stage"`
	Type        string                 `json:"type"` // prompt, frd, code, tests, etc.
	Artifact    string                 `json:"artifact"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	Version     int                    `json:"version"`
}

// DropCollection represents a collection of drops for a workflow
type DropCollection struct {
	WorkflowID string        `json:"workflow_id"`
	RequestID  string        `json:"request_id"`
	Drops      []QuantumDrop `json:"drops"`
	TotalDrops int           `json:"total_drops"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// DropSummary provides overview of all drops
type DropSummary struct {
	ID         string    `json:"id"`
	Stage      string    `json:"stage"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
	Size       int       `json:"size"`
}

var db *sql.DB

func main() {
	// Initialize database connection
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "postgres-ha.quantumlayer.svc.cluster.local"
	}
	
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "quantumlayer"
	}
	
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		dbPass = "quantum2024"
	}
	
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "quantumdrops"
	}

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbUser, dbPass, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Create tables if not exists
	createTables()

	// Setup Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// QuantumDrops API endpoints
	r.POST("/api/v1/drops", createDrop)
	r.GET("/api/v1/drops/:id", getDrop)
	r.GET("/api/v1/workflows/:workflow_id/drops", getWorkflowDrops)
	r.GET("/api/v1/workflows/:workflow_id/drops/:stage", getDropByStage)
	r.GET("/api/v1/workflows/:workflow_id/summary", getDropsSummary)
	r.POST("/api/v1/workflows/:workflow_id/rollback/:drop_id", rollbackToDrop)
	r.DELETE("/api/v1/drops/:id", deleteDrop)

	// Batch operations
	r.POST("/api/v1/drops/batch", createBatchDrops)
	r.GET("/api/v1/drops/search", searchDrops)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	log.Printf("Starting QuantumDrops service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func createTables() {
	// Create tables
	query := `
	CREATE TABLE IF NOT EXISTS quantum_drops (
		id VARCHAR(255) PRIMARY KEY,
		workflow_id VARCHAR(255) NOT NULL,
		request_id VARCHAR(255) NOT NULL,
		stage VARCHAR(100) NOT NULL,
		type VARCHAR(50) NOT NULL,
		artifact TEXT NOT NULL,
		metadata JSONB,
		version INT DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Warning: Failed to create quantum_drops table: %v", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_workflow_id ON quantum_drops(workflow_id);",
		"CREATE INDEX IF NOT EXISTS idx_request_id ON quantum_drops(request_id);",
		"CREATE INDEX IF NOT EXISTS idx_stage ON quantum_drops(stage);",
		"CREATE INDEX IF NOT EXISTS idx_type ON quantum_drops(type);",
	}

	for _, idx := range indexes {
		_, err := db.Exec(idx)
		if err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}

	// Create collections table
	collectionQuery := `
	CREATE TABLE IF NOT EXISTS drop_collections (
		workflow_id VARCHAR(255) PRIMARY KEY,
		request_id VARCHAR(255) NOT NULL,
		total_drops INT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(collectionQuery)
	if err != nil {
		log.Printf("Warning: Failed to create drop_collections table: %v", err)
	}
}

// API Handlers

func createDrop(c *gin.Context) {
	var drop QuantumDrop
	if err := c.ShouldBindJSON(&drop); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate ID if not provided
	if drop.ID == "" {
		drop.ID = fmt.Sprintf("drop-%s-%s-%d", drop.WorkflowID, drop.Stage, time.Now().Unix())
	}
	drop.CreatedAt = time.Now()

	// Store in database
	metadataJSON, _ := json.Marshal(drop.Metadata)
	query := `INSERT INTO quantum_drops (id, workflow_id, request_id, stage, type, artifact, metadata, version, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := db.Exec(query, drop.ID, drop.WorkflowID, drop.RequestID, drop.Stage, drop.Type, 
		drop.Artifact, metadataJSON, drop.Version, drop.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store drop", "details": err.Error()})
		return
	}

	// Update collection
	updateCollection(drop.WorkflowID, drop.RequestID)

	c.JSON(http.StatusCreated, drop)
}

func getDrop(c *gin.Context) {
	dropID := c.Param("id")

	var drop QuantumDrop
	var metadataJSON []byte
	query := `SELECT id, workflow_id, request_id, stage, type, artifact, metadata, version, created_at
			  FROM quantum_drops WHERE id = $1`
	
	err := db.QueryRow(query, dropID).Scan(&drop.ID, &drop.WorkflowID, &drop.RequestID, 
		&drop.Stage, &drop.Type, &drop.Artifact, &metadataJSON, &drop.Version, &drop.CreatedAt)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Drop not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve drop"})
		return
	}

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &drop.Metadata)
	}

	c.JSON(http.StatusOK, drop)
}

func getWorkflowDrops(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	
	query := `SELECT id, workflow_id, request_id, stage, type, artifact, metadata, version, created_at
			  FROM quantum_drops WHERE workflow_id = $1 ORDER BY created_at ASC`
	
	rows, err := db.Query(query, workflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve drops"})
		return
	}
	defer rows.Close()

	drops := []QuantumDrop{}
	for rows.Next() {
		var drop QuantumDrop
		var metadataJSON []byte
		
		err := rows.Scan(&drop.ID, &drop.WorkflowID, &drop.RequestID, &drop.Stage, 
			&drop.Type, &drop.Artifact, &metadataJSON, &drop.Version, &drop.CreatedAt)
		if err != nil {
			continue
		}
		
		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &drop.Metadata)
		}
		drops = append(drops, drop)
	}

	c.JSON(http.StatusOK, DropCollection{
		WorkflowID: workflowID,
		Drops:      drops,
		TotalDrops: len(drops),
		UpdatedAt:  time.Now(),
	})
}

func getDropByStage(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	stage := c.Param("stage")

	var drop QuantumDrop
	var metadataJSON []byte
	query := `SELECT id, workflow_id, request_id, stage, type, artifact, metadata, version, created_at
			  FROM quantum_drops WHERE workflow_id = $1 AND stage = $2 
			  ORDER BY created_at DESC LIMIT 1`
	
	err := db.QueryRow(query, workflowID, stage).Scan(&drop.ID, &drop.WorkflowID, &drop.RequestID,
		&drop.Stage, &drop.Type, &drop.Artifact, &metadataJSON, &drop.Version, &drop.CreatedAt)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Drop not found for stage"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve drop"})
		return
	}

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &drop.Metadata)
	}

	c.JSON(http.StatusOK, drop)
}

func getDropsSummary(c *gin.Context) {
	workflowID := c.Param("workflow_id")

	query := `SELECT id, stage, type, created_at, LENGTH(artifact) as size
			  FROM quantum_drops WHERE workflow_id = $1 ORDER BY created_at ASC`
	
	rows, err := db.Query(query, workflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve summary"})
		return
	}
	defer rows.Close()

	summaries := []DropSummary{}
	for rows.Next() {
		var summary DropSummary
		err := rows.Scan(&summary.ID, &summary.Stage, &summary.Type, &summary.CreatedAt, &summary.Size)
		if err != nil {
			continue
		}
		summaries = append(summaries, summary)
	}

	c.JSON(http.StatusOK, gin.H{
		"workflow_id": workflowID,
		"total_drops": len(summaries),
		"summaries":   summaries,
	})
}

func rollbackToDrop(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	dropID := c.Param("drop_id")

	// Get the drop to rollback to
	var drop QuantumDrop
	var metadataJSON []byte
	query := `SELECT id, workflow_id, request_id, stage, type, artifact, metadata, version, created_at
			  FROM quantum_drops WHERE id = $1 AND workflow_id = $2`
	
	err := db.QueryRow(query, dropID, workflowID).Scan(&drop.ID, &drop.WorkflowID, &drop.RequestID,
		&drop.Stage, &drop.Type, &drop.Artifact, &metadataJSON, &drop.Version, &drop.CreatedAt)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Drop not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve drop"})
		return
	}

	// Create a new drop representing the rollback
	rollbackDrop := QuantumDrop{
		ID:         fmt.Sprintf("rollback-%s-%d", dropID, time.Now().Unix()),
		WorkflowID: workflowID,
		RequestID:  drop.RequestID,
		Stage:      "rollback",
		Type:       drop.Type,
		Artifact:   drop.Artifact,
		Version:    drop.Version + 1,
		CreatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"rollback_from": dropID,
			"original_stage": drop.Stage,
		},
	}

	// Store rollback drop
	rollbackMetadataJSON, _ := json.Marshal(rollbackDrop.Metadata)
	insertQuery := `INSERT INTO quantum_drops (id, workflow_id, request_id, stage, type, artifact, metadata, version, created_at)
					 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err = db.Exec(insertQuery, rollbackDrop.ID, rollbackDrop.WorkflowID, rollbackDrop.RequestID,
		rollbackDrop.Stage, rollbackDrop.Type, rollbackDrop.Artifact, rollbackMetadataJSON,
		rollbackDrop.Version, rollbackDrop.CreatedAt)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rollback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Rollback successful",
		"rollback_drop": rollbackDrop,
		"original_drop": drop,
	})
}

func deleteDrop(c *gin.Context) {
	dropID := c.Param("id")

	query := `DELETE FROM quantum_drops WHERE id = $1`
	result, err := db.Exec(query, dropID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete drop"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Drop not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Drop deleted successfully"})
}

func createBatchDrops(c *gin.Context) {
	var drops []QuantumDrop
	if err := c.ShouldBindJSON(&drops); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	for _, drop := range drops {
		if drop.ID == "" {
			drop.ID = fmt.Sprintf("drop-%s-%s-%d", drop.WorkflowID, drop.Stage, time.Now().UnixNano())
		}
		drop.CreatedAt = time.Now()

		metadataJSON, _ := json.Marshal(drop.Metadata)
		query := `INSERT INTO quantum_drops (id, workflow_id, request_id, stage, type, artifact, metadata, version, created_at)
				  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		
		_, err := tx.Exec(query, drop.ID, drop.WorkflowID, drop.RequestID, drop.Stage, drop.Type,
			drop.Artifact, metadataJSON, drop.Version, drop.CreatedAt)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store drops", "details": err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Batch drops created successfully",
		"count":   len(drops),
	})
}

func searchDrops(c *gin.Context) {
	stage := c.Query("stage")
	dropType := c.Query("type")
	workflowID := c.Query("workflow_id")
	limit := c.DefaultQuery("limit", "100")

	query := `SELECT id, workflow_id, request_id, stage, type, artifact, metadata, version, created_at
			  FROM quantum_drops WHERE 1=1`
	args := []interface{}{}
	argCount := 0

	if stage != "" {
		argCount++
		query += fmt.Sprintf(" AND stage = $%d", argCount)
		args = append(args, stage)
	}
	if dropType != "" {
		argCount++
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, dropType)
	}
	if workflowID != "" {
		argCount++
		query += fmt.Sprintf(" AND workflow_id = $%d", argCount)
		args = append(args, workflowID)
	}

	query += " ORDER BY created_at DESC"
	argCount++
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search drops"})
		return
	}
	defer rows.Close()

	drops := []QuantumDrop{}
	for rows.Next() {
		var drop QuantumDrop
		var metadataJSON []byte
		
		err := rows.Scan(&drop.ID, &drop.WorkflowID, &drop.RequestID, &drop.Stage,
			&drop.Type, &drop.Artifact, &metadataJSON, &drop.Version, &drop.CreatedAt)
		if err != nil {
			continue
		}
		
		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &drop.Metadata)
		}
		drops = append(drops, drop)
	}

	c.JSON(http.StatusOK, gin.H{
		"results": drops,
		"count":   len(drops),
	})
}

func updateCollection(workflowID, requestID string) {
	query := `INSERT INTO drop_collections (workflow_id, request_id, total_drops, updated_at)
			  VALUES ($1, $2, 1, $3)
			  ON CONFLICT (workflow_id) 
			  DO UPDATE SET total_drops = drop_collections.total_drops + 1, updated_at = $3`
	
	db.Exec(query, workflowID, requestID, time.Now())
}