package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sdp_dev/internal/beads"
)

// AggregateResult holds the outcome of aggregating subtask results for a feature.
type AggregateResult struct {
	FeatureID   string
	SubtaskIDs  []string
	AllClosed   bool
	Merged      bool
	PRURL       string
	IntegrationPassed bool
}

// Aggregate checks subtask statuses for a feature and merges branches when all are closed.
func Aggregate(adapter *beads.Adapter, featureID string, subtaskIDs []string, workDir string) (AggregateResult, error) {
	res := AggregateResult{FeatureID: featureID, SubtaskIDs: subtaskIDs}
	allClosed := true
	for _, id := range subtaskIDs {
		iss, err := adapter.Show(id)
		if err != nil {
			return res, fmt.Errorf("show %s: %w", id, err)
		}
		if iss.Status != "closed" {
			allClosed = false
			break
		}
	}
	res.AllClosed = allClosed
	if !allClosed {
		return res, nil
	}

	baseBranch := "master"
	runGit := func(args ...string) ([]byte, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = workDir
		return cmd.CombinedOutput()
	}

	featureBranch := "feat/" + featureID
	if out, err := runGit("checkout", "-b", featureBranch, baseBranch); err != nil {
		if !strings.Contains(string(out), "already exists") {
			return res, fmt.Errorf("create feature branch: %w: %s", err, out)
		}
		_, _ = runGit("checkout", featureBranch)
	}

	for _, id := range subtaskIDs {
		subBranch := "feat/" + id
		if out, err := runGit("merge", "--no-edit", subBranch); err != nil {
			return res, fmt.Errorf("merge %s: %w: %s", subBranch, err, out)
		}
	}
	res.Merged = true

	if out, err := runGit("diff", "--name-only", baseBranch+"..HEAD"); err == nil && len(out) > 0 {
		if out2, err2 := runGit("go", "test", "./..."); err2 != nil {
			res.IntegrationPassed = false
			_ = out2
		} else {
			res.IntegrationPassed = true
		}
	}

	return res, nil
}

// PublishFeaturePR creates a PR for the aggregated feature branch.
func PublishFeaturePR(workDir, featureID, title string, bodyPath string) (string, error) {
	featureBranch := "feat/" + featureID
	cmd := exec.Command("gh", "pr", "create", "--head", featureBranch, "--base", "master", "--title", title, "--body-file", bodyPath)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create: %w: %s", err, out)
	}
	url := strings.TrimSpace(string(out))
	return url, nil
}

// WriteFeaturePRBody writes a PR body for the aggregated feature.
func WriteFeaturePRBody(workDir, featureID string, subtaskIDs []string) (string, error) {
	path := filepath.Join(workDir, ".sdp", "pr-body-feature-"+featureID+".md")
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)
	var b strings.Builder
	b.WriteString("## Feature: " + featureID + "\n\n")
	b.WriteString("Aggregated subtasks:\n")
	for _, id := range subtaskIDs {
		b.WriteString("- " + id + "\n")
	}
	content := b.String()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
