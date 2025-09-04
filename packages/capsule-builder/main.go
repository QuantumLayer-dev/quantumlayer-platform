package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BuildRequest represents a request to build a structured capsule
type BuildRequest struct {
	WorkflowID   string                 `json:"workflow_id" binding:"required"`
	Language     string                 `json:"language" binding:"required"`
	Framework    string                 `json:"framework,omitempty"`
	Type         string                 `json:"type" binding:"required"` // api, web, cli, library
	Name         string                 `json:"name" binding:"required"`
	Description  string                 `json:"description,omitempty"`
	Code         string                 `json:"code" binding:"required"`
	Tests        string                 `json:"tests,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// StructuredCapsule represents a fully organized project
type StructuredCapsule struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	Name        string                 `json:"name"`
	Language    string                 `json:"language"`
	Framework   string                 `json:"framework"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Structure   map[string]FileContent `json:"structure"`
	Metadata    CapsuleMetadata        `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	Size        int64                  `json:"size"`
}

// FileContent represents a file in the capsule
type FileContent struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	Type        string `json:"type"` // source, test, config, doc, asset
	Executable  bool   `json:"executable,omitempty"`
	Description string `json:"description,omitempty"`
}

// CapsuleMetadata contains capsule metadata
type CapsuleMetadata struct {
	Version      string            `json:"version"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	Repository   string            `json:"repository,omitempty"`
	Keywords     []string          `json:"keywords,omitempty"`
	Scripts      map[string]string `json:"scripts,omitempty"`
	Dependencies []string          `json:"dependencies"`
	DevDeps      []string          `json:"dev_dependencies,omitempty"`
	BuildCommand string            `json:"build_command,omitempty"`
	StartCommand string            `json:"start_command,omitempty"`
	TestCommand  string            `json:"test_command,omitempty"`
}

// ProjectTemplate defines language-specific project structures
type ProjectTemplate struct {
	Language  string
	Framework string
	Type      string
	Files     []FileTemplate
}

// FileTemplate defines a template file
type FileTemplate struct {
	Path       string
	Template   string
	Type       string
	Executable bool
}

var (
	// Storage for built capsules (in production, use S3/MinIO)
	capsuleStorage = make(map[string]*StructuredCapsule)
)

