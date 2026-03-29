// Package sdk provides adapter SDK interfaces for contract producers and consumers.
package sdk

import (
	"context"
	"time"
)

// EventProducer defines the interface for emitting orchestration events.
type EventProducer interface {
	// EmitEvent emits an orchestration event to the event bus.
	EmitEvent(ctx context.Context, event *OrchestrationEvent) error

	// EmitEventAsync emits an event asynchronously and returns immediately.
	EmitEventAsync(ctx context.Context, event *OrchestrationEvent) error

	// Close closes the producer and releases resources.
	Close() error
}

// EventConsumer defines the interface for consuming orchestration events.
type EventConsumer interface {
	// Subscribe subscribes to events matching the given filter.
	Subscribe(ctx context.Context, filter EventFilter, handler EventHandler) error

	// Unsubscribe unsubscribes from events.
	Unsubscribe(ctx context.Context, subscriptionID string) error

	// Close closes the consumer and releases resources.
	Close() error
}

// EventFilter defines criteria for filtering events.
type EventFilter struct {
	EventTypes   []string          `json:"event_types,omitempty"`
	Source       *EventSource      `json:"source,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	WorkstreamID string            `json:"workstream_id,omitempty"`
	FeatureID    string            `json:"feature_id,omitempty"`
}

// EventHandler handles received events.
type EventHandler func(ctx context.Context, event *OrchestrationEvent) error

// DecisionMaker defines the interface for making runtime governance decisions.
type DecisionMaker interface {
	// MakeDecision evaluates a request and returns a decision.
	MakeDecision(ctx context.Context, request *DecisionRequest, context *DecisionContext) (*RuntimeDecision, error)

	// MakeDecisionAsync makes a decision asynchronously and returns the decision ID.
	MakeDecisionAsync(ctx context.Context, request *DecisionRequest, context *DecisionContext) (decisionID string, err error)

	// Close closes the decision maker and releases resources.
	Close() error
}

// DecisionAuditor defines the interface for auditing governance decisions.
type DecisionAuditor interface {
	// RecordDecision records a decision for audit purposes.
	RecordDecision(ctx context.Context, decision *RuntimeDecision) error

	// QueryDecisions queries historical decisions.
	QueryDecisions(ctx context.Context, query DecisionQuery) ([]*RuntimeDecision, error)

	// Close closes the auditor and releases resources.
	Close() error
}

// DecisionQuery defines criteria for querying decisions.
type DecisionQuery struct {
	DecisionTypes []string `json:"decision_types,omitempty"`
	Decisions     []string `json:"decisions,omitempty"` // allow, ask, deny
	Actor         *Actor   `json:"actor,omitempty"`
	WorkstreamID  string   `json:"workstream_id,omitempty"`
	StartTime     string   `json:"start_time,omitempty"`
	EndTime       string   `json:"end_time,omitempty"`
	Limit         int      `json:"limit,omitempty"`
}

// AdapterConfig contains configuration for an adapter.
type AdapterConfig struct {
	// System identifies the adapter system (omo, gas-town, beads, sdp-lab)
	System string `json:"system"`

	// Component identifies the component within the system
	Component string `json:"component"`

	// Version is the adapter version
	Version string `json:"version"`

	// Environment is the execution environment (oss, enterprise, development, production)
	Environment string `json:"environment"`

	// EventBus configures the event bus connection
	EventBus *EventBusConfig `json:"event_bus,omitempty"`

	// PolicyEngine configures the policy engine connection
	PolicyEngine *PolicyEngineConfig `json:"policy_engine,omitempty"`

	// Audit configures the audit trail connection
	Audit *AuditConfig `json:"audit,omitempty"`
}

// EventBusConfig configures event bus connection.
type EventBusConfig struct {
	Type     string `json:"type"` // nats, kafka, memory
	Endpoint string `json:"endpoint"`
	Topic    string `json:"topic"`
}

// PolicyEngineConfig configures policy engine connection.
type PolicyEngineConfig struct {
	Type     string `json:"type"` // opa, local
	Endpoint string `json:"endpoint"`
	Policies string `json:"policies"`
}

// AuditConfig configures audit trail connection.
type AuditConfig struct {
	Type     string `json:"type"` // postgres, sqlite, file
	Endpoint string `json:"endpoint"`
	Table    string `json:"table,omitempty"`
}

// AdapterBuilder provides a fluent interface for building adapters.
type AdapterBuilder interface {
	// WithConfig sets the adapter configuration.
	WithConfig(config *AdapterConfig) AdapterBuilder

	// WithEventProducer configures the event producer.
	WithEventProducer(producer EventProducer) AdapterBuilder

	// WithEventConsumer configures the event consumer.
	WithEventConsumer(consumer EventConsumer) AdapterBuilder

	// WithDecisionMaker configures the decision maker.
	WithDecisionMaker(maker DecisionMaker) AdapterBuilder

	// WithDecisionAuditor configures the decision auditor.
	WithDecisionAuditor(auditor DecisionAuditor) AdapterBuilder

	// Build builds and returns the configured adapter.
	Build() (Adapter, error)
}

// Adapter is the main interface for a system adapter.
type Adapter interface {
	// EventProducer returns the event producer for emitting events.
	EventProducer() EventProducer

	// EventConsumer returns the event consumer for receiving events.
	EventConsumer() EventConsumer

	// DecisionMaker returns the decision maker for governance decisions.
	DecisionMaker() DecisionMaker

	// DecisionAuditor returns the decision auditor for audit trails.
	DecisionAuditor() DecisionAuditor

	// Start starts the adapter.
	Start(ctx context.Context) error

	// Stop stops the adapter gracefully.
	Stop(ctx context.Context) error

	// Health returns the adapter health status.
	Health() HealthStatus
}

// HealthStatus represents the health status of an adapter.
type HealthStatus struct {
	Status      string            `json:"status"` // healthy, unhealthy, degraded
	Details     map[string]string `json:"details,omitempty"`
	LastChecked time.Time         `json:"last_checked"`
}
