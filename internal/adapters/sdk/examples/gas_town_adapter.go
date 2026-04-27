// Package examples provides reference adapter implementations.
package examples

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/fall-out-bug/sdp_lab/internal/adapters/sdk"
)

// GasTownAdapter is a reference implementation for Gas Town enterprise integration.
type GasTownAdapter struct {
	config       *sdk.AdapterConfig
	eventProducer sdk.EventProducer
	eventConsumer sdk.EventConsumer
	decisionMaker sdk.DecisionMaker
	auditor       sdk.DecisionAuditor
}

// NewGasTownAdapter creates a new Gas Town adapter instance.
func NewGasTownAdapter(config *sdk.AdapterConfig) *GasTownAdapter {
	return &GasTownAdapter{
		config: config,
	}
}

// EventProducer returns the event producer.
func (a *GasTownAdapter) EventProducer() sdk.EventProducer {
	return a.eventProducer
}

// EventConsumer returns the event consumer.
func (a *GasTownAdapter) EventConsumer() sdk.EventConsumer {
	return a.eventConsumer
}

// DecisionMaker returns the decision maker.
func (a *GasTownAdapter) DecisionMaker() sdk.DecisionMaker {
	return a.decisionMaker
}

// DecisionAuditor returns the decision auditor.
func (a *GasTownAdapter) DecisionAuditor() sdk.DecisionAuditor {
	return a.auditor
}

// Start starts the adapter.
func (a *GasTownAdapter) Start(ctx context.Context) error {
	a.eventProducer = NewGasTownEventProducer(a.config)
	a.eventConsumer = NewGasTownEventConsumer(a.config)
	a.decisionMaker = NewGasTownDecisionMaker(a.config)
	a.auditor = NewGasTownDecisionAuditor(a.config)
	return nil
}

