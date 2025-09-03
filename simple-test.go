package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"
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

	fmt.Println("✅ Successfully connected to Temporal!")
	fmt.Printf("Namespace: quantumlayer\n")
	fmt.Printf("Host: 192.168.1.177:30733\n")
	
	// List workflows to test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	resp, err := c.ListWorkflow(ctx, &client.ListWorkflowExecutionsRequest{
		PageSize: 1,
	})
	if err != nil {
		log.Printf("Error listing workflows: %v\n", err)
	} else {
		fmt.Printf("✅ API test successful! Found %d workflows\n", len(resp.Executions))
	}
}