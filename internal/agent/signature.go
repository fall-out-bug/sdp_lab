package agent

import (
	"encoding/json"

	"sdp_dev/internal/bus"
)

// AgentSignature is included in every published envelope for auditability.
type AgentSignature struct {
	AgentID        string   `json:"agent_id"`
	Role           string   `json:"role"`
	ModelUsed      string   `json:"model_used"`
	BoundaryHash   string   `json:"boundary_hash"`
	SkillsLoaded   []string `json:"skills_loaded"`
	HooksExecuted  []string `json:"hooks_executed"`
	TraceLink      string   `json:"trace_link"`
	EvidenceLink   string   `json:"evidence_link"`
	ProvenanceHash string   `json:"provenance_hash"`
}

// signatureKey is the envelope payload key for agent signature (nested in payload).
const signatureKey = "agent_signature"

// WithAgentSignature embeds AgentSignature into the envelope payload.
func WithAgentSignature(env bus.Envelope, sig AgentSignature) bus.Envelope {
	var payload map[string]any
	if len(env.Payload) > 0 {
		_ = json.Unmarshal(env.Payload, &payload)
	}
	if payload == nil {
		payload = make(map[string]any)
	}
	payload[signatureKey] = sig
	b, _ := json.Marshal(payload)
	env.Payload = b
	return env
}

// ExtractAgentSignature reads AgentSignature from an envelope payload if present.
func ExtractAgentSignature(env bus.Envelope) (AgentSignature, bool) {
	var payload map[string]any
	if len(env.Payload) == 0 {
		return AgentSignature{}, false
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return AgentSignature{}, false
	}
	v, ok := payload[signatureKey]
	if !ok {
		return AgentSignature{}, false
	}
	b, err := json.Marshal(v)
	if err != nil {
		return AgentSignature{}, false
	}
	var sig AgentSignature
	if err := json.Unmarshal(b, &sig); err != nil {
		return AgentSignature{}, false
	}
	return sig, true
}
