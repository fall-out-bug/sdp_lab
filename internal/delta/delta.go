// Package delta implements phase delta artifacts in OpenSpec-style ADDED/MODIFIED/REMOVED format.
// Delta artifacts capture what changed during a phase transition, providing traceability
// and auditability for AI-SDLC workflows.
package delta

import (
	"fmt"
	"strings"
	"time"
)

// Disclosure represents the visibility level of a block.
type Disclosure string

const (
	DisclosurePublic       Disclosure = "public"
	DisclosurePrivate      Disclosure = "private"
	DisclosureConfidential Disclosure = "confidential"
)

// Block represents a logical group of related files (e.g., a feature, component, or module).
type Block struct {
	Title       string     `json:"title" yaml:"title"`
	Description string     `json:"description" yaml:"description"`
	Files       []string   `json:"files" yaml:"files"`
	Disclosure  Disclosure `json:"disclosure" yaml:"disclosure"`
}

// Delta represents a phase transition delta artifact.
type Delta struct {
	Phase     string    `json:"phase" yaml:"phase"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// Optional identifiers for traceability
	FeatureID     string `json:"feature_id,omitempty" yaml:"feature_id,omitempty"`
	WorkstreamID  string `json:"ws_id,omitempty" yaml:"ws_id,omitempty"`
	RunID         string `json:"run_id,omitempty" yaml:"run_id,omitempty"`

	Added     []Block `json:"added" yaml:"added"`
	Modified  []Block `json:"modified" yaml:"modified"`
	Removed   []Block `json:"removed" yaml:"removed"`
	Rationale string  `json:"rationale" yaml:"rationale"`
}

// Option configures a Delta during construction.
type Option func(*Delta)

// WithFeatureID sets the feature identifier.
func WithFeatureID(id string) Option {
	return func(d *Delta) {
		d.FeatureID = id
	}
}

// WithWorkstreamID sets the workstream identifier.
func WithWorkstreamID(id string) Option {
	return func(d *Delta) {
		d.WorkstreamID = id
	}
}

// WithRunID sets the run identifier.
func WithRunID(id string) Option {
	return func(d *Delta) {
		d.RunID = id
	}
}

// NewDelta creates a new Delta for the given phase.
func NewDelta(phase string, opts ...Option) *Delta {
	d := &Delta{
		Phase:     phase,
		Timestamp: time.Now().UTC(),
		Added:     make([]Block, 0),
		Modified:  make([]Block, 0),
		Removed:   make([]Block, 0),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Add adds a new block to the delta.
func (d *Delta) Add(block Block) {
	d.Added = append(d.Added, block)
}

// AddModified adds a modified block to the delta.
func (d *Delta) AddModified(block Block) {
	d.Modified = append(d.Modified, block)
}

// AddRemoved adds a removed block to the delta.
func (d *Delta) AddRemoved(block Block) {
	d.Removed = append(d.Removed, block)
}

// SetRationale sets the rationale for the delta.
func (d *Delta) SetRationale(rationale string) {
	d.Rationale = rationale
}

// RenderMarkdown renders the delta as a markdown document with YAML frontmatter.
func (d *Delta) RenderMarkdown() string {
	var sb strings.Builder

	// Write frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("phase: %s\n", d.Phase))
	sb.WriteString(fmt.Sprintf("feature_id: %s\n", d.FeatureID))
	sb.WriteString(fmt.Sprintf("ws_id: %s\n", d.WorkstreamID))
	sb.WriteString(fmt.Sprintf("run_id: %s\n", d.RunID))
	sb.WriteString(fmt.Sprintf("timestamp: %s\n", d.Timestamp.Format(time.RFC3339)))
	sb.WriteString("---\n")
	sb.WriteString("\n")

	// Title
	sb.WriteString(fmt.Sprintf("# Phase: %s — Delta\n\n", d.Phase))

	// Added section
	if len(d.Added) > 0 {
		sb.WriteString("## ADDED\n\n")
		for _, block := range d.Added {
			disclosureTag := ""
			if block.Disclosure != "" {
				disclosureTag = fmt.Sprintf(" [%s]", block.Disclosure)
			}
			sb.WriteString(fmt.Sprintf("### %s%s\n", block.Title, disclosureTag))
			if block.Description != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", block.Description))
			}
			if len(block.Files) > 0 {
				sb.WriteString(fmt.Sprintf("Files: %s\n\n", strings.Join(block.Files, ", ")))
			}
		}
	}

	// Modified section
	if len(d.Modified) > 0 {
		sb.WriteString("## MODIFIED\n\n")
		for _, block := range d.Modified {
			disclosureTag := ""
			if block.Disclosure != "" {
				disclosureTag = fmt.Sprintf(" [%s]", block.Disclosure)
			}
			sb.WriteString(fmt.Sprintf("### %s%s\n", block.Title, disclosureTag))
			if block.Description != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", block.Description))
			}
			if len(block.Files) > 0 {
				sb.WriteString(fmt.Sprintf("Files: %s\n\n", strings.Join(block.Files, ", ")))
			}
		}
	}

	// Removed section
	if len(d.Removed) > 0 {
		sb.WriteString("## REMOVED\n\n")
		for _, block := range d.Removed {
			disclosureTag := ""
			if block.Disclosure != "" {
				disclosureTag = fmt.Sprintf(" [%s]", block.Disclosure)
			}
			sb.WriteString(fmt.Sprintf("### %s%s\n", block.Title, disclosureTag))
			if block.Description != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", block.Description))
			}
			if len(block.Files) > 0 {
				sb.WriteString(fmt.Sprintf("Files: %s\n\n", strings.Join(block.Files, ", ")))
			}
		}
	}

	// Rationale section
	if d.Rationale != "" {
		sb.WriteString("## Rationale\n\n")
		sb.WriteString(d.Rationale)
		sb.WriteString("\n")
	}

	return sb.String()
}

// IsEmpty returns true if the delta contains no changes.
func (d *Delta) IsEmpty() bool {
	return len(d.Added) == 0 && len(d.Modified) == 0 && len(d.Removed) == 0
}

// HasChanges returns true if the delta contains any changes.
func (d *Delta) HasChanges() bool {
	return !d.IsEmpty()
}

// TotalBlocks returns the total number of blocks across all categories.
func (d *Delta) TotalBlocks() int {
	return len(d.Added) + len(d.Modified) + len(d.Removed)
}

// TotalFiles returns the total number of files across all blocks.
func (d *Delta) TotalFiles() int {
	total := 0
	for _, b := range d.Added {
		total += len(b.Files)
	}
	for _, b := range d.Modified {
		total += len(b.Files)
	}
	for _, b := range d.Removed {
		total += len(b.Files)
	}
	return total
}

// FilterByDisclosure returns all blocks matching the given disclosure level
// across Added, Modified, and Removed.
func (d *Delta) FilterByDisclosure(level Disclosure) []Block {
	var result []Block

	for _, block := range d.Added {
		if block.Disclosure == level {
			result = append(result, block)
		}
	}
	for _, block := range d.Modified {
		if block.Disclosure == level {
			result = append(result, block)
		}
	}
	for _, block := range d.Removed {
		if block.Disclosure == level {
			result = append(result, block)
		}
	}

	return result
}
