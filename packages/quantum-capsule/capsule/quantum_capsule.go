package quantumcapsule

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// QuantumCapsule represents a self-contained package of generated code
type QuantumCapsule struct {
	ID              string                 `json:"id"`
	WorkflowID      string                 `json:"workflow_id"`
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description"`
	CreatedAt       time.Time              `json:"created_at"`
	Language        string                 `json:"language"`
	Framework       string                 `json:"framework"`
	Files           []CapsuleFile          `json:"files"`
	Dependencies    []string               `json:"dependencies"`
	TestResults     TestResults            `json:"test_results,omitempty"`
	SecurityReport  SecurityReport         `json:"security_report,omitempty"`
	QuantumDrops    []string               `json:"quantum_drops"` // IDs of associated QuantumDrops
	Metadata        map[string]interface{} `json:"metadata"`
	Checksum        string                 `json:"checksum"`
	Size            int64                  `json:"size"`
}

// CapsuleFile represents a file within the capsule
type CapsuleFile struct {
	Path        string    `json:"path"`
	Content     string    `json:"content"`
	Mode        int       `json:"mode"`
	Size        int64     `json:"size"`
	Hash        string    `json:"hash"`
	Type        string    `json:"type"` // source, test, config, doc
	LastModified time.Time `json:"last_modified"`
}

// TestResults for the capsule
type TestResults struct {
	Passed       int      `json:"passed"`
	Failed       int      `json:"failed"`
	Skipped      int      `json:"skipped"`
	Coverage     float64  `json:"coverage"`
	TestOutput   string   `json:"test_output"`
	TestDuration string   `json:"test_duration"`
}

// SecurityReport for the capsule
type SecurityReport struct {
	Score           float64         `json:"score"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	ScanDate        time.Time       `json:"scan_date"`
	Scanner         string          `json:"scanner"`
}

// Vulnerability details
type Vulnerability struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"` // critical, high, medium, low
	Title       string `json:"title"`
	Description string `json:"description"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Fix         string `json:"fix,omitempty"`
}

// CapsuleManifest describes the capsule contents
type CapsuleManifest struct {
	Version      string            `json:"version"`
	Created      time.Time         `json:"created"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	EntryPoint   string            `json:"entry_point"`
	BuildCommand string            `json:"build_command"`
	RunCommand   string            `json:"run_command"`
	TestCommand  string            `json:"test_command"`
	Environment  map[string]string `json:"environment"`
	Requirements Requirements      `json:"requirements"`
}

// Requirements for running the capsule
type Requirements struct {
	Runtime     string   `json:"runtime"`
	MinVersion  string   `json:"min_version"`
	MaxVersion  string   `json:"max_version,omitempty"`
	SystemDeps  []string `json:"system_deps,omitempty"`
	Services    []string `json:"services,omitempty"` // e.g., postgres, redis
}

// CreateCapsule creates a new QuantumCapsule from generated files
func CreateCapsule(workflowID string, files []CapsuleFile, metadata map[string]interface{}) (*QuantumCapsule, error) {
	capsule := &QuantumCapsule{
		ID:         fmt.Sprintf("capsule-%s-%d", workflowID, time.Now().Unix()),
		WorkflowID: workflowID,
		CreatedAt:  time.Now(),
		Files:      files,
		Metadata:   metadata,
		Version:    "1.0.0",
	}

	// Extract metadata
	if name, ok := metadata["project_name"].(string); ok {
		capsule.Name = name
	}
	if lang, ok := metadata["language"].(string); ok {
		capsule.Language = lang
	}
	if fw, ok := metadata["framework"].(string); ok {
		capsule.Framework = fw
	}
	if desc, ok := metadata["description"].(string); ok {
		capsule.Description = desc
	}
	if deps, ok := metadata["dependencies"].([]string); ok {
		capsule.Dependencies = deps
	}

	// Calculate size
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}
	capsule.Size = totalSize

	return capsule, nil
}

