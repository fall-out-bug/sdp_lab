package delta

import (
	"strings"
	"testing"
	"time"
)

func TestNewDelta(t *testing.T) {
	d := NewDelta("plan")
	if d.Phase != "plan" {
		t.Errorf("Phase = %q, want %q", d.Phase, "plan")
	}
	if d.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if !d.IsEmpty() {
		t.Error("New delta should be empty")
	}
}

func TestNewDeltaWithOptions(t *testing.T) {
	d := NewDelta("plan",
		WithFeatureID("F134"),
		WithWorkstreamID("00-134-02"),
		WithRunID("run-123"),
	)

	if d.FeatureID != "F134" {
		t.Errorf("FeatureID = %q, want %q", d.FeatureID, "F134")
	}
	if d.WorkstreamID != "00-134-02" {
		t.Errorf("WorkstreamID = %q, want %q", d.WorkstreamID, "00-134-02")
	}
	if d.RunID != "run-123" {
		t.Errorf("RunID = %q, want %q", d.RunID, "run-123")
	}
}

func TestDelta_Add(t *testing.T) {
	d := NewDelta("plan")

	block := Block{
		Title:       "Test Component",
		Description: "A test component",
		Files:       []string{"test.go", "test_test.go"},
	}

	d.Add(block)

	if len(d.Added) != 1 {
		t.Errorf("Added length = %d, want 1", len(d.Added))
	}
	if d.Added[0].Title != "Test Component" {
		t.Errorf("Added[0].Title = %q, want %q", d.Added[0].Title, "Test Component")
	}
	if d.IsEmpty() {
		t.Error("Delta with additions should not be empty")
	}
	if !d.HasChanges() {
		t.Error("Delta with additions should have changes")
	}
}

func TestDelta_AddModified(t *testing.T) {
	d := NewDelta("plan")

	block := Block{
		Title: "Modified Component",
		Files: []string{"existing.go"},
	}

	d.AddModified(block)

	if len(d.Modified) != 1 {
		t.Errorf("Modified length = %d, want 1", len(d.Modified))
	}
	if d.Modified[0].Title != "Modified Component" {
		t.Errorf("Modified[0].Title = %q, want %q", d.Modified[0].Title, "Modified Component")
	}
	if d.IsEmpty() {
		t.Error("Delta with modifications should not be empty")
	}
}

func TestDelta_AddRemoved(t *testing.T) {
	d := NewDelta("plan")

	block := Block{
		Title: "Deprecated Component",
		Files: []string{"old.go"},
	}

	d.AddRemoved(block)

	if len(d.Removed) != 1 {
		t.Errorf("Removed length = %d, want 1", len(d.Removed))
	}
	if d.Removed[0].Title != "Deprecated Component" {
		t.Errorf("Removed[0].Title = %q, want %q", d.Removed[0].Title, "Deprecated Component")
	}
	if d.IsEmpty() {
		t.Error("Delta with removals should not be empty")
	}
}

func TestDelta_SetRationale(t *testing.T) {
	d := NewDelta("plan")
	rationale := "Implemented initial feature set"
	d.SetRationale(rationale)

	if d.Rationale != rationale {
		t.Errorf("Rationale = %q, want %q", d.Rationale, rationale)
	}
}

func TestDelta_TotalBlocks(t *testing.T) {
	d := NewDelta("plan")
	d.Add(Block{Title: "A"})
	d.Add(Block{Title: "B"})
	d.AddModified(Block{Title: "C"})
	d.AddRemoved(Block{Title: "D"})

	if got := d.TotalBlocks(); got != 4 {
		t.Errorf("TotalBlocks() = %d, want 4", got)
	}
}

func TestDelta_TotalFiles(t *testing.T) {
	d := NewDelta("plan")
	d.Add(Block{Files: []string{"a.go", "b.go"}})
	d.AddModified(Block{Files: []string{"c.go"}})
	d.AddRemoved(Block{Files: []string{"d.go", "e.go", "f.go"}})

	if got := d.TotalFiles(); got != 6 {
		t.Errorf("TotalFiles() = %d, want 6", got)
	}
}

func TestDelta_RenderMarkdown_Empty(t *testing.T) {
	d := NewDelta("plan")
	md := d.RenderMarkdown()

	// Check frontmatter
	if !strings.Contains(md, "phase: plan") {
		t.Error("Rendered markdown should contain phase in frontmatter")
	}
	if !strings.Contains(md, "feature_id: ") {
		t.Error("Rendered markdown should contain feature_id in frontmatter")
	}
	if !strings.Contains(md, "timestamp:") {
		t.Error("Rendered markdown should contain timestamp in frontmatter")
	}

	// Check title
	if !strings.Contains(md, "# Phase: plan — Delta") {
		t.Error("Rendered markdown should contain phase title")
	}

	// Empty delta should not have change sections
	if strings.Contains(md, "## ADDED") {
		t.Error("Empty delta should not render ADDED section")
	}
	if strings.Contains(md, "## MODIFIED") {
		t.Error("Empty delta should not render MODIFIED section")
	}
	if strings.Contains(md, "## REMOVED") {
		t.Error("Empty delta should not render REMOVED section")
	}
}

