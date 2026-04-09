package cli

import (
	"cmp"
	"fmt"
	"slices"
	"sort"
	"strings"

	"sdp_dev/internal/control"
)

func RenderDoctorControl(report *control.DoctorReport) string {
	var b strings.Builder
	issues := append([]control.DoctorCheck(nil), report.Checks...)
	sortDoctorChecks(issues)

	warnings := 0
	errors := 0
	byCheck := map[string]int{}
	for _, check := range issues {
		byCheck[check.CheckID]++
		if check.Severity == "error" {
			errors++
		} else {
			warnings++
		}
	}

	b.WriteString("DOCTOR CONTROL\n")
	fmt.Fprintf(&b, "Checks: %d total | %d passed | %d issues\n", report.TotalChecks, report.Passed, report.Failed)

	if len(issues) == 0 {
		b.WriteString("Status: healthy\n")
		b.WriteString("Next action: none — control store looks clean.\n")
		return strings.TrimSpace(b.String())
	}

	fmt.Fprintf(&b, "Status: action needed (%d errors, %d warnings)\n", errors, warnings)
	b.WriteString("Next action: fix errors first, then clear the oldest/stalest warnings.\n")
	b.WriteString("\nIssue groups:\n")
	for _, line := range summarizeCheckCounts(byCheck) {
		b.WriteString("- " + line + "\n")
	}

	b.WriteString("\nTop issues:\n")
	for _, check := range issues {
		fmt.Fprintf(&b, "- [%s] %s", strings.ToUpper(check.Severity), humanizeCheckID(check.CheckID))
		if check.ProjectID != "" || check.CardID != "" {
			b.WriteString(" — ")
			if check.ProjectID != "" {
				b.WriteString(check.ProjectID)
			}
			if check.CardID != "" {
				if check.ProjectID != "" {
					b.WriteString("/")
				}
				b.WriteString(check.CardID)
			}
		}
		b.WriteString("\n")
		b.WriteString("  " + check.Message + "\n")
		b.WriteString("  Next: " + recommendedDoctorAction(check) + "\n")
	}

	return strings.TrimSpace(b.String())
}

func sortDoctorChecks(checks []control.DoctorCheck) {
	slices.SortFunc(checks, func(a, b control.DoctorCheck) int {
		if c := cmp.Compare(severityRank(a.Severity), severityRank(b.Severity)); c != 0 {
			return c
		}
		if c := cmp.Compare(a.CheckID, b.CheckID); c != 0 {
			return c
		}
		if c := cmp.Compare(a.ProjectID, b.ProjectID); c != 0 {
			return c
		}
		return cmp.Compare(a.CardID, b.CardID)
	})
}

func severityRank(severity string) int {
	if severity == "error" {
		return 0
	}
	return 1
}

func summarizeCheckCounts(byCheck map[string]int) []string {
	keys := make([]string, 0, len(byCheck))
	for key := range byCheck {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %d", humanizeCheckID(key), byCheck[key]))
	}
	return lines
}

func recommendedDoctorAction(check control.DoctorCheck) string {
	scope := check.ProjectID
	if check.CardID != "" {
		scope += "/" + check.CardID
	}
	switch check.CheckID {
	case "missing-intake-artifact", "intake-artifact-not-found":
		return "restore or regenerate the intake artifact so the card has traceable intake context."
	case "ready-gate-missing":
		return "finish clarification, fill the ready-gate fields, then mark the card ready again."
	case "executing-without-beads":
		return "re-dispatch or repair Beads linkage before trusting this execution state."
	case "executing-without-dispatch-metadata":
		return "write dispatched_at / dispatched_to / dispatched_packet_path so operators can trace execution."
	case "executing-without-session":
		return "record the executor session id once a real runtime exists, or reconcile why execution is still only pending."
	case "executing-without-heartbeat":
		return "record the first executor heartbeat or relaunch/reconcile the runtime if nothing actually started."
	case "stale-executor-heartbeat":
		return "refresh the executor heartbeat or mark the runtime lost if the session is gone."
	case "executing-runtime-lost":
		return "either relaunch the executor and heartbeat it, or move the card out of executing so the board stops lying."
	case "needs-input-without-questions":
		return "add explicit feedback questions or decisions so the human knows what to answer."
	case "stale-ready-card":
		return "dispatch this ready card or park it if it should not move yet."
	case "stale-needs-input-card":
		return "nudge the requested human/admin and capture the missing answer."
	case "stale-blocked-card":
		return "make the blocking reason explicit and decide whether to unblock, escalate, or park it."
	case "done-without-result-summary":
		return "ingest or write the executor result summary so completion is auditable."
	default:
		if scope != "" {
			return "inspect " + scope + " and clear this hygiene issue."
		}
		return "inspect the affected card and clear this hygiene issue."
	}
}