// PackageAsTarGz packages the capsule as a compressed tar archive
func (c *QuantumCapsule) PackageAsTarGz() ([]byte, error) {
	var buf bytes.Buffer
	
	// Create gzip writer
	gzipWriter := gzip.NewWriter(&buf)
	defer gzipWriter.Close()
	
	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Add manifest file
	manifest := CapsuleManifest{
		Version:    c.Version,
		Created:    c.CreatedAt,
		Author:     "QuantumLayer Platform",
		License:    "MIT",
		EntryPoint: getEntryPoint(c.Language, c.Framework),
		RunCommand: getRunCommand(c.Language, c.Framework),
		TestCommand: getTestCommand(c.Language),
		Requirements: Requirements{
			Runtime: c.Language,
		},
	}
	
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write manifest to tar
	manifestHeader := &tar.Header{
		Name:    "QUANTUM_MANIFEST.json",
		Mode:    0644,
		Size:    int64(len(manifestJSON)),
		ModTime: time.Now(),
	}
	
	if err := tarWriter.WriteHeader(manifestHeader); err != nil {
		return nil, fmt.Errorf("failed to write manifest header: %w", err)
	}
	
	if _, err := tarWriter.Write(manifestJSON); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	// Add all files to tar
	for _, file := range c.Files {
		header := &tar.Header{
			Name:    file.Path,
			Mode:    int64(file.Mode),
			Size:    int64(len(file.Content)),
			ModTime: file.LastModified,
		}
		
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("failed to write header for %s: %w", file.Path, err)
		}
		
		if _, err := io.WriteString(tarWriter, file.Content); err != nil {
			return nil, fmt.Errorf("failed to write content for %s: %w", file.Path, err)
		}
	}

	// Add metadata file
	metadataJSON, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataHeader := &tar.Header{
		Name:    "QUANTUM_CAPSULE.json",
		Mode:    0644,
		Size:    int64(len(metadataJSON)),
		ModTime: time.Now(),
	}
	
	if err := tarWriter.WriteHeader(metadataHeader); err != nil {
		return nil, fmt.Errorf("failed to write metadata header: %w", err)
	}
	
	if _, err := tarWriter.Write(metadataJSON); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	return buf.Bytes(), nil
}

// Helper functions
func getEntryPoint(language, framework string) string {
	switch language {
	case "python":
		if framework == "fastapi" {
			return "main.py"
		}
		return "app.py"
	case "javascript", "typescript":
		return "index.js"
	case "go":
		return "main.go"
	case "java":
		return "Main.java"
	default:
		return "main"
	}
}

func getRunCommand(language, framework string) string {
	switch language {
	case "python":
		if framework == "fastapi" {
			return "uvicorn main:app --reload"
		}
		return "python main.py"
	case "javascript", "typescript":
		return "npm start"
	case "go":
		return "go run main.go"
	case "java":
		return "java Main"
	default:
		return "./run.sh"
	}
}

func getTestCommand(language string) string {
	switch language {
	case "python":
		return "pytest"
	case "javascript", "typescript":
		return "npm test"
	case "go":
		return "go test ./..."
	case "java":
		return "mvn test"
	default:
		return "./test.sh"
	}
}

// ValidateCapsule validates the integrity of a capsule
func ValidateCapsule(capsuleData []byte) (*QuantumCapsule, error) {
	// Create gzip reader
	gzipReader, err := gzip.NewReader(bytes.NewReader(capsuleData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	var capsule *QuantumCapsule
	var manifest *CapsuleManifest

	// Read all files from tar
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Read file content
		content := make([]byte, header.Size)
		if _, err := io.ReadFull(tarReader, content); err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", header.Name, err)
		}

		// Parse special files
		switch header.Name {
		case "QUANTUM_CAPSULE.json":
			if err := json.Unmarshal(content, &capsule); err != nil {
				return nil, fmt.Errorf("failed to unmarshal capsule metadata: %w", err)
			}
		case "QUANTUM_MANIFEST.json":
			if err := json.Unmarshal(content, &manifest); err != nil {
				return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
			}
		}
	}

	if capsule == nil {
		return nil, fmt.Errorf("no capsule metadata found")
	}
	if manifest == nil {
		return nil, fmt.Errorf("no manifest found")
	}

	return capsule, nil
}