package evals

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	legacyeval "sdp_dev/internal/eval"
	"sdp_dev/internal/kernel"

	"gopkg.in/yaml.v3"
)

// Case defines one eval scenario. Trace assertions are preferred.
// Transcript pattern checks remain as a compatibility fallback.
type Case struct {
	Name                     string                      `yaml:"name"`
	Skill                    string                      `yaml:"skill"`
	InputTrace               string                      `yaml:"input_trace,omitempty"`
	InputTranscript          string                      `yaml:"input_transcript,omitempty"`
	ExpectedTraceKinds       []kernel.TraceEventKind     `yaml:"expected_trace_kinds,omitempty"`
	ExpectedRoutingProviders []kernel.ProviderID         `yaml:"expected_routing_providers,omitempty"`
	ExpectedToolDecisions    []kernel.ToolPolicyDecision `yaml:"expected_tool_decisions,omitempty"`
	ExpectedArtifacts        []kernel.ArtifactType       `yaml:"expected_artifacts,omitempty"`
	ExpectedMemoryScopes     []kernel.MemoryScope        `yaml:"expected_memory_scopes,omitempty"`
	ForbiddenPatterns        []string                    `yaml:"forbidden_patterns,omitempty"`
	RequiredPatterns         []string                    `yaml:"required_patterns,omitempty"`
	Verdict                  string                      `yaml:"verdict,omitempty"` // PASS or FAIL for transcript fallback
}

// Result is the outcome of running one case.
type Result struct {
	Case   string
	Pass   bool
	Reason string
}

func (c Case) hasTraceAssertions() bool {
	return c.InputTrace != "" ||
		len(c.ExpectedTraceKinds) > 0 ||
		len(c.ExpectedRoutingProviders) > 0 ||
		len(c.ExpectedToolDecisions) > 0 ||
		len(c.ExpectedArtifacts) > 0 ||
		len(c.ExpectedMemoryScopes) > 0
}

func (c Case) toEvalCase() kernel.EvalCase {
	return kernel.EvalCase{
		ID:                       c.Name,
		Scenario:                 c.Name,
		Inputs:                   map[string]any{"skill": c.Skill},
		ExpectedTraceKinds:       c.ExpectedTraceKinds,
		ExpectedRoutingProviders: c.ExpectedRoutingProviders,
		ExpectedToolDecisions:    c.ExpectedToolDecisions,
		ExpectedArtifacts:        c.ExpectedArtifacts,
		ExpectedMemoryScopes:     c.ExpectedMemoryScopes,
	}
}

func (c Case) toLegacyCase() *legacyeval.Case {
	return &legacyeval.Case{
		Name:              c.Name,
		Skill:             c.Skill,
		InputTranscript:   c.InputTranscript,
		ForbiddenPatterns: c.ForbiddenPatterns,
		RequiredPatterns:  c.RequiredPatterns,
		Verdict:           c.Verdict,
	}
}

// LoadCases reads YAML case files from a directory.
func LoadCases(casesDir, skill string) ([]Case, error) {
	pattern := filepath.Join(casesDir, "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var cases []Case
	for _, p := range matches {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var c Case
		if err := yaml.Unmarshal(data, &c); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		if skill != "" && c.Skill != skill {
			continue
		}
		cases = append(cases, c)
	}
	return cases, nil
}

// Run executes all cases from a directory.
func Run(projectRoot, casesDir, skill string) ([]Result, error) {
	cases, err := LoadCases(casesDir, skill)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(cases))
	for _, c := range cases {
		results = append(results, RunCase(&c, projectRoot))
	}
	return results, nil
}

// RunCase executes a single case. Trace assertions take priority.
func RunCase(c *Case, projectRoot string) Result {
	if c.hasTraceAssertions() {
		return runTraceCase(c, projectRoot)
	}
	return legacyResult(legacyeval.RunCase(c.toLegacyCase(), projectRoot))
}

func legacyResult(r legacyeval.Result) Result {
	return Result{Case: r.Case, Pass: r.Pass, Reason: r.Reason}
}

