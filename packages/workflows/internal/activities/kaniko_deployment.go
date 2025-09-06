package activities

import (
	"context"
	"fmt"
	"time"
	"strings"
	"encoding/base64"

	"go.temporal.io/sdk/activity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KanikoDeploymentStrategy implements enterprise-grade Docker-less container builds
type KanikoDeploymentStrategy struct {
	clientset         *kubernetes.Clientset
	registry          string
	namespace         string
	serviceAccount    string
	builderImage      string
	pushSecret        string
}

// KanikoBuildConfig represents Kaniko build configuration
type KanikoBuildConfig struct {
	// Build Context
	ContextConfigMap  string            `json:"context_configmap"`
	DockerfileContent string            `json:"dockerfile_content"`
	BuildArgs         map[string]string `json:"build_args"`
	
	// Registry Configuration  
	RegistryURL       string            `json:"registry_url"`
	ImageName         string            `json:"image_name"`
	ImageTag          string            `json:"image_tag"`
	PushSecret        string            `json:"push_secret"`
	
	// Build Options
	Cache             bool              `json:"cache"`
	CacheRepo         string            `json:"cache_repo"`
	Compression       string            `json:"compression"`
	CleanupContext    bool              `json:"cleanup_context"`
	Verbosity         string            `json:"verbosity"`
	
	// Security & Compliance
	ScanImage         bool              `json:"scan_image"`
	SignImage         bool              `json:"sign_image"`
	SBOMGeneration    bool              `json:"sbom_generation"`
	
	// Resource Limits
	CPULimit          string            `json:"cpu_limit"`
	MemoryLimit       string            `json:"memory_limit"`
	BuildTimeout      time.Duration     `json:"build_timeout"`
}

// NewKanikoDeploymentStrategy creates a new Kaniko deployment strategy
func NewKanikoDeploymentStrategy() (*KanikoDeploymentStrategy, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}
	
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	
	return &KanikoDeploymentStrategy{
		clientset:      clientset,
		registry:       "ghcr.io/quantumlayer-dev",
		namespace:      "quantumlayer-builds",
		serviceAccount: "kaniko-builder",
		builderImage:   "gcr.io/kaniko-project/executor:v1.9.0-debug",
		pushSecret:     "ghcr-push-secret",
	}, nil
}

// ExecuteKanikoBuild performs an enterprise-grade container build using Kaniko
func ExecuteKanikoBuild(ctx context.Context, request DeploymentRequest) (*UniversalDeploymentResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting Kaniko-based enterprise container build",
		"workflow_id", request.WorkflowID,
		"language", request.Language,
		"framework", request.Framework)

	strategy, err := NewKanikoDeploymentStrategy()
	if err != nil {
		return &UniversalDeploymentResult{Success: false}, 
			fmt.Errorf("failed to initialize Kaniko strategy: %w", err)
	}

	// Step 1: Prepare build context and configuration
	buildConfig, err := strategy.prepareBuildConfig(ctx, request)
	if err != nil {
		return &UniversalDeploymentResult{Success: false}, 
			fmt.Errorf("failed to prepare build config: %w", err)
	}

	// Step 2: Create build context ConfigMap
	err = strategy.createBuildContext(ctx, request, buildConfig)
	if err != nil {
		return &UniversalDeploymentResult{Success: false}, 
			fmt.Errorf("failed to create build context: %w", err)
	}

	// Step 3: Execute Kaniko build job
	jobName, err := strategy.executeBuildJob(ctx, buildConfig)
	if err != nil {
		return &UniversalDeploymentResult{Success: false}, 
			fmt.Errorf("failed to execute build job: %w", err)
	}

	// Step 4: Monitor build progress
	buildResult, err := strategy.monitorBuildProgress(ctx, jobName, buildConfig.BuildTimeout)
	if err != nil {
		return &UniversalDeploymentResult{Success: false}, 
			fmt.Errorf("build failed: %w", err)
	}

	// Step 5: Post-build security scanning and signing (if enabled)
	if buildConfig.ScanImage {
		scanResult, err := strategy.scanContainerImage(ctx, buildConfig)
		if err != nil {
			logger.Warn("Container image scan failed", "error", err)
			// Continue - scanning failure shouldn't fail deployment in dev
		} else {
			buildResult.SecurityScan = scanResult
		}
	}

	// Step 6: Generate deployment manifests
	manifests, err := strategy.generateDeploymentManifests(ctx, request, buildConfig)
	if err != nil {
		return buildResult, fmt.Errorf("failed to generate deployment manifests: %w", err)
	}

	// Step 7: Deploy to Kubernetes
	deploymentInfo, err := strategy.deployToKubernetes(ctx, manifests, request)
	if err != nil {
		return buildResult, fmt.Errorf("failed to deploy to Kubernetes: %w", err)
	}

	// Step 8: Setup monitoring and health checks
	err = strategy.setupMonitoringAndHealthChecks(ctx, deploymentInfo)
	if err != nil {
		logger.Warn("Failed to setup monitoring", "error", err)
		// Continue - monitoring setup failure shouldn't fail deployment
	}

	// Step 9: Cleanup build resources (if configured)
	if buildConfig.CleanupContext {
		err = strategy.cleanupBuildResources(ctx, buildConfig.ContextConfigMap, jobName)
		if err != nil {
			logger.Warn("Failed to cleanup build resources", "error", err)
		}
	}

	buildResult.Success = true
	buildResult.Strategy = StrategyKaniko
	buildResult.DeploymentID = deploymentInfo.DeploymentID
	buildResult.LiveURL = deploymentInfo.LiveURL
	buildResult.DashboardURL = deploymentInfo.DashboardURL
	buildResult.HealthURL = deploymentInfo.HealthURL
	buildResult.Provider = "kubernetes-kaniko"
	buildResult.Status = "deployed"

	logger.Info("Kaniko-based deployment completed successfully",
		"image", fmt.Sprintf("%s:%s", buildConfig.ImageName, buildConfig.ImageTag),
		"live_url", buildResult.LiveURL,
		"deployment_id", buildResult.DeploymentID)

	return buildResult, nil
}

