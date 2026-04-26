package harness

import (
	"reflect"
	"testing"
)

// TestOpenCodeHarness_buildArgs_PassesProviderModel verifies that an explicit
// provider/model pair reaches the CLI as `-m provider/model`.
func TestOpenCodeHarness_buildArgs_PassesProviderModel(t *testing.T) {
	h := NewOpenCodeHarness()
	got := h.buildArgs(SpawnOpts{Agent: "implementer", Model: "ollama/qwen2.5-coder:7b"})
	want := []string{"run", "--agent", "implementer", "-m", "ollama/qwen2.5-coder:7b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs = %v, want %v", got, want)
	}
}

// TestOpenCodeHarness_buildArgs_AutoPrefixesOllama verifies that bare model
// strings (no `provider/` prefix) are auto-prefixed with `ollama/` so the
// caller can address local models without juggling provider strings.
func TestOpenCodeHarness_buildArgs_AutoPrefixesOllama(t *testing.T) {
	h := NewOpenCodeHarness()
	got := h.buildArgs(SpawnOpts{Agent: "implementer", Model: "qwen2.5-coder:7b"})
	want := []string{"run", "--agent", "implementer", "-m", "ollama/qwen2.5-coder:7b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs = %v, want %v", got, want)
	}
}

// TestOpenCodeHarness_buildArgs_KeepsExplicitProvider verifies that a model
// that already contains a `/` is passed through verbatim (no double-prefix).
func TestOpenCodeHarness_buildArgs_KeepsExplicitProvider(t *testing.T) {
	h := NewOpenCodeHarness()
	got := h.buildArgs(SpawnOpts{Agent: "implementer", Model: "openai/gpt-4o-mini"})
	want := []string{"run", "--agent", "implementer", "-m", "openai/gpt-4o-mini"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs = %v, want %v", got, want)
	}
}

// TestOpenCodeHarness_buildArgs_NoModelKept verifies back-compat: empty Model
// produces no `-m` flag, so existing opencode.config-driven flows still work.
func TestOpenCodeHarness_buildArgs_NoModelKept(t *testing.T) {
	h := NewOpenCodeHarness()
	got := h.buildArgs(SpawnOpts{Agent: "implementer"})
	for _, a := range got {
		if a == "-m" {
			t.Errorf("buildArgs unexpectedly emitted -m when Model was empty: %v", got)
		}
	}
	want := []string{"run", "--agent", "implementer"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs = %v, want %v", got, want)
	}
}

// TestOpenCodeHarness_buildArgs_PreservesExtraArgs verifies that ExtraArgs
// remain at the tail.
func TestOpenCodeHarness_buildArgs_PreservesExtraArgs(t *testing.T) {
	h := NewOpenCodeHarness()
	got := h.buildArgs(SpawnOpts{Agent: "implementer", Model: "ollama/q", ExtraArgs: []string{"--quiet"}})
	want := []string{"run", "--agent", "implementer", "-m", "ollama/q", "--quiet"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs = %v, want %v", got, want)
	}
}
