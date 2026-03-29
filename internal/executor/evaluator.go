package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/control"
)

const (
	evalVerdictPass        = "pass"
	evalVerdictFail        = "fail"
	evalVerdictNeedsReview = "needs_review"
	evalVerdictBlocked     = "blocked" // OmO unavailable or no evidence — hard fail
)

type EvaluatorConfig struct {
	Enabled  bool
	Model    string
	Criteria []EvalCriterion
}

type EvalCriterion struct {
	Name        string
	Description string
	Weight      float64
}

type EvalResult struct {
	Verdict     string          `json:"verdict"`
	Score       float64         `json:"score"`
	Findings    []string        `json:"findings,omitempty"`
	Passed      map[string]bool `json:"passed,omitempty"`
	RawFeedback string          `json:"raw_feedback,omitempty"`
}

type buildEvidence struct {
	Phase        string                       `json:"phase,omitempty"`
	CardID       string                       `json:"card_id,omitempty"`
	Timestamp    string                       `json:"timestamp,omitempty"`
	Executor     string                       `json:"executor,omitempty"`
	ExitCode     int                          `json:"exit_code,omitempty"`
	Status       control.ExecutorResultStatus `json:"status,omitempty"`
	Summary      string                       `json:"summary,omitempty"`
	FilesChanged []string                     `json:"files_changed,omitempty"`
	Artifacts    []control.ExecutorArtifact   `json:"artifacts,omitempty"`
	Findings     []string                     `json:"findings,omitempty"`
}

type llmEvalPayload struct {
	Verdict  string          `json:"verdict"`
	Score    float64         `json:"score"`
	Findings []string        `json:"findings"`
	Passed   map[string]bool `json:"passed"`
}

func DefaultEvaluatorConfig() EvaluatorConfig {
	model := strings.TrimSpace(os.Getenv("SDP_EVAL_MODEL"))
	if model == "" {
		model = "default"
	}
	return EvaluatorConfig{
		Enabled: true,
		Model:   model,
		Criteria: []EvalCriterion{
			{Name: "tests_pass", Description: "Did tests pass based on execution evidence?", Weight: 0.30},
			{Name: "scope_adherence", Description: "Were ScopeOut constraints respected and changes kept inside scope?", Weight: 0.25},
			{Name: "evidence_complete", Description: "Are required build artifacts and evidence files present?", Weight: 0.20},
			{Name: "code_quality", Description: "Does the diff look clean, structured, and free of obvious hacks?", Weight: 0.25},
		},
	}
}

func verdictForScore(score float64, hardFailure bool) string {
	if hardFailure || score < 0.50 {
		return evalVerdictFail
	}
	if score >= 0.80 {
		return evalVerdictPass
	}
	return evalVerdictNeedsReview
}

func scoreCriteria(criteria []EvalCriterion, passed map[string]bool) (float64, bool) {
	var score float64
	hardFailure := false
	for _, criterion := range criteria {
		if passed[criterion.Name] {
			score += criterion.Weight
			continue
		}
		if criterion.Name == "tests_pass" || criterion.Name == "scope_adherence" {
			hardFailure = true
		}
	}
	if score < 0 {
		return 0, hardFailure
	}
	if score > 1 {
		return 1, hardFailure
	}
	return score, hardFailure
}

func loadBuildEvidence(projectRoot, cardID string) (*buildEvidence, string, error) {
	path := filepath.Join(projectRoot, ".sdp", "artifacts", cardID, "build.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, path, nil
		}
		return nil, path, err
	}
	var evidence buildEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		return nil, path, fmt.Errorf("parse build evidence: %w", err)
	}
	if len(evidence.FilesChanged) == 0 && len(evidence.Artifacts) > 0 {
		for _, artifact := range evidence.Artifacts {
			if strings.TrimSpace(artifact.Reference) != "" {
				evidence.FilesChanged = append(evidence.FilesChanged, strings.TrimSpace(artifact.Reference))
			}
		}
	}
	return &evidence, path, nil
}

func detectChangedFiles(projectRoot string, evidence *buildEvidence) []string {
	if evidence != nil && len(evidence.FilesChanged) > 0 {
		return dedupeStrings(evidence.FilesChanged)
	}
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--name-only", "HEAD~1", "HEAD")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return dedupeStrings(strings.Fields(string(out)))
}

func gitDiff(projectRoot string) string {
	cmd := exec.CommandContext(context.Background(), "git", "diff", "HEAD~1")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	diff := string(out)
	if len(diff) > 12000 {
		return diff[:12000]
	}
	return diff
}

func detectEvidenceComplete(projectRoot, cardID string, card *control.FeatureCard) bool {
	buildPath := filepath.Join(projectRoot, ".sdp", "artifacts", cardID, "build.json")
	if _, err := os.Stat(buildPath); err != nil {
		return false
	}
	for _, required := range card.RequiredArtifacts {
		required = strings.TrimSpace(required)
		if required == "" {
			continue
		}
		candidate := required
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(projectRoot, candidate)
		}
		if _, err := os.Stat(candidate); err != nil {
			return false
		}
	}
	return true
}