func main() {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API endpoints
	v1 := r.Group("/api/v1")
	{
		// Build structured capsule from drops
		v1.POST("/build", handleBuildCapsule)
		
		// Get capsule structure
		v1.GET("/capsules/:id", handleGetCapsule)
		
		// Download capsule as tar.gz
		v1.GET("/capsules/:id/download", handleDownloadCapsule)
		
		// Get file from capsule
		v1.GET("/capsules/:id/files/*path", handleGetFile)
		
		// List available templates
		v1.GET("/templates", handleListTemplates)
		
		// Preview capsule structure (without building)
		v1.POST("/preview", handlePreviewStructure)
		
		// Build from workflow result
		v1.POST("/build-from-workflow", handleBuildFromWorkflow)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8092"
	}

	log.Printf("Starting Capsule Builder on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func handleBuildCapsule(c *gin.Context) {
	var req BuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate capsule ID
	capsuleID := fmt.Sprintf("capsule-%s", uuid.New().String())

	// Build structured capsule
	capsule := buildStructuredCapsule(capsuleID, req)

	// Store capsule
	capsuleStorage[capsuleID] = capsule

	c.JSON(http.StatusCreated, capsule)
}

func buildStructuredCapsule(id string, req BuildRequest) *StructuredCapsule {
	structure := make(map[string]FileContent)
	
	// Get template for the language/framework/type combination
	template := getProjectTemplate(req.Language, req.Framework, req.Type)
	
	// Apply template to generate structure
	for _, file := range template.Files {
		content := generateFileContent(file, req)
		structure[file.Path] = FileContent{
			Path:       file.Path,
			Content:    content,
			Type:       file.Type,
			Executable: file.Executable,
		}
	}

	// Add main code file
	mainFile := getMainFilePath(req.Language, req.Type)
	structure[mainFile] = FileContent{
		Path:    mainFile,
		Content: req.Code,
		Type:    "source",
	}

	// Add test file if provided
	if req.Tests != "" {
		testFile := getTestFilePath(req.Language)
		structure[testFile] = FileContent{
			Path:    testFile,
			Content: req.Tests,
			Type:    "test",
		}
	}

	// Create metadata
	metadata := CapsuleMetadata{
		Version:      "1.0.0",
		Author:       "QuantumLayer Platform",
		License:      "MIT",
		Dependencies: req.Dependencies,
		Scripts:      getScripts(req.Language, req.Type),
		BuildCommand: getBuildCommand(req.Language, req.Framework),
		StartCommand: getStartCommand(req.Language, req.Type),
		TestCommand:  getTestCommand(req.Language),
	}

	// Calculate total size
	var totalSize int64
	for _, file := range structure {
		totalSize += int64(len(file.Content))
	}

	return &StructuredCapsule{
		ID:          id,
		WorkflowID:  req.WorkflowID,
		Name:        req.Name,
		Language:    req.Language,
		Framework:   req.Framework,
		Type:        req.Type,
		Description: req.Description,
		Structure:   structure,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
		Size:        totalSize,
	}
}

func getProjectTemplate(language, framework, projectType string) ProjectTemplate {
	// Define templates based on language/framework/type
	switch strings.ToLower(language) {
	case "python":
		return getPythonTemplate(framework, projectType)
	case "javascript", "typescript":
		return getNodeTemplate(language, framework, projectType)
	case "go":
		return getGoTemplate(framework, projectType)
	case "java":
		return getJavaTemplate(framework, projectType)
	default:
		return getDefaultTemplate(language, projectType)
	}
}

func getPythonTemplate(framework, projectType string) ProjectTemplate {
	files := []FileTemplate{
		{
			Path:     "README.md",
			Template: readmeTemplate,
			Type:     "doc",
		},
		{
			Path:     "requirements.txt",
			Template: "{{range .Dependencies}}{{.}}\n{{end}}",
			Type:     "config",
		},
		{
			Path:     ".gitignore",
			Template: pythonGitignore,
			Type:     "config",
		},
		{
			Path:     "Dockerfile",
			Template: pythonDockerfile,
			Type:     "config",
		},
		{
			Path:     ".env.example",
			Template: envTemplate,
			Type:     "config",
		},
	}

	if framework == "fastapi" && projectType == "api" {
		files = append(files, FileTemplate{
			Path:     "app/__init__.py",
			Template: "",
			Type:     "source",
		})
		files = append(files, FileTemplate{
			Path:     "app/models.py",
			Template: pythonModelsTemplate,
			Type:     "source",
		})
		files = append(files, FileTemplate{
			Path:     "app/routes.py",
			Template: pythonRoutesTemplate,
			Type:     "source",
		})
	}

	if projectType == "cli" {
		files = append(files, FileTemplate{
			Path:       "run.sh",
			Template:   "#!/bin/bash\npython main.py \"$@\"",
			Type:       "config",
			Executable: true,
		})
	}

	return ProjectTemplate{
		Language:  "python",
		Framework: framework,
		Type:      projectType,
		Files:     files,
	}
}

func getNodeTemplate(language, framework, projectType string) ProjectTemplate {
	files := []FileTemplate{
		{
			Path:     "README.md",
			Template: readmeTemplate,
			Type:     "doc",
		},
		{
			Path:     "package.json",
			Template: packageJSONTemplate,
			Type:     "config",
		},
		{
			Path:     ".gitignore",
			Template: nodeGitignore,
			Type:     "config",
		},
		{
			Path:     "Dockerfile",
			Template: nodeDockerfile,
			Type:     "config",
		},
		{
			Path:     ".env.example",
			Template: envTemplate,
			Type:     "config",
		},
	}

	if language == "typescript" {
		files = append(files, FileTemplate{
			Path:     "tsconfig.json",
			Template: tsConfigTemplate,
			Type:     "config",
		})
	}

	if framework == "express" && projectType == "api" {
		files = append(files, FileTemplate{
			Path:     "src/routes/index.js",
			Template: expressRoutesTemplate,
			Type:     "source",
		})
		files = append(files, FileTemplate{
			Path:     "src/middleware/auth.js",
			Template: expressMiddlewareTemplate,
			Type:     "source",
		})
	}

	if framework == "react" && projectType == "web" {
		files = append(files, FileTemplate{
			Path:     "public/index.html",
			Template: reactIndexHTML,
			Type:     "asset",
		})
		files = append(files, FileTemplate{
			Path:     "src/App.jsx",
			Template: reactAppTemplate,
			Type:     "source",
		})
		files = append(files, FileTemplate{
			Path:     "src/index.css",
			Template: reactStylesTemplate,
			Type:     "asset",
		})
	}

	return ProjectTemplate{
		Language:  language,
		Framework: framework,
		Type:      projectType,
		Files:     files,
	}
}

func getGoTemplate(framework, projectType string) ProjectTemplate {
	files := []FileTemplate{
		{
			Path:     "README.md",
			Template: readmeTemplate,
			Type:     "doc",
		},
		{
			Path:     "go.mod",
			Template: goModTemplate,
			Type:     "config",
		},
		{
			Path:     ".gitignore",
			Template: goGitignore,
			Type:     "config",
		},
		{
			Path:     "Dockerfile",
			Template: goDockerfile,
			Type:     "config",
		},
		{
			Path:     "Makefile",
			Template: goMakefile,
			Type:     "config",
		},
	}

	if framework == "gin" && projectType == "api" {
		files = append(files, FileTemplate{
			Path:     "handlers/handlers.go",
			Template: goHandlersTemplate,
			Type:     "source",
		})
		files = append(files, FileTemplate{
			Path:     "models/models.go",
			Template: goModelsTemplate,
			Type:     "source",
		})
		files = append(files, FileTemplate{
			Path:     "middleware/middleware.go",
			Template: goMiddlewareTemplate,
			Type:     "source",
		})
	}

	return ProjectTemplate{
		Language:  "go",
		Framework: framework,
		Type:      projectType,
		Files:     files,
	}
}

func getJavaTemplate(framework, projectType string) ProjectTemplate {
	files := []FileTemplate{
		{
			Path:     "README.md",
			Template: readmeTemplate,
			Type:     "doc",
		},
		{
			Path:     ".gitignore",
			Template: javaGitignore,
			Type:     "config",
		},
		{
			Path:     "Dockerfile",
			Template: javaDockerfile,
			Type:     "config",
		},
	}

	if framework == "spring" {
		files = append(files, FileTemplate{
			Path:     "pom.xml",
			Template: springPomTemplate,
			Type:     "config",
		})
		files = append(files, FileTemplate{
			Path:     "src/main/resources/application.properties",
			Template: springPropertiesTemplate,
			Type:     "config",
		})
	} else {
		files = append(files, FileTemplate{
			Path:     "build.gradle",
			Template: gradleTemplate,
			Type:     "config",
		})
	}

	return ProjectTemplate{
		Language:  "java",
		Framework: framework,
		Type:      projectType,
		Files:     files,
	}
}

func getDefaultTemplate(language, projectType string) ProjectTemplate {
	return ProjectTemplate{
		Language: language,
		Type:     projectType,
		Files: []FileTemplate{
			{
				Path:     "README.md",
				Template: readmeTemplate,
				Type:     "doc",
			},
			{
				Path:     ".gitignore",
				Template: defaultGitignore,
				Type:     "config",
			},
			{
				Path:     "Dockerfile",
				Template: defaultDockerfile,
				Type:     "config",
			},
		},
	}
}

func generateFileContent(file FileTemplate, req BuildRequest) string {
	tmpl, err := template.New("file").Parse(file.Template)
	if err != nil {
		return file.Template // Return raw template if parsing fails
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Name":         req.Name,
		"Description":  req.Description,
		"Language":     req.Language,
		"Framework":    req.Framework,
		"Type":         req.Type,
		"Dependencies": req.Dependencies,
		"Metadata":     req.Metadata,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return file.Template // Return raw template if execution fails
	}

	return buf.String()
}

func getMainFilePath(language, projectType string) string {
	switch strings.ToLower(language) {
	case "python":
		if projectType == "library" {
			return "src/__init__.py"
		}
		return "main.py"
	case "javascript":
		if projectType == "web" {
			return "src/index.js"
		}
		return "index.js"
	case "typescript":
		if projectType == "web" {
			return "src/index.ts"
		}
		return "index.ts"
	case "go":
		return "main.go"
	case "java":
		return "src/main/java/Main.java"
	case "rust":
		return "src/main.rs"
	case "ruby":
		return "main.rb"
	case "php":
		return "index.php"
	default:
		return "main." + strings.ToLower(language)
	}
}

func getTestFilePath(language string) string {
	switch strings.ToLower(language) {
	case "python":
		return "tests/test_main.py"
	case "javascript":
		return "tests/main.test.js"
	case "typescript":
		return "tests/main.test.ts"
	case "go":
		return "main_test.go"
	case "java":
		return "src/test/java/MainTest.java"
	case "rust":
		return "src/tests.rs"
	default:
		return "test_main." + strings.ToLower(language)
	}
}

func getScripts(language, projectType string) map[string]string {
	switch strings.ToLower(language) {
	case "javascript", "typescript":
		scripts := map[string]string{
			"start": "node index.js",
			"test":  "jest",
			"dev":   "nodemon index.js",
		}
		if projectType == "web" {
			scripts["build"] = "webpack --mode production"
			scripts["start"] = "webpack-dev-server --open"
		}
		return scripts
	case "python":
		return map[string]string{
			"start": "python main.py",
			"test":  "pytest",
			"lint":  "pylint main.py",
		}
	default:
		return map[string]string{}
	}
}

func getBuildCommand(language, framework string) string {
	switch strings.ToLower(language) {
	case "go":
		return "go build -o app"
	case "java":
		if framework == "spring" {
			return "mvn clean package"
		}
		return "gradle build"
	case "typescript":
		return "tsc"
	case "rust":
		return "cargo build --release"
	default:
		return ""
	}
}

func getStartCommand(language, projectType string) string {
	switch strings.ToLower(language) {
	case "python":
		if projectType == "api" {
			return "uvicorn main:app --reload"
		}
		return "python main.py"
	case "javascript", "typescript":
		if projectType == "api" {
			return "node index.js"
		}
		return "npm start"
	case "go":
		return "./app"
	case "java":
		return "java -jar target/app.jar"
	default:
		return ""
	}
}

func getTestCommand(language string) string {
	switch strings.ToLower(language) {
	case "python":
		return "pytest"
	case "javascript", "typescript":
		return "npm test"
	case "go":
		return "go test ./..."
	case "java":
		return "mvn test"
	case "rust":
		return "cargo test"
	default:
		return ""
	}
}

func handleGetCapsule(c *gin.Context) {
	id := c.Param("id")

	capsule, exists := capsuleStorage[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "capsule not found"})
		return
	}

	c.JSON(http.StatusOK, capsule)
}

