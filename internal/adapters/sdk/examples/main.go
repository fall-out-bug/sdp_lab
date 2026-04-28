// Package examples demonstrates usage of the adapter SDK.
package examples

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/fall-out-bug/sdp_lab/internal/adapters/sdk"
)

// ExampleOMOAdapterUsage demonstrates using the OMO adapter.
func ExampleOMOAdapterUsage() {
	ctx := context.Background()

	// Configure the adapter
	config := &sdk.AdapterConfig{
		System:    "omo",
		Component: "agent",
		Version:   "1.0.0",
		Environment: "development",
		EventBus: &sdk.EventBusConfig{
			Type:     "memory",
			Endpoint: "",
			Topic:    "sdp.events",
		},
		PolicyEngine: &sdk.PolicyEngineConfig{
			Type:     "local",
			Endpoint: "",
			Policies: "./policies",
		},
	}

	// Create and start the adapter
	adapter := NewOMOAdapter(config)
	if err := adapter.Start(ctx); err != nil {
		log.Fatalf("Failed to start adapter: %v", err)
	}
	defer adapter.Stop(ctx)

	// Emit an orchestration event
	event := &sdk.OrchestrationEvent{
		SpecVersion: "v1.0",
		EventID:     "omo-event-001",
		Timestamp:   time.Now(),
		Source: sdk.EventSource{
			System:    "omo",
			Component: "agent",
			Version:   "1.0.0",
		},
		EventType: "task.completed",
		Payload: map[string]interface{}{
			"task_id":       "F068-01",
			"workstream_id": "00-068-01",
			"feature_id":    "F068",
			"status":        "success",
		},
		Context: &sdk.ExecutionContext{
			WorkstreamID: "00-068-01",
			FeatureID:    "F068",
		},
	}

	producer := adapter.EventProducer()
	if err := producer.EmitEvent(ctx, event); err != nil {
		log.Printf("Failed to emit event: %v", err)
	}

	// Make a runtime decision
	request := &sdk.DecisionRequest{
		Action:   "write_file",
		Resource: "/path/to/file.go",
	}

	decisionCtx := &sdk.DecisionContext{
		Request:     *request,
		Environment: "development",
		WorkstreamID: "00-068-01",
		FeatureID:   "F068",
		Actor: &sdk.Actor{
			Type: "agent",
			ID:   "omo-agent-001",
		},
	}

	maker := adapter.DecisionMaker()
	decision, err := maker.MakeDecision(ctx, request, decisionCtx)
	if err != nil {
		log.Printf("Failed to make decision: %v", err)
	} else {
		fmt.Printf("Decision: %s (%s)\n", decision.Decision, decision.Reason.Message)
	}

	// Check health
	health := adapter.Health()
	fmt.Printf("Adapter health: %s\n", health.Status)
}

// ExampleBeadsAdapterUsage demonstrates using the Beads adapter.
func ExampleBeadsAdapterUsage() {
	ctx := context.Background()

	config := &sdk.AdapterConfig{
		System:    "beads",
		Component: "bridge",
		Version:   "1.0.0",
		Environment: "oss",
	}

	adapter := NewBeadsAdapter(config)
	if err := adapter.Start(ctx); err != nil {
		log.Fatalf("Failed to start adapter: %v", err)
	}
	defer adapter.Stop(ctx)

	// Emit an issue created event
	event := &sdk.OrchestrationEvent{
		SpecVersion: "v1.0",
		EventID:     "beads-event-001",
		Timestamp:   time.Now(),
		Source: sdk.EventSource{
			System:    "beads",
			Component: "bridge",
			Version:   "1.0.0",
		},
		EventType: "issue.created",
		Payload: map[string]interface{}{
			"issue_id": "sdplab-abc123",
			"title":    "F068-01: Implement Adapter SDK",
			"priority": "P1",
		},
	}

	producer := adapter.EventProducer()
	if err := producer.EmitEvent(ctx, event); err != nil {
		log.Printf("Failed to emit event: %v", err)
	}

	fmt.Printf("Beads adapter health: %s\n", adapter.Health().Status)
}

