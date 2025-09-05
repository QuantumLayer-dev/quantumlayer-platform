package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	
	"github.com/quantumlayer/infra-workflow-worker/activities"
	"github.com/quantumlayer/infra-workflow-worker/workflows"
)

func main() {
	// Get Temporal configuration from environment
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "temporal-frontend.temporal.svc.cluster.local:7233"
	}

	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "quantumlayer"
	}

	taskQueue := os.Getenv("TASK_QUEUE")
	if taskQueue == "" {
		taskQueue = "infrastructure-generation"
	}

	// Create Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatal("Unable to create Temporal client:", err)
	}
	defer temporalClient.Close()

	// Create worker
	w := worker.New(temporalClient, taskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: 10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Register workflows
	w.RegisterWorkflow(workflows.InfrastructureGenerationWorkflow)
	
	// Register activities
	w.RegisterActivity(activities.AnalyzeCodeActivity)
	w.RegisterActivity(activities.GenerateInfrastructureActivity)
	w.RegisterActivity(activities.BuildGoldenImageActivity)
	w.RegisterActivity(activities.GenerateSOPActivity)
	w.RegisterActivity(activities.ValidateComplianceActivity)
	w.RegisterActivity(activities.EstimateCostActivity)
	w.RegisterActivity(activities.StoreInfraDropActivity)
	w.RegisterActivity(activities.DeployInfrastructureActivity)

	// Start worker
	go func() {
		err := w.Run(worker.InterruptCh())
		if err != nil {
			log.Fatal("Unable to start worker:", err)
		}
	}()

	log.Printf("Infrastructure workflow worker started on task queue: %s", taskQueue)
	log.Printf("Connected to Temporal at: %s", temporalHost)
	log.Printf("Namespace: %s", namespace)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down worker...")
	w.Stop()
	log.Println("Worker stopped")
}