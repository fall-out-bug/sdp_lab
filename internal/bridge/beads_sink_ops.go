package bridge

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func (s *BeadsSink) createBeadsIssue(ctx context.Context, title, description string, priority int, labels []string) (string, error) {
	args := []string{
		"create",
		"--silent",
		"-p", fmt.Sprintf("%d", priority),
		"-t", "task",
		"-l", strings.Join(labels, ","),
		"-d", description,
	}
	if strings.TrimSpace(s.prefix) != "" {
		args = append(args, "--prefix", s.prefix)
	}
	args = append(args, title)

	cmd := exec.CommandContext(ctx, "bd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bd create failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	issueID := strings.TrimSpace(string(output))
	if issueID == "" {
		return "", fmt.Errorf("bd create returned empty issue id")
	}

	return issueID, nil
}

func (s *BeadsSink) updateBeadsIssue(ctx context.Context, issueID, title, description string, priority int, labels []string) error {
	args := []string{"update", issueID, "--title", title, "-d", description, "-p", fmt.Sprintf("%d", priority)}
	for _, label := range labels {
		args = append(args, "--add-label", label)
	}

	cmd := exec.CommandContext(ctx, "bd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd update %s failed: %w: %s", issueID, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (s *BeadsSink) reopenBeadsIssue(ctx context.Context, issueID, reason string) error {
	args := []string{"reopen", issueID}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	cmd := exec.CommandContext(ctx, "bd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd reopen %s failed: %w: %s", issueID, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (s *BeadsSink) closeBeadsIssue(ctx context.Context, issueID, reason string) error {
	args := []string{"close", issueID}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	cmd := exec.CommandContext(ctx, "bd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd close %s failed: %w: %s", issueID, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (s *BeadsSink) handleDryRunDecision(decision DedupeDecision, findingHash, payloadHash, title string) {
	switch decision.Action {
	case DedupeCreate:
		fmt.Printf("[DRY-RUN] Would create: %s\n", title)
		s.dedupe.RecordCreated(findingHash, payloadHash, "dry-run")
		s.mu.Lock()
		s.stats.Created++
		s.mu.Unlock()
	case DedupeUpdate:
		fmt.Printf("[DRY-RUN] Would update: %s\n", decision.IssueID)
		s.dedupe.RecordUpdated(findingHash, payloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
	case DedupeReopenUpdate:
		fmt.Printf("[DRY-RUN] Would reopen+update: %s\n", decision.IssueID)
		s.dedupe.RecordUpdated(findingHash, payloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
	default:
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
	}
}

func (s *BeadsSink) applyDecision(ctx context.Context, decision DedupeDecision, title, description string, priority int, labels []string) (string, error) {
	switch decision.Action {
	case DedupeCreate:
		issueID, err := s.createBeadsIssue(ctx, title, description, priority, labels)
		if err != nil {
			return "", err
		}
		s.dedupe.RecordCreated(decision.FindingHash, decision.PayloadHash, issueID)
		s.mu.Lock()
		s.stats.Created++
		s.mu.Unlock()
		return issueID, nil
	case DedupeUpdate:
		if err := s.updateBeadsIssue(ctx, decision.IssueID, title, description, priority, labels); err != nil {
			return "", err
		}
		s.dedupe.RecordUpdated(decision.FindingHash, decision.PayloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
		return decision.IssueID, nil
	case DedupeReopenUpdate:
		if err := s.reopenBeadsIssue(ctx, decision.IssueID, "finding payload changed"); err != nil {
			return "", err
		}
		if err := s.updateBeadsIssue(ctx, decision.IssueID, title, description, priority, labels); err != nil {
			return "", err
		}
		s.dedupe.RecordUpdated(decision.FindingHash, decision.PayloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
		return decision.IssueID, nil
	default:
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return decision.IssueID, nil
	}
}

// bdListAll runs bd list --all --json -n 0 and returns the raw output.
func bdListAll(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "bd", "list", "--all", "--json", "-n", "0")
	return cmd.Output()
}