// prepareBuildConfig creates the Kaniko build configuration
func (k *KanikoDeploymentStrategy) prepareBuildConfig(ctx context.Context, request DeploymentRequest) (*KanikoBuildConfig, error) {
	// Generate intelligent Dockerfile
	dockerfile, err := generateIntelligentDockerfile(request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	imageName := fmt.Sprintf("%s/apps", k.registry)
	imageTag := fmt.Sprintf("app-%s", request.WorkflowID[:8])
	
	return &KanikoBuildConfig{
		ContextConfigMap:  fmt.Sprintf("build-context-%s", request.WorkflowID[:8]),
		DockerfileContent: dockerfile,
		BuildArgs: map[string]string{
			"BUILDKIT_INLINE_CACHE": "1",
			"WORKSPACE":             "/workspace",
		},
		RegistryURL:    k.registry,
		ImageName:      imageName,
		ImageTag:       imageTag,
		PushSecret:     k.pushSecret,
		Cache:          true,
		CacheRepo:      fmt.Sprintf("%s/cache", k.registry),
		Compression:    "gzip",
		CleanupContext: true,
		Verbosity:      "info",
		ScanImage:      true,
		SignImage:      false, // Enable in production
		SBOMGeneration: true,
		CPULimit:       "2",
		MemoryLimit:    "4Gi",
		BuildTimeout:   10 * time.Minute,
	}, nil
}

// createBuildContext creates a ConfigMap with the build context
func (k *KanikoDeploymentStrategy) createBuildContext(ctx context.Context, request DeploymentRequest, config *KanikoBuildConfig) error {
	logger := activity.GetLogger(ctx)
	
	// Prepare build context data
	contextData := make(map[string]string)
	
	// Add Dockerfile
	contextData["Dockerfile"] = config.DockerfileContent
	
	// Add all application files
	for filePath, content := range request.Files {
		contextData[filePath] = content
	}
	
	// Add dependency files based on language
	if err := k.addDependencyFiles(contextData, request); err != nil {
		return fmt.Errorf("failed to add dependency files: %w", err)
	}
	
	// Create ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.ContextConfigMap,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":         "kaniko-build",
				"workflow-id": request.WorkflowID,
				"language":    request.Language,
				"framework":   request.Framework,
			},
			Annotations: map[string]string{
				"quantumlayer.io/build-id":    request.WorkflowID,
				"quantumlayer.io/created-by":  "kaniko-deployment-strategy",
				"quantumlayer.io/created-at":  time.Now().Format(time.RFC3339),
			},
		},
		Data: contextData,
	}
	
	_, err := k.clientset.CoreV1().ConfigMaps(k.namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create build context ConfigMap: %w", err)
	}
	
	logger.Info("Build context ConfigMap created successfully", 
		"configmap", config.ContextConfigMap,
		"files_count", len(contextData))
	
	return nil
}