func TestDelta_RenderMarkdown_Full(t *testing.T) {
	d := NewDelta("plan",
		WithFeatureID("F134"),
		WithWorkstreamID("00-134-02"),
		WithRunID("run-abc"),
	)

	d.Add(Block{
		Title:       "New Feature",
		Description: "Initial implementation",
		Files:       []string{"feature.go", "feature_test.go"},
	})

	d.AddModified(Block{
		Title: "Existing Component",
		Files: []string{"existing.go"},
	})

	d.AddRemoved(Block{
		Title: "Deprecated Code",
		Files: []string{"old.go"},
	})

	d.SetRationale("Implemented core functionality as per requirements")

	md := d.RenderMarkdown()

	// Check frontmatter with IDs
	if !strings.Contains(md, "feature_id: F134") {
		t.Error("Rendered markdown should contain feature_id F134")
	}
	if !strings.Contains(md, "ws_id: 00-134-02") {
		t.Error("Rendered markdown should contain ws_id")
	}
	if !strings.Contains(md, "run_id: run-abc") {
		t.Error("Rendered markdown should contain run_id")
	}

	// Check sections
	if !strings.Contains(md, "## ADDED") {
		t.Error("Rendered markdown should contain ADDED section")
	}
	if !strings.Contains(md, "## MODIFIED") {
		t.Error("Rendered markdown should contain MODIFIED section")
	}
	if !strings.Contains(md, "## REMOVED") {
		t.Error("Rendered markdown should contain REMOVED section")
	}
	if !strings.Contains(md, "## Rationale") {
		t.Error("Rendered markdown should contain Rationale section")
	}

	// Check content
	if !strings.Contains(md, "### New Feature") {
		t.Error("Rendered markdown should contain New Feature block")
	}
	if !strings.Contains(md, "Initial implementation") {
		t.Error("Rendered markdown should contain block description")
	}
	if !strings.Contains(md, "Files: feature.go, feature_test.go") {
		t.Error("Rendered markdown should contain file list")
	}
	if !strings.Contains(md, "Implemented core functionality") {
		t.Error("Rendered markdown should contain rationale text")
	}
}

func TestDelta_RenderMarkdown_BlockWithoutDescription(t *testing.T) {
	d := NewDelta("plan")
	d.Add(Block{
		Title: "Minimal Block",
		Files: []string{"file.go"},
	})

	md := d.RenderMarkdown()

	if !strings.Contains(md, "### Minimal Block") {
		t.Error("Should render block title")
	}
	// Files should still be rendered
	if !strings.Contains(md, "Files: file.go") {
		t.Error("Should render files even without description")
	}
}

func TestDelta_RenderMarkdown_BlockWithoutFiles(t *testing.T) {
	d := NewDelta("plan")
	d.Add(Block{
		Title:       "Metadata Only",
		Description: "Just a note",
	})

	md := d.RenderMarkdown()

	if !strings.Contains(md, "### Metadata Only") {
		t.Error("Should render block title")
	}
	if !strings.Contains(md, "Just a note") {
		t.Error("Should render description")
	}
	// Should not have "Files:" line
	if strings.Contains(md, "Files:") {
		t.Error("Should not render Files: line when no files")
	}
}

func TestDelta_RenderMarkdown_TimestampFormat(t *testing.T) {
	// Create delta with fixed timestamp
	fixedTime := time.Date(2026, 4, 19, 12, 30, 45, 0, time.UTC)
	d := NewDelta("plan")
	d.Timestamp = fixedTime

	md := d.RenderMarkdown()

	// Check for RFC3339 format
	if !strings.Contains(md, "2026-04-19T12:30:45Z") {
		t.Errorf("Timestamp should be in RFC3339 format, got: %s", md)
	}
}

func TestDelta_MultipleBlocksPerSection(t *testing.T) {
	d := NewDelta("plan")
	d.Add(Block{Title: "Feature A", Files: []string{"a.go"}})
	d.Add(Block{Title: "Feature B", Files: []string{"b.go"}})
	d.Add(Block{Title: "Feature C", Files: []string{"c.go"}})

	md := d.RenderMarkdown()

	// Count occurrences of "### " which indicates block titles
	count := strings.Count(md, "### ")
	if count != 3 {
		t.Errorf("Expected 3 block titles, got %d", count)
	}
}

func TestDelta_IsEmpty_True(t *testing.T) {
	d := NewDelta("plan")
	if !d.IsEmpty() {
		t.Error("New delta should be empty")
	}
	if d.HasChanges() {
		t.Error("New delta should not have changes")
	}
}

func TestDelta_IsEmpty_False(t *testing.T) {
	d := NewDelta("plan")
	d.Add(Block{Title: "X"})
	if d.IsEmpty() {
		t.Error("Delta with additions should not be empty")
	}
}

func TestDelta_TotalBlocks_Empty(t *testing.T) {
	d := NewDelta("plan")
	if got := d.TotalBlocks(); got != 0 {
		t.Errorf("TotalBlocks() = %d, want 0", got)
	}
}

func TestDelta_TotalFiles_Empty(t *testing.T) {
	d := NewDelta("plan")
	if got := d.TotalFiles(); got != 0 {
		t.Errorf("TotalFiles() = %d, want 0", got)
	}

	// Blocks with no files
	d.Add(Block{Title: "No files"})
	if got := d.TotalFiles(); got != 0 {
		t.Errorf("TotalFiles() with empty blocks = %d, want 0", got)
	}
}
