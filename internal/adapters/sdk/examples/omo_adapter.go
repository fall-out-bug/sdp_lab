// Package examples provides reference adapter implementations.
package examples

import (
	"context"
	"fmt"
	"time"

	sdk "sdp_dev/internal/adapters/sdk"
)

// OMOAdapter is a reference implementation for OhMyOpenCode integration.
type OMOAdapter struct {
	config       *sdk.AdapterConfig
	eventProducer sdk.EventProducer
	eventConsumer sdk.EventConsumer
	decisionMaker sdk.DecisionMaker
	auditor       sdk.DecisionAuditor
}

// NewOMOAdapter creates a new OMO adapter instance.
func NewOMOAdapter(config *sdk.AdapterConfig) *OMOAdapter {
	return &OMOAdapter{
		config: config,
	}
}

// EventProducer returns the event producer.
func (a *OMOAdapter) EventProducer() sdk.EventProducer {
	return a.eventProducer
}

// EventConsumer returns the event consumer.
func (a *OMOAdapter) EventConsumer() sdk.EventConsumer {
	return a.eventConsumer
}

// DecisionMaker returns the decision maker.
func (a *OMOAdapter) DecisionMaker() sdk.DecisionMaker {
	return a.decisionMaker
}

// DecisionAuditor returns the decision auditor.
func (a *OMOAdapter) DecisionAuditor() sdk.DecisionAuditor {
	return a.auditor
}

// Start starts the adapter.
func (a *OMOAdapter) Start(ctx context.Context) error {
	// Initialize event producer
	a.eventProducer = NewOMOEventProducer(a.config)

	// Initialize event consumer
	a.eventConsumer = NewOMOEventConsumer(a.config)

	// Initialize decision maker
	a.decisionMaker = NewOMODecisionMaker(a.config)

	// Initialize auditor
	a.auditor = NewOMODecisionAuditor(a.config)

	return nil
}

// Stop stops the adapter gracefully.
func (a *OMOAdapter) Stop(ctx context.Context) error {
	if a.eventProducer != nil {
		_ = a.eventProducer.Close()
	}
	if a.eventConsumer != nil {
		_ = a.eventConsumer.Close()
	}
	if a.decisionMaker != nil {
		_ = a.decisionMaker.Close()
	}
	if a.auditor != nil {
		_ = a.auditor.Close()
	}
	return nil
}

// Health returns the adapter health status.
func (a *OMOAdapter) Health() sdk.HealthStatus {
	return sdk.HealthStatus{
		Status:      "healthy",
		Details:     map[string]string{"system": a.config.System, "component": a.config.Component},
		LastChecked: time.Now(),
	}
}

// OMOEventProducer implements EventProducer for OMO.
type OMOEventProducer struct {
	config *sdk.AdapterConfig
}

// NewOMOEventProducer creates a new OMO event producer.
func NewOMOEventProducer(config *sdk.AdapterConfig) *OMOEventProducer {
	return &OMOEventProducer{config: config}
}

// EmitEvent emits an orchestration event.
func (p *OMOEventProducer) EmitEvent(ctx context.Context, event *sdk.OrchestrationEvent) error {
	// Validate the event
	validator, err := sdk.NewValidator()
	if err != nil {
		return fmt.Errorf("create validator: %w", err)
	}

	if err := validator.ValidateOrchestrationEvent(event); err != nil {
		return fmt.Errorf("validate event: %w", err)
	}

	// In a real implementation, this would publish to the event bus
	fmt.Printf("[OMO] Emitting event: %s (type: %s)\n", event.EventID, event.EventType)
	return nil
}

// EmitEventAsync emits an event asynchronously.
func (p *OMOEventProducer) EmitEventAsync(ctx context.Context, event *sdk.OrchestrationEvent) error {
	// For simplicity, we're doing synchronous emission
	// In production, this would use a message queue
	return p.EmitEvent(ctx, event)
}

// Close closes the producer.
func (p *OMOEventProducer) Close() error {
	return nil
}

