package providers

import (
	"context"
	"testing"
	"time"

	"sdp_dev/internal/dispatch/harness"
)

// fakeProvider implements harness.Provider for testing.
type fakeProvider struct {
	name   string
	models []string
}

func (f *fakeProvider) Name() string     { return f.name }
func (f *fakeProvider) Models() []string { return f.models }
func (f *fakeProvider) CheckLimits(_ context.Context) (*harness.Limits, error) {
	return &harness.Limits{Source: "fake", CheckedAt: time.Now()}, nil
}

func TestProviderRegistry_Get(t *testing.T) {
	tests := []struct {
		name     string
		register []string
		get      string
		wantNil  bool
	}{
		{"get registered", []string{"openai"}, "openai", false},
		{"get nonexistent", []string{"openai"}, "kimi", true},
		{"empty registry", []string{}, "any", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			for _, n := range tc.register {
				r.Register(&fakeProvider{name: n})
			}
			got := r.Get(tc.get)
			if tc.wantNil && got != nil {
				t.Errorf("Get(%q) = %v, want nil", tc.get, got)
			}
			if !tc.wantNil && got == nil {
				t.Errorf("Get(%q) = nil, want non-nil", tc.get)
			}
		})
	}
}

func TestProviderRegistry_All(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeProvider{name: "openai"})
	r.Register(&fakeProvider{name: "anthropic"})
	r.Register(&fakeProvider{name: "kimi"})
	got := r.All()
	if len(got) != 3 {
		t.Fatalf("All() returned %d names, want 3", len(got))
	}
	want := map[string]bool{"openai": true, "anthropic": true, "kimi": true}
	for _, n := range got {
		if !want[n] {
			t.Errorf("All() returned unexpected name %q", n)
		}
	}
}

func TestProviderRegistry_DuplicateOverwrite(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeProvider{name: "openai", models: []string{"gpt-5"}})
	r.Register(&fakeProvider{name: "openai", models: []string{"gpt-5", "o3"}})
	got := r.Get("openai")
	if got == nil {
		t.Fatal("Get(openai) = nil after duplicate register")
	}
	if len(got.Models()) != 2 {
		t.Errorf("expected overwritten provider with 2 models, got %d", len(got.Models()))
	}
}

func TestDefault_Singleton(t *testing.T) {
	a := Default()
	b := Default()
	if a != b {
		t.Errorf("Default() returned different instances on two calls")
	}
}
