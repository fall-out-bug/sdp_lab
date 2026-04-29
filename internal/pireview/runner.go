package pireview

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/executil"
)

// ReviewRun tracks a single pi-review execution.
type ReviewRun struct {
	RunID        string        `json:"run_id"`
	Feature      string        `json:"feature,omitempty"`
	Workstreams  []string      `json:"workstreams,omitempty"`
	Round        int           `json:"round"`
	Timestamp    string        `json:"timestamp"`
	Scope        RunScope      `json:"scope"`
	Context      RunContext    `json:"context_packet"`
	TestEvidence TestEvidence  `json:"test_evidence"`
	Models       []ModelResult `json:"models"`
	VerdictRef   ArtifactRef   `json:"verdict_ref"`
}

// RunScope mirrors schema/pi-review-run.schema.json scope.
type RunScope struct {
	Mode          ScopeMode `json:"mode"`
	Base          string    `json:"base,omitempty"`
	Branch        string    `json:"branch,omitempty"`
	HeadSHA       string    `json:"head_sha,omitempty"`
	PRNumber      int       `json:"pr_number,omitempty"`
	ReviewedFiles []string  `json:"reviewed_files"`
	OmittedFiles  []string  `json:"omitted_files,omitempty"`
}

// RunContext tracks context packet artifact metadata.
type RunContext struct {
	Path               string            `json:"path"`
	SHA256             string            `json:"sha256"`
	DiffSHA256         string            `json:"diff_sha256"`
	RulesSHA256        string            `json:"rules_sha256"`
	TestEvidenceSHA256 string            `json:"test_evidence_sha256,omitempty"`
	FileHashes         map[string]string `json:"file_hashes"`
}

// ModelResult mirrors schema pi-review-run modelRun.
type ModelResult struct {
	Slot         string      `json:"slot"`
	Provider     string      `json:"provider"`
	Model        string      `json:"model"`
	Role         string      `json:"role"`
	Status       string      `json:"status"`
	ArtifactPath string      `json:"artifact_path"`
	LatencyMs    int64       `json:"latency_ms,omitempty"`
	Usage        *ModelUsage `json:"usage,omitempty"`
	Error        string      `json:"error,omitempty"`
}

// ModelUsage tracks token and cost data.
type ModelUsage struct {
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
}