func handleDownloadCapsule(c *gin.Context) {
	id := c.Param("id")

	capsule, exists := capsuleStorage[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "capsule not found"})
		return
	}

	// Create tar.gz archive
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	// Add all files to archive
	for path, file := range capsule.Structure {
		header := &tar.Header{
			Name:    path,
			Mode:    0644,
			Size:    int64(len(file.Content)),
			ModTime: capsule.CreatedAt,
		}

		if file.Executable {
			header.Mode = 0755
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write tar header"})
			return
		}

		if _, err := tarWriter.Write([]byte(file.Content)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write tar content"})
			return
		}
	}

	// Add metadata file
	metadataJSON, _ := json.MarshalIndent(capsule.Metadata, "", "  ")
	metadataHeader := &tar.Header{
		Name:    ".quantum/metadata.json",
		Mode:    0644,
		Size:    int64(len(metadataJSON)),
		ModTime: capsule.CreatedAt,
	}

	tarWriter.WriteHeader(metadataHeader)
	tarWriter.Write(metadataJSON)

	tarWriter.Close()
	gzipWriter.Close()

	// Send file
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.tar.gz", capsule.Name))
	c.Data(http.StatusOK, "application/gzip", buf.Bytes())
}

func handleGetFile(c *gin.Context) {
	id := c.Param("id")
	filePath := c.Param("path")

	capsule, exists := capsuleStorage[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "capsule not found"})
		return
	}

	file, exists := capsule.Structure[strings.TrimPrefix(filePath, "/")]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.String(http.StatusOK, file.Content)
}

