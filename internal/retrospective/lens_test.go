package retrospective

import (
	"testing"
)

func TestDefaultLenses(t *testing.T) {
	lenses := DefaultLenses()
	if len(lenses) != 5 {
		t.Errorf("DefaultLenses: want 5, got %d", len(lenses))
	}
	ids := make(map[LensID]bool)
	for _, l := range lenses {
		ids[l.ID] = true
		if l.Description == "" {
			t.Errorf("lens %s: empty description", l.ID)
		}
		if len(l.Focus) == 0 {
			t.Errorf("lens %s: empty focus", l.ID)
		}
	}
	for _, id := range []LensID{LensProtocol, LensInfra, LensCodeQuality, LensOperator, LensDX} {
		if !ids[id] {
			t.Errorf("missing lens %s", id)
		}
	}
}