// ArtifactRef references a stored artifact by path and hash.
type ArtifactRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Finding represents a structured review finding.
type Finding struct {
	Priority     string `json:"priority"`
	Title        string `json:"title"`
	File         string `json:"file,omitempty"`
	StartLine    int    `json:"start_line,omitempty"`
	EndLine      int    `json:"end_line,omitempty"`
	Reviewer     string `json:"reviewer,omitempty"`
	Rationale    string `json:"rationale,omitempty"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
	DedupeKey    string `json:"dedupe_key,omitempty"`
}

// Verdict represents the compact review verdict.
type Verdict struct {
	Feature         string                `json:"feature"`
	Verdict         string                `json:"verdict"`
	Round           int                   `json:"round"`
	Timestamp       string                `json:"timestamp"`
	Reviewers       map[string]RoleResult `json:"reviewers"`
	P0Count         int                   `json:"p0_count"`
	P1Count         int                   `json:"p1_count"`
	FindingIDs      []string              `json:"finding_ids,omitempty"`
	BlockingIDs     []string              `json:"blocking_ids,omitempty"`
	Summary         string                `json:"summary,omitempty"`
	ReviewerRuntime string                `json:"reviewer_runtime,omitempty"`
	ModelPanel      []ModelResult         `json:"model_panel,omitempty"`
	FindingsDetail  []Finding             `json:"findings_detail,omitempty"`
}

// RoleResult represents a single reviewer role's verdict.
type RoleResult struct {
	Verdict  string   `json:"verdict"`
	Findings []string `json:"findings"`
	Notes    string   `json:"notes,omitempty"`
}

// ReviewerSlot defines a model reviewer configuration.
type ReviewerSlot struct {
	Slot     string
	Provider string
	Model    string
	Role     string
	Required bool
}

// DefaultSlots returns the MVP reviewer panel.
func DefaultSlots() []ReviewerSlot {
	return []ReviewerSlot{
		{Slot: "zai", Provider: "zai", Model: "glm-5.1", Role: "reviewer", Required: true},
		{Slot: "kimi", Provider: "kimi-coding", Model: "k2p6", Role: "reviewer", Required: true},
		{Slot: "minimax", Provider: "minimax", Model: "MiniMax-M2.7", Role: "reviewer", Required: true},
	}
}

// Runner executes pi-review runs.
type Runner struct {
	cfg    Config
	runner executil.CommandRunner
	slots  []ReviewerSlot
}

// NewRunner creates a new pi-review runner.
func NewRunner(cfg Config, slots []ReviewerSlot) (*Runner, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if len(slots) == 0 {
		slots = DefaultSlots()
	}
	return &Runner{cfg: cfg, runner: cfg.Runner, slots: slots}, nil
}

// Run executes a complete pi-review cycle.
func (r *Runner) Run(ctx context.Context) (*ReviewRun, *Verdict, error) {
	// Build context packet
	pkt, err := BuildContextPacket(ctx, r.cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("context: %w", err)
	}

	runID := fmt.Sprintf("pireview-%s-%d", hashString(pkt.Branch + pkt.UnifiedDiff)[:12], time.Now().UnixMilli())
	runDir := filepath.Join(r.cfg.ProjectRoot, ".sdp", "runs", "pi-review", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("run %s: mkdir: %w", runID, err)
	}

	// Collect test evidence
	evidence, err := CollectTestEvidence(ctx, r.cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("run %s: evidence: %w", runID, err)
	}

	// Run model panel
	modelResults := r.runModelPanel(ctx, runID, pkt, evidence)

	// Check quorum: at least one required reviewer must succeed
	requiredOK := 0
	requiredTotal := 0
	for _, slot := range r.slots {
		if slot.Required {
			requiredTotal++
		}
	}
	for _, mr := range modelResults {
		for _, slot := range r.slots {
			if slot.Required && slot.Slot == mr.Slot && mr.Status == "ok" {
				requiredOK++
			}
		}
	}

	// Synthesize findings
	findings := synthesizeFindings(modelResults)

	// Build run
	run := &ReviewRun{
		RunID:     runID,
		Feature:   r.cfg.Feature,
		Round:     1,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Scope: RunScope{
			Mode:          r.cfg.Scope,
			Base:          r.cfg.BaseRef,
			Branch:        pkt.Branch,
			HeadSHA:       pkt.HeadSHA,
			ReviewedFiles: pkt.ReviewedFiles,
			OmittedFiles:  pkt.OmittedFiles,
		},
		Context: RunContext{
			Path:        fmt.Sprintf(".sdp/runs/pi-review/%s/context.json", runID),
			SHA256:      hashString(pkt.GitStatus),
			DiffSHA256:  hashString(pkt.UnifiedDiff),
			RulesSHA256: hashString(rulesContent(pkt.ProjectRules)),
			FileHashes:  pkt.FileHashes,
		},
		TestEvidence: *evidence,
		Models:       modelResults,
	}

	// Persist context and evidence artifacts
	ctxJSON, err := json.MarshalIndent(pkt, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("run %s: marshal context: %w", runID, err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "context.json"), ctxJSON, 0o644); err != nil {
		return nil, nil, fmt.Errorf("run %s: write context: %w", runID, err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "context.diff"), []byte(pkt.UnifiedDiff), 0o644); err != nil {
		return nil, nil, fmt.Errorf("run %s: write diff: %w", runID, err)
	}
	evJSON, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("run %s: marshal evidence: %w", runID, err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "test-evidence.json"), evJSON, 0o644); err != nil {
		return nil, nil, fmt.Errorf("run %s: write evidence: %w", runID, err)
	}

	// Update context hash with actual artifact content
	run.Context.SHA256 = hashString(string(ctxJSON))
	run.Context.DiffSHA256 = hashString(pkt.UnifiedDiff)

	// Build verdict
	verdict := buildVerdict(r.cfg.Feature, run.Round, findings, modelResults, pkt, requiredOK, requiredTotal)

	run.VerdictRef = ArtifactRef{
		Path:   fmt.Sprintf(".sdp/review_verdict.json"),
		SHA256: hashString(verdict.Summary),
	}

	return run, verdict, nil
}

// runModelPanel invokes each configured model reviewer.
func (r *Runner) runModelPanel(ctx context.Context, runID string, pkt *ContextPacket, evidence *TestEvidence) []ModelResult {
	results := make([]ModelResult, 0, len(r.slots))

	for _, slot := range r.slots {
		start := time.Now()
		result := ModelResult{
			Slot:     slot.Slot,
			Provider: slot.Provider,
			Model:    slot.Model,
			Role:     slot.Role,
		}

		output, err := r.invokePi(ctx, slot, pkt, evidence)
		result.LatencyMs = time.Since(start).Milliseconds()

		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.ArtifactPath = modelArtifactPath(r.cfg.ProjectRoot, runID, slot.Slot)
			if slot.Required && !errors.Is(err, context.DeadlineExceeded) {
				// Try OpenRouter fallback for required slots
				fbOutput, fbErr := r.invokeFallback(ctx, slot, pkt, evidence)
				if fbErr == nil {
					result.Status = "ok"
					result.ArtifactPath = writeModelArtifact(r.cfg.ProjectRoot, runID, slot.Slot, fbOutput)
					result.Provider = "openrouter"
					result.Model = fallbackModel(slot)
				} else {
					result.Error = fmt.Sprintf("%s; fallback failed: %v", result.Error, fbErr)
				}
			}
		} else {
			artifactPath := writeModelArtifact(r.cfg.ProjectRoot, runID, slot.Slot, output)
			if artifactPath == "" {
				result.Status = "failed"
				result.Error = "artifact write failed"
				result.ArtifactPath = modelArtifactPath(r.cfg.ProjectRoot, runID, slot.Slot)
			} else {
				result.Status = "ok"
				result.ArtifactPath = artifactPath
			}
		}

		results = append(results, result)
	}

	return results
}

// invokePi calls the local pi binary with the review context.
func (r *Runner) invokePi(ctx context.Context, slot ReviewerSlot, pkt *ContextPacket, evidence *TestEvidence) (string, error) {
	reviewPrompt := buildReviewPrompt(slot, pkt, evidence)
	promptArg, cleanup, err := writeTempPrompt(reviewPrompt)
	if err != nil {
		return "", err
	}
	defer cleanup()
	modelCtx, cancel := context.WithTimeout(ctx, r.cfg.effectiveModelTimeout())
	defer cancel()
	out, err := r.runner.CombinedOutput(modelCtx, r.cfg.ProjectRoot,
		"pi", "--provider", slot.Provider, "--model", slot.Model,
		"--no-tools", "--no-context-files", "--no-session", "-p", promptArg)
	if err != nil {
		return "", fmt.Errorf("pi run %s/%s: %w", slot.Provider, slot.Model, err)
	}
	return string(out), nil
}

// invokeFallback attempts OpenRouter when a primary provider fails.
func (r *Runner) invokeFallback(ctx context.Context, slot ReviewerSlot, pkt *ContextPacket, evidence *TestEvidence) (string, error) {
	reviewPrompt := buildReviewPrompt(slot, pkt, evidence)
	promptArg, cleanup, err := writeTempPrompt(reviewPrompt)
	if err != nil {
		return "", err
	}
	defer cleanup()
	modelCtx, cancel := context.WithTimeout(ctx, r.cfg.effectiveModelTimeout())
	defer cancel()
	out, err := r.runner.CombinedOutput(modelCtx, r.cfg.ProjectRoot,
		"pi", "--provider", "openrouter", "--model", fallbackModel(slot),
		"--no-tools", "--no-context-files", "--no-session", "-p", promptArg)
	if err != nil {
		return "", fmt.Errorf("openrouter fallback: %w", err)
	}
	return string(out), nil
}

func writeTempPrompt(prompt string) (string, func(), error) {
	f, err := os.CreateTemp("", "sdp-pi-review-*.md")
	if err != nil {
		return "", func() {}, fmt.Errorf("write pi prompt temp file: %w", err)
	}
	path := f.Name()
	cleanup := func() { _ = os.Remove(path) }
	if _, err := f.WriteString(prompt); err != nil {
		_ = f.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("write pi prompt temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("close pi prompt temp file: %w", err)
	}
	return "@" + path, cleanup, nil
}

func fallbackModel(slot ReviewerSlot) string {
	switch slot.Slot {
	case "zai":
		return "z-ai/glm-5.1"
	case "kimi":
		return "moonshotai/kimi-k2.6"
	case "minimax":
		return "minimax/minimax-m2.7"
	default:
		return "z-ai/glm-5.1"
	}
}

// buildReviewPrompt constructs the prompt for the model reviewer.
func buildReviewPrompt(slot ReviewerSlot, pkt *ContextPacket, evidence *TestEvidence) string {
	var b strings.Builder
	b.WriteString("You are an expert code reviewer.\n")
	b.WriteString(fmt.Sprintf("Review role: %s\n", slot.Role))
	b.WriteString(fmt.Sprintf("Feature: %s\n\n", pkt.Branch))

	b.WriteString("## Changed Files\n")
	for _, f := range pkt.ReviewedFiles {
		b.WriteString(fmt.Sprintf("- %s\n", f))
	}

	if len(pkt.FileContents) > 0 {
		b.WriteString("\n## File Contents\n--- BEGIN FILES ---\n")
		for f, content := range pkt.FileContents {
			b.WriteString(fmt.Sprintf("\n### %s\n", f))
			b.WriteString(content)
			b.WriteString("\n")
		}
		b.WriteString("--- END FILES ---\n")
	}

	if len(pkt.ProjectRules) > 0 {
		b.WriteString("\n## Project Rules\n--- BEGIN RULES ---\n")
		for name, content := range pkt.ProjectRules {
			b.WriteString(fmt.Sprintf("\n### %s\n%s\n", name, content))
		}
		b.WriteString("--- END RULES ---\n")
	}

	if pkt.BeadContext != "" {
		b.WriteString("\n## Bead Context\n--- BEGIN BEADS ---\n")
		b.WriteString(pkt.BeadContext)
		b.WriteString("\n--- END BEADS ---\n")
	}

	b.WriteString("\n## Diff\n")
	b.WriteString("--- BEGIN DIFF ---\n")
	b.WriteString(pkt.UnifiedDiff)
	b.WriteString("\n--- END DIFF ---\n")

	b.WriteString("\n## Test Evidence\n")
	b.WriteString("--- BEGIN EVIDENCE ---\n")
	b.WriteString(fmt.Sprintf("Status: %s\n", evidence.Status))
	if evidence.Command != "" {
		b.WriteString(fmt.Sprintf("Command: %s\n", evidence.Command))
	}
	if evidence.ExitCode != 0 {
		b.WriteString(fmt.Sprintf("Exit code: %d\n", evidence.ExitCode))
	}
	b.WriteString("--- END EVIDENCE ---\n")

	b.WriteString("\n## Instructions\n")
	b.WriteString("Return findings as JSON array. Each finding must have: priority (P0-P3), title, file, start_line, end_line, rationale, suggested_fix.\n")
	b.WriteString("P0/P1 findings block approval. P2/P3 are advisory.\n")

	return b.String()
}

// synthesizeFindings parses model outputs into structured findings.
func synthesizeFindings(results []ModelResult) []Finding {
	var all []Finding

	for _, mr := range results {
		if mr.Status != "ok" || mr.ArtifactPath == "" {
			continue
		}
		data, err := os.ReadFile(mr.ArtifactPath)
		if err != nil {
			// Required reviewer's artifact missing is a serious issue
			all = append(all, Finding{
				Priority:  "P1",
				Title:     fmt.Sprintf("model %s artifact unreadable: %v", mr.Slot, err),
				Reviewer:  "synthesizer",
				DedupeKey: fmt.Sprintf("P1:artifact:%s:unreadable", mr.Slot),
			})
			continue
		}
		findings := parseFindingsFromOutput(string(data), mr.Slot)
		all = append(all, findings...)
	}

	return dedupeFindings(all)
}

// parseFindingsFromOutput extracts structured findings from model output.
func parseFindingsFromOutput(output string, slot string) []Finding {
	// Try to extract JSON array from the output
	start := strings.Index(output, "[")
	end := strings.LastIndex(output, "]")
	if start < 0 || end < 0 || end <= start {
		return nil
	}

	jsonStr := output[start : end+1]
	var raw []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil
	}

	findings := make([]Finding, 0, len(raw))
	for _, r := range raw {
		f := Finding{Reviewer: slot}
		if p, ok := r["priority"].(string); ok {
			f.Priority = p
		}
		if t, ok := r["title"].(string); ok {
			f.Title = t
		}
		if fi, ok := r["file"].(string); ok {
			f.File = fi
		}
		if sl, ok := r["start_line"].(float64); ok {
			f.StartLine = int(sl)
		}
		if el, ok := r["end_line"].(float64); ok {
			f.EndLine = int(el)
		}
		if rat, ok := r["rationale"].(string); ok {
			f.Rationale = rat
		}
		if sf, ok := r["suggested_fix"].(string); ok {
			f.SuggestedFix = sf
		}
		f.DedupeKey = fmt.Sprintf("%s:%s:%s", f.Priority, f.File, f.Title)
		findings = append(findings, f)
	}
	return findings
}

// dedupeFindings removes duplicate findings based on dedupe key.
func dedupeFindings(findings []Finding) []Finding {
	seen := make(map[string]bool)
	var deduped []Finding
	for _, f := range findings {
		if seen[f.DedupeKey] {
			continue
		}
		seen[f.DedupeKey] = true
		deduped = append(deduped, f)
	}
	return deduped
}

// buildVerdict creates the compact review verdict.
func buildVerdict(feature string, round int, findings []Finding, models []ModelResult, pkt *ContextPacket, requiredOK, requiredTotal int) *Verdict {
	v := &Verdict{
		Feature:         feature,
		Round:           round,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Reviewers:       make(map[string]RoleResult),
		ReviewerRuntime: "pi",
		ModelPanel:      models,
		FindingsDetail:  findings,
	}

	p0, p1 := 0, 0
	for _, f := range findings {
		switch f.Priority {
		case "P0":
			p0++
		case "P1":
			p1++
		}
	}
	v.P0Count = p0
	v.P1Count = p1
	requiredQuorum := requiredTotal
	if requiredTotal > 2 {
		requiredQuorum = (requiredTotal / 2) + 1
	}

	if p0 > 0 || p1 > 0 {
		v.Verdict = "CHANGES_REQUESTED"
	} else if requiredOK < requiredQuorum {
		v.Verdict = "ESCALATED"
	} else {
		v.Verdict = "APPROVED"
	}

	// Distribute findings across seven reviewer roles
	allPass := RoleResult{Verdict: "PASS", Findings: []string{}}
	v.Reviewers = map[string]RoleResult{
		"qa":        allPass,
		"security":  allPass,
		"devops":    allPass,
		"sre":       allPass,
		"techlead":  allPass,
		"docs":      allPass,
		"promptops": allPass,
	}

	if p0 > 0 || p1 > 0 {
		v.Reviewers["qa"] = RoleResult{
			Verdict:  "FAIL",
			Findings: []string{},
			Notes:    fmt.Sprintf("pi-review found %d P0 and %d P1 issues", p0, p1),
		}
		v.Summary = fmt.Sprintf("CHANGES_REQUESTED: %d P0, %d P1, %d total findings", p0, p1, len(findings))
	} else if requiredOK < requiredQuorum {
		v.Reviewers["qa"] = RoleResult{
			Verdict:  "BLOCKED",
			Findings: []string{},
			Notes:    fmt.Sprintf("quorum failure: %d/%d required reviewers succeeded; quorum=%d", requiredOK, requiredTotal, requiredQuorum),
		}
		v.Summary = fmt.Sprintf("ESCALATED: quorum failure (%d/%d required reviewers; quorum=%d)", requiredOK, requiredTotal, requiredQuorum)
	} else {
		v.Summary = fmt.Sprintf("APPROVED: %d advisory findings (P2/P3), reviewer quorum %d/%d", len(findings), requiredOK, requiredTotal)
	}

	return v
}

// writeModelArtifact persists raw model output to disk.
func writeModelArtifact(projectRoot, runID, slot string, output string) string {
	path := modelArtifactPath(projectRoot, runID, slot)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	if err := os.WriteFile(path, []byte(output), 0o644); err != nil {
		return ""
	}
	return path
}

func modelArtifactPath(projectRoot, runID, slot string) string {
	return filepath.Join(projectRoot, ".sdp", "runs", "pi-review", runID, "models", slot+".json")
}

// hashString returns SHA-256 hex of a string.
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:])
}

// rulesContent concatenates project rules for hashing.
func rulesContent(rules map[string]string) string {
	var b strings.Builder
	keys := make([]string, 0, len(rules))
	for k := range rules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(0)
		b.WriteString(rules[k])
		b.WriteByte(0)
	}
	return b.String()
}