func handleListTemplates(c *gin.Context) {
	templates := []map[string]interface{}{
		{
			"language":  "python",
			"framework": "fastapi",
			"type":      "api",
			"name":      "Python FastAPI REST API",
		},
		{
			"language":  "python",
			"framework": "flask",
			"type":      "web",
			"name":      "Python Flask Web App",
		},
		{
			"language":  "javascript",
			"framework": "express",
			"type":      "api",
			"name":      "Node.js Express API",
		},
		{
			"language":  "javascript",
			"framework": "react",
			"type":      "web",
			"name":      "React Web Application",
		},
		{
			"language":  "typescript",
			"framework": "express",
			"type":      "api",
			"name":      "TypeScript Express API",
		},
		{
			"language":  "go",
			"framework": "gin",
			"type":      "api",
			"name":      "Go Gin REST API",
		},
		{
			"language":  "java",
			"framework": "spring",
			"type":      "api",
			"name":      "Spring Boot API",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"total":     len(templates),
	})
}

func handlePreviewStructure(c *gin.Context) {
	var req BuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get template
	template := getProjectTemplate(req.Language, req.Framework, req.Type)

	// Build file list
	files := make([]map[string]interface{}, 0, len(template.Files)+2)

	// Add template files
	for _, file := range template.Files {
		files = append(files, map[string]interface{}{
			"path": file.Path,
			"type": file.Type,
			"size": len(generateFileContent(file, req)),
		})
	}

	// Add main code file
	mainFile := getMainFilePath(req.Language, req.Type)
	files = append(files, map[string]interface{}{
		"path": mainFile,
		"type": "source",
		"size": len(req.Code),
	})

	// Add test file if provided
	if req.Tests != "" {
		testFile := getTestFilePath(req.Language)
		files = append(files, map[string]interface{}{
			"path": testFile,
			"type": "test",
			"size": len(req.Tests),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"name":      req.Name,
		"language":  req.Language,
		"framework": req.Framework,
		"type":      req.Type,
		"files":     files,
		"total":     len(files),
	})
}

func handleBuildFromWorkflow(c *gin.Context) {
	var req struct {
		WorkflowID string `json:"workflow_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch workflow result from QuantumDrops service
	dropsURL := os.Getenv("QUANTUM_DROPS_URL")
	if dropsURL == "" {
		dropsURL = "http://quantum-drops.quantumlayer.svc.cluster.local:8090"
	}

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/workflows/%s/drops", dropsURL, req.WorkflowID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch workflow drops"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow drops not found"})
		return
	}

	// Parse drops
	var drops struct {
		Drops []struct {
			Type     string `json:"type"`
			Artifact string `json:"artifact"`
			Stage    string `json:"stage"`
		} `json:"drops"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&drops); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse drops"})
		return
	}

	// Extract code and test drops
	var code, tests string
	var language, framework, projectType string

	for _, drop := range drops.Drops {
		switch drop.Type {
		case "code":
			code = drop.Artifact
		case "tests":
			tests = drop.Artifact
		case "frd":
			// Parse FRD for project details
			// TODO: Extract language, framework, type from FRD
		}
	}

	// Build request from drops
	buildReq := BuildRequest{
		WorkflowID:  req.WorkflowID,
		Language:    language,
		Framework:   framework,
		Type:        projectType,
		Name:        fmt.Sprintf("project-%s", req.WorkflowID),
		Code:        code,
		Tests:       tests,
	}

	// Build capsule
	capsuleID := fmt.Sprintf("capsule-%s", uuid.New().String())
	capsule := buildStructuredCapsule(capsuleID, buildReq)

	// Store capsule
	capsuleStorage[capsuleID] = capsule

	c.JSON(http.StatusCreated, capsule)
}

