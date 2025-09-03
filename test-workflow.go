package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"test-workflow/packages/workflows/internal/types"
	"test-workflow/packages/workflows/internal/workflows"
)

func main() {
	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  "192.168.1.177:30733",
		Namespace: "quantumlayer",
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	// Create workflow options
	workflowID := fmt.Sprintf("test-codegen-%s", uuid.New().String())
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: workflows.CodeGenerationTaskQueue,
		WorkflowExecutionTimeout: 5 * time.Minute,
	}

	// Create test request
	request := types.CodeGenerationRequest{
		ID:          uuid.New().String(),
		Prompt:      "Create a REST API service in Go that manages a todo list with CRUD operations",
		Language:    "go",
		Framework:   "gin",
		Type:        "api",
		GenerateTests: true,
		GenerateDocs:  true,
		Requirements: map[string]interface{}{
			"database":     "postgresql",
			"authentication": "jwt",
			"features": []string{
				"Create todo",
				"List todos",
				"Update todo",
				"Delete todo",
				"Mark as complete",
			},
		},
	}

	// Start workflow
	fmt.Printf("Starting workflow with ID: %s\n", workflowID)
	we, err := c.ExecuteWorkflow(context.Background(), options, workflows.CodeGenerationWorkflow, request)
	if err != nil {
		log.Fatalln("Unable to start workflow", err)
	}

	fmt.Printf("Workflow started successfully!\n")
	fmt.Printf("WorkflowID: %s\n", we.GetID())
	fmt.Printf("RunID: %s\n", we.GetRunID())

	// Wait for result
	fmt.Println("\nWaiting for workflow to complete...")
	var result types.CodeGenerationResult
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Unable to get workflow result", err)
	}

	// Display results
	fmt.Println("\n=== Workflow Completed Successfully! ===")
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Message: %s\n", result.Message)
	fmt.Printf("Generated Files: %d\n", len(result.Files))
	
	for _, file := range result.Files {
		fmt.Printf("\n--- File: %s ---\n", file.Name)
		fmt.Printf("Type: %s\n", file.Type)
		fmt.Printf("Size: %d bytes\n", len(file.Content))
		if len(file.Content) > 200 {
			fmt.Printf("Content Preview:\n%s...\n", file.Content[:200])
		} else {
			fmt.Printf("Content:\n%s\n", file.Content)
		}
	}

	fmt.Printf("\n✅ Test completed! Check Temporal Web UI at http://192.168.1.177:30888\n")
}