package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"sdp_dev/internal/control"
	"sdp_dev/internal/kernel"
	"sdp_dev/internal/orchestrate"
)

// ExecutorBridge connects dispatched execution packets to the configured LLM invoker.
type ExecutorBridge struct {
	Store       *control.Store
	Invoker     kernel.RuntimeAdapter
	ProjectRoot string
}

func (b *ExecutorBridge) DispatchAndRun(ctx context.Context, projectID, cardID string) (*control.ExecutorResultPacket, error) {
	if b == nil || b.Store == nil {
		return nil, fmt.Errorf("nil executor bridge/store")
	}
	if b.Invoker == nil {
		b.Invoker = orchestrate.DefaultLLMInvoker
	}

	card, err := b.Store.LoadCard(projectID, cardID)
	if err != nil {
		return nil, fmt.Errorf("load card: %w", err)
	}
	if strings.TrimSpace(card.DispatchedPacketPath) == "" {
		return nil, fmt.Errorf("card %s has no dispatched packet path", cardID)
	}

	packet, err := loadExecutionPacket(card.DispatchedPacketPath)
	if err != nil {
		return nil, fmt.Errorf("load execution packet: %w", err)
	}

	agent := mapExecutorRole(packet.ExecutorRole)
	prompt := buildDispatchPrompt(packet)
	if err := RecordDispatchProvenance(b.ProjectRoot, card, packet, prompt); err != nil {
		return nil, fmt.Errorf("record dispatch provenance: %w", err)
	}
	promptHash := orchestrate.ComputePromptHash(prompt)
	_ = orchestrate.WritePromptProvenance(b.ProjectRoot, promptHash, orchestrate.BuildContextSources(b.ProjectRoot, card.ID, "", nil))

	now := time.Now().UTC()
	sessionID := fmt.Sprintf("exec-%d", now.UnixNano())
	card.ExecutorRuntimeState = control.ExecutorRuntimeRunning
	card.ExecutorSessionID = sessionID
	card.ExecutorStartedAt = now.Format(time.RFC3339)
	card.LastExecutorHeartbeatAt = now.Format(time.RFC3339)
	card.ExecutorProgressSummary = "Executor launched from dispatched packet"
	if err := b.Store.SaveCard(card); err != nil {
		return nil, fmt.Errorf("save running card state: %w", err)
	}

	runtimeResult, err := b.Invoker.Invoke(ctx, kernel.RuntimeInvocation{
		WorkDir: b.ProjectRoot,
		Agent:   agent,
		Prompt:  prompt,
	})
	if err != nil {
		card.ExecutorRuntimeState = "failed"
		card.ExecutorProgressSummary = strings.TrimSpace(err.Error())
		card.ExecutorResult = &control.ExecutorResultSummary{
			Status:     string(control.ResultStatusFailed),
			Summary:    strings.TrimSpace(err.Error()),
			ReceivedAt: time.Now().UTC().Format(time.RFC3339),
		}
		if saveErr := b.Store.SaveCard(card); saveErr != nil {
			fmt.Fprintf(os.Stderr, "warning: save card on failure: %v\n", saveErr)
		}
		return nil, fmt.Errorf("invoke executor: %w", err)
	}

	result := translateResult(packet, runtimeResult.Output, runtimeResult.ExitCode)
	if err := RouteFindingsToCard(b.Store, projectID, cardID, result); err != nil {
		return nil, fmt.Errorf("route findings to card: %w", err)
	}
	card, err = b.Store.LoadCard(projectID, cardID)
	if err != nil {
		return nil, fmt.Errorf("reload card after findings routing: %w", err)
	}
	resultPath, writeErr := b.writeResult(cardID, result)
	if writeErr != nil {
		return nil, writeErr
	}

	completedAt := time.Now().UTC()
	card.LastExecutorHeartbeatAt = completedAt.Format(time.RFC3339)
	card.ExecutorProgressSummary = result.Summary
	card.ExecutorResult = summarizeResult(result, completedAt)
	if result.Status == control.ResultStatusSuccess {
		card.ExecutorRuntimeState = control.ExecutorRuntimeCompleted
	} else {
		card.ExecutorRuntimeState = "failed"
	}
	if err := b.Store.SaveCard(card); err != nil {
		return nil, fmt.Errorf("save completed card state: %w", err)
	}

	_ = resultPath
	return result, nil
}

