package executor

import "testing"

func TestResolveAgent_KnownPhases(t *testing.T) {
	t.Setenv("SDP_DEFAULT_AGENT", "")

	tests := []struct {
		phase string
		want  string
	}{
		{phase: "build", want: "hephaestus"},
		{phase: "review", want: "momus"},
		{phase: "qa", want: "oracle"},
	}

	for _, tc := range tests {
		if got := ResolveAgent(tc.phase); got != tc.want {
			t.Fatalf("ResolveAgent(%q) = %q, want %q", tc.phase, got, tc.want)
		}
	}
}

func TestResolveAgent_UnknownPhase(t *testing.T) {
	t.Setenv("SDP_DEFAULT_AGENT", "")
	if got := ResolveAgent("bogus"); got != "sisyphus" {
		t.Fatalf("ResolveAgent(unknown) = %q, want sisyphus", got)
	}
}

func TestResolveAgent_EmptyPhase(t *testing.T) {
	t.Setenv("SDP_DEFAULT_AGENT", "")
	if got := ResolveAgent(""); got != "sisyphus" {
		t.Fatalf("ResolveAgent(empty) = %q, want sisyphus", got)
	}
}

func TestResolveAgent_EnvOverride(t *testing.T) {
	t.Setenv("SDP_DEFAULT_AGENT", "hephaestus")
	for _, phase := range []string{"build", "review", "", "bogus"} {
		if got := ResolveAgent(phase); got != "hephaestus" {
			t.Fatalf("ResolveAgent(%q) with override = %q, want hephaestus", phase, got)
		}
	}
}
