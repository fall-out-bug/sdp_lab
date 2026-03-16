package orchestrate

import "testing"

func TestNormalizeFindingStatus(t *testing.T) {
	if got := normalizeFindingStatus(" Open "); got != "open" {
		t.Fatalf("normalizeFindingStatus() = %q, want open", got)
	}
	if got := normalizeFindingStatus("CLOSED"); got != "closed" {
		t.Fatalf("normalizeFindingStatus() = %q, want closed", got)
	}
}
