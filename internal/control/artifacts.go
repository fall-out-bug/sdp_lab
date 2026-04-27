package control

import (
	"context"

	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

// ArtifactType classifies artifact kinds.
type ArtifactType = kernel.ArtifactType

const (
	ArtifactDispatchPacket = kernel.ArtifactDispatchPacket
	ArtifactResultPacket   = kernel.ArtifactResultPacket
	ArtifactEvidence       = kernel.ArtifactEvidence
	ArtifactProvenance     = kernel.ArtifactProvenance
	ArtifactContract       = kernel.ArtifactContract
	ArtifactIntake         = kernel.ArtifactIntake
)

// ArtifactRef is a reference to a file-based artifact.
type ArtifactRef = kernel.ArtifactRef

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
