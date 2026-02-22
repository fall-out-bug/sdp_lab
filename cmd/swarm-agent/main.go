package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type claimResult struct {
	IssueID string `json:"issue_id"`
	Title   string `json:"title"`
	Model   string `json:"model"`
	Branch  string `json:"branch"`
}

type issueDetail struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Labels []string `json:"labels"`
}

func runIn(repo, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func currentBranch(repo string) (string, error) {
	out, err := runIn(repo, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func parseClaim(out []byte) (claimResult, error) {
	var r claimResult
	if err := json.Unmarshal(out, &r); err != nil {
		return r, err
	}
	if r.IssueID == "" || r.Branch == "" {
		return r, errors.New("invalid claim result")
	}
	return r, nil
}

func loadIssue(repo, issueID string) (issueDetail, error) {
	out, err := runIn(repo, "bd", "show", issueID, "--json")
	if err != nil {
		return issueDetail{}, err
	}
	var list []issueDetail
	if err := json.Unmarshal(out, &list); err == nil && len(list) > 0 {
		return list[0], nil
	}
	var single issueDetail
	if err := json.Unmarshal(out, &single); err != nil {
		return issueDetail{}, err
	}
	return single, nil
}

func handlerFromLabels(labels []string) string {
	for _, l := range labels {
		if strings.HasPrefix(l, "handler:") {
			return strings.TrimPrefix(l, "handler:")
		}
	}
	return ""
}

func ensureBranch(repo, branch string) error {
	_, err := runIn(repo, "git", "checkout", "-b", branch)
	if err == nil {
		return nil
	}
	_, err = runIn(repo, "git", "checkout", branch)
	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func write(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func addFlowInspectHandler(repo string) ([]string, error) {
	changed := make([]string, 0)
	cmdPath := filepath.Join(repo, "cmd", "flow-inspect", "main.go")
	if !fileExists(cmdPath) {
		source := `package main

import (
  "encoding/json"
  "flag"
  "fmt"
  "os"
  "path/filepath"
)

func main() {
  issueID := flag.String("issue", "", "Issue ID")
  flag.Parse()
  if *issueID == "" {
    fmt.Fprintln(os.Stderr, "--issue is required")
    os.Exit(2)
  }
  wd, _ := os.Getwd()
  runPath := filepath.Join(wd, ".sdp", "runs", *issueID+".json")
  payload, err := os.ReadFile(runPath)
  if err != nil {
    fmt.Fprintf(os.Stderr, "read run packet: %v\n", err)
    os.Exit(1)
  }
  var run map[string]any
  if err := json.Unmarshal(payload, &run); err != nil {
    fmt.Fprintf(os.Stderr, "parse run packet: %v\n", err)
    os.Exit(1)
  }
  flow, _ := run["flow"].(string)
  if flow == "" {
    flow, _ = run["status"].(string)
  }
  out := map[string]any{
    "issue": *issueID,
    "flow": flow,
    "run": run,
  }
  b, _ := json.MarshalIndent(out, "", "  ")
  fmt.Println(string(b))
}
`
		if err := write(cmdPath, source); err != nil {
			return nil, err
		}
		changed = append(changed, "cmd/flow-inspect/main.go")
	}

	readme := filepath.Join(repo, "README.md")
	b, err := os.ReadFile(readme)
	if err != nil {
		return nil, err
	}
	content := string(b)
	line := "- `cmd/flow-inspect/` - inspects protocol flow state from run packets."
	if !strings.Contains(content, line) {
		content += "\n" + line + "\n"
		if err := os.WriteFile(readme, []byte(content), 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, "README.md")
	}
	return changed, nil
}

func setEvidenceExecution(repo, issueID, branch string, changed []string) error {
	path := filepath.Join(repo, ".sdp", "evidence", issueID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return err
	}

	execSection, _ := payload["execution"].(map[string]any)
	if execSection == nil {
		execSection = map[string]any{}
		payload["execution"] = execSection
	}
	execSection["branch"] = branch
	execSection["changed_files"] = changed
	execSection["claimed_issue_ids"] = []string{issueID}

	verification, _ := payload["verification"].(map[string]any)
	if verification == nil {
		verification = map[string]any{}
		payload["verification"] = verification
	}
	verification["tests"] = []string{"go test ./..."}

	trace, _ := payload["trace"].(map[string]any)
	if trace == nil {
		trace = map[string]any{}
		payload["trace"] = trace
	}
	trace["branch"] = branch
	trace["beads_ids"] = []string{issueID}

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

func writePRBody(repo, issueID, title string) (string, error) {
	path := filepath.Join(repo, ".sdp", "pr-body-"+issueID+".md")
	body := "## Summary\n\n- autonomous swarm agent execution for " + issueID + "\n- implemented: " + title + "\n- policy: strict-evidence, go-first, manual merge\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func main() {
	noPR := flag.Bool("no-pr", false, "Skip PR publishing")
	flag.Parse()
	repo, _ := os.Getwd()

	baseBranch, err := currentBranch(repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	claimOut, err := runIn(repo, "go", "run", "./cmd/autonomy-worker")
	if err != nil {
		if strings.Contains(err.Error(), "No eligible autonomy tasks found") {
			fmt.Println("No eligible autonomy tasks found")
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	claim, err := parseClaim(claimOut)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := ensureBranch(repo, claim.Branch); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	detail, err := loadIssue(repo, claim.IssueID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	handler := handlerFromLabels(detail.Labels)
	if handler == "" {
		fmt.Fprintf(os.Stderr, "issue %s has no handler label\n", claim.IssueID)
		os.Exit(1)
	}

	var changed []string
	switch handler {
	case "add-flow-inspect-command":
		changed, err = addFlowInspectHandler(repo)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unsupported handler: %s\n", handler)
		os.Exit(1)
	}

	if _, err := runIn(repo, "go", "test", "./..."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := setEvidenceExecution(repo, claim.IssueID, claim.Branch, changed); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if _, err := runIn(repo, "go", "run", "./cmd/beads-fsm", "--issue", claim.IssueID, "--to", "review", "--apply"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := runIn(repo, "go", "run", "./cmd/pr-gate", "--issue", claim.IssueID, "--prepublish"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if _, err := runIn(repo, "git", "add", "-A"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := runIn(repo, "git", "commit", "-m", "agent: implement "+claim.IssueID, "-m", "Automated handler execution under SDP protocol."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := runIn(repo, "git", "push", "-u", "origin", claim.Branch); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if !*noPR {
		bodyPath, err := writePRBody(repo, claim.IssueID, claim.Title)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if _, err := runIn(repo, "go", "run", "./cmd/pr-publish", "--issue", claim.IssueID, "--title", "Agent: "+claim.Title, "--head", claim.Branch, "--base", baseBranch, "--body-file", bodyPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if _, err := runIn(repo, "go", "run", "./cmd/pr-gate", "--issue", claim.IssueID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if _, err := runIn(repo, "go", "run", "./cmd/beads-fsm", "--issue", claim.IssueID, "--to", "verified", "--apply"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := runIn(repo, "go", "run", "./cmd/beads-fsm", "--issue", claim.IssueID, "--to", "done", "--apply"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if _, err := runIn(repo, "git", "add", "-A"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := runIn(repo, "git", "commit", "-m", "agent: finalize "+claim.IssueID, "-m", "Record PR trace and close task via SDP flow transitions."); err != nil {
		if !strings.Contains(err.Error(), "nothing to commit") {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		if _, err := runIn(repo, "git", "push"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	summary := map[string]any{
		"issue":  claim.IssueID,
		"branch": claim.Branch,
		"base":   baseBranch,
		"status": "done",
	}
	b, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(b))
}
