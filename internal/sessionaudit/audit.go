package sessionaudit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	nudgeRe     = regexp.MustCompile(`(?i)^(да|ок|окей|yes|ok|continue|продолжай|погнали|давай|go|next|готово|хорошо|верно|гут|кайф|разумно)$`)
	ctxLossRe   = regexp.MustCompile(`(?i)continue from where you left off`)
	reviewSkill = regexp.MustCompile(`(?i)(review|reality.?check|verify.?workstream)`)
	bdCloseRe   = regexp.MustCompile(`bd close\s+([\w-]+)`)
)

// Session holds per-session metrics.
type Session struct {
	ID              string         `json:"session_id"`
	SizeMB          float64        `json:"size_mb"`
	UserMsgs        int            `json:"user_msgs"`
	AssistantMsgs   int            `json:"assistant_msgs"`
	Nudges          []string       `json:"nudges"`
	CtxLossCount    int            `json:"context_loss_count"`
	SkillCalls      map[string]int `json:"skill_calls"`
	ToolCalls       map[string]int `json:"tool_calls"`
	ReviewIter      int            `json:"review_iterations"`
	IssuesClosed    []string       `json:"issues_closed"`
	FirstTS         string         `json:"first_ts,omitempty"`
	LastTS          string         `json:"last_ts,omitempty"`
}

// NudgeRate returns nudge+ctx-loss percentage of user messages.
func (s *Session) NudgeRate() float64 {
	if s.UserMsgs == 0 {
		return 0
	}
	return float64(len(s.Nudges)+s.CtxLossCount) / float64(s.UserMsgs) * 100
}

// Productivity returns closed issues per 100 user messages.
func (s *Session) Productivity() float64 {
	if s.UserMsgs == 0 {
		return 0
	}
	return float64(len(s.IssuesClosed)) / float64(s.UserMsgs) * 100
}

// Aggregate holds cross-session summary.
type Aggregate struct {
	SessionsAnalyzed    int            `json:"sessions_analyzed"`
	TotalUserMsgs       int            `json:"total_user_msgs"`
	TotalNudges         int            `json:"total_nudges"`
	TotalCtxLoss        int            `json:"total_context_loss"`
	TotalReviewIter     int            `json:"total_review_iterations"`
	TotalIssuesClosed   int            `json:"total_issues_closed"`
	TopSkills           []SkillCount   `json:"top_skills"`
	NudgeHeavy          []string       `json:"nudge_heavy_sessions"`
	CtxLossSessions     []string       `json:"ctx_loss_sessions"`
	MostProductive      []string       `json:"most_productive_sessions"`
}

// SkillCount is a name+count pair for JSON serialisation.
type SkillCount struct {
	Skill string `json:"skill"`
	Count int    `json:"count"`
}

// jsonRecord mirrors the minimal JSONL structure we need.
type jsonRecord struct {
	Timestamp string `json:"timestamp"`
	Message   struct {
		Role      string          `json:"role"`
		Content   json.RawMessage `json:"content"`
		Timestamp string          `json:"timestamp"`
	} `json:"message"`
}

// ParseSession reads one JSONL file and returns a Session.
func ParseSession(path string) (*Session, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	s := &Session{
		ID:           strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		SizeMB:       float64(info.Size()) / 1024 / 1024,
		SkillCalls:   make(map[string]int),
		ToolCalls:    make(map[string]int),
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var rec jsonRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}

		ts := rec.Timestamp
		if ts == "" {
			ts = rec.Message.Timestamp
		}
		if ts != "" {
			if s.FirstTS == "" {
				s.FirstTS = ts
			}
			s.LastTS = ts
		}

		switch rec.Message.Role {
		case "user":
			s.UserMsgs++
			text := extractText(rec.Message.Content)
			if text == "" {
				continue
			}
			if ctxLossRe.MatchString(text) {
				s.CtxLossCount++
			} else if isNudge(text) {
				s.Nudges = append(s.Nudges, truncate(strings.TrimSpace(text), 60))
			}

		case "assistant":
			s.AssistantMsgs++
			processToolUses(rec.Message.Content, s)
		}
	}

	return s, scanner.Err()
}

