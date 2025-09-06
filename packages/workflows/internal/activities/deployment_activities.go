package activities

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
)

// DeploymentRequest represents a deployment request
type DeploymentRequest struct {
	WorkflowID   string                 `json:"workflow_id"`
	CapsuleID    string                 `json:"capsule_id"`
	Language     string                 `json:"language"`
	Framework    string                 `json:"framework"`
	Files        map[string]string      `json:"files"`
	Dependencies []string               `json:"dependencies"`
	Environment  map[string]string      `json:"environment"`
	Resources    ContainerResources     `json:"resources"`
	TTLMinutes   int                    `json:"ttl_minutes"`
}

// ContainerResources defines resource requirements
type ContainerResources struct {
	CPU    string `json:"cpu"`    // e.g., "200m"
	Memory string `json:"memory"` // e.g., "256Mi"
}

// ContainerBuildResult represents the result of container building
type ContainerBuildResult struct {
	Success      bool     `json:"success"`
	ImageName    string   `json:"image_name"`
	ImageTag     string   `json:"image_tag"`
	ImageDigest  string   `json:"image_digest,omitempty"`
	BuildTime    float64  `json:"build_time_seconds"`
	ImageSize    int64    `json:"image_size_bytes"`
	Warnings     []string `json:"warnings,omitempty"`
	Message      string   `json:"message"`
}

// KubernetesDeploymentResult represents the result of Kubernetes deployment
type KubernetesDeploymentResult struct {
	Success       bool      `json:"success"`
	DeploymentID  string    `json:"deployment_id"`
	ServiceName   string    `json:"service_name"`
	IngressName   string    `json:"ingress_name"`
	Namespace     string    `json:"namespace"`
	LiveURL       string    `json:"live_url"`
	DashboardURL  string    `json:"dashboard_url"`
	ExpiresAt     time.Time `json:"expires_at"`
	Message       string    `json:"message"`
}

// BuildContainerImageActivity builds a Docker container image from generated code
func BuildContainerImageActivity(ctx context.Context, request DeploymentRequest) (*ContainerBuildResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting container build", "workflow_id", request.WorkflowID, "language", request.Language)

	startTime := time.Now()
	
	// Create temporary directory for build context
	buildDir, err := createBuildContext(request)
	if err != nil {
		return &ContainerBuildResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create build context: %v", err),
		}, nil
	}
	defer os.RemoveAll(buildDir)

	// Generate appropriate Dockerfile
	dockerfile, err := generateIntelligentDockerfile(request)
	if err != nil {
		return &ContainerBuildResult{
			Success: false,
			Message: fmt.Sprintf("Failed to generate Dockerfile: %v", err),
		}, nil
	}

	// Write Dockerfile to build directory
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return &ContainerBuildResult{
			Success: false,
			Message: fmt.Sprintf("Failed to write Dockerfile: %v", err),
		}, nil
	}

	// Build image name and tag
	imageName := fmt.Sprintf("ghcr.io/quantumlayer/apps")
	imageTag := fmt.Sprintf("app-%s", request.WorkflowID[:8])
	fullImageName := fmt.Sprintf("%s:%s", imageName, imageTag)

	// Build Docker image
	buildCmd := exec.CommandContext(ctx, "docker", "build", 
		"--tag", fullImageName,
		"--label", fmt.Sprintf("workflow-id=%s", request.WorkflowID),
		"--label", fmt.Sprintf("language=%s", request.Language),
		"--label", fmt.Sprintf("framework=%s", request.Framework),
		"--label", fmt.Sprintf("created=%s", time.Now().Format(time.RFC3339)),
		buildDir)

	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		logger.Error("Docker build failed", "error", err, "output", string(buildOutput))
		return &ContainerBuildResult{
			Success: false,
			Message: fmt.Sprintf("Docker build failed: %v\nOutput: %s", err, string(buildOutput)),
		}, nil
	}

	// Push image to registry
	pushCmd := exec.CommandContext(ctx, "docker", "push", fullImageName)
	pushOutput, err := pushCmd.CombinedOutput()
	if err != nil {
		logger.Error("Docker push failed", "error", err, "output", string(pushOutput))
		return &ContainerBuildResult{
			Success: false,
			Message: fmt.Sprintf("Docker push failed: %v\nOutput: %s", err, string(pushOutput)),
		}, nil
	}

	// Get image information
	imageSize, _ := getImageSize(fullImageName)
	buildDuration := time.Since(startTime).Seconds()

	logger.Info("Container build completed successfully", 
		"image", fullImageName, 
		"build_time", buildDuration,
		"size_mb", imageSize/1024/1024)

	return &ContainerBuildResult{
		Success:   true,
		ImageName: imageName,
		ImageTag:  imageTag,
		BuildTime: buildDuration,
		ImageSize: imageSize,
		Message:   fmt.Sprintf("Successfully built and pushed %s", fullImageName),
	}, nil
}

