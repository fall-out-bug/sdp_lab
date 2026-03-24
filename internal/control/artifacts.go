package control

import (
	"context"
	"time"
)

// ArtifactType classifies artifact kinds.
type ArtifactType string

const (
	ArtifactDispatchPacket ArtifactType = "dispatch_packet"
	ArtifactResultPacket   ArtifactType = "result_packet"
	ArtifactEvidence        ArtifactType = "evidence"
	ArtifactProvenance      ArtifactType = "provenance"
	ArtifactContract        ArtifactType = "contract"
	ArtifactIntake          ArtifactType = "intake"
)

// ArtifactRef is a reference to a file-based artifact.
type ArtifactRef struct {
	Type      ArtifactType `json:"type"`
	Path      string       `json:"path"`
	Hash      string       `json:"hash,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	Size      int64        `json:"size,omitempty"`
}

// ArtifactStore manages file-based artifacts (dispatch packets, results,
// evidence envelopes, provenance files, contracts, intake docs).
// Artifacts remain file-based — they are NOT stored in Beads.
// Beads metadata holds references (path + hash) to these files.
type ArtifactStore interface {
	// Store writes an artifact and returns its reference.
	// cardID is the feature card this artifact belongs to.
	Store(ctx context.Context, cardID string, ref ArtifactRef, data []byte) (ArtifactRef, error)

	// Load retrieves artifact data by reference.
	Load(ctx context.Context, ref ArtifactRef) ([]byte, error)

	// List returns all artifacts for a given feature card.
	List(ctx context.Context, cardID string) ([]ArtifactRef, error)

	// Delete removes an artifact.
	Delete(ctx context.Context, ref ArtifactRef) error
}