func isNudge(text string) bool {
	t := strings.TrimSpace(text)
	if len(t) > 80 {
		return false
	}
	return nudgeRe.MatchString(t)
}

func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try string first.
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	// Try array of content blocks.
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, " ")
}

func processToolUses(raw json.RawMessage, s *Session) {
	var blocks []struct {
		Type  string          `json:"type"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return
	}
	for _, b := range blocks {
		if b.Type != "tool_use" {
			continue
		}
		s.ToolCalls[b.Name]++

		if b.Name == "Skill" {
			var inp struct {
				Skill string `json:"skill"`
			}
			if json.Unmarshal(b.Input, &inp) == nil && inp.Skill != "" {
				s.SkillCalls[inp.Skill]++
				if reviewSkill.MatchString(inp.Skill) {
					s.ReviewIter++
				}
			}
		}

		if b.Name == "Bash" {
			var inp struct {
				Command string `json:"command"`
			}
			if json.Unmarshal(b.Input, &inp) == nil {
				for _, m := range bdCloseRe.FindAllStringSubmatch(inp.Command, -1) {
					s.IssuesClosed = append(s.IssuesClosed, m[1])
				}
			}
		}
	}
}

// LoadSessions reads all JSONL files from dir, optionally filtered by since.
func LoadSessions(dir string, since time.Time, limit int) ([]*Session, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	type entry struct {
		path string
		size int64
		mod  time.Time
	}
	var files []entry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !since.IsZero() && info.ModTime().Before(since) {
			continue
		}
		files = append(files, entry{
			path: filepath.Join(dir, e.Name()),
			size: info.Size(),
			mod:  info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].size > files[j].size })
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	var sessions []*Session
	for _, f := range files {
		s, err := ParseSession(f.path)
		if err != nil || s.UserMsgs < 3 {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// Summarise computes the Aggregate across sessions.
func Summarise(sessions []*Session) *Aggregate {
	agg := &Aggregate{}
	skillTotals := make(map[string]int)

	for _, s := range sessions {
		agg.SessionsAnalyzed++
		agg.TotalUserMsgs += s.UserMsgs
		agg.TotalNudges += len(s.Nudges)
		agg.TotalCtxLoss += s.CtxLossCount
		agg.TotalReviewIter += s.ReviewIter
		agg.TotalIssuesClosed += len(s.IssuesClosed)
		for k, v := range s.SkillCalls {
			skillTotals[k] += v
		}
	}

	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range skillTotals {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })
	max := 10
	if len(sorted) < max {
		max = len(sorted)
	}
	for _, kv := range sorted[:max] {
		agg.TopSkills = append(agg.TopSkills, SkillCount{kv.k, kv.v})
	}

	top3 := func(less func(a, b *Session) bool) []string {
		cp := make([]*Session, len(sessions))
		copy(cp, sessions)
		sort.Slice(cp, func(i, j int) bool { return less(cp[i], cp[j]) })
		n := 3
		if len(cp) < n {
			n = len(cp)
		}
		ids := make([]string, n)
		for i, s := range cp[:n] {
			end := 12
			if len(s.ID) < end {
				end = len(s.ID)
			}
			ids[i] = s.ID[:end]
		}
		return ids
	}

	agg.NudgeHeavy = top3(func(a, b *Session) bool { return a.NudgeRate() > b.NudgeRate() })
	agg.CtxLossSessions = top3(func(a, b *Session) bool { return a.CtxLossCount > b.CtxLossCount })
	agg.MostProductive = top3(func(a, b *Session) bool { return a.Productivity() > b.Productivity() })

	return agg
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
