package main

import (
	"fmt"
	"time"

	"sdp_dev/internal/adapters/sdk"
)

func main() {
	fmt.Println("=== Adapter SDK Examples ===")
	fmt.Println()

	// Example 1: Create and validate an task completion event
	fmt.Println("1. Task completion event:")
	event := &sdk.OrchestrationEvent{
		SpecVersion: "v1.0",
		EventID:     "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:   time.Now(),
		Source: sdk.EventSource{
			System:    "omo",
			Component: "coder-agent",
		},
		EventType: "task.completed",
		Payload: map[string]interface{}{
			"task_id": "F068-02",
			"status":  "success",
		},
	}

	validator, err := sdk.NewValidator()
	if err != nil {
		fmt.Printf("Error creating validator: %v\n", err)
		return
	}

	if err := validator.ValidateOrchestrationEvent(event); err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Event validated successfully")
	}

	fmt.Println()
	fmt.Println("2. Scope boundary decision:")
	decision := &sdk.RuntimeDecision{
		SpecVersion:  "v1.0",
		DecisionID:   "660e8400-e29b-41d4-a716-446655440000",
		Timestamp:    time.Now(),
		DecisionType: "scope.boundary",
		Decision:     "deny",
		Reason: sdk.DecisionReason{
			Code:    "SCOPE_EXCEEDED",
			Message: "Changes exceed workstream scope",
		},
	}

	if err := validator.ValidateRuntimeDecision(decision); err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Decision: %s\n", decision.Decision)
		fmt.Printf("   Reason: %s\n", decision.Reason.Message)
	}

	fmt.Println()
	fmt.Println("=== Examples completed ===")
}
