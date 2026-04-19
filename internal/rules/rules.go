package rules

import (
	"fmt"
	"sort"
	"strings"
)

// Source constants -- only these are valid origins for rules.
const (
	SourceObservedFailure = "observed-failure"
	SourceHumanAnnotated  = "human-annotated"
)

// allowedSources is the set of valid Source values.
var allowedSources = map[string]bool{
	SourceObservedFailure: true,
	SourceHumanAnnotated:  true,
}

// Severity constants.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Rule represents an observed-only rule generated from evidence.
type Rule struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Source      string `json:"source"`
	EvidenceRef string `json:"evidence_ref"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// Valid reports whether the rule has a recognized source and non-empty
// required fields.
func (r Rule) Valid() bool {
	if !allowedSources[r.Source] {
		return false
	}
	return r.ID != "" && r.Title != "" && r.EvidenceRef != ""
}

// Generator reads evidence sources and produces rules.
type Generator struct {
	evidenceDir string
}

// NewGenerator creates a Generator that reads from evidenceDir.
func NewGenerator(evidenceDir string) *Generator {
	return &Generator{evidenceDir: evidenceDir}
}

// Generate scans the evidence directory for failure patterns and produces
// deterministic rules. Each rule is backed by observed evidence; speculative
// rules are never emitted.
func (g *Generator) Generate() ([]Rule, error) {
	entries, err := ReadEvidenceDir(g.evidenceDir)
	if err != nil {
		return nil, fmt.Errorf("read evidence: %w", err)
	}

	failures := filterFailures(entries)
	if len(failures) == 0 {
		return nil, nil
	}

	groups := groupFailures(failures)
	rules := buildRules(groups, g.evidenceDir)

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	})

	return rules, nil
}

// failureGroup collects evidence entries that share a phase+summary pattern.
type failureGroup struct {
	Phase   string
	Summary string
	Entries []EvidenceEntry
}

// filterFailures returns only entries with verdict "fail" or "error".
func filterFailures(entries []EvidenceEntry) []EvidenceEntry {
	var out []EvidenceEntry
	for _, e := range entries {
		if e.Verdict == "fail" || e.Verdict == "error" {
			out = append(out, e)
		}
	}
	return out
}

// groupFailures clusters failures by (phase, summary) so similar failures
// produce a single rule.
func groupFailures(failures []EvidenceEntry) []failureGroup {
	index := make(map[string]*failureGroup)
	var keys []string

	for _, f := range failures {
		key := f.Phase + "|" + f.Summary
		if g, ok := index[key]; ok {
			g.Entries = append(g.Entries, f)
			continue
		}
		keys = append(keys, key)
		index[key] = &failureGroup{
			Phase:   f.Phase,
			Summary: f.Summary,
			Entries: []EvidenceEntry{f},
		}
	}

	groups := make([]failureGroup, 0, len(keys))
	for _, k := range keys {
		groups = append(groups, *index[k])
	}
	return groups
}

// buildRules converts failure groups into Rule values with deterministic IDs.
func buildRules(groups []failureGroup, evidenceDir string) []Rule {
	// Sort groups for deterministic ordering: by phase then summary.
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Phase != groups[j].Phase {
			return groups[i].Phase < groups[j].Phase
		}
		return groups[i].Summary < groups[j].Summary
	})

	rules := make([]Rule, 0, len(groups))
	for i, g := range groups {
		id := fmt.Sprintf("RULE-%03d", i+1)
		ref := buildEvidenceRef(g, evidenceDir)
		severity := severityForPhase(g.Phase)
		rules = append(rules, Rule{
			ID:          id,
			Title:       fmt.Sprintf("Observed failure in phase %s", g.Phase),
			Source:      SourceObservedFailure,
			EvidenceRef: ref,
			Severity:    severity,
			Description: g.Summary,
		})
	}
	return rules
}

// buildEvidenceRef constructs the evidence reference string for a group.
func buildEvidenceRef(g failureGroup, evidenceDir string) string {
	refs := make([]string, 0, len(g.Entries))
	for _, e := range g.Entries {
		path := e.FilePath
		if path == "" {
			path = fmt.Sprintf("%s/%s.json", evidenceDir, e.RunID)
		}
		refs = append(refs, path)
	}
	return SourceObservedFailure + ":" + strings.Join(refs, ",")
}

// severityForPhase maps a phase name to a severity level.
func severityForPhase(phase string) string {
	switch phase {
	case "build", "compile":
		return SeverityError
	case "test":
		return SeverityError
	case "lint", "vet":
		return SeverityWarning
	default:
		return SeverityWarning
	}
}