// Template strings (simplified versions - in production, use embedded files)
const (
	readmeTemplate = `# {{.Name}}

{{.Description}}

## Installation

\` + "``bash" + `
# Install dependencies
{{if eq .Language "python"}}pip install -r requirements.txt{{end}}
{{if eq .Language "javascript"}}npm install{{end}}
{{if eq .Language "go"}}go mod download{{end}}
\` + "``" + `

## Usage

\` + "``bash" + `
# Run the application
{{if eq .Language "python"}}python main.py{{end}}
{{if eq .Language "javascript"}}npm start{{end}}
{{if eq .Language "go"}}go run main.go{{end}}
\` + "``" + `

## Testing

\` + "``bash" + `
# Run tests
{{if eq .Language "python"}}pytest{{end}}
{{if eq .Language "javascript"}}npm test{{end}}
{{if eq .Language "go"}}go test ./...{{end}}
\` + "``" + `

## License

MIT`

	pythonDockerfile = `FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["python", "main.py"]`

	nodeDockerfile = `FROM node:18-alpine

WORKDIR /app

COPY package*.json ./
RUN npm ci --only=production

COPY . .

EXPOSE 3000
CMD ["node", "index.js"]`

	goDockerfile = `FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/app .
CMD ["./app"]`

	javaDockerfile = `FROM openjdk:17-alpine AS builder

WORKDIR /app
COPY . .
RUN ./gradlew build

FROM openjdk:17-alpine
COPY --from=builder /app/build/libs/*.jar app.jar
CMD ["java", "-jar", "app.jar"]`

	defaultDockerfile = `FROM alpine:latest
WORKDIR /app
COPY . .
CMD ["./run.sh"]`

	pythonGitignore = `__pycache__/
*.py[cod]
*$py.class
*.so
.Python
env/
venv/
.venv
.env
*.egg-info/
dist/
build/`

	nodeGitignore = `node_modules/
.env
.env.local
npm-debug.log*
yarn-debug.log*
dist/
build/
*.log
.DS_Store`

	goGitignore = `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
vendor/
.env`

	javaGitignore = `*.class
*.log
*.jar
*.war
*.ear
target/
build/
.gradle/
.idea/
*.iml`

	defaultGitignore = `.env
*.log
*.tmp
.DS_Store`

	envTemplate = `# Environment Variables
PORT=8080
DATABASE_URL=
API_KEY=
DEBUG=false`

	packageJSONTemplate = `{
  "name": "{{.Name}}",
  "version": "1.0.0",
  "description": "{{.Description}}",
  "main": "index.js",
  "scripts": {
    "start": "node index.js",
    "test": "jest",
    "dev": "nodemon index.js"
  },
  "dependencies": {
    {{range $i, $dep := .Dependencies}}{{if $i}},{{end}}
    "{{$dep}}": "latest"{{end}}
  },
  "devDependencies": {
    "jest": "^29.0.0",
    "nodemon": "^3.0.0"
  }
}`

	tsConfigTemplate = `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "lib": ["ES2020"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}`

	goModTemplate = `module {{.Name}}

go 1.21

require (
	{{range .Dependencies}}{{.}}
	{{end}}
)`

	goMakefile = `.PHONY: build run test clean

build:
	go build -o app

run:
	go run main.go

test:
	go test ./...

clean:
	rm -f app`

	springPomTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    
    <groupId>com.quantumlayer</groupId>
    <artifactId>{{.Name}}</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>
    
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
    </parent>
    
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
    </dependencies>
</project>`

	gradleTemplate = `plugins {
    id 'java'
    id 'application'
}

group = 'com.quantumlayer'
version = '1.0.0'

repositories {
    mavenCentral()
}

dependencies {
    {{range .Dependencies}}implementation '{{.}}'
    {{end}}
}

application {
    mainClass = 'Main'
}`

	springPropertiesTemplate = `server.port=8080
spring.application.name={{.Name}}
logging.level.root=INFO`

	pythonModelsTemplate = `from pydantic import BaseModel
from typing import Optional
from datetime import datetime

class Item(BaseModel):
    id: Optional[int] = None
    name: str
    description: Optional[str] = None
    created_at: Optional[datetime] = None`

	pythonRoutesTemplate = `from fastapi import APIRouter, HTTPException
from typing import List

router = APIRouter()

@router.get("/health")
def health_check():
    return {"status": "healthy"}

@router.get("/items")
def get_items():
    return {"items": []}`

	expressRoutesTemplate = `const express = require('express');
const router = express.Router();

router.get('/health', (req, res) => {
    res.json({ status: 'healthy' });
});

router.get('/items', (req, res) => {
    res.json({ items: [] });
});

module.exports = router;`

	expressMiddlewareTemplate = `module.exports = {
    authenticate: (req, res, next) => {
        // Authentication logic here
        next();
    },
    
    errorHandler: (err, req, res, next) => {
        console.error(err.stack);
        res.status(500).json({ error: 'Internal Server Error' });
    }
};`

	reactIndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Name}}</title>
</head>
<body>
    <div id="root"></div>
</body>
</html>`

	reactAppTemplate = `import React from 'react';
import './App.css';

function App() {
    return (
        <div className="App">
            <h1>{{.Name}}</h1>
            <p>{{.Description}}</p>
        </div>
    );
}

export default App;`

	reactStylesTemplate = `* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    background-color: #f5f5f5;
}

.App {
    text-align: center;
    padding: 2rem;
}`

	goHandlersTemplate = `package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}`

	goModelsTemplate = `package models

import "time"

type Item struct {
    ID          uint      ` + "`json:\"id\"`" + `
    Name        string    ` + "`json:\"name\"`" + `
    Description string    ` + "`json:\"description\"`" + `
    CreatedAt   time.Time ` + "`json:\"created_at\"`" + `
}`

	goMiddlewareTemplate = `package middleware

import (
    "github.com/gin-gonic/gin"
)

func Authentication() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Authentication logic here
        c.Next()
    }
}`
)