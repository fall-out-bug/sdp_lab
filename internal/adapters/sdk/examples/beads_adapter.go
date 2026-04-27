// Package examples provides reference adapter implementations.
package examples

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/fall-out-bug/sdp_lab/internal/adapters/sdk"
)

// BeadsAdapter is a reference implementation for Beads integration.
type BeadsAdapter struct {
	config       *sdk.AdapterConfig
	eventProducer sdk.EventProducer
	eventConsumer sdk.EventConsumer
	decisionMaker sdk.DecisionMaker
	auditor       sdk.DecisionAuditor
}

// NewBeadsAdapter creates a new Beads adapter instance.
func NewBeadsAdapter(config *sdk.AdapterConfig) *BeadsAdapter {
	return &BeadsAdapter{
		config: config,
	}
}

// EventProducer returns the event producer.
func (a *BeadsAdapter) EventProducer() sdk.EventProducer {
	return a.eventProducer
}

// EventConsumer returns the event consumer.
func (a *BeadsAdapter) EventConsumer() sdk.EventConsumer {
	return a.eventConsumer
}

// DecisionMaker returns the decision maker.
func (a *BeadsAdapter) DecisionMaker() sdk.DecisionMaker {
	return a.decisionMaker
}

// DecisionAuditor returns the decision auditor.
func (a *BeadsAdapter) DecisionAuditor() sdk.DecisionAuditor {
	return a.auditor
}

// Start starts the adapter.
func (a *BeadsAdapter) Start(ctx context.Context) error {
	a.eventProducer = NewBeadsEventProducer(a.config)
	a.eventConsumer = NewBeadsEventConsumer(a.config)
	a.decisionMaker = NewBeadsDecisionMaker(a.config)
	a.auditor = NewBeadsDecisionAuditor(a.config)
	return nil
}

// Stop stops the adapter gracefully.
func (a *BeadsAdapter) Stop(ctx context.Context) error {
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
func (a *BeadsAdapter) Health() sdk.HealthStatus {
	return sdk.HealthStatus{
		Status:      "healthy",
		Details:     map[string]string{"system": a.config.System, "component": a.config.Component},
		LastChecked: time.Now(),
	}
}

// BeadsEventProducer implements EventProducer for Beads.
type BeadsEventProducer struct {
	config *sdk.AdapterConfig
}

// NewBeadsEventProducer creates a new Beads event producer.
func NewBeadsEventProducer(config *sdk.AdapterConfig) *BeadsEventProducer {
	return &BeadsEventProducer{config: config}
}

// EmitEvent emits an orchestration event.
func (p *BeadsEventProducer) EmitEvent(ctx context.Context, event *sdk.OrchestrationEvent) error {
	validator, err := sdk.NewValidator()
	if err != nil {
		return fmt.Errorf("create validator: %w", err)
	}

	if err := validator.ValidateOrchestrationEvent(event); err != nil {
		return fmt.Errorf("validate event: %w", err)
	}

	fmt.Printf("[Beads] Emitting event: %s (type: %s)\n", event.EventID, event.EventType)
	return nil
}

// EmitEventAsync emits an event asynchronously.
func (p *BeadsEventProducer) EmitEventAsync(ctx context.Context, event *sdk.OrchestrationEvent) error {
	return p.EmitEvent(ctx, event)
}

// Close closes the producer.
func (p *BeadsEventProducer) Close() error {
	return nil
}

// BeadsEventConsumer implements EventConsumer for Beads.
type BeadsEventConsumer struct {
	config *sdk.AdapterConfig
}

// NewBeadsEventConsumer creates a new Beads event consumer.
func NewBeadsEventConsumer(config *sdk.AdapterConfig) *BeadsEventConsumer {
	return &BeadsEventConsumer{config: config}
}

// Subscribe subscribes to events.
func (c *BeadsEventConsumer) Subscribe(ctx context.Context, filter sdk.EventFilter, handler sdk.EventHandler) error {
	fmt.Printf("[Beads] Subscribed to events with filter: %+v\n", filter)
	return nil
}

// Unsubscribe unsubscribes from events.
func (c *BeadsEventConsumer) Unsubscribe(ctx context.Context, subscriptionID string) error {
	fmt.Printf("[Beads] Unsubscribed: %s\n", subscriptionID)
	return nil
}

// Close closes the consumer.
func (c *BeadsEventConsumer) Close() error {
	return nil
}

// BeadsDecisionMaker implements DecisionMaker for Beads.
type BeadsDecisionMaker struct {
	config *sdk.AdapterConfig
}

// NewBeadsDecisionMaker creates a new Beads decision maker.
func NewBeadsDecisionMaker(config *sdk.AdapterConfig) *BeadsDecisionMaker {
	return &BeadsDecisionMaker{config: config}
}

// MakeDecision makes a governance decision.
func (m *BeadsDecisionMaker) MakeDecision(ctx context.Context, request *sdk.DecisionRequest, context *sdk.DecisionContext) (*sdk.RuntimeDecision, error) {
	decision := &sdk.RuntimeDecision{
		SpecVersion:  "v1.0",
		DecisionID:   fmt.Sprintf("beads-decision-%d", time.Now().Unix()),
		Timestamp:    time.Now(),
		DecisionType: "quality.gate",
		Decision:     "allow",
		Reason: sdk.DecisionReason{
			Code:    "TESTS_PASSING",
			Message: "All quality gates passing for Beads integration",
		},
		Context: *context,
	}

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
func (m *BeadsDecisionMaker) MakeDecisionAsync(ctx context.Context, request *sdk.DecisionRequest, context *sdk.DecisionContext) (string, error) {
	decisionID := fmt.Sprintf("beads-async-decision-%d", time.Now().Unix())
	fmt.Printf("[Beads] Async decision initiated: %s\n", decisionID)
	return decisionID, nil
}

// Close closes the decision maker.
func (m *BeadsDecisionMaker) Close() error {
	return nil
}

// BeadsDecisionAuditor implements DecisionAuditor for Beads.
type BeadsDecisionAuditor struct {
	config *sdk.AdapterConfig
}

// NewBeadsDecisionAuditor creates a new Beads decision auditor.
func NewBeadsDecisionAuditor(config *sdk.AdapterConfig) *BeadsDecisionAuditor {
	return &BeadsDecisionAuditor{config: config}
}

// RecordDecision records a decision for audit.
func (a *BeadsDecisionAuditor) RecordDecision(ctx context.Context, decision *sdk.RuntimeDecision) error {
	fmt.Printf("[Beads] Recorded decision: %s (type: %s, decision: %s)\n",
		decision.DecisionID, decision.DecisionType, decision.Decision)
	return nil
}

// QueryDecisions queries historical decisions.
func (a *BeadsDecisionAuditor) QueryDecisions(ctx context.Context, query sdk.DecisionQuery) ([]*sdk.RuntimeDecision, error) {
	fmt.Printf("[Beads] Querying decisions with filter: %+v\n", query)
	return []*sdk.RuntimeDecision{}, nil
}

// Close closes the auditor.
func (a *BeadsDecisionAuditor) Close() error {
	return nil
}
