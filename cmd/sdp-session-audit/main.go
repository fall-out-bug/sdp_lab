package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/sessionaudit"
)

func main() {
	top := flag.Int("top", 0, "Limit to N largest sessions (0 = all)")
	sessionID := flag.String("session", "", "Analyse single session by ID prefix")
	asJSON := flag.Bool("json", false, "Output JSON")
	since := flag.String("since", "", "Only sessions newer than e.g. 7d, 24h, 2w")
	detail := flag.Bool("detail", false, "Show nudge message text per session")
	flag.Parse()

	dir := filepath.Join(os.Getenv("HOME"), ".claude/projects/-Users-fall-out-bug-projects-vibe-coding-sdp-lab")
	if d := os.Getenv("SESSION_AUDIT_DIR"); d != "" {
		dir = d
	}

	var sinceTime time.Time
	if *since != "" {
		d, err := parseSince(*since)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR:", err)
			os.Exit(1)
		}
		sinceTime = time.Now().Add(-d)
	}

	sessions, err := sessionaudit.LoadSessions(dir, sinceTime, *top)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}

	if *sessionID != "" {
		sessions = filterByID(sessions, *sessionID)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return
	}

	if *sessionID != "" && len(sessions) == 1 {
		if *asJSON {
			printJSON(sessions[0])
		} else {
			fmt.Println(fmtSession(sessions[0], true))
		}
		return
	}

	agg := sessionaudit.Summarise(sessions)

	if *asJSON {
		printJSON(map[string]any{"aggregate": agg, "sessions": sessions})
		return
	}

	fmt.Println(fmtAggregate(agg))
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("PER-SESSION BREAKDOWN")
	fmt.Println(strings.Repeat("=", 60))
	for _, s := range sessions {
		fmt.Println()
		fmt.Println(fmtSession(s, *detail))
	}
}

func filterByID(sessions []*sessionaudit.Session, prefix string) []*sessionaudit.Session {
	var out []*sessionaudit.Session
	for _, s := range sessions {
		if strings.HasPrefix(s.ID, prefix) {
			out = append(out, s)
		}
	}
	return out
}

func fmtAggregate(agg *sessionaudit.Aggregate) string {
	sep := strings.Repeat("─", 56)
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }

	line(strings.Repeat("=", 60))
	line("SESSION AUDIT — AGGREGATE")
	line(strings.Repeat("=", 60))
	line("Sessions analyzed  : %d", agg.SessionsAnalyzed)
	line("Total user messages: %d", agg.TotalUserMsgs)
	line("")
	line("── Autonomy health %s", sep[18:])
	line("Nudges ('ок'/'да'/'давай')    : %d", agg.TotalNudges)
	line("Context-loss ('Continue from'): %d", agg.TotalCtxLoss)
	line("Review iterations             : %d", agg.TotalReviewIter)
	line("Issues closed                 : %d", agg.TotalIssuesClosed)
	line("")
	line("── Worst autonomy (most nudges) %s", sep[31:])
	for _, id := range agg.NudgeHeavy {
		line("  %s", id)
	}
	line("")
	line("── Context-loss sessions %s", sep[24:])
	for _, id := range agg.CtxLossSessions {
		line("  %s", id)
	}
	line("")
	line("── Most productive sessions %s", sep[27:])
	for _, id := range agg.MostProductive {
		line("  %s", id)
	}
	line("")
	line("── Top skills %s", sep[13:])
	for _, sc := range agg.TopSkills {
		line("  %4dx  %s", sc.Count, sc.Skill)
	}
	return strings.TrimRight(b.String(), "\n")
}

func fmtSession(s *sessionaudit.Session, detail bool) string {
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }

	line("Session %s  (%.1f MB)", s.ID[:min(12, len(s.ID))], s.SizeMB)
	line("  Messages : user=%d  assistant=%d", s.UserMsgs, s.AssistantMsgs)
	line("  Nudges   : %d (%.0f%% of user msgs)", len(s.Nudges), s.NudgeRate())
	line("  CtxLoss  : %dx 'Continue from where'", s.CtxLossCount)
	line("  Reviews  : %d review skill calls", s.ReviewIter)
	line("  Closed   : %d issues  (productivity=%.2f per 100 msgs)", len(s.IssuesClosed), s.Productivity())

	if len(s.SkillCalls) > 0 {
		line("  Skills   : %s", topN(s.SkillCalls, 5))
	}
	if len(s.ToolCalls) > 0 {
		line("  Tools    : %s", topN(s.ToolCalls, 5))
	}
	if detail && len(s.Nudges) > 0 {
		line("  Nudge msgs:")
		for i, n := range s.Nudges {
			if i >= 10 {
				break
			}
			line("    · %s", n)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func topN(m map[string]int, n int) string {
	type kv struct {
		k string
		v int
	}
	var pairs []kv
	for k, v := range m {
		pairs = append(pairs, kv{k, v})
	}
	// simple selection sort for small n
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].v > pairs[i].v {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	if len(pairs) > n {
		pairs = pairs[:n]
	}
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = fmt.Sprintf("%s×%d", p.k, p.v)
	}
	return strings.Join(parts, ", ")
}

func parseSince(spec string) (time.Duration, error) {
	if len(spec) < 2 {
		return 0, fmt.Errorf("invalid --since %q: use e.g. 7d, 24h, 2w", spec)
	}
	unit := spec[len(spec)-1]
	var n int
	if _, err := fmt.Sscanf(spec[:len(spec)-1], "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid --since %q", spec)
	}
	switch unit {
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit %q in --since (use h/d/w)", string(unit))
	}
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v) //nolint:errcheck
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
