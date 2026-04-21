package delta

import (
	"testing"
)

func TestFilterByDisclosure_Public(t *testing.T) {
	d := NewDelta("test")
	d.Add(Block{Title: "Public Block 1", Disclosure: DisclosurePublic})
	d.Add(Block{Title: "Private Block", Disclosure: DisclosurePrivate})
	d.AddModified(Block{Title: "Public Block 2", Disclosure: DisclosurePublic})
	d.AddRemoved(Block{Title: "Confidential Block", Disclosure: DisclosureConfidential})

	result := d.FilterByDisclosure(DisclosurePublic)

	if len(result) != 2 {
		t.Errorf("Expected 2 public blocks, got %d", len(result))
	}

	for _, block := range result {
		if block.Disclosure != DisclosurePublic {
			t.Errorf("Expected all blocks to be public, got %s", block.Disclosure)
		}
	}
}

func TestFilterByDisclosure_Private(t *testing.T) {
	d := NewDelta("test")
	d.Add(Block{Title: "Public Block", Disclosure: DisclosurePublic})
	d.Add(Block{Title: "Private Block", Disclosure: DisclosurePrivate})
	d.AddModified(Block{Title: "Another Private", Disclosure: DisclosurePrivate})

	result := d.FilterByDisclosure(DisclosurePrivate)

	if len(result) != 2 {
		t.Errorf("Expected 2 private blocks, got %d", len(result))
	}

	for _, block := range result {
		if block.Disclosure != DisclosurePrivate {
			t.Errorf("Expected all blocks to be private, got %s", block.Disclosure)
		}
	}
}

func TestFilterByDisclosure_Confidential(t *testing.T) {
	d := NewDelta("test")
	d.Add(Block{Title: "Public Block", Disclosure: DisclosurePublic})
	d.Add(Block{Title: "Confidential Block", Disclosure: DisclosureConfidential})
	d.AddRemoved(Block{Title: "Another Confidential", Disclosure: DisclosureConfidential})

	result := d.FilterByDisclosure(DisclosureConfidential)

	if len(result) != 2 {
		t.Errorf("Expected 2 confidential blocks, got %d", len(result))
	}

	for _, block := range result {
		if block.Disclosure != DisclosureConfidential {
			t.Errorf("Expected all blocks to be confidential, got %s", block.Disclosure)
		}
	}
}

func TestFilterByDisclosure_EmptyResult(t *testing.T) {
	d := NewDelta("test")
	d.Add(Block{Title: "Public Block", Disclosure: DisclosurePublic})
	d.Add(Block{Title: "Private Block", Disclosure: DisclosurePrivate})

	result := d.FilterByDisclosure(DisclosureConfidential)

	if len(result) != 0 {
		t.Errorf("Expected 0 confidential blocks, got %d", len(result))
	}
}

func TestFilterByDisclosure_DefaultEmptyString(t *testing.T) {
	d := NewDelta("test")
	d.Add(Block{Title: "No Disclosure", Disclosure: ""})
	d.Add(Block{Title: "Public Block", Disclosure: DisclosurePublic})

	// Filter for empty disclosure (default)
	result := d.FilterByDisclosure("")

	if len(result) != 1 {
		t.Errorf("Expected 1 block with empty disclosure, got %d", len(result))
	}

	if result[0].Title != "No Disclosure" {
		t.Errorf("Expected 'No Disclosure' block, got %s", result[0].Title)
	}
}

func TestFilterByDisclosure_AcrossAllCategories(t *testing.T) {
	d := NewDelta("test")
	d.Add(Block{Title: "Added Public", Disclosure: DisclosurePublic})
	d.AddModified(Block{Title: "Modified Public", Disclosure: DisclosurePublic})
	d.AddRemoved(Block{Title: "Removed Public", Disclosure: DisclosurePublic})
	d.Add(Block{Title: "Added Private", Disclosure: DisclosurePrivate})

	result := d.FilterByDisclosure(DisclosurePublic)

	if len(result) != 3 {
		t.Errorf("Expected 3 public blocks across all categories, got %d", len(result))
	}

	// Verify we got blocks from all three categories
	titles := make(map[string]bool)
	for _, block := range result {
		titles[block.Title] = true
	}

	if !titles["Added Public"] {
		t.Error("Missing 'Added Public' block")
	}
	if !titles["Modified Public"] {
		t.Error("Missing 'Modified Public' block")
	}
	if !titles["Removed Public"] {
		t.Error("Missing 'Removed Public' block")
	}
}

func TestRenderMarkdown_WithDisclosure(t *testing.T) {
	d := NewDelta("test", WithFeatureID("F001"))
	d.Add(Block{
		Title:       "Public Feature",
		Description: "This is public",
		Files:       []string{"file1.go"},
		Disclosure:  DisclosurePublic,
	})
	d.Add(Block{
		Title:       "Confidential Feature",
		Description: "This is confidential",
		Files:       []string{"file2.go"},
		Disclosure:  DisclosureConfidential,
	})

	markdown := d.RenderMarkdown()

	// Check that disclosure tags are present
	if !contains(markdown, "### Public Feature [public]") {
		t.Error("Expected '### Public Feature [public]' in markdown")
	}
	if !contains(markdown, "### Confidential Feature [confidential]") {
		t.Error("Expected '### Confidential Feature [confidential]' in markdown")
	}
}

func TestRenderMarkdown_WithoutDisclosure(t *testing.T) {
	d := NewDelta("test")
	d.Add(Block{
		Title:       "No Disclosure",
		Description: "This has no disclosure",
		Files:       []string{"file1.go"},
		Disclosure:  "",
	})

	markdown := d.RenderMarkdown()

	// Check that title appears without disclosure tag
	if !contains(markdown, "### No Disclosure") {
		t.Error("Expected '### No Disclosure' in markdown")
	}
	// Should NOT have the bracket format
	if contains(markdown, "### No Disclosure [") {
		t.Error("Should not have disclosure tag when disclosure is empty")
	}
}

func TestDisclosureConstants(t *testing.T) {
	// Verify constants have expected values
	if DisclosurePublic != "public" {
		t.Errorf("Expected DisclosurePublic to be 'public', got '%s'", DisclosurePublic)
	}
	if DisclosurePrivate != "private" {
		t.Errorf("Expected DisclosurePrivate to be 'private', got '%s'", DisclosurePrivate)
	}
	if DisclosureConfidential != "confidential" {
		t.Errorf("Expected DisclosureConfidential to be 'confidential', got '%s'", DisclosureConfidential)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
