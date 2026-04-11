package architect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// nullByte is the delimiter for deterministic IDs. Null bytes are invalid in
// POSIX paths, NPM package names, Maven coordinates, Cargo crate names, and
// PyPI package names — zero collision risk across all target ecosystems.
const nullByte = '\x00'

// JoinID constructs a deterministic ID from segments separated by null bytes.
// Any null byte within a segment is percent-encoded as %00.
// Returns an error if any segment is empty.
// Idempotent: SplitID(JoinID(segs)) == segs for valid segments.
func JoinID(segments ...string) (string, error) {
	if len(segments) == 0 {
		return "", fmt.Errorf("id: at least one segment required")
	}
	encoded := make([]string, len(segments))
	for i, seg := range segments {
		if seg == "" {
			return "", fmt.Errorf("id: segment %d is empty", i)
		}
		encoded[i] = strings.ReplaceAll(seg, "\x00", "%00")
	}
	return strings.Join(encoded, string(nullByte)), nil
}

// SplitID splits a deterministic ID into its segments.
// Decodes %00 back to \x00 within each segment.
// Returns an error if the ID format is invalid.
// Idempotent: JoinID(SplitID(id)) == id for well-formed IDs.
func SplitID(id string) ([]string, error) {
	if id == "" {
		return nil, fmt.Errorf("id: empty")
	}
	segments := strings.Split(id, string(nullByte))
	for i, seg := range segments {
		if seg == "" {
			return nil, fmt.Errorf("id: empty segment at position %d", i)
		}
		segments[i] = strings.ReplaceAll(seg, "%00", "\x00")
	}
	return segments, nil
}

// NormalizeID parses and re-joins an ID to its canonical form.
// All ID comparisons MUST use NormalizeID on both sides.
func NormalizeID(id string) (string, error) {
	segments, err := SplitID(id)
	if err != nil {
		return "", err
	}
	return JoinID(segments...)
}

// ContentHashSuffix returns an 8-char hex suffix for disambiguation.
// Used when path alone is ambiguous (multiple modules at same path).
// The suffix goes into the 3rd segment: "name~abc12345".
func ContentHashSuffix(canonicalJSON string) string {
	h := sha256.Sum256([]byte(canonicalJSON))
	return hex.EncodeToString(h[:4]) // 8 hex chars
}