// Stop stops the adapter gracefully.
func (a *GasTownAdapter) Stop(ctx context.Context) error {
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
func (a *GasTownAdapter) Health() sdk.HealthStatus {
	return sdk.HealthStatus{
		Status:      "healthy",
		Details:     map[string]string{"system": a.config.System, "component": a.config.Component},
		LastChecked: time.Now(),
	}
}

// GasTownEventProducer implements EventProducer for Gas Town.
type GasTownEventProducer struct {
	config *sdk.AdapterConfig
}

// NewGasTownEventProducer creates a new Gas Town event producer.
func NewGasTownEventProducer(config *sdk.AdapterConfig) *GasTownEventProducer {
	return &GasTownEventProducer{config: config}
}

// EmitEvent emits an orchestration event.
func (p *GasTownEventProducer) EmitEvent(ctx context.Context, event *sdk.OrchestrationEvent) error {
	validator, err := sdk.NewValidator()
	if err != nil {
		return fmt.Errorf("create validator: %w", err)
	}

	if err := validator.ValidateOrchestrationEvent(event); err != nil {
		return fmt.Errorf("validate event: %w", err)
	}

	fmt.Printf("[GasTown] Emitting event: %s (type: %s)\n", event.EventID, event.EventType)
	return nil
}

// EmitEventAsync emits an event asynchronously.
func (p *GasTownEventProducer) EmitEventAsync(ctx context.Context, event *sdk.OrchestrationEvent) error {
	return p.EmitEvent(ctx, event)
}

// Close closes the producer.
func (p *GasTownEventProducer) Close() error {
	return nil
}

// GasTownEventConsumer implements EventConsumer for Gas Town.
type GasTownEventConsumer struct {
	config *sdk.AdapterConfig
}

// NewGasTownEventConsumer creates a new Gas Town event consumer.
func NewGasTownEventConsumer(config *sdk.AdapterConfig) *GasTownEventConsumer {
	return &GasTownEventConsumer{config: config}
}

// Subscribe subscribes to events.
func (c *GasTownEventConsumer) Subscribe(ctx context.Context, filter sdk.EventFilter, handler sdk.EventHandler) error {
	fmt.Printf("[GasTown] Subscribed to events with filter: %+v\n", filter)
	return nil
}

// Unsubscribe unsubscribes from events.
func (c *GasTownEventConsumer) Unsubscribe(ctx context.Context, subscriptionID string) error {
	fmt.Printf("[GasTown] Unsubscribed: %s\n", subscriptionID)
	return nil
}

// Close closes the consumer.
func (c *GasTownEventConsumer) Close() error {
	return nil
}

// GasTownDecisionMaker implements DecisionMaker for Gas Town.
type GasTownDecisionMaker struct {
	config *sdk.AdapterConfig
}

// NewGasTownDecisionMaker creates a new Gas Town decision maker.
func NewGasTownDecisionMaker(config *sdk.AdapterConfig) *GasTownDecisionMaker {
	return &GasTownDecisionMaker{config: config}
}

// MakeDecision makes a governance decision.
func (m *GasTownDecisionMaker) MakeDecision(ctx context.Context, request *sdk.DecisionRequest, context *sdk.DecisionContext) (*sdk.RuntimeDecision, error) {
	decision := &sdk.RuntimeDecision{
		SpecVersion:  "v1.0",
		DecisionID:   fmt.Sprintf("gt-decision-%d", time.Now().Unix()),
		Timestamp:    time.Now(),
		DecisionType: "security.approval",
		Decision:     "allow",
		Reason: sdk.DecisionReason{
			Code:    "ENTERPRISE_APPROVED",
			Message: "Action approved by enterprise policy engine",
		},
		Context: *context,
		PolicyReference: &sdk.PolicyReference{
			PolicyID:      "enterprise/security-approval",
			PolicyVersion: "v1.2.0",
			RuleName:      "allow-approved-actions",
			OPADecisionID: fmt.Sprintf("opa-decision-%d", time.Now().Unix()),
		},
		Evidence: []sdk.DecisionEvidence{
			{
				Type:      "review_approval",
				Reference: "https://enterprise.reviews/approvals/12345",
				Summary:   "Approved by enterprise security reviewer",
			},
		},
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
func (m *GasTownDecisionMaker) MakeDecisionAsync(ctx context.Context, request *sdk.DecisionRequest, context *sdk.DecisionContext) (string, error) {
	decisionID := fmt.Sprintf("gt-async-decision-%d", time.Now().Unix())
	fmt.Printf("[GasTown] Async decision initiated: %s\n", decisionID)
	return decisionID, nil
}

// Close closes the decision maker.
func (m *GasTownDecisionMaker) Close() error {
	return nil
}

// GasTownDecisionAuditor implements DecisionAuditor for Gas Town.
type GasTownDecisionAuditor struct {
	config *sdk.AdapterConfig
}

// NewGasTownDecisionAuditor creates a new Gas Town decision auditor.
func NewGasTownDecisionAuditor(config *sdk.AdapterConfig) *GasTownDecisionAuditor {
	return &GasTownDecisionAuditor{config: config}
}

// RecordDecision records a decision for audit.
func (a *GasTownDecisionAuditor) RecordDecision(ctx context.Context, decision *sdk.RuntimeDecision) error {
	fmt.Printf("[GasTown] Recorded decision: %s (type: %s, decision: %s)\n",
		decision.DecisionID, decision.DecisionType, decision.Decision)
	return nil
}

// QueryDecisions queries historical decisions.
func (a *GasTownDecisionAuditor) QueryDecisions(ctx context.Context, query sdk.DecisionQuery) ([]*sdk.RuntimeDecision, error) {
	fmt.Printf("[GasTown] Querying decisions with filter: %+v\n", query)
	return []*sdk.RuntimeDecision{}, nil
}

// Close closes the auditor.
func (a *GasTownDecisionAuditor) Close() error {
	return nil
}
