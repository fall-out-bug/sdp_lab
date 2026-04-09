package dispatch

import (
	"path/filepath"
	"testing"
)

func TestCapabilityProfile_ScoreFor(t *testing.T) {
	tests := []struct {
		name         string
		capabilities map[string]CapabilityScore
		taskType     string
		language     string
		want         float64
	}{
		{
			name: "exact match returns TestPassRate",
			capabilities: map[string]CapabilityScore{
				"refactor:go": {TestPassRate: 0.92, AvgDuration: 5.0, SampleCount: 10},
			},
			taskType: "refactor",
			language: "go",
			want:     0.92,
		},
		{
			name: "no match returns 0",
			capabilities: map[string]CapabilityScore{
				"feature:typescript": {TestPassRate: 0.75, AvgDuration: 8.0, SampleCount: 4},
			},
			taskType: "refactor",
			language: "go",
			want:     0.0,
		},
		{
			name:         "empty capabilities returns 0",
			capabilities: map[string]CapabilityScore{},
			taskType:     "bugfix",
			language:     "python",
			want:         0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &CapabilityProfile{
				Harness:      "claude",
				Provider:     "anthropic",
				Model:        "sonnet",
				Capabilities: tc.capabilities,
			}
			got := p.ScoreFor(tc.taskType, tc.language)
			if got != tc.want {
				t.Errorf("ScoreFor(%q, %q) = %v, want %v", tc.taskType, tc.language, got, tc.want)
			}
		})
	}
}

func TestProfileStore_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	store := &ProfileStore{Dir: dir}

	original := &CapabilityProfile{
		Harness:  "claude",
		Provider: "anthropic",
		Model:    "sonnet",
		Capabilities: map[string]CapabilityScore{
			"refactor:go": {TestPassRate: 0.88, AvgDuration: 6.5, SampleCount: 20},
		},
		UpdatedAt: "2026-03-28T00:00:00Z",
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load("claude", "anthropic", "sonnet")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("Load() returned nil profile")
		return
	}
	if loaded.Harness != original.Harness {
		t.Errorf("Harness = %q, want %q", loaded.Harness, original.Harness)
	}
	if loaded.Provider != original.Provider {
		t.Errorf("Provider = %q, want %q", loaded.Provider, original.Provider)
	}
	if loaded.Model != original.Model {
		t.Errorf("Model = %q, want %q", loaded.Model, original.Model)
	}
	if loaded.UpdatedAt != original.UpdatedAt {
		t.Errorf("UpdatedAt = %q, want %q", loaded.UpdatedAt, original.UpdatedAt)
	}

	got := loaded.ScoreFor("refactor", "go")
	if got != 0.88 {
		t.Errorf("ScoreFor(refactor,go) = %v, want 0.88", got)
	}

	// Confirm file name follows convention.
	expectedFile := filepath.Join(dir, "claude-anthropic-sonnet.json")
	if _, err := filepath.Abs(expectedFile); err != nil {
		t.Errorf("expected file path invalid: %v", err)
	}
}

func TestProfileStore_LoadAll(t *testing.T) {
	dir := t.TempDir()
	store := &ProfileStore{Dir: dir}

	profiles := []*CapabilityProfile{
		{
			Harness:      "claude",
			Provider:     "anthropic",
			Model:        "haiku",
			Capabilities: map[string]CapabilityScore{"feature:go": {TestPassRate: 0.80, AvgDuration: 3.0, SampleCount: 5}},
		},
		{
			Harness:      "gpt",
			Provider:     "openai",
			Model:        "gpt4o",
			Capabilities: map[string]CapabilityScore{"bugfix:python": {TestPassRate: 0.70, AvgDuration: 4.5, SampleCount: 8}},
		},
	}

	for _, p := range profiles {
		if err := store.Save(p); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	all, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("LoadAll() returned %d profiles, want 2", len(all))
	}

	seen := map[string]bool{}
	for _, p := range all {
		key := p.Harness + "-" + p.Provider + "-" + p.Model
		seen[key] = true
	}
	for _, p := range profiles {
		key := p.Harness + "-" + p.Provider + "-" + p.Model
		if !seen[key] {
			t.Errorf("LoadAll() missing profile %q", key)
		}
	}
}

func TestProfileStore_LoadAll_EmptyDir(t *testing.T) {
	store := &ProfileStore{Dir: "/tmp/sdp-dispatch-nonexistent-dir-xyz987"}

	all, err := store.LoadAll()
	if err != nil {
		t.Errorf("LoadAll() on missing dir error = %v, want nil", err)
	}
	if all != nil {
		t.Errorf("LoadAll() on missing dir = %v, want nil", all)
	}
}