// executeBuildJob creates and runs a Kaniko build job
func (k *KanikoDeploymentStrategy) executeBuildJob(ctx context.Context, config *KanikoBuildConfig) (string, error) {
	logger := activity.GetLogger(ctx)
	
	jobName := fmt.Sprintf("kaniko-build-%d", time.Now().Unix())
	
	// Prepare Kaniko arguments
	args := []string{
		fmt.Sprintf("--dockerfile=/workspace/Dockerfile"),
		fmt.Sprintf("--context=/workspace"),
		fmt.Sprintf("--destination=%s:%s", config.ImageName, config.ImageTag),
		fmt.Sprintf("--registry-certificate=%s=false", config.RegistryURL),
		fmt.Sprintf("--verbosity=%s", config.Verbosity),
		fmt.Sprintf("--compression=%s", config.Compression),
	}
	
	if config.Cache {
		args = append(args, fmt.Sprintf("--cache=true"))
		args = append(args, fmt.Sprintf("--cache-repo=%s", config.CacheRepo))
	}
	
	// Add build args
	for key, value := range config.BuildArgs {
		args = append(args, fmt.Sprintf("--build-arg=%s=%s", key, value))
	}
	
	// Create the Kaniko build job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":     "kaniko-build",
				"type":    "container-build",
				"stage":   "build",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: k.serviceAccount,
					Containers: []corev1.Container{
						{
							Name:  "kaniko",
							Image: k.builderImage,
							Args:  args,
							Env: []corev1.EnvVar{
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: "/secret/credentials.json",
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    parseQuantity(config.CPULimit),
									corev1.ResourceMemory: parseQuantity(config.MemoryLimit),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    parseQuantity("500m"),
									corev1.ResourceMemory: parseQuantity("1Gi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "build-context",
									MountPath: "/workspace",
								},
								{
									Name:      "registry-secret",
									MountPath: "/kaniko/.docker",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "build-context",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: config.ContextConfigMap,
									},
								},
							},
						},
						{
							Name: "registry-secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: config.PushSecret,
									Items: []corev1.KeyToPath{
										{
											Key:  ".dockerconfigjson",
											Path: "config.json",
										},
									},
								},
							},
						},
					},
				},
			},
			BackoffLimit: int32Ptr(2),
		},
	}
	
	_, err := k.clientset.BatchV1().Jobs(k.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create Kaniko build job: %w", err)
	}
	
	logger.Info("Kaniko build job created successfully", "job", jobName)
	return jobName, nil
}

// monitorBuildProgress monitors the Kaniko build job progress
func (k *KanikoDeploymentStrategy) monitorBuildProgress(ctx context.Context, jobName string, timeout time.Duration) (*UniversalDeploymentResult, error) {
	logger := activity.GetLogger(ctx)
	
	startTime := time.Now()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()
	
	for {
		select {
		case <-timeoutTimer.C:
			return &UniversalDeploymentResult{Success: false}, 
				fmt.Errorf("build timed out after %v", timeout)
				
		case <-ticker.C:
			job, err := k.clientset.BatchV1().Jobs(k.namespace).Get(ctx, jobName, metav1.GetOptions{})
			if err != nil {
				logger.Warn("Failed to get job status", "error", err)
				continue
			}
			
			if job.Status.CompletionTime != nil {
				// Job completed successfully
				buildDuration := time.Since(startTime)
				logger.Info("Kaniko build completed successfully", 
					"job", jobName,
					"duration", buildDuration)
				
				return &UniversalDeploymentResult{
					Success:   true,
					Strategy:  StrategyKaniko,
					Provider:  "kubernetes-kaniko",
					Status:    "build-completed",
					Metrics: DeploymentMetrics{
						BuildDuration: buildDuration,
						BuildMethod:   "kaniko",
					},
				}, nil
			}
			
			if job.Status.Failed > 0 {
				// Job failed - get logs for debugging
				logs, _ := k.getBuildLogs(ctx, jobName)
				return &UniversalDeploymentResult{Success: false}, 
					fmt.Errorf("build failed: %s", logs)
			}
			
			// Job still running - continue monitoring
			logger.Info("Build in progress", 
				"job", jobName,
				"active", job.Status.Active,
				"duration", time.Since(startTime))
		}
	}
}

// Helper functions
func (k *KanikoDeploymentStrategy) addDependencyFiles(contextData map[string]string, request DeploymentRequest) error {
	switch request.Language {
	case "python":
		// Create requirements.txt from dependencies
		if len(request.Dependencies) > 0 {
			contextData["requirements.txt"] = strings.Join(request.Dependencies, "\n")
		}
	case "javascript", "typescript":
		// Create package.json
		if len(request.Dependencies) > 0 {
			packageJSON := generatePackageJSON(request)
			contextData["package.json"] = packageJSON
		}
	case "go":
		// Go modules handling
		contextData["go.mod"] = generateGoMod(request)
	case "java":
		// Maven pom.xml
		contextData["pom.xml"] = generatePomXML(request)
	}
	return nil
}

func (k *KanikoDeploymentStrategy) getBuildLogs(ctx context.Context, jobName string) (string, error) {
	// Get pod associated with job
	pods, err := k.clientset.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil || len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", jobName)
	}
	
	// Get logs from the first pod
	podName := pods.Items[0].Name
	req := k.clientset.CoreV1().Pods(k.namespace).GetLogs(podName, &corev1.PodLogOptions{})
	
	logs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer logs.Close()
	
	// Read logs (implement proper streaming)
	return "Build logs would be streamed here", nil
}

// Additional helper functions would be implemented here for:
// - generatePackageJSON
// - generateGoMod  
// - generatePomXML
// - parseQuantity
// - scanContainerImage
// - generateDeploymentManifests
// - deployToKubernetes
// - setupMonitoringAndHealthChecks
// - cleanupBuildResources