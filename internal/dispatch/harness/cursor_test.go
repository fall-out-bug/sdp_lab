package harness

import (
	"reflect"
	"testing"
)

// TestCursorHarness_buildArgs_PassesModel verifies that opts.Model is plumbed
// through as `--model <id>` ahead of `-p`. Closes the silent bug where the
// Router-selected model was discarded by the harness.
func TestCursorHarness_buildArgs_PassesModel(t *testing.T) {
	h := NewCursorHarness()
	got := h.buildArgs(SpawnOpts{Model: "composer-2-fast", Prompt: "hi"})
	want := []string{"agent", "--model", "composer-2-fast", "-p", "hi"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs = %v, want %v", got, want)
	}
}

// TestCursorHarness_buildArgs_NoModelKept verifies back-compat: when Model is
// empty, --model is NOT injected so existing default-model flows behave as
// before this fix.
func TestCursorHarness_buildArgs_NoModelKept(t *testing.T) {
	h := NewCursorHarness()
	got := h.buildArgs(SpawnOpts{Prompt: "hi"})
	for _, a := range got {
		if a == "--model" {
			t.Errorf("buildArgs unexpectedly emitted --model when Model was empty: %v", got)
		}
	}
	want := []string{"agent", "-p", "hi"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs = %v, want %v", got, want)
	}
}

// TestCursorHarness_buildArgs_PreservesExtraArgs verifies that ExtraArgs
// remain at the tail and are not re-ordered by the model-flag injection.
func TestCursorHarness_buildArgs_PreservesExtraArgs(t *testing.T) {
	h := NewCursorHarness()
	got := h.buildArgs(SpawnOpts{Model: "composer-2", Prompt: "hi", ExtraArgs: []string{"--foo", "bar"}})
	want := []string{"agent", "--model", "composer-2", "-p", "hi", "--foo", "bar"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs = %v, want %v", got, want)
	}
}
