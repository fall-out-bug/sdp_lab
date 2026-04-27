package decompose

// Stitcher enforces a strict wire format on stage outputs: it validates that
// the stage's return value conforms to the expected shape, and serializes it
// for evidence logging and inter-stage handoff.
type Stitcher interface {
	// Name returns a human-readable identifier (e.g. "enum/verdict", "json/extract", "toon/aggregate").
	Name() string
	// Validate checks that out conforms to the stitcher's schema.
	// Returns nil if the value is valid; a descriptive error otherwise.
	Validate(out any) error
	// Marshal serializes out to a string representation.
	// The representation is stable: round-trip Marshal → parse → Validate must succeed.
	Marshal(out any) (string, error)
}