func runTraceCase(c *Case, projectRoot string) Result {
	if c.InputTrace == "" {
		return Result{Case: c.Name, Pass: false, Reason: "trace case missing input_trace"}
	}

	events, err := loadTraceEvents(filepath.Join(projectRoot, c.InputTrace))
	if err != nil {
		return Result{Case: c.Name, Pass: false, Reason: fmt.Sprintf("load trace: %v", err)}
	}

	ev := c.toEvalCase()
	var failures []string

	if missing := missingTraceKinds(events, ev.ExpectedTraceKinds); len(missing) > 0 {
		failures = append(failures, "missing trace kinds: "+joinKinds(missing))
	}
	if missing := missingRoutingProviders(events, ev.ExpectedRoutingProviders); len(missing) > 0 {
		failures = append(failures, "missing routing providers: "+joinStrings(missing))
	}
	if missing := missingToolDecisions(events, ev.ExpectedToolDecisions); len(missing) > 0 {
		failures = append(failures, "missing tool decisions: "+joinStrings(missing))
	}
	if missing := missingArtifacts(events, ev.ExpectedArtifacts); len(missing) > 0 {
		failures = append(failures, "missing artifacts: "+joinStrings(missing))
	}
	if missing := missingMemoryScopes(events, ev.ExpectedMemoryScopes); len(missing) > 0 {
		failures = append(failures, "missing memory scopes: "+joinStrings(missing))
	}

	if len(failures) > 0 {
		return Result{Case: c.Name, Pass: false, Reason: strings.Join(failures, "; ")}
	}
	return Result{Case: c.Name, Pass: true}
}

func loadTraceEvents(path string) ([]kernel.TraceEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var runDoc struct {
		Events []kernel.TraceEvent `json:"events"`
	}
	if err := json.Unmarshal(data, &runDoc); err == nil && len(runDoc.Events) > 0 {
		return runDoc.Events, nil
	}

	var events []kernel.TraceEvent
	if err := json.Unmarshal(data, &events); err == nil && len(events) > 0 {
		return events, nil
	}

	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var evt kernel.TraceEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			return nil, fmt.Errorf("parse jsonl trace event: %w", err)
		}
		events = append(events, evt)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("no trace events found")
	}
	return events, nil
}

func missingTraceKinds(events []kernel.TraceEvent, expected []kernel.TraceEventKind) []kernel.TraceEventKind {
	seen := make(map[kernel.TraceEventKind]bool)
	for _, evt := range events {
		if evt.Kind != "" {
			seen[evt.Kind] = true
		}
	}
	var missing []kernel.TraceEventKind
	for _, want := range expected {
		if !seen[want] {
			missing = append(missing, want)
		}
	}
	return missing
}

func missingToolDecisions(events []kernel.TraceEvent, expected []kernel.ToolPolicyDecision) []string {
	seen := make(map[string]bool)
	for _, evt := range events {
		if evt.Kind != kernel.TraceEventTool {
			continue
		}
		if decision, ok := payloadString(evt.Payload, "decision", "tool_decision.decision", "result.decision"); ok {
			seen[decision] = true
		}
	}
	var missing []string
	for _, want := range expected {
		if !seen[string(want)] {
			missing = append(missing, string(want))
		}
	}
	return missing
}

func missingRoutingProviders(events []kernel.TraceEvent, expected []kernel.ProviderID) []string {
	seen := make(map[string]bool)
	for _, evt := range events {
		if evt.Kind != kernel.TraceEventRouting {
			continue
		}
		if provider, ok := payloadString(evt.Payload, "selected_provider", "decision.selected_provider", "route.selected_provider"); ok {
			seen[provider] = true
		}
	}
	var missing []string
	for _, want := range expected {
		if !seen[string(want)] {
			missing = append(missing, string(want))
		}
	}
	return missing
}

func missingArtifacts(events []kernel.TraceEvent, expected []kernel.ArtifactType) []string {
	seen := make(map[string]bool)
	for _, evt := range events {
		if evt.Kind != kernel.TraceEventArtifact {
			continue
		}
		if artifactType, ok := payloadString(evt.Payload, "type", "artifact_type", "artifact.type"); ok {
			seen[artifactType] = true
		}
	}
	var missing []string
	for _, want := range expected {
		if !seen[string(want)] {
			missing = append(missing, string(want))
		}
	}
	return missing
}

func missingMemoryScopes(events []kernel.TraceEvent, expected []kernel.MemoryScope) []string {
	seen := make(map[string]bool)
	for _, evt := range events {
		if evt.Kind != kernel.TraceEventMemory {
			continue
		}
		if scope, ok := payloadString(evt.Payload, "scope", "memory_scope", "memory.scope"); ok {
			seen[scope] = true
		}
	}
	var missing []string
	for _, want := range expected {
		if !seen[string(want)] {
			missing = append(missing, string(want))
		}
	}
	return missing
}

func payloadString(raw json.RawMessage, paths ...string) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", false
	}
	for _, path := range paths {
		if value, ok := nestedString(payload, strings.Split(path, ".")...); ok {
			return value, true
		}
	}
	return "", false
}

func nestedString(value any, path ...string) (string, bool) {
	current := value
	for _, step := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = m[step]
		if !ok {
			return "", false
		}
	}
	s, ok := current.(string)
	return s, ok
}

func joinKinds(kinds []kernel.TraceEventKind) string {
	items := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		items = append(items, string(kind))
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}

func joinStrings(items []string) string {
	sorted := append([]string(nil), items...)
	sort.Strings(sorted)
	return strings.Join(sorted, ", ")
}
