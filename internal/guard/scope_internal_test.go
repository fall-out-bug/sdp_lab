package guard

import "testing"

func TestInScope(t *testing.T) {
	tests := []struct {
		file   string
		scope  []string
		inScope bool
	}{
		{"internal/guard/scope_check.go", []string{"internal/guard/scope_check.go"}, true},
		{"internal/guard/other.go", []string{"internal/guard/scope_check.go"}, false},
		{"sdp/sdp-plugin/internal/verify/verifier.go", []string{"sdp/sdp-plugin/internal/verify/"}, true},
		{"sdp/sdp-plugin/internal/verify/adapters.go", []string{"sdp/sdp-plugin/internal/verify/"}, true},
		{"sdp/sdp-plugin/internal/quality/coverage.go", []string{"sdp/sdp-plugin/internal/quality/"}, true},
		{"sdp", []string{"sdp"}, true},
		{"sdp/sdp-plugin/cmd/sdp/main.go", []string{"sdp"}, true},
		{"docs/workstreams/INDEX.md", []string{"docs/workstreams/INDEX.md"}, true},
		{"docs/workstreams/backlog/00-053-01.md", []string{"docs/workstreams/backlog/"}, true},
	}
	for _, tt := range tests {
		got := inScope(tt.file, tt.scope)
		if got != tt.inScope {
			t.Errorf("inScope(%q, %v) = %v, want %v", tt.file, tt.scope, got, tt.inScope)
		}
	}
}
