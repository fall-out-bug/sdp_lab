package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
)

type AutoAttestOptions struct {
	BaseBranch string
	PRNumber   string
	PRURL      string
	RepoRoot   string
}

func AutoAttest(opts AutoAttestOptions) (CodingWorkflowStatement, error) {
	changedFiles, err := gitChangedFiles(opts.RepoRoot, opts.BaseBranch)
	if err != nil {
		return CodingWorkflowStatement{}, fmt.Errorf("git changed files: %w", err)
	}

	branch, err := gitCurrentBranch(opts.RepoRoot)
	if err != nil {
		return CodingWorkflowStatement{}, fmt.Errorf("git branch: %w", err)
	}

	headSHA, err := gitHeadSHA(opts.RepoRoot)
	if err != nil {
		return CodingWorkflowStatement{}, fmt.Errorf("git head SHA: %w", err)
	}

	testResults, coverage := collectTestResults(opts.RepoRoot)
	lintResults := collectLintResults(opts.RepoRoot)

	subjectName := fmt.Sprintf("PR #%s", opts.PRNumber)
	if opts.PRURL != "" {
		subjectName = opts.PRURL
	}

	subjects := []intoto.Subject{{
		Name:   subjectName,
		Digest: common.DigestSet{"sha256": headSHA},
	}}

	predicate := CodingWorkflowPredicate{
		Intent: Intent{
			Trigger: "ci-auto-attestation",
		},
		Execution: Execution{
			Branch:       branch,
			ChangedFiles: changedFiles,
		},
		Verification: Verification{
			Tests: testResults,
			Lint:  lintResults,
			Coverage: func() *Coverage {
				if coverage >= 0 {
					return &Coverage{Value: coverage, Threshold: 80}
				}
				return nil
			}(),
		},
		Boundary: Boundary{
			Observed: ObservedBoundary{
				TouchedPaths: changedFiles,
			},
			Compliance: BoundaryCompliance{
				OK:     true,
				Reason: "auto-attested from CI observation",
			},
		},
		Provenance: Provenance{
			RunID:        fmt.Sprintf("ci-auto-%s-%s", opts.PRNumber, headSHA[:8]),
			Orchestrator: "github-actions",
			Runtime:      "ci",
			CapturedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		Trace: Trace{
			Branch:  branch,
			Commits: []string{headSHA},
			PRURL:   opts.PRURL,
		},
	}

	return NewStatement(subjects, predicate), nil
}

func gitChangedFiles(repoRoot, baseBranch string) ([]string, error) {
	if baseBranch == "" {
		baseBranch = "master"
	}
	out, err := runGit(repoRoot, "diff", "--name-only", "origin/"+baseBranch+"...HEAD")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func gitCurrentBranch(repoRoot string) (string, error) {
	out, err := runGit(repoRoot, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func gitHeadSHA(repoRoot string) (string, error) {
	out, err := runGit(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func collectTestResults(repoRoot string) ([]GateResult, float64) {
	cmd := exec.Command("go", "test", "./...", "-count=1", "-json")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return []GateResult{{Name: "go-test", Status: "fail"}}, -1
	}

	passed := 0
	failed := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var evt map[string]any
		if json.Unmarshal([]byte(line), &evt) != nil {
			continue
		}
		action, _ := evt["Action"].(string)
		if action == "pass" {
			if _, hasTest := evt["Test"]; hasTest {
				passed++
			}
		}
		if action == "fail" {
			if _, hasTest := evt["Test"]; hasTest {
				failed++
			}
		}
	}

	status := "pass"
	if failed > 0 {
		status = "fail"
	}
	return []GateResult{{
		Name:   "go-test",
		Status: fmt.Sprintf("%s (%d passed, %d failed)", status, passed, failed),
	}}, -1
}

func collectLintResults(repoRoot string) []GateResult {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return []GateResult{{Name: "go-vet", Status: "fail"}}
	}
	return []GateResult{{Name: "go-vet", Status: "pass"}}
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

func splitLines(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	result := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

func WriteAutoAttestationReport(outputPath string, stmt CodingWorkflowStatement) error {
	report := map[string]any{
		"type":           "ci-auto-attestation",
		"generated_at":   stmt.Predicate.Provenance.CapturedAt,
		"changed_files":  len(stmt.Predicate.Execution.ChangedFiles),
		"test_results":   stmt.Predicate.Verification.Tests,
		"lint_results":   stmt.Predicate.Verification.Lint,
		"branch":         stmt.Predicate.Trace.Branch,
		"head_commit":    firstOrEmpty(stmt.Predicate.Trace.Commits),
		"attestation_id": stmt.Predicate.Provenance.RunID,
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(outputPath, b, 0o644)
}

func firstOrEmpty(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}
