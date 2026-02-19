package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sdp_dev/internal/evidence"
)

var allowedTransitions = map[string]map[string]bool{
	"open":        {"in_progress": true, "cancelled": true},
	"in_progress": {"review": true, "blocked": true, "escalated": true, "cancelled": true},
	"review":      {"verified": true, "blocked": true, "escalated": true, "cancelled": true},
	"verified":    {"done": true, "escalated": true, "cancelled": true},
	"blocked":     {"in_progress": true, "escalated": true, "cancelled": true},
	"escalated":   {"in_progress": true, "cancelled": true},
}

type dep struct {
	IssueID        string `json:"issue_id"`
	DependsOnID    string `json:"depends_on_id"`
	ID             string `json:"id"`
	Type           string `json:"type"`
	DependencyType string `json:"dependency_type"`
	IssueType      string `json:"issue_type"`
	Status         string `json:"status"`
}

func (d dep) refID() string {
	if d.DependsOnID != "" {
		return d.DependsOnID
	}
	return d.ID
}

func (d dep) kind() string {
	if d.Type != "" {
		return d.Type
	}
	if d.DependencyType != "" {
		return d.DependencyType
	}
	if d.IssueType == "epic" || d.IssueType == "feature" {
		return "parent-child"
	}
	return d.DependencyType
}

type issue struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	Labels       []string `json:"labels"`
	Dependencies []dep    `json:"dependencies"`
}

func runBD(args ...string) ([]byte, error) {
	cmd := exec.Command("bd", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bd %s: %w: %s", strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func loadIssue(issueID string) (issue, error) {
	out, err := runBD("show", issueID, "--json")
	if err != nil {
		return issue{}, err
	}
	var list []issue
	if err := json.Unmarshal(out, &list); err == nil && len(list) > 0 {
		return list[0], nil
	}
	var it issue
	if err := json.Unmarshal(out, &it); err != nil {
		return issue{}, err
	}
	return it, nil
}

func listIssueStatuses() (map[string]string, error) {
	out, err := runBD("list", "--json")
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	statuses := make(map[string]string, len(raw))
	for _, row := range raw {
		id, _ := row["id"].(string)
		status, _ := row["status"].(string)
		statuses[id] = status
	}
	return statuses, nil
}

func canTransition(current, target string) bool {
	nexts, ok := allowedTransitions[current]
	if !ok {
		return false
	}
	return nexts[target]
}

func currentFlow(it issue, root string) string {
	if flow, ok := readRunFlow(root, it.ID); ok {
		return flow
	}
	switch it.Status {
	case "in_progress":
		return "in_progress"
	case "closed", "done":
		return "done"
	default:
		return "open"
	}
}

func validate(issueID, target, root string) (bool, string, error) {
	it, err := loadIssue(issueID)
	if err != nil {
		return false, "", err
	}
	current := currentFlow(it, root)
	if !canTransition(current, target) {
		return false, fmt.Sprintf("transition %s -> %s is not allowed", current, target), nil
	}

	if target == "in_progress" {
		statuses, err := listIssueStatuses()
		if err != nil {
			return false, "", err
		}
		for _, d := range it.Dependencies {
			if d.IssueID != "" && d.IssueID != it.ID {
				continue
			}
			if d.kind() == "parent-child" {
				continue
			}
			st := d.Status
			if st == "" {
				st = statuses[d.refID()]
			}
			if st != "closed" && st != "done" {
				return false, "dependency not satisfied: " + d.refID(), nil
			}
		}
	}

	if target == "review" {
		if _, err := os.Stat(filepath.Join(root, ".sdp", "runs", issueID+".json")); err != nil {
			return false, "missing run packet", nil
		}
	}

	if target == "verified" || target == "done" {
		path := filepath.Join(root, ".sdp", "evidence", issueID+".json")
		res, err := evidence.ValidateStrictFile(path, target == "done")
		if err != nil {
			if os.IsNotExist(err) {
				return false, "missing evidence file", nil
			}
			return false, "invalid evidence json", nil
		}
		if !res.OK {
			if len(res.Missing) > 0 {
				return false, evidence.FormatMissing(res.Missing), nil
			}
			return false, res.Reason, nil
		}
	}

	return true, "ok", nil
}

func main() {
	issueID := flag.String("issue", "", "Issue ID")
	target := flag.String("to", "", "Target status")
	apply := flag.Bool("apply", false, "Apply transition when valid")
	flag.Parse()
	if *issueID == "" || *target == "" {
		fmt.Fprintln(os.Stderr, "--issue and --to are required")
		os.Exit(2)
	}

	root, _ := os.Getwd()
	ok, reason, err := validate(*issueID, *target, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := map[string]any{"issue": *issueID, "target": *target, "ok": ok, "reason": reason, "applied": false}
	if ok && *apply {
		if err := applyTransition(root, *issueID, *target); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		result["applied"] = true
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
	if !ok {
		os.Exit(2)
	}
}

func applyTransition(root, issueID, target string) error {
	switch target {
	case "open":
		if _, err := runBD("update", issueID, "--status", "open"); err != nil {
			return err
		}
		return writeRunFlow(root, issueID, "open")
	case "in_progress":
		if _, err := runBD("update", issueID, "--status", "in_progress"); err != nil {
			return err
		}
		return writeRunFlow(root, issueID, "in_progress")
	case "review", "verified", "blocked", "escalated":
		if _, err := runBD("update", issueID, "--status", "in_progress"); err != nil {
			return err
		}
		if err := writeRunFlow(root, issueID, target); err != nil {
			return err
		}
		_, err := runBD("update", issueID, "--append-notes", "protocol flow -> "+target)
		return err
	case "done":
		if err := writeRunFlow(root, issueID, "done"); err != nil {
			return err
		}
		_, err := runBD("close", issueID, "--reason", "protocol done")
		return err
	case "cancelled":
		if err := writeRunFlow(root, issueID, "cancelled"); err != nil {
			return err
		}
		_, err := runBD("close", issueID, "--reason", "protocol cancelled")
		return err
	default:
		return fmt.Errorf("unsupported target state: %s", target)
	}
}

func readRunFlow(root, issueID string) (string, bool) {
	path := filepath.Join(root, ".sdp", "runs", issueID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", false
	}
	v, _ := payload["flow"].(string)
	if v == "" {
		v, _ = payload["status"].(string)
	}
	if v == "" {
		return "", false
	}
	return v, true
}

func writeRunFlow(root, issueID, flow string) error {
	path := filepath.Join(root, ".sdp", "runs", issueID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return err
	}
	payload["flow"] = flow
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}
