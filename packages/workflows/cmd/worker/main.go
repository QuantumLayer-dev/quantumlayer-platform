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

	// Register activities
	w.RegisterActivity(activities.EnhancePromptActivity)
	w.RegisterActivity(activities.ParseRequirementsActivity)
	w.RegisterActivity(activities.GenerateCodeActivity)
	w.RegisterActivity(activities.ValidateCodeActivity)
	w.RegisterActivity(activities.GenerateTestsActivity)
	w.RegisterActivity(activities.GenerateDocumentationActivity)

	logger.Info("Starting Temporal worker",
		zap.String("taskQueue", workflows.CodeGenerationTaskQueue),
		zap.String("temporalHost", temporalHost))

	// Start worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal("Unable to start worker", zap.Error(err))
	}
}