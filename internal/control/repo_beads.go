package control

// BeadsCardRepository is a placeholder for a future Beads-backed repository.
// When implemented, this will read/write lifecycle state to Beads instead of
// YAML files. FileCardRepository will serve as fallback during migration.
//
// See docs/BEADS_FIRST_CONTROL_TOWER_ROADMAP.md phases R3-R5.
type BeadsCardRepository struct{}

// NewBeadsCardRepository creates a placeholder Beads repository.
// NOT YET IMPLEMENTED — will be wired when Beads public API is available.
func NewBeadsCardRepository() *BeadsCardRepository {
	return &BeadsCardRepository{}
}