// createBuildContext creates the build directory with all necessary files
func createBuildContext(request DeploymentRequest) (string, error) {
	// Create temporary directory
	buildDir, err := os.MkdirTemp("", fmt.Sprintf("build-%s-", request.WorkflowID[:8]))
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Write all files to build directory
	for filePath, content := range request.Files {
		fullPath := filepath.Join(buildDir, filePath)
		
		// Create directory if it doesn't exist
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write file content
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return buildDir, nil
}

// generateIntelligentDockerfile generates an optimized Dockerfile
func generateIntelligentDockerfile(request DeploymentRequest) (string, error) {
	// Auto-detect framework if not provided
	framework := request.Framework
	if framework == "" || framework == "generic" {
		framework = DetectFramework(request.Language, request.Files)
	}

	// Generate Dockerfile using the generator
	dockerfile, err := GenerateDockerfile(request.Language, framework, request.Dependencies, request.Files)
	if err != nil {
		return "", fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	return dockerfile, nil
}

// getImageSize gets the size of a Docker image
func getImageSize(imageName string) (int64, error) {
	cmd := exec.Command("docker", "image", "inspect", imageName, "--format", "{{.Size}}")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var size int64
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &size)
	return size, err
}

// GenerateK8sManifestsActivity generates Kubernetes deployment manifests
func GenerateK8sManifestsActivity(ctx context.Context, request DeploymentRequest, imageTag string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating Kubernetes manifests", "workflow_id", request.WorkflowID)

	// Set resource defaults if not provided
	cpu := request.Resources.CPU
	memory := request.Resources.Memory
	if cpu == "" {
		cpu = "200m"
	}
	if memory == "" {
		memory = "256Mi"
	}

	// Generate unique names
	appName := fmt.Sprintf("app-%s", request.WorkflowID[:8])
	namespace := "quantumlayer-apps"
	
	// Detect port from framework
	port := getFrameworkPort(request.Language, request.Framework)
	
	// Generate Kubernetes YAML manifest
	manifest := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    workflow-id: %s
    language: %s
    framework: %s
    managed-by: quantumlayer
  annotations:
    ttl: "%d"
    created: "%s"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
        workflow-id: %s
    spec:
      containers:
      - name: app
        image: ghcr.io/quantumlayer/apps:%s
        ports:
        - containerPort: %d
          name: http
        env:
%s
        resources:
          requests:
            memory: %s
            cpu: %s
          limits:
            memory: %s
            cpu: %s
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
spec:
  selector:
    app: %s
  ports:
  - port: 80
    targetPort: http
    protocol: TCP
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/cors-allow-origin: "*"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - %s.demo.quantumlayer.io
    secretName: %s-tls
  rules:
  - host: %s.demo.quantumlayer.io
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: 80`,
		namespace, // namespace creation
		appName, namespace, appName, request.WorkflowID, request.Language, request.Framework, request.TTLMinutes, time.Now().Format(time.RFC3339), // deployment metadata
		appName, // deployment selector
		appName, request.WorkflowID, // pod labels
		imageTag, port, // container spec
		generateEnvVars(request.Environment), // environment variables
		memory, cpu, memory, cpu, // resources
		appName, namespace, appName, appName, // service
		appName, namespace, appName, // ingress
		appName, appName, appName, appName) // ingress rules

	return manifest, nil
}

// generateEnvVars generates environment variable YAML
func generateEnvVars(env map[string]string) string {
	if len(env) == 0 {
		return "        - name: NODE_ENV\n          value: production"
	}

	var envVars strings.Builder
	for key, value := range env {
		envVars.WriteString(fmt.Sprintf("        - name: %s\n          value: %q\n", key, value))
	}
	
	return strings.TrimSuffix(envVars.String(), "\n")
}

// getFrameworkPort returns the default port for a framework
func getFrameworkPort(language, framework string) int {
	switch language {
	case "python":
		switch framework {
		case "fastapi", "django":
			return 8000
		case "flask":
			return 5000
		}
	case "javascript", "typescript":
		switch framework {
		case "nextjs", "express", "react":
			return 3000
		}
	case "go":
		return 8080
	case "java":
		return 8080
	}
	
	return 8080 // default
}

// DeployToKubernetesActivity deploys the application to Kubernetes
func DeployToKubernetesActivity(ctx context.Context, request DeploymentRequest, manifest string, imageTag string) (*KubernetesDeploymentResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Deploying to Kubernetes", "workflow_id", request.WorkflowID)

	// Write manifest to temporary file
	manifestFile, err := os.CreateTemp("", fmt.Sprintf("k8s-manifest-%s-*.yaml", request.WorkflowID[:8]))
	if err != nil {
		return &KubernetesDeploymentResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create manifest file: %v", err),
		}, nil
	}
	defer os.Remove(manifestFile.Name())

	if _, err := manifestFile.WriteString(manifest); err != nil {
		return &KubernetesDeploymentResult{
			Success: false,
			Message: fmt.Sprintf("Failed to write manifest: %v", err),
		}, nil
	}
	manifestFile.Close()

	// Apply manifest to Kubernetes
	applyCmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", manifestFile.Name())
	applyOutput, err := applyCmd.CombinedOutput()
	if err != nil {
		logger.Error("Kubernetes deployment failed", "error", err, "output", string(applyOutput))
		return &KubernetesDeploymentResult{
			Success: false,
			Message: fmt.Sprintf("Kubernetes deployment failed: %v\nOutput: %s", err, string(applyOutput)),
		}, nil
	}

	// Generate deployment details
	appName := fmt.Sprintf("app-%s", request.WorkflowID[:8])
	namespace := "quantumlayer-apps"
	liveURL := fmt.Sprintf("https://%s.demo.quantumlayer.io", appName)
	dashboardURL := fmt.Sprintf("https://dashboard.quantumlayer.io/apps/%s", request.WorkflowID)
	
	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(request.TTLMinutes) * time.Minute)
	if request.TTLMinutes == 0 {
		expiresAt = time.Now().Add(60 * time.Minute) // default 1 hour
	}

	logger.Info("Kubernetes deployment completed successfully", 
		"deployment_id", appName, 
		"live_url", liveURL,
		"expires_at", expiresAt)

	return &KubernetesDeploymentResult{
		Success:      true,
		DeploymentID: appName,
		ServiceName:  appName,
		IngressName:  appName,
		Namespace:    namespace,
		LiveURL:      liveURL,
		DashboardURL: dashboardURL,
		ExpiresAt:    expiresAt,
		Message:      fmt.Sprintf("Successfully deployed %s to Kubernetes", appName),
	}, nil
}

// HealthCheckActivity performs health checks on the deployed application
func HealthCheckActivity(ctx context.Context, liveURL string, maxAttempts int) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting health check", "url", liveURL, "max_attempts", maxAttempts)

	if maxAttempts == 0 {
		maxAttempts = 30 // default 5 minutes with 10s intervals
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		logger.Info("Health check attempt", "attempt", attempt, "url", liveURL)
		
		// Try to reach the health endpoint
		healthURL := liveURL + "/health"
		curlCmd := exec.CommandContext(ctx, "curl", "-f", "-s", "--max-time", "10", healthURL)
		
		if err := curlCmd.Run(); err == nil {
			logger.Info("Health check successful", "attempt", attempt, "url", healthURL)
			return true, nil
		}

		// Try the root endpoint as fallback
		curlCmd = exec.CommandContext(ctx, "curl", "-f", "-s", "--max-time", "10", liveURL)
		if err := curlCmd.Run(); err == nil {
			logger.Info("Health check successful (root endpoint)", "attempt", attempt, "url", liveURL)
			return true, nil
		}

		if attempt < maxAttempts {
			time.Sleep(10 * time.Second)
		}
	}

	logger.Warn("Health check failed after all attempts", "url", liveURL, "attempts", maxAttempts)
	return false, nil
}