package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session holds all pipeline state for one discovery run.
type Session struct {
	Slug  string
	Date  string
	Frame *FrameResult
	Scan  *ScanResult
}

// NewSession creates a new Session with today's date and a slug derived from idea.
func NewSession(idea string) *Session {
	return &Session{
		Slug: slugify(idea),
		Date: time.Now().Format("2006-01-02"),
	}
}

// slugify converts a string to a URL-safe slug (max 40 chars).
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-':
			b.WriteRune('-')
		}
	}
	slug := b.String()
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return strings.Trim(slug, "-")
}

// WriteArtifacts writes frame and scan markdown files to dir.
func WriteArtifacts(dir string, s *Session) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	prefix := filepath.Join(dir, s.Date+"-"+s.Slug)

	if s.Frame != nil {
		if err := writeFrame(prefix+"-frame.md", s.Frame); err != nil {
			return err
		}
	}
	if s.Scan != nil {
		if err := writeScan(prefix+"-scan.md", s.Scan); err != nil {
			return err
		}
	}
	return nil
}

func writeFrame(path string, f *FrameResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Frame\n\n")
	fmt.Fprintf(&b, "**Raw idea:** %s\n\n", f.RawIdea)
	fmt.Fprintf(&b, "## Problem Statement\n\n%s\n\n", f.ProblemStatement)
	fmt.Fprintf(&b, "## Jobs to Be Done\n\n")
	for _, j := range f.Jobs {
		fmt.Fprintf(&b, "- %s\n", j)
	}
	fmt.Fprintf(&b, "\n**Appetite:** %s\n\n**Scope:** %s\n", f.Appetite, f.Scope)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeScan(path string, r *ScanResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery Scan\n\n")
	fmt.Fprintf(&b, "**Whitespace:** %s\n\n", r.Whitespace)
	fmt.Fprintf(&b, "## Landscape\n\n")
	fmt.Fprintf(&b, "| Tool | Disposition | Coverage | Flagged |\n")
	fmt.Fprintf(&b, "|---|---|---|---|\n")
	for _, item := range r.Items {
		flagged := "—"
		if item.DepthFlag != nil && item.DepthFlag.Flagged {
			flagged = "⚠️ " + item.DepthFlag.Reason
		}
		fmt.Fprintf(&b, "| %s | %s | %.2f | %s |\n",
			item.Name, item.Disposition, item.CoverageScore, flagged)
	}
	// Append raw JSON for downstream use
	raw, _ := json.MarshalIndent(r, "", "  ")
	fmt.Fprintf(&b, "\n```json\n%s\n```\n", raw)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