// ExampleGasTownAdapterUsage demonstrates using the Gas Town adapter.
func ExampleGasTownAdapterUsage() {
	ctx := context.Background()

	config := &sdk.AdapterConfig{
		System:    "gas-town",
		Component: "k8s-agent-controller",
		Version:   "2.1.0",
		Environment: "enterprise",
		PolicyEngine: &sdk.PolicyEngineConfig{
			Type:     "opa",
			Endpoint: "https://opa.example.com",
			Policies: "enterprise",
		},
		Audit: &sdk.AuditConfig{
			Type:     "postgres",
			Endpoint: "postgres://audit-db.example.com:5432/audit",
			Table:    "decisions",
		},
	}

	adapter := NewGasTownAdapter(config)
	if err := adapter.Start(ctx); err != nil {
		log.Fatalf("Failed to start adapter: %v", err)
	}
	defer adapter.Stop(ctx)

	// Emit a phase transition event
	event := &sdk.OrchestrationEvent{
		SpecVersion: "v1.0",
		EventID:     "gt-event-001",
		Timestamp:   time.Now(),
		Source: sdk.EventSource{
			System:    "gas-town",
			Component: "k8s-agent-controller",
			Version:   "2.1.0",
		},
		EventType: "phase.transition",
		Payload: map[string]interface{}{
			"from_phase": "analyst",
			"to_phase":   "coder",
			"handoff": map[string]interface{}{
				"analyst_output": "/artifacts/analyst/F068-01.json",
				"coder_prompt":   "/prompts/coder/feature-implementation.yaml",
			},
		},
	}

	producer := adapter.EventProducer()
	if err := producer.EmitEvent(ctx, event); err != nil {
		log.Printf("Failed to emit event: %v", err)
	}

	// Make a security approval decision
	request := &sdk.DecisionRequest{
		Action:   "deploy",
		Resource: "production",
		Parameters: map[string]interface{}{
			"target_env": "production",
			"artifact":   "sdp-lab:v1.2.3",
		},
	}

	decisionCtx := &sdk.DecisionContext{
		Request:     *request,
		Environment: "enterprise",
		Actor: &sdk.Actor{
			Type:  "agent",
			ID:    "gt-agent-001",
			Roles: []string{"deployer"},
		},
	}

	maker := adapter.DecisionMaker()
	decision, err := maker.MakeDecision(ctx, request, decisionCtx)
	if err != nil {
		log.Printf("Failed to make decision: %v", err)
	} else {
		fmt.Printf("Decision: %s (Policy: %s)\n", decision.Decision, decision.PolicyReference.PolicyID)
	}

	// Audit the decision
	auditor := adapter.DecisionAuditor()
	if err := auditor.RecordDecision(ctx, decision); err != nil {
		log.Printf("Failed to record decision: %v", err)
	}
}

// ExampleValidation demonstrates contract validation.
func ExampleValidation() {
	// Create a validator
	validator, err := sdk.NewValidator()
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Validate an orchestration event
	event := &sdk.OrchestrationEvent{
		SpecVersion: "v1.0",
		EventID:     "test-event-001",
		Timestamp:   time.Now(),
		Source: sdk.EventSource{
			System:    "test",
			Component: "validator",
		},
		EventType: "test.event",
		Payload:   map[string]interface{}{"test": "data"},
	}

	if err := validator.ValidateOrchestrationEvent(event); err != nil {
		log.Printf("Event validation failed: %v", err)
	} else {
		fmt.Println("Event validation passed")
	}

	// Validate a runtime decision
	decision := &sdk.RuntimeDecision{
		SpecVersion:  "v1.0",
		DecisionID:   "test-decision-001",
		Timestamp:    time.Now(),
		DecisionType: "scope.boundary",
		Decision:     "allow",
		Reason: sdk.DecisionReason{
			Code:    "TEST",
			Message: "Test decision",
		},
		Context: sdk.DecisionContext{
			Request: sdk.DecisionRequest{
				Action: "test",
			},
		},
	}

	if err := validator.ValidateRuntimeDecision(decision); err != nil {
		log.Printf("Decision validation failed: %v", err)
	} else {
		fmt.Println("Decision validation passed")
	}
}

// ExampleEventConsumption demonstrates consuming events.
func ExampleEventConsumption() {
	ctx := context.Background()

	config := &sdk.AdapterConfig{
		System:    "consumer",
		Component: "example",
		Version:   "1.0.0",
	}

	adapter := NewOMOAdapter(config)
	if err := adapter.Start(ctx); err != nil {
		log.Fatalf("Failed to start adapter: %v", err)
	}
	defer adapter.Stop(ctx)

	// Subscribe to events
	consumer := adapter.EventConsumer()
	filter := sdk.EventFilter{
		EventTypes: []string{"task.completed", "task.failed"},
		Labels: map[string]string{
			"priority": "high",
		},
	}

	handler := func(ctx context.Context, event *sdk.OrchestrationEvent) error {
		fmt.Printf("Received event: %s (type: %s)\n", event.EventID, event.EventType)
		// Process the event
		return nil
	}

	if err := consumer.Subscribe(ctx, filter, handler); err != nil {
		log.Printf("Failed to subscribe: %v", err)
	}
}

func main() {
	fmt.Println("=== OMO Adapter Example ===")
	ExampleOMOAdapterUsage()

	fmt.Println("\n=== Beads Adapter Example ===")
	ExampleBeadsAdapterUsage()

	fmt.Println("\n=== Gas Town Adapter Example ===")
	ExampleGasTownAdapterUsage()

	fmt.Println("\n=== Validation Example ===")
	ExampleValidation()

	fmt.Println("\n=== Event Consumption Example ===")
	ExampleEventConsumption()
}
