package bus

import (
	"encoding/json"
	"time"

	"sdp_dev/internal/artifact"
)

// Envelope extends artifact.ArtifactEnvelope with SDP bus metadata.
// Used for NATS publish/subscribe with typed payloads.
type Envelope struct {
	// Core artifact fields (matches artifact.ArtifactEnvelope)
	IssueID       string          `json:"issue_id"`
	ArtifactID    string          `json:"artifact_id"`
	ArtifactClass string          `json:"artifact_class"`
	Phase         string          `json:"phase"`
	Role          string          `json:"role"`
	CapturedAt    string          `json:"captured_at"`
	Payload       json.RawMessage `json:"payload"`
	Provenance    artifact.ProvenanceRecord `json:"provenance"`

	// Bus-specific metadata
	RunID     string `json:"run_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	Subject   string `json:"subject,omitempty"`
}

// FromArtifactEnvelope converts artifact.ArtifactEnvelope to bus.Envelope.
func FromArtifactEnvelope(ae artifact.ArtifactEnvelope, runID, projectID string) Envelope {
	return Envelope{
		IssueID:       ae.IssueID,
		ArtifactID:    ae.ArtifactID,
		ArtifactClass: ae.ArtifactClass,
		Phase:         ae.Phase,
		Role:          ae.Role,
		CapturedAt:    ae.CapturedAt,
		Payload:       ae.Payload,
		Provenance:    ae.Provenance,
		RunID:         runID,
		ProjectID:     projectID,
	}
}

// ToArtifactEnvelope converts bus.Envelope to artifact.ArtifactEnvelope.
func (e Envelope) ToArtifactEnvelope() artifact.ArtifactEnvelope {
	return artifact.ArtifactEnvelope{
		IssueID:       e.IssueID,
		ArtifactID:    e.ArtifactID,
		ArtifactClass: e.ArtifactClass,
		Phase:         e.Phase,
		Role:          e.Role,
		CapturedAt:    e.CapturedAt,
		Payload:       e.Payload,
		Provenance:    e.Provenance,
	}
}

// ToIngestRequest converts envelope to artifact.IngestRequest for BusService.Ingest.
func (e Envelope) ToIngestRequest() artifact.IngestRequest {
	var payload any = map[string]any{}
	if len(e.Payload) > 0 {
		_ = json.Unmarshal(e.Payload, &payload)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return artifact.IngestRequest{
		IssueID:       e.IssueID,
		ArtifactID:    e.ArtifactID,
		ArtifactClass: e.ArtifactClass,
		Phase:         e.Phase,
		Role:          e.Role,
		CapturedAt:    e.CapturedAt,
		Payload:       payload,
	}
}

// Timestamp returns CapturedAt as RFC3339 or now if empty.
func (e Envelope) Timestamp() string {
	if e.CapturedAt != "" {
		return e.CapturedAt
	}
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000Z07:00")
}
