package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/activities"
	"github.com/QuantumLayer-dev/quantumlayer-platform/packages/workflows/internal/workflows"
)

func main() {
	// Create logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// Get Temporal host from environment
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "temporal-frontend.temporal.svc.cluster.local:7233"
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: "quantumlayer",
		// Use default Temporal logger
	})
	if err != nil {
		logger.Fatal("Unable to create Temporal client", zap.Error(err))
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, workflows.CodeGenerationTaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: 10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Register workflows
	w.RegisterWorkflow(workflows.CodeGenerationWorkflow)
	w.RegisterWorkflow(workflows.ExtendedCodeGenerationWorkflow) // New extended workflow
	w.RegisterWorkflow(workflows.IntelligentCodeGenerationWorkflow) // Intelligent workflow v2

	// Register activities - original
	w.RegisterActivity(activities.EnhancePromptActivity)
	w.RegisterActivity(activities.ParseRequirementsActivity)
	w.RegisterActivity(activities.GenerateCodeActivity)
	w.RegisterActivity(activities.ValidateCodeActivity)
	w.RegisterActivity(activities.GenerateTestsActivity)
	w.RegisterActivity(activities.GenerateDocumentationActivity)
	
	// Register activities - extended
	w.RegisterActivity(activities.GenerateFRDActivity)
	w.RegisterActivity(activities.GenerateProjectStructureActivity)
	w.RegisterActivity(activities.ValidateSemanticActivity)
	w.RegisterActivity(activities.ApplyFeedbackLoopActivity)
	w.RegisterActivity(activities.ResolveDependenciesActivity)
	w.RegisterActivity(activities.GenerateTestPlanActivity)
	w.RegisterActivity(activities.PerformSecurityScanActivity)
	w.RegisterActivity(activities.AnalyzePerformanceActivity)
	w.RegisterActivity(activities.GenerateReadmeActivity)
	w.RegisterActivity(activities.StoreQuantumDropActivity)
	
	// Register intelligent code generation
	w.RegisterActivity(activities.GenerateIntelligentCodeActivity)
	
	// Register preview activities
	w.RegisterActivity(activities.GeneratePreviewActivity)
	w.RegisterActivity(activities.StorePreviewMetadataActivity)
	
	// Register deployment activities
	w.RegisterActivity(activities.BuildContainerImageActivity)
	w.RegisterActivity(activities.GenerateK8sManifestsActivity)
	w.RegisterActivity(activities.DeployToKubernetesActivity)
	w.RegisterActivity(activities.HealthCheckActivity)

	logger.Info("Starting Temporal worker",
		zap.String("taskQueue", workflows.CodeGenerationTaskQueue),
		zap.String("temporalHost", temporalHost))

	// Start worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal("Unable to start worker", zap.Error(err))
	}
}