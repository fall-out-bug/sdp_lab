package sdk

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Validator validates contracts against JSON schemas.
type Validator struct {
	orchestrationSchema *jsonschema.Schema
	decisionSchema      *jsonschema.Schema
}

// NewValidator creates a new contract validator.
func NewValidator() (*Validator, error) {
	v := &Validator{}

	// Load schemas (compiled from embedded files or file system)
	orchSchema, err := loadOrchestrationEventSchema()
	if err != nil {
		return nil, fmt.Errorf("load orchestration schema: %w", err)
	}
	v.orchestrationSchema = orchSchema

	decSchema, err := loadDecisionSchema()
	if err != nil {
		return nil, fmt.Errorf("load decision schema: %w", err)
	}
	v.decisionSchema = decSchema

	return v, nil
}

// ValidateOrchestrationEvent validates an OrchestrationEvent against its schema.
func (v *Validator) ValidateOrchestrationEvent(event *OrchestrationEvent) error {
	// Convert to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Validate against schema
	var m interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	if err := v.orchestrationSchema.Validate(m); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}

	// Additional semantic validation
	return v.validateOrchestrationSemantics(event)
}

// ValidateRuntimeDecision validates a RuntimeDecision against its schema.
func (v *Validator) ValidateRuntimeDecision(decision *RuntimeDecision) error {
	// Convert to JSON
	data, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("marshal decision: %w", err)
	}

	// Validate against schema
	var m interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("unmarshal decision: %w", err)
	}

	if err := v.decisionSchema.Validate(m); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}

	// Additional semantic validation
	return v.validateDecisionSemantics(decision)
}

// ValidateJSON validates raw JSON against a contract type.
func (v *Validator) ValidateJSON(contractType string, data []byte) error {
	var m interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("unmarshal JSON: %w", err)
	}

	switch strings.ToLower(contractType) {
	case "orchestrationevent", "event":
		return v.orchestrationSchema.Validate(m)
	case "runtimedecision", "decision":
		return v.decisionSchema.Validate(m)
	default:
		return fmt.Errorf("unknown contract type: %s", contractType)
	}
}

// validateOrchestrationSemantics performs semantic validation on events.
func (v *Validator) validateOrchestrationSemantics(event *OrchestrationEvent) error {
	// Validate spec version format
	if !isValidSpecVersion(event.SpecVersion) {
		return fmt.Errorf("invalid spec_version format: %s (expected vMAJOR.MINOR)", event.SpecVersion)
	}

	// Validate event ID is not empty
	if event.EventID == "" {
		return fmt.Errorf("event_id is required")
	}

	// Validate timestamp is not zero
	if event.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// Validate source
	if event.Source.System == "" {
		return fmt.Errorf("source.system is required")
	}
	if event.Source.Component == "" {
		return fmt.Errorf("source.component is required")
	}

	// Validate event type is known
	if !isValidEventType(event.EventType) {
		return fmt.Errorf("unknown event_type: %s", event.EventType)
	}

	return nil
}

// validateDecisionSemantics performs semantic validation on decisions.
func (v *Validator) validateDecisionSemantics(decision *RuntimeDecision) error {
	// Validate spec version format
	if !isValidSpecVersion(decision.SpecVersion) {
		return fmt.Errorf("invalid spec_version format: %s (expected vMAJOR.MINOR)", decision.SpecVersion)
	}

	// Validate decision ID is not empty
	if decision.DecisionID == "" {
		return fmt.Errorf("decision_id is required")
	}

	// Validate timestamp is not zero
	if decision.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// Validate decision type is known
	if !isValidDecisionType(decision.DecisionType) {
		return fmt.Errorf("unknown decision_type: %s", decision.DecisionType)
	}

	// Validate decision value
	if !isValidDecisionValue(decision.Decision) {
		return fmt.Errorf("invalid decision value: %s (expected allow, ask, or deny)", decision.Decision)
	}

	// Validate reason
	if decision.Reason.Code == "" {
		return fmt.Errorf("reason.code is required")
	}
	if decision.Reason.Message == "" {
		return fmt.Errorf("reason.message is required")
	}

	return nil
}

// isValidSpecVersion checks if a spec version follows vMAJOR.MINOR format.
func isValidSpecVersion(version string) bool {
	if len(version) < 4 || version[0] != 'v' {
		return false
	}

	parts := strings.Split(version[1:], ".")
	if len(parts) != 2 {
		return false
	}

	// Check both parts are numeric
	for _, part := range parts {
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}

	return true
}

// isValidEventType checks if an event type is known.
func isValidEventType(eventType string) bool {
	// Allow any non-empty event type - different adapters may emit different events
	return eventType != ""
}

// isValidDecisionType checks if a decision type is known.
func isValidDecisionType(decisionType string) bool {
	validDecisionTypes := map[string]bool{
		"scope.boundary":    true,
		"test.coverage":     true,
		"security.approval": true,
		"review.status":     true,
		"evidence.validity": true,
		"quality.gate":      true,
		"resource.limit":    true,
		"model.selection":   true,
	}
	return validDecisionTypes[decisionType]
}

// isValidDecisionValue checks if a decision value is valid.
func isValidDecisionValue(decision string) bool {
	switch decision {
	case "allow", "ask", "deny":
		return true
	default:
		return false
	}
}

// loadOrchestrationEventSchema loads the OrchestrationEvent JSON schema.
func loadOrchestrationEventSchema() (*jsonschema.Schema, error) {
	// In production, this would load from embedded files or file system
	// For now, we'll use a simplified compiler
	compiler := jsonschema.NewCompiler()

	// Add schema from string (in production, load from file)
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://sdp.dev/contracts/orchestration-event/v1",
		"type": "object"
	}`

	if err := compiler.AddResource("https://sdp.dev/contracts/orchestration-event/v1", strings.NewReader(schemaJSON)); err != nil {
		return nil, err
	}

	return compiler.Compile("https://sdp.dev/contracts/orchestration-event/v1")
}

// loadDecisionSchema loads the RuntimeDecision JSON schema.
func loadDecisionSchema() (*jsonschema.Schema, error) {
	// In production, this would load from embedded files or file system
	compiler := jsonschema.NewCompiler()

	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://sdp.dev/contracts/runtime-decision/v1",
		"type": "object"
	}`

	if err := compiler.AddResource("https://sdp.dev/contracts/runtime-decision/v1", strings.NewReader(schemaJSON)); err != nil {
		return nil, err
	}

	return compiler.Compile("https://sdp.dev/contracts/runtime-decision/v1")
}
