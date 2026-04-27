package dispatch

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

// TestAvailabilityFactor validates the availability factor bucketing logic.
func TestAvailabilityFactor(t *testing.T) {
	tests := []struct {
		name   string
		limits *harness.Limits
		want   float64
	}{
		{
			name:   "nil limits returns 1.0",
			limits: nil,
			want:   1.0,
		},
		{
			name:   "Total=0 returns 1.0",
			limits: &harness.Limits{Total: 0, Used: 0},
			want:   1.0,
		},
		{
			name:   "0% used returns 1.0",
			limits: &harness.Limits{Total: 100, Used: 0},
			want:   1.0,
		},
		{
			name:   "50% used returns 1.0",
			limits: &harness.Limits{Total: 100, Used: 50},
			want:   1.0,
		},
		{
			name:   "70% used returns 1.0 (boundary <=70 is full)",
			limits: &harness.Limits{Total: 100, Used: 70},
			want:   1.0,
		},
		{
			name:   "71% used returns 0.5",
			limits: &harness.Limits{Total: 100, Used: 71},
			want:   0.5,
		},
		{
			name:   "85% used returns 0.5",
			limits: &harness.Limits{Total: 100, Used: 85},
			want:   0.5,
		},
		{
			name:   "90% used returns 0.5 (boundary <=90 is 0.5)",
			limits: &harness.Limits{Total: 100, Used: 90},
			want:   0.5,
		},
		{
			name:   "91% used returns 0.1",
			limits: &harness.Limits{Total: 100, Used: 91},
			want:   0.1,
		},
		{
			name:   "100% used returns 0.0",
			limits: &harness.Limits{Total: 100, Used: 100},
			want:   0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AvailabilityFactor(tc.limits)
			if got != tc.want {
				t.Errorf("AvailabilityFactor(%+v) = %v, want %v", tc.limits, got, tc.want)
			}
		})
	}
}

// fakeProvider is a test double for harness.Provider.
type fakeProvider struct {
	name   string
	limits *harness.Limits
	err    error
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) CheckLimits(_ context.Context) (*harness.Limits, error) {
	return f.limits, f.err
}

func (f *fakeProvider) Models() []string { return nil }

// TestLimitsCheckerCheckAll validates that CheckAll aggregates provider results correctly.
func TestLimitsCheckerCheckAll(t *testing.T) {
	okLimits := &harness.Limits{Total: 1000, Used: 500, Window: "1h", Source: "api"}
	warnLimits := &harness.Limits{Total: 100, Used: 80, Window: "1h", Source: "api"}

	t.Run("all providers succeed", func(t *testing.T) {
		lc := &LimitsChecker{
			Providers: []harness.Provider{
				&fakeProvider{name: "provider-a", limits: okLimits},
				&fakeProvider{name: "provider-b", limits: warnLimits},
			},
		}

		result := lc.CheckAll(context.Background())

		if len(result) != 2 {
			t.Fatalf("expected 2 results, got %d", len(result))
		}
		if result["provider-a"] != okLimits {
			t.Errorf("provider-a limits mismatch")
		}
		if result["provider-b"] != warnLimits {
			t.Errorf("provider-b limits mismatch")
		}
	})

	t.Run("failed provider is skipped", func(t *testing.T) {
		lc := &LimitsChecker{
			Providers: []harness.Provider{
				&fakeProvider{name: "provider-ok", limits: okLimits},
				&fakeProvider{name: "provider-fail", err: errors.New("connection refused")},
			},
		}

		result := lc.CheckAll(context.Background())

		if len(result) != 1 {
			t.Fatalf("expected 1 result (failed provider skipped), got %d", len(result))
		}
		if _, ok := result["provider-ok"]; !ok {
			t.Errorf("expected provider-ok to be present")
		}
		if _, ok := result["provider-fail"]; ok {
			t.Errorf("expected provider-fail to be absent")
		}
	})

	t.Run("empty providers returns empty map", func(t *testing.T) {
		lc := &LimitsChecker{Providers: []harness.Provider{}}
		result := lc.CheckAll(context.Background())
		if len(result) != 0 {
			t.Fatalf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("all providers fail returns empty map", func(t *testing.T) {
		lc := &LimitsChecker{
			Providers: []harness.Provider{
				&fakeProvider{name: "p1", err: errors.New("timeout")},
				&fakeProvider{name: "p2", err: errors.New("auth error")},
			},
		}
		result := lc.CheckAll(context.Background())
		if len(result) != 0 {
			t.Fatalf("expected empty map, got %d entries", len(result))
		}
	})
}

// TestFormatLimitsTable validates the human-readable output of FormatLimitsTable.
func TestFormatLimitsTable(t *testing.T) {
	tests := []struct {
		name        string
		limits      map[string]*harness.Limits
		wantContain []string
	}{
		{
			name:        "empty map returns header only",
			limits:      map[string]*harness.Limits{},
			wantContain: []string{"Provider"},
		},
		{
			name:        "ok status for low usage",
			limits:      map[string]*harness.Limits{"provider-a": {Total: 1000, Used: 100, Window: "1h"}},
			wantContain: []string{"provider-a", "ok"},
		},
		{
			name:        "warning status for 71-90% usage",
			limits:      map[string]*harness.Limits{"p": {Total: 100, Used: 80, Window: "1h"}},
			wantContain: []string{"warning"},
		},
		{
			name:        "critical status for 91-99% usage",
			limits:      map[string]*harness.Limits{"p": {Total: 100, Used: 95, Window: "1h"}},
			wantContain: []string{"critical"},
		},
		{
			name:        "exhausted status for 100% usage",
			limits:      map[string]*harness.Limits{"p": {Total: 100, Used: 100, Window: "1h"}},
			wantContain: []string{"exhausted"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := FormatLimitsTable(tc.limits)
			for _, want := range tc.wantContain {
				if !strings.Contains(out, want) {
					t.Errorf("FormatLimitsTable output missing %q, got: %q", want, out)
				}
			}
		})
	}
}
