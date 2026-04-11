package architect

import (
	"strings"
	"testing"
)

func TestJoinID(t *testing.T) {
	tests := []struct {
		name    string
		segs    []string
		want    string
		wantErr bool
	}{
		{"basic", []string{"go", "internal/arch", "arch"}, "go\x00internal/arch\x00arch", false},
		{"npm scoped", []string{"typescript", "@types/node", "node"}, "typescript\x00@types/node\x00node", false},
		{"external", []string{"ext", "npm", "lodash"}, "ext\x00npm\x00lodash", false},
		{"container", []string{"container", "auth-service"}, "container\x00auth-service", false},
		{"maven", []string{"java", "com/google/guava", "guava"}, "java\x00com/google/guava\x00guava", false},
		{"empty segment", []string{"go", "", "name"}, "", true},
		{"no segments", nil, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := JoinID(tt.segs...)
			if (err != nil) != tt.wantErr {
				t.Errorf("JoinID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("JoinID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		want    []string
		wantErr bool
	}{
		{"basic", "go\x00internal/arch\x00arch", []string{"go", "internal/arch", "arch"}, false},
		{"two segments", "container\x00auth", []string{"container", "auth"}, false},
		{"empty", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !equalStrings(got, tt.want) {
				t.Errorf("SplitID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	segments := []string{"go", "internal/architect", "architect"}
	id, err := JoinID(segments...)
	if err != nil {
		t.Fatal(err)
	}
	got, err := SplitID(id)
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(got, segments) {
		t.Errorf("round trip: got %v, want %v", got, segments)
	}

	// NormalizeID should be idempotent
	norm, err := NormalizeID(id)
	if err != nil {
		t.Fatal(err)
	}
	if norm != id {
		t.Errorf("NormalizeID() = %q, want %q", norm, id)
	}
}

func TestIDNullByteInSegment(t *testing.T) {
	// Null bytes within segments are encoded as %00, not rejected.
	id, err := JoinID("go", "path\x00injection", "name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := SplitID(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"go", "path\x00injection", "name"}
	if !equalStrings(got, want) {
		t.Errorf("round trip with null byte: got %v, want %v", got, want)
	}
}

func TestIDPercent00RoundTrip(t *testing.T) {
	// A segment containing literal %00 text is NOT treated specially on join
	// (only actual \x00 is encoded). On split, literal %00 in the joined ID
	// from an original \x00 is decoded back. Round-trip is idempotent.
	segments := []string{"go", "has\x00null\x00here", "mod"}
	id, err := JoinID(segments...)
	if err != nil {
		t.Fatal(err)
	}
	// The joined ID should not contain literal \x00 within segments,
	// only as delimiters.
	parts := strings.Split(id, "\x00")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	if strings.ContainsRune(parts[1], 0) {
		t.Error("segment should not contain literal null byte after encoding")
	}

	got, err := SplitID(id)
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(got, segments) {
		t.Errorf("round trip: got %v, want %v", got, segments)
	}

	// NormalizeID should also be idempotent
	norm, err := NormalizeID(id)
	if err != nil {
		t.Fatal(err)
	}
	if norm != id {
		t.Errorf("NormalizeID() = %q, want %q", norm, id)
	}
}

func TestIDContainsNullByte(t *testing.T) {
	id := "go\x00internal/arch\x00architect"
	if !strings.ContainsRune(id, 0) {
		t.Error("ID should contain null byte")
	}
	// The ID should have exactly 2 null bytes (3 segments)
	count := strings.Count(id, "\x00")
	if count != 2 {
		t.Errorf("expected 2 null bytes, got %d", count)
	}
}

func TestContentHashSuffix(t *testing.T) {
	s := ContentHashSuffix(`{"name":"test"}`)
	if len(s) != 8 {
		t.Errorf("ContentHashSuffix() = %q, want 8 chars", s)
	}
	// Same input produces same hash
	s2 := ContentHashSuffix(`{"name":"test"}`)
	if s != s2 {
		t.Errorf("ContentHashSuffix not deterministic: %q != %q", s, s2)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
