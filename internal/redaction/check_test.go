package redaction

import "testing"

func TestCheckContentFindsViolations(t *testing.T) {
	content := "Public text\nInternal hostnames are private\nNo secrets here"
	violations := CheckContent(content)
	if len(violations) < 2 {
		t.Fatalf("expected violations, got %d", len(violations))
	}
}

func TestCheckContentClean(t *testing.T) {
	content := "Protocol interfaces and public schemas only"
	violations := CheckContent(content)
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d", len(violations))
	}
}