func loadExecutionPacket(path string) (*control.ExecutionPacket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var packet control.ExecutionPacket
	if err := json.Unmarshal(data, &packet); err != nil {
		return nil, err
	}
	return &packet, nil
}

func mapExecutorRole(role string) string {
	switch strings.TrimSpace(role) {
	case string(control.ExecutorRoleReview):
		return "reviewer"
	case string(control.ExecutorRoleOmOImplementation):
		return "implementer"
	default:
		return "implementer"
	}
}

func buildDispatchPrompt(packet *control.ExecutionPacket) string {
	var b strings.Builder
	b.WriteString("Objective:\n")
	b.WriteString(strings.TrimSpace(packet.Objective))
	b.WriteString("\n\n")
	if len(packet.ScopeIn) > 0 {
		b.WriteString("Scope In:\n")
		for _, item := range packet.ScopeIn {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(item))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(packet.ScopeOut) > 0 {
		b.WriteString("Scope Out:\n")
		for _, item := range packet.ScopeOut {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(item))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(packet.Constraints) > 0 {
		b.WriteString("Constraints:\n")
		for _, item := range packet.Constraints {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(item))
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func translateResult(packet *control.ExecutionPacket, output string, exitCode int) *control.ExecutorResultPacket {
	result := &control.ExecutorResultPacket{
		BeadsTaskID:         packet.BeadsTaskID,
		ParentFeatureID:     packet.ParentFeatureID,
		ExecutorRole:        packet.ExecutorRole,
		RecommendedNextStep: packet.NextHandoffTarget,
	}
	trimmed := strings.TrimSpace(output)
	if exitCode == 0 {
		result.Status = control.ResultStatusSuccess
		result.Summary = "Execution completed successfully"
		if commit := extractCommitHash(trimmed); commit != "" {
			result.Artifacts = append(result.Artifacts, control.ExecutorArtifact{Type: "commit", Reference: commit, Description: "Commit produced by executor"})
			result.Summary = fmt.Sprintf("Execution completed successfully (%s)", commit)
		}
		return result
	}
	result.Status = control.ResultStatusFailed
	if trimmed == "" {
		result.Summary = fmt.Sprintf("Execution failed with exit code %d", exitCode)
	} else {
		result.Summary = trimmed
	}
	return result
}

var commitHashRe = regexp.MustCompile(`\b[0-9a-fA-F]{40}\b|\b[0-9a-fA-F]{7,12}\b`)

func extractCommitHash(output string) string {
	matches := commitHashRe.FindAllString(output, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

func (b *ExecutorBridge) writeResult(cardID string, result *control.ExecutorResultPacket) (string, error) {
	resultsDir := filepath.Join(b.Store.ControlRoot, "executor-results")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir executor-results: %w", err)
	}
	filename := fmt.Sprintf("%s-%d.json", cardID, time.Now().UTC().Unix())
	path := filepath.Join(resultsDir, filename)
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal executor result: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write executor result: %w", err)
	}
	return path, nil
}

func summarizeResult(result *control.ExecutorResultPacket, at time.Time) *control.ExecutorResultSummary {
	summary := &control.ExecutorResultSummary{
		Status:              string(result.Status),
		Summary:             result.Summary,
		ReceivedAt:          at.Format(time.RFC3339),
		Findings:            result.Findings,
		OpenRisks:           result.OpenRisks,
		RecommendedNextStep: result.RecommendedNextStep,
	}
	for _, artifact := range result.Artifacts {
		summary.Artifacts = append(summary.Artifacts, fmt.Sprintf("%s: %s", artifact.Type, artifact.Reference))
	}
	return summary
}
