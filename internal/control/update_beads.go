package control

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

var createBeadsIssueFn = createBeadsIssue

type bdCreateOutput struct {
	ID string `json:"id"`
}

func createBeadsIssue(card *FeatureCard) (string, error) {
	title := fmt.Sprintf("%s [%s]", card.Title, card.ID)
	description := buildBeadsDescription(card)

	priority := "2"
	switch strings.ToLower(card.RiskLevel) {
	case "high":
		priority = "1"
	case "low":
		priority = "3"
	}

	args := []string{
		"create", title,
		"--description=" + description,
		"--type=feature",
		"--priority=" + priority,
		"--json",
	}

	cmd := exec.CommandContext(context.Background(), "bd", args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	jsonOutput := findJSONInOutput(outputStr)
	if jsonOutput == "" {
		if err != nil {
			return "", fmt.Errorf("bd create failed: %w, output: %s", err, outputStr)
		}
		return "", fmt.Errorf("no JSON found in bd create output: %s", outputStr)
	}

	var result bdCreateOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		return "", fmt.Errorf("parse bd create output: %w, output: %s", err, jsonOutput)
	}

	return result.ID, nil
}

func findJSONInOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") {
			var buf strings.Builder
			for j := i; j < len(lines); j++ {
				buf.WriteString(lines[j])
				if strings.HasSuffix(strings.TrimSpace(lines[j]), "}") {
					break
				}
				if j < len(lines)-1 {
					buf.WriteString("\n")
				}
			}
			candidate := buf.String()
			if json.Valid([]byte(candidate)) {
				return candidate
			}
		}
	}
	return ""
}

func buildBeadsDescription(card *FeatureCard) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "FeatureCard: %s\n\n", card.ID)
	fmt.Fprintf(&sb, "Project: %s\n\n", card.ProjectID)

	if card.NormalizedIntent != "" {
		fmt.Fprintf(&sb, "Normalized Intent: %s\n\n", card.NormalizedIntent)
	}

	if card.TaskType != "" {
		fmt.Fprintf(&sb, "Task Type: %s\n", card.TaskType)
	}
	if card.ExecutionMode != "" {
		fmt.Fprintf(&sb, "Execution Mode: %s\n", card.ExecutionMode)
	}
	if card.TargetRepo != "" {
		fmt.Fprintf(&sb, "Target Repo: %s\n", card.TargetRepo)
	}
	if card.TargetArea != "" {
		fmt.Fprintf(&sb, "Target Area: %s\n", card.TargetArea)
	}
	sb.WriteString("\n")

	if len(card.ScopeIn) > 0 {
		sb.WriteString("Scope In:\n")
		for _, item := range card.ScopeIn {
			fmt.Fprintf(&sb, "- %s\n", item)
		}
		sb.WriteString("\n")
	}

	if len(card.ScopeOut) > 0 {
		sb.WriteString("Scope Out:\n")
		for _, item := range card.ScopeOut {
			fmt.Fprintf(&sb, "- %s\n", item)
		}
		sb.WriteString("\n")
	}

	if len(card.NonGoals) > 0 {
		sb.WriteString("Non-goals:\n")
		for _, item := range card.NonGoals {
			fmt.Fprintf(&sb, "- %s\n", item)
		}
		sb.WriteString("\n")
	}

	if len(card.AcceptanceShape) > 0 {
		sb.WriteString("Acceptance Criteria:\n")
		for _, item := range card.AcceptanceShape {
			fmt.Fprintf(&sb, "- %s\n", item)
		}
		sb.WriteString("\n")
	}

	if card.RecommendedNext != "" {
		fmt.Fprintf(&sb, "Recommended Next Step: %s\n", card.RecommendedNext)
	}

	return sb.String()
}

func SetCreateBeadsIssueFn(fn func(*FeatureCard) (string, error)) {
	createBeadsIssueFn = fn
}

func MockCreateBeadsIssue(id string) func(*FeatureCard) (string, error) {
	return func(*FeatureCard) (string, error) { return id, nil }
}