// OMOEventConsumer implements EventConsumer for OMO.
type OMOEventConsumer struct {
	config *sdk.AdapterConfig
}

// NewOMOEventConsumer creates a new OMO event consumer.
func NewOMOEventConsumer(config *sdk.AdapterConfig) *OMOEventConsumer {
	return &OMOEventConsumer{config: config}
}

// Subscribe subscribes to events.
func (c *OMOEventConsumer) Subscribe(ctx context.Context, filter sdk.EventFilter, handler sdk.EventHandler) error {
	fmt.Printf("[OMO] Subscribed to events with filter: %+v\n", filter)
	return nil
}

// Unsubscribe unsubscribes from events.
func (c *OMOEventConsumer) Unsubscribe(ctx context.Context, subscriptionID string) error {
	fmt.Printf("[OMO] Unsubscribed: %s\n", subscriptionID)
	return nil
}

// Close closes the consumer.
func (c *OMOEventConsumer) Close() error {
	return nil
}

// OMODecisionMaker implements DecisionMaker for OMO.
type OMODecisionMaker struct {
	config *sdk.AdapterConfig
}

// NewOMODecisionMaker creates a new OMO decision maker.
func NewOMODecisionMaker(config *sdk.AdapterConfig) *OMODecisionMaker {
	return &OMODecisionMaker{config: config}
}

// MakeDecision makes a governance decision.
func (m *OMODecisionMaker) MakeDecision(ctx context.Context, request *sdk.DecisionRequest, context *sdk.DecisionContext) (*sdk.RuntimeDecision, error) {
	decision := &sdk.RuntimeDecision{
		SpecVersion:  "v1.0",
		DecisionID:   fmt.Sprintf("omo-decision-%d", time.Now().Unix()),
		Timestamp:    time.Now(),
		DecisionType: "scope.boundary",
		Decision:     "allow",
		Reason: sdk.DecisionReason{
			Code:    "WITHIN_SCOPE",
			Message: "Action is within OMO scope boundaries",
		},
		Context: *context,
	}

	// Validate the decision
	validator, err := sdk.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("create validator: %w", err)
	}

	if err := validator.ValidateRuntimeDecision(decision); err != nil {
		return nil, fmt.Errorf("validate decision: %w", err)
	}

	return decision, nil
}

// MakeDecisionAsync makes a decision asynchronously.
func (m *OMODecisionMaker) MakeDecisionAsync(ctx context.Context, request *sdk.DecisionRequest, context *sdk.DecisionContext) (string, error) {
	decisionID := fmt.Sprintf("omo-async-decision-%d", time.Now().Unix())
	fmt.Printf("[OMO] Async decision initiated: %s\n", decisionID)
	return decisionID, nil
}

// Close closes the decision maker.
func (m *OMODecisionMaker) Close() error {
	return nil
}

// OMODecisionAuditor implements DecisionAuditor for OMO.
type OMODecisionAuditor struct {
	config *sdk.AdapterConfig
}

// NewOMODecisionAuditor creates a new OMO decision auditor.
func NewOMODecisionAuditor(config *sdk.AdapterConfig) *OMODecisionAuditor {
	return &OMODecisionAuditor{config: config}
}

// RecordDecision records a decision for audit.
func (a *OMODecisionAuditor) RecordDecision(ctx context.Context, decision *sdk.RuntimeDecision) error {
	fmt.Printf("[OMO] Recorded decision: %s (type: %s, decision: %s)\n",
		decision.DecisionID, decision.DecisionType, decision.Decision)
	return nil
}

// QueryDecisions queries historical decisions.
func (a *OMODecisionAuditor) QueryDecisions(ctx context.Context, query sdk.DecisionQuery) ([]*sdk.RuntimeDecision, error) {
	fmt.Printf("[OMO] Querying decisions with filter: %+v\n", query)
	return []*sdk.RuntimeDecision{}, nil
}

// Close closes the auditor.
func (a *OMODecisionAuditor) Close() error {
	return nil
}
