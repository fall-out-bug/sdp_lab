package agentloop

import (
	"fmt"
	"strings"
	"sync"

	"sdp_dev/internal/harness"
)

// EvidenceAccumulator collects structured evidence from tool results.
// The agent cannot self-report gate passage — only tool outcomes count (I9).
// Fix Q2: quality map initialized in constructor — no nil map panic in OnToolResult.
// Fix A6: Reset() explicitly defined — called by transitionTo on phase change.
type EvidenceAccumulator struct {
	mu       sync.Mutex
	evidence []string
	claims   []harness.Claim
	quality  map[string]bool
}

// NewEvidenceAccumulator creates an EvidenceAccumulator with initialized maps (Fix Q2).
func NewEvidenceAccumulator() *EvidenceAccumulator {
	return &EvidenceAccumulator{
		quality: make(map[string]bool),
	}
}

// OnToolResult is called via AfterToolCall hook after each tool execution (Fix N5: full ToolResult).
// Tool errors are recorded as negative evidence, not ignored.
// Structured per-tool extraction — no LLM summarization.
func (ea *EvidenceAccumulator) OnToolResult(r ToolResult) error {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	if r.Err != nil {
		// Tool failure: negative evidence — EvidenceAccumulator must know about failures.
		ea.evidence = append(ea.evidence, fmt.Sprintf("tool_error:%s:%s", r.Name, r.Err.Error()))
		return nil
	}

	switch r.Name {
	case "bash":
		// PASS in output → quality["test"] = true; FAIL → false (explicit).
		ea.quality["test"] = extractTestPass(r.Output)
	case "edit_file":
		ea.evidence = append(ea.evidence, "file_modified:"+extractFilePath(r.Output))
	case "bd_create":
		ea.evidence = append(ea.evidence, "card_created:"+extractCardID(r.Output))
	}
	return nil
}

// Reset clears all accumulated evidence, claims, and quality without reallocation (Fix A6).
// Called by transitionTo on every phase change so the next phase starts fresh.
func (ea *EvidenceAccumulator) Reset() {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.evidence = ea.evidence[:0]
	ea.claims = ea.claims[:0]
	for k := range ea.quality {
		delete(ea.quality, k)
	}
}

// Snapshot returns a point-in-time copy of accumulated evidence for the given phase.
// Thread-safe — copies slices so callers cannot race on the original.
func (ea *EvidenceAccumulator) Snapshot(phase Role) PhaseSnapshot {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	evidence := make([]string, len(ea.evidence))
	copy(evidence, ea.evidence)

	claims := make([]harness.Claim, len(ea.claims))
	copy(claims, ea.claims)

	quality := make(map[string]bool, len(ea.quality))
	for k, v := range ea.quality {
		quality[k] = v
	}

	return PhaseSnapshot{
		Phase:    phase,
		Evidence: evidence,
		Claims:   claims,
		Quality:  quality,
	}
}

// ---- per-tool extractors ----

// extractTestPass returns true if the bash output indicates test success.
// Heuristic: presence of " PASS" or "ok " prefix (go test output convention).
func extractTestPass(output string) bool {
	return strings.Contains(output, "PASS") && !strings.Contains(output, "FAIL")
}

// extractFilePath extracts a file path from edit_file output.
// edit_file outputs something like "wrote path/to/file.go" or the path directly.
func extractFilePath(output string) string {
	// Heuristic: last word often contains the path.
	parts := strings.Fields(output)
	if len(parts) == 0 {
		return output
	}
	return parts[len(parts)-1]
}

// extractCardID extracts a card ID from bd_create output.
// bd_create outputs something like "created card PROJ-42".
func extractCardID(output string) string {
	parts := strings.Fields(output)
	if len(parts) == 0 {
		return output
	}
	return parts[len(parts)-1]
}