func detectScopeAdherence(card *control.FeatureCard, changed []string) (bool, []string) {
	if card == nil {
		return true, nil
	}
	var findings []string
	for _, file := range changed {
		if matchesAnyScope(file, card.ScopeOut) {
			findings = append(findings, fmt.Sprintf("changed file is explicitly out of scope: %s", file))
			continue
		}
		if len(card.ScopeIn) > 0 && !matchesAnyScope(file, card.ScopeIn) {
			findings = append(findings, fmt.Sprintf("changed file is outside scope_in: %s", file))
		}
	}
	return len(findings) == 0, findings
}

func matchesAnyScope(file string, scope []string) bool {
	for _, p := range scope {
		p = filepath.ToSlash(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		file = filepath.ToSlash(file)
		if ok, _ := filepath.Match(p, file); ok {
			return true
		}
		if strings.HasSuffix(p, "/**") {
			prefix := strings.TrimSuffix(p, "/**")
			if file == prefix || strings.HasPrefix(file, prefix+"/") {
				return true
			}
		}
		if strings.HasSuffix(p, "/") {
			if strings.HasPrefix(file, p) {
				return true
			}
			continue
		}
		if file == p || strings.HasPrefix(file, p+"/") {
			return true
		}
	}
	return false
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseEvalResult(raw string, cfg EvaluatorConfig, localPassed map[string]bool, localFindings []string) EvalResult {
	result := EvalResult{Passed: map[string]bool{}, RawFeedback: raw}
	payload := llmEvalPayload{}
	decoder := json.NewDecoder(strings.NewReader(extractJSONObject(raw)))
	if err := decoder.Decode(&payload); err == nil {
		result.Passed = payload.Passed
		result.Findings = dedupeStrings(payload.Findings)
		result.Score = payload.Score
		result.Verdict = payload.Verdict
	}
	if result.Passed == nil {
		result.Passed = map[string]bool{}
	}
	for name, passed := range localPassed {
		if _, ok := result.Passed[name]; !ok {
			result.Passed[name] = passed
		}
	}
	result.Findings = dedupeStrings(append(localFindings, result.Findings...))
	score, _ := scoreCriteria(cfg.Criteria, result.Passed)
	if result.Score <= 0 || result.Score > 1 {
		result.Score = score
	}
	if result.Verdict == "" || (result.Verdict != evalVerdictPass && result.Verdict != evalVerdictFail && result.Verdict != evalVerdictNeedsReview) {
		result.Verdict = verdictForScore(result.Score, false)
	}
	return result
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func saveEvaluationEvidence(projectRoot, cardID string, result EvalResult) (string, error) {
	path := filepath.Join(projectRoot, ".sdp", "artifacts", cardID, "evaluation.json")
	payload := map[string]any{
		"phase":        "evaluation",
		"card_id":      cardID,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"result":       result,
		"verdict":      result.Verdict,
		"score":        result.Score,
		"findings":     result.Findings,
		"passed":       result.Passed,
		"raw_feedback": result.RawFeedback,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func EvaluateBuild(ctx context.Context, projectRoot string, card *control.FeatureCard, cfg EvaluatorConfig) (EvalResult, error) {
	if card == nil {
		return EvalResult{}, fmt.Errorf("nil card")
	}
	if !cfg.Enabled {
		return EvalResult{Verdict: evalVerdictFail, Findings: []string{"evaluator is disabled; treating as hard fail"}}, nil
	}

	evidence, _, err := loadBuildEvidence(projectRoot, card.ID)
	if err != nil {
		return EvalResult{}, err
	}
	if evidence == nil {
		return EvalResult{Verdict: evalVerdictBlocked, Findings: []string{"build evidence not found — cannot evaluate"}}, nil
	}

	passed := map[string]bool{}
	findings := []string{}
	passed["tests_pass"] = evidence.ExitCode == 0 && evidence.Status == control.ResultStatusSuccess
	if !passed["tests_pass"] {
		findings = append(findings, fmt.Sprintf("build/test evidence indicates failure (exit_code=%d status=%s)", evidence.ExitCode, evidence.Status))
	}
	changed := detectChangedFiles(projectRoot, evidence)
	scopePass, scopeFindings := detectScopeAdherence(card, changed)
	passed["scope_adherence"] = scopePass
	findings = append(findings, scopeFindings...)
	passed["evidence_complete"] = detectEvidenceComplete(projectRoot, card.ID, card)
	if !passed["evidence_complete"] {
		findings = append(findings, "required evidence artifacts are missing")
	}

	prompt := BuildEvaluationPrompt(cfg.Model, card, evidence, changed, gitDiff(projectRoot))
	raw, exitCode, invokeErr := InvokeWithFallback(ctx, projectRoot, "sisyphus", prompt)
	if invokeErr != nil || exitCode != 0 {
		score, _ := scoreCriteria(cfg.Criteria, passed)
		return EvalResult{Verdict: evalVerdictBlocked, Score: score, Findings: dedupeStrings(append(findings, fmt.Sprintf("evaluation failed: %v", invokeErr)))}, nil
	}

	result := parseEvalResult(raw, cfg, passed, findings)
	if _, ok := result.Passed["code_quality"]; !ok {
		result.Passed["code_quality"] = result.Verdict == evalVerdictPass || result.Score >= 0.80
	}
	result.Score, _ = scoreCriteria(cfg.Criteria, result.Passed)
	result.Verdict = verdictForScore(result.Score, !result.Passed["tests_pass"] || !result.Passed["scope_adherence"])
	return result, nil
}
