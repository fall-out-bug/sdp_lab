package telemetry

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sdp_dev/internal/llm"
	"sdp_dev/internal/observability"
)

// Analyzer analyzes closed issues and generates backlog proposals.
type Analyzer struct {
	WorkDir     string
	Model       string
	MaxPerCycle int
	mu          sync.Mutex
	cycleCount  int
	cycleReset  time.Time
}

// NewAnalyzer creates an Analyzer.
func NewAnalyzer(workDir, model string, maxPerCycle int) *Analyzer {
	if maxPerCycle <= 0 {
		maxPerCycle = 5
	}
	if model == "" {
		model = "glm-4.7"
	}
	return &Analyzer{
		WorkDir:     workDir,
		Model:       model,
		MaxPerCycle: maxPerCycle,
		cycleReset:  time.Now(),
	}
}

// HandleClosed processes a closed issue event: read evidence, optionally analyze, create proposal.
func (a *Analyzer) HandleClosed(ctx context.Context, issueID, projectID string) (created bool, err error) {
	a.mu.Lock()
	if time.Since(a.cycleReset) > 30*time.Minute {
		a.cycleCount = 0
		a.cycleReset = time.Now()
	}
	if a.cycleCount >= a.MaxPerCycle {
		a.mu.Unlock()
		return false, nil
	}
	a.mu.Unlock()

	workDir := a.workDirForProject(projectID)
	evPath := filepath.Join(workDir, ".sdp", "evidence", issueID+".json")
	data, err := os.ReadFile(evPath)
	if err != nil {
		return false, nil
	}

	var ev map[string]any
	if err := json.Unmarshal(data, &ev); err != nil {
		return false, nil
	}

	summary := a.summarizeEvidence(ev)
	proposal, err := a.analyzeWithLLM(ctx, summary, issueID)
	if err != nil {
		return false, err
	}
	if proposal == "" {
		return false, nil
	}

	if a.isDuplicate(proposal, workDir) {
		return false, nil
	}

	if err := a.createBeadsIssue(issueID, proposal, workDir); err != nil {
		return false, err
	}

	a.mu.Lock()
	a.cycleCount++
	a.mu.Unlock()
	return true, nil
}

func (a *Analyzer) workDirForProject(projectID string) string {
	if projectID == "" {
		projectID = "default"
	}
	return filepath.Join(a.WorkDir, projectID)
}

func (a *Analyzer) summarizeEvidence(ev map[string]any) string {
	var b strings.Builder
	if intent, ok := ev["intent"].(map[string]any); ok {
		if obj, ok := intent["objective"].(string); ok {
			b.WriteString("Objective: ")
			b.WriteString(obj)
			b.WriteString("\n")
		}
	}
	if exec, ok := ev["execution"].(map[string]any); ok {
		if changed, ok := exec["changed_files"].([]any); ok {
			b.WriteString("Changed files: ")
			for i, c := range changed {
				if s, ok := c.(string); ok {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString(s)
				}
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (a *Analyzer) analyzeWithLLM(ctx context.Context, summary, issueID string) (string, error) {
	client := llm.NewOpenRouterClient()
	if client.APIKey == "" {
		return "Backlog proposal from closed issue " + issueID, nil
	}
	prompt := "Based on this closed task evidence, suggest ONE short backlog item (feature or task) if there is a clear follow-up. Reply with only the title, or 'none' if no follow-up.\n\n" + summary
	msg, result, err := client.ChatWithUsage(ctx, a.Model, []llm.OpenRouterMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return "", err
	}
	if result != nil && (result.PromptTokens > 0 || result.CompletionTokens > 0) {
		costUSD := estimateCostUSD(result.PromptTokens, result.CompletionTokens)
		observability.ObserveLLMUsage("default", "telemetry-analyzer", a.Model, result.PromptTokens, result.CompletionTokens, costUSD)
	}
	msg = strings.TrimSpace(msg)
	lower := strings.ToLower(msg)
	if lower == "" || lower == "none" || len(msg) < 10 {
		return "", nil
	}
	return msg, nil
}

// isDuplicate returns true if proposal matches an existing open issue title.
func (a *Analyzer) isDuplicate(proposal, workDir string) bool {
	cmd := exec.Command("bd", "list", "--status", "open")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	proposal = strings.ToLower(strings.TrimSpace(proposal))
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		idx := strings.LastIndex(line, " - ")
		if idx < 0 {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(line[idx+3:]))
		if title == proposal || strings.Contains(title, proposal) || strings.Contains(proposal, title) {
			return true
		}
	}
	return false
}

// estimateCostUSD returns rough cost in USD (placeholder: ~$0.002/1K tokens).
func estimateCostUSD(prompt, completion int) float64 {
	total := prompt + completion
	if total <= 0 {
		return 0
	}
	return float64(total) / 1000 * 0.002
}

func (a *Analyzer) createBeadsIssue(closedIssueID, title, workDir string) error {
	cmd := exec.Command("bd", "create",
		"--title", title,
		"--labels", "source:telemetry-analyzer",
		"--notes", "Generated from closed issue "+closedIssueID,
	)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "BD_ACTOR=telemetry-analyzer")
	return cmd.Run()
}
