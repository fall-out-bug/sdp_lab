package policy

import "testing"

func TestNextFallback(t *testing.T) {
	next, ok := NextFallback("glm-5")
	if !ok || next != "glm-4.7" {
		t.Fatalf("expected glm-4.7 fallback, got %q ok=%v", next, ok)
	}

	next, ok = NextFallback("glm-4.7")
	if ok || next != "" {
		t.Fatalf("expected no fallback from glm-4.7, got %q ok=%v", next, ok)
	}
}

func TestResolveFallbackSequence(t *testing.T) {
	seq := ResolveFallbackSequence("glm-5")
	if len(seq) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(seq))
	}
	if seq[0] != "glm-5" || seq[1] != "glm-4.7" || seq[2] != "escalated" {
		t.Fatalf("unexpected sequence: %#v", seq)
	}
}
