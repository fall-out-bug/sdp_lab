package harness_test

import (
	"context"
	"testing"
	"time"

	"sdp_dev/internal/dispatch/harness"
)

// fakeHarness implements the Harness interface for testing.
type fakeHarness struct {
	name       string
	available  bool
	providers  []string
}

func (f *fakeHarness) Name() string { return f.name }

func (f *fakeHarness) Available() bool { return f.available }

func (f *fakeHarness) SupportedProviders() []string { return f.providers }

func (f *fakeHarness) Spawn(_ context.Context, _ harness.SpawnOpts) (*harness.Process, error) {
	ch := make(chan harness.Result, 1)
	ch <- harness.Result{ExitCode: 0, Duration: time.Millisecond, Output: "ok"}
	return &harness.Process{
		HarnessName: f.name,
		PID:         1234,
		Worktree:    "/tmp/test",
		StartedAt:   time.Now(),
		Done:        ch,
	}, nil
}

// TestRegistry_Get verifies register, get, and get-nonexistent behaviour.
func TestRegistry_Get(t *testing.T) {
	tests := []struct {
		name     string
		register []string
		get      string
		wantNil  bool
	}{
		{
			name:     "get registered harness returns harness",
			register: []string{"alpha"},
			get:      "alpha",
			wantNil:  false,
		},
		{
			name:     "get nonexistent returns nil",
			register: []string{"alpha"},
			get:      "beta",
			wantNil:  true,
		},
		{
			name:     "empty registry returns nil",
			register: []string{},
			get:      "any",
			wantNil:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := harness.NewRegistry()
			for _, n := range tc.register {
				r.Register(&fakeHarness{name: n, available: true})
			}
			got := r.Get(tc.get)
			if tc.wantNil && got != nil {
				t.Errorf("Get(%q) = %v, want nil", tc.get, got)
			}
			if !tc.wantNil && got == nil {
				t.Errorf("Get(%q) = nil, want non-nil", tc.get)
			}
			if !tc.wantNil && got != nil && got.Name() != tc.get {
				t.Errorf("Get(%q).Name() = %q, want %q", tc.get, got.Name(), tc.get)
			}
		})
	}
}

// TestRegistry_Available verifies that only available harnesses are returned.
func TestRegistry_Available(t *testing.T) {
	tests := []struct {
		name      string
		harnesses []struct {
			n string
			a bool
		}
		wantNames []string
	}{
		{
			name: "all available",
			harnesses: []struct {
				n string
				a bool
			}{{"h1", true}, {"h2", true}},
			wantNames: []string{"h1", "h2"},
		},
		{
			name: "none available",
			harnesses: []struct {
				n string
				a bool
			}{{"h1", false}, {"h2", false}},
			wantNames: []string{},
		},
		{
			name: "mixed availability",
			harnesses: []struct {
				n string
				a bool
			}{{"h1", true}, {"h2", false}, {"h3", true}},
			wantNames: []string{"h1", "h3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := harness.NewRegistry()
			for _, h := range tc.harnesses {
				r.Register(&fakeHarness{name: h.n, available: h.a})
			}
			got := r.Available()
			if len(got) != len(tc.wantNames) {
				t.Fatalf("Available() returned %d harnesses, want %d", len(got), len(tc.wantNames))
			}
			nameSet := make(map[string]bool)
			for _, h := range got {
				nameSet[h.Name()] = true
			}
			for _, wn := range tc.wantNames {
				if !nameSet[wn] {
					t.Errorf("Available() missing expected harness %q", wn)
				}
			}
		})
	}
}

// TestRegistry_ForProvider verifies that harnesses supporting a provider are returned.
func TestRegistry_ForProvider(t *testing.T) {
	tests := []struct {
		name      string
		harnesses []struct {
			n         string
			providers []string
		}
		provider  string
		wantNames []string
	}{
		{
			name: "single match",
			harnesses: []struct {
				n         string
				providers []string
			}{
				{"h1", []string{"openai", "zai"}},
				{"h2", []string{"anthropic"}},
			},
			provider:  "zai",
			wantNames: []string{"h1"},
		},
		{
			name: "multiple matches",
			harnesses: []struct {
				n         string
				providers []string
			}{
				{"h1", []string{"openai", "zai"}},
				{"h2", []string{"zai"}},
				{"h3", []string{"anthropic"}},
			},
			provider:  "zai",
			wantNames: []string{"h1", "h2"},
		},
		{
			name: "no match",
			harnesses: []struct {
				n         string
				providers []string
			}{
				{"h1", []string{"openai"}},
			},
			provider:  "zai",
			wantNames: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := harness.NewRegistry()
			for _, h := range tc.harnesses {
				r.Register(&fakeHarness{name: h.n, available: true, providers: h.providers})
			}
			got := r.ForProvider(tc.provider)
			if len(got) != len(tc.wantNames) {
				t.Fatalf("ForProvider(%q) returned %d harnesses, want %d", tc.provider, len(got), len(tc.wantNames))
			}
			nameSet := make(map[string]bool)
			for _, h := range got {
				nameSet[h.Name()] = true
			}
			for _, wn := range tc.wantNames {
				if !nameSet[wn] {
					t.Errorf("ForProvider(%q) missing expected harness %q", tc.provider, wn)
				}
			}
		})
	}
}

// TestLimits_UsagePercent verifies percentage calculation including edge cases.
func TestLimits_UsagePercent(t *testing.T) {
	tests := []struct {
		name  string
		total int
		used  int
		want  float64
	}{
		{
			name:  "zero used of 1000",
			total: 1000,
			used:  0,
			want:  0.0,
		},
		{
			name:  "500 used of 1000",
			total: 1000,
			used:  500,
			want:  0.5,
		},
		{
			name:  "total zero avoids division by zero",
			total: 0,
			used:  0,
			want:  0.0,
		},
		{
			name:  "full usage",
			total: 100,
			used:  100,
			want:  1.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := &harness.Limits{Total: tc.total, Used: tc.used}
			got := l.UsagePercent()
			if got != tc.want {
				t.Errorf("UsagePercent() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRegistry_All verifies All() returns all registered harness names.
func TestRegistry_All(t *testing.T) {
	r := harness.NewRegistry()
	r.Register(&fakeHarness{name: "h1"})
	r.Register(&fakeHarness{name: "h2"})
	r.Register(&fakeHarness{name: "h3"})

	got := r.All()
	if len(got) != 3 {
		t.Fatalf("All() returned %d names, want 3", len(got))
	}
	nameSet := map[string]bool{"h1": true, "h2": true, "h3": true}
	for _, n := range got {
		if !nameSet[n] {
			t.Errorf("All() contains unexpected name %q", n)
		}
	}
}
