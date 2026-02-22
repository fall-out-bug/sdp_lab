package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/beads"
)


// ImprovementTask is a suggested self-improvement task.
type ImprovementTask struct {
	Title       string
	Description string
	Pattern     string // e.g. "frequent_failures", "boundary_violation", "timeout"
	Evidence    []string
}

// AnalyzeRuns scans .sdp/runs/*.json and detects improvement patterns.
func AnalyzeRuns(workDir string, minRuns int) ([]ImprovementTask, error) {
	runsDir := filepath.Join(workDir, ".sdp", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var failures []string
	var timeouts []string
	var boundaryViolations []string

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(runsDir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc RunDoc
		if err := json.Unmarshal(b, &doc); err != nil {
			continue
		}
		if doc.LastState == "failed" {
			failures = append(failures, doc.IssueID)
			for _, ev := range doc.Events {
				if ev.State == "failed" && strings.Contains(strings.ToLower(ev.Message), "timeout") {
					timeouts = append(timeouts, doc.IssueID)
					break
				}
				if strings.Contains(strings.ToLower(ev.Message), "boundary") {
					boundaryViolations = append(boundaryViolations, doc.IssueID)
					break
				}
			}
		}
	}

	if len(entries) < minRuns {
		return nil, nil
	}

	var tasks []ImprovementTask
	threshold := len(entries) / 3
	if threshold < 2 {
		threshold = 2
	}
	if len(failures) >= threshold {
		tasks = append(tasks, ImprovementTask{
			Title:       "Self-improvement: Reduce frequent run failures",
			Description: fmt.Sprintf("Observed %d failures in recent runs. Investigate and improve error handling or retry logic.", len(failures)),
			Pattern:     "frequent_failures",
			Evidence:    failures,
		})
	}
	if len(timeouts) >= 2 {
		tasks = append(tasks, ImprovementTask{
			Title:       "Self-improvement: Address timeout patterns",
			Description: fmt.Sprintf("Observed %d timeout failures. Consider increasing timeout or optimizing long-running tasks.", len(timeouts)),
			Pattern:     "timeout",
			Evidence:    timeouts,
		})
	}
	if len(boundaryViolations) >= 1 {
		tasks = append(tasks, ImprovementTask{
			Title:       "Self-improvement: Fix boundary violation handling",
			Description: fmt.Sprintf("Observed %d boundary violations. Review workstream path constraints and LLM prompts.", len(boundaryViolations)),
			Pattern:     "boundary_violation",
			Evidence:    boundaryViolations,
		})
	}
	return tasks, nil
}

// EmitImprovementTasks creates beads issues for each improvement task.
func EmitImprovementTasks(adapter *beads.Adapter, tasks []ImprovementTask) ([]string, error) {
	labels := []string{"autonomy", "strict-evidence", "workstream:self-improvement", "lane:commit"}
	var ids []string
	for _, t := range tasks {
		desc := t.Description
		if len(t.Evidence) > 0 {
			desc += "\n\nEvidence: " + strings.Join(t.Evidence, ", ")
		}
		id, err := adapter.Create(beads.CreateOpts{
			Title:       t.Title,
			Type:        "task",
			Priority:    2,
			Description: desc,
			Acceptance:  "Task completed and verified",
			Labels:      labels,
		})
		if err != nil {
			return ids, fmt.Errorf("create improvement task: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
