package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type issue struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	IssueType string   `json:"issue_type"`
	Status    string   `json:"status"`
	Labels    []string `json:"labels"`
}

func run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func extractJSON(out []byte) []byte {
	for i, b := range out {
		if b == '[' || b == '{' {
			return out[i:]
		}
	}
	return out
}

func runComponent(binary string, goPkg string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(binary); err == nil {
		return run(binary, args...)
	}
	goArgs := append([]string{"run", goPkg}, args...)
	return run("go", goArgs...)
}

func commitBeadsState(issueID string, outcome string) error {
	if _, err := run("git", "add", ".beads/issues.jsonl", ".beads/metadata.json"); err != nil {
		return err
	}
	msg := "reviewer: update " + issueID
	body := "Record reviewer outcome: " + outcome + "."
	if _, err := run("git", "commit", "-m", msg, "-m", body); err != nil {
		if strings.Contains(err.Error(), "nothing to commit") {
			return nil
		}
		return err
	}
	if _, err := run("git", "push"); err != nil {
		return err
	}
	return nil
}

func listOpenTasks() ([]issue, error) {
	out, err := run("bd", "list", "--json")
	if err != nil {
		return nil, err
	}
	var items []issue
	if err := json.Unmarshal(extractJSON(out), &items); err != nil {
		return nil, err
	}
	result := make([]issue, 0)
	for _, it := range items {
		if it.IssueType != "task" || it.Status != "in_progress" {
			continue
		}
		if !hasLabel(it.Labels, "autonomy") || !hasLabel(it.Labels, "strict-evidence") {
			continue
		}
		result = append(result, it)
	}
	return result, nil
}

func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}

func runFlow(issueID string) (string, error) {
	path := filepath.Join(".sdp", "runs", issueID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", err
	}
	flow, _ := payload["flow"].(string)
	if flow == "" {
		flow, _ = payload["status"].(string)
	}
	return flow, nil
}

func evidencePath(issueID string) string {
	return filepath.Join(".sdp", "evidence", issueID+".json")
}

func reviewerApprove(issueID string) (bool, error) {
	path := evidencePath(issueID)
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return false, err
	}
	execSection, _ := payload["execution"].(map[string]any)
	if execSection == nil {
		return false, errors.New("missing execution section")
	}
	rawFiles, _ := execSection["changed_files"].([]any)
	hasTest := false
	for _, rf := range rawFiles {
		f, _ := rf.(string)
		if strings.HasSuffix(f, "_test.go") {
			hasTest = true
			break
		}
	}

	if _, err := run("go", "test", "./..."); err != nil {
		return false, nil
	}

	review, _ := payload["review"].(map[string]any)
	if review == nil {
		review = map[string]any{}
		payload["review"] = review
	}
	if hasTest {
		review["self_review"] = []string{"worker changed code and test files"}
		review["adversarial_review"] = []string{"go test ./... passed", "reviewer verdict: approved"}
	} else {
		review["self_review"] = []string{"worker changed code without test files"}
		review["adversarial_review"] = []string{"reviewer verdict: needs_changes (missing *_test.go)"}
	}

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return false, err
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return false, err
	}

	return hasTest, nil
}

func issueBranchFromEvidence(issueID string) (string, error) {
	b, err := os.ReadFile(evidencePath(issueID))
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", err
	}
	trace, _ := payload["trace"].(map[string]any)
	if trace == nil {
		return "", errors.New("missing trace section")
	}
	branch, _ := trace["branch"].(string)
	if strings.TrimSpace(branch) == "" {
		return "", errors.New("missing trace.branch")
	}
	return strings.TrimSpace(branch), nil
}

func issueTitle(issueID string) (string, error) {
	out, err := run("bd", "show", issueID, "--json")
	if err != nil {
		return "", err
	}
	var items []issue
	if err := json.Unmarshal(extractJSON(out), &items); err != nil {
		return "", err
	}
	if len(items) == 0 || strings.TrimSpace(items[0].Title) == "" {
		return "", fmt.Errorf("unable to resolve title for %s", issueID)
	}
	return strings.TrimSpace(items[0].Title), nil
}

func recoverMissingPR(issueID string) error {
	branch, err := issueBranchFromEvidence(issueID)
	if err != nil {
		return err
	}
	title, err := issueTitle(issueID)
	if err != nil {
		return err
	}
	_, err = runComponent("pr-publish", "./cmd/pr-publish", "--issue", issueID, "--title", "Worker: "+title, "--head", branch, "--base", "master")
	if err != nil {
		return err
	}
	return nil
}

func main() {
	items, err := listOpenTasks()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(items) == 0 {
		fmt.Println("No reviewer-eligible tasks found")
		return
	}

	target := ""
	for _, it := range items {
		flow, err := runFlow(it.ID)
		if err != nil {
			continue
		}
		if flow == "review" {
			target = it.ID
			break
		}
	}
	if target == "" {
		fmt.Println("No tasks in review flow")
		return
	}

	approved, err := reviewerApprove(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if approved {
		if _, err := runComponent("beads-fsm", "./cmd/beads-fsm", "--issue", target, "--to", "verified", "--apply"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if _, err := runComponent("pr-gate", "./cmd/pr-gate", "--issue", target); err != nil {
			if strings.Contains(err.Error(), "missing trace.pr_url") {
				if recErr := recoverMissingPR(target); recErr != nil {
					fmt.Fprintln(os.Stderr, recErr)
					os.Exit(1)
				}
				if _, err := runComponent("pr-gate", "./cmd/pr-gate", "--issue", target); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			} else {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		if _, err := runComponent("beads-fsm", "./cmd/beads-fsm", "--issue", target, "--to", "done", "--apply"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := commitBeadsState(target, "approved"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		if _, err := runComponent("beads-fsm", "./cmd/beads-fsm", "--issue", target, "--to", "blocked", "--apply"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_, _ = run("bd", "update", target, "--append-notes", "reviewer verdict: needs_changes (missing test changes)")
		if err := commitBeadsState(target, "blocked"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	out, _ := json.MarshalIndent(map[string]any{
		"issue":    target,
		"approved": approved,
	}, "", "  ")
	fmt.Println(string(out))
}
