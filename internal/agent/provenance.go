package agent

import (
	"encoding/json"
	"time"

	"sdp_dev/internal/artifact"
	"sdp_dev/internal/bus"
)

func marshalPayload(v any) (json.RawMessage, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// ProvenanceSigner adds agent identity and hash-chain provenance to envelopes.
type ProvenanceSigner struct {
	agentID string
	role    string
}

// NewProvenanceSigner creates a ProvenanceSigner.
func NewProvenanceSigner(agentID, role string) *ProvenanceSigner {
	return &ProvenanceSigner{agentID: agentID, role: role}
}

// SignInput holds data needed to sign an artifact envelope.
type SignInput struct {
	IssueID       string
	ArtifactID    string
	ArtifactClass string
	Phase         string
	Payload       any
	ModelUsed     string
	BoundaryHash  string
	SkillsLoaded  []string
	HooksExecuted []string
	TraceLink     string
	EvidenceLink  string
	Sequence      uint64
	HashPrev      string
}

// Sign builds a provenance record and agent signature, returns a signed envelope.
func (p *ProvenanceSigner) Sign(in SignInput) (bus.Envelope, error) {
	capturedAt := time.Now().UTC().Format(time.RFC3339Nano)

	prov, err := artifact.BuildProvenanceRecord(artifact.ProvenanceInput{
		IssueID:       in.IssueID,
		ArtifactID:    in.ArtifactID,
		ArtifactClass: in.ArtifactClass,
		Phase:         in.Phase,
		Role:          p.role,
		CapturedAt:    capturedAt,
		Sequence:      in.Sequence,
		HashPrev:      in.HashPrev,
	}, in.Payload)
	if err != nil {
		return bus.Envelope{}, err
	}

	payloadBytes, err := marshalPayload(in.Payload)
	if err != nil {
		return bus.Envelope{}, err
	}

	sig := AgentSignature{
		AgentID:        p.agentID,
		Role:           p.role,
		ModelUsed:      in.ModelUsed,
		BoundaryHash:   in.BoundaryHash,
		SkillsLoaded:   in.SkillsLoaded,
		HooksExecuted:  in.HooksExecuted,
		TraceLink:      in.TraceLink,
		EvidenceLink:   in.EvidenceLink,
		ProvenanceHash: prov.Hash,
	}

	env := bus.Envelope{
		IssueID:       in.IssueID,
		ArtifactID:    in.ArtifactID,
		ArtifactClass: in.ArtifactClass,
		Phase:         in.Phase,
		Role:          p.role,
		CapturedAt:    capturedAt,
		Payload:       payloadBytes,
		Provenance:    prov,
	}
	env = WithAgentSignature(env, sig)
	return env, nil
}
