package finetune

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// beadsRecord is the subset of fields we care about from .beads/issues.jsonl.
type beadsRecord struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Body     string   `json:"description"`
	Priority int      `json:"priority"`
	Type     string   `json:"issue_type"`
	Labels   []string `json:"labels"`
	Status   string   `json:"status"`
}

// LoadBeads reads JSONL and returns one Sample per usable issue. Lines with
// missing labels are skipped (counted in skipped).
func LoadBeads(path string) (samples []Sample, skipped map[string]int, err error) {
	skipped = map[string]int{}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("finetune: open beads jsonl: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r beadsRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			skipped["json_parse_error"]++
			continue
		}
		s, reason := beadsToSample(r)
		if reason != "" {
			skipped[reason]++
			continue
		}
		samples = append(samples, s)
	}
	if err := sc.Err(); err != nil {
		return nil, nil, fmt.Errorf("finetune: scan beads jsonl: %w", err)
	}
	return samples, skipped, nil
}

func beadsToSample(r beadsRecord) (Sample, string) {
	if r.Title == "" {
		return Sample{}, "empty_title"
	}

	risk := DeriveRiskFromPriority(fmtPriority(r.Priority))
	if risk == "" {
		return Sample{}, "missing_priority"
	}

	taskType := taskTypeFromBeads(r)
	if taskType == "" {
		return Sample{}, "task_type_unresolved"
	}

	complexity := complexityFromBeads(r)

	label := Label{Complexity: complexity, TaskType: taskType, Risk: risk}
	user := buildUserPrompt(r.Title, r.Body)
	assistant := mustMarshalLabel(label)

	return Sample{
		Messages: []Message{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: user},
			{Role: "assistant", Content: assistant},
		},
		Meta: SampleMeta{
			Source:   "beads",
			SourceID: r.ID,
			Real:     true,
			Label:    label,
			InputKey: hashStr(user),
		},
	}, ""
}

// fmtPriority returns "P0".."P4" for numeric priorities so DeriveRiskFromPriority
// can reuse its mapping.
func fmtPriority(p int) string {
	if p < 0 || p > 9 {
		return ""
	}
	return fmt.Sprintf("P%d", p)
}

// taskTypeFromBeads prefers explicit issue_type, falls back to label/title heuristics.
func taskTypeFromBeads(r beadsRecord) string {
	switch strings.ToLower(r.Type) {
	case "bug":
		return "bugfix"
	case "feature":
		return "feature"
	case "task":
		// task is generic — fall through to keyword detection
	}
	if t := DeriveTaskType(r.Title, r.Body); t != "" {
		return t
	}
	for _, l := range r.Labels {
		ll := strings.ToLower(l)
		if ll == "bug" || ll == "fix" {
			return "bugfix"
		}
		if ll == "test" || ll == "tests" {
			return "test"
		}
		if ll == "docs" || ll == "doc" {
			return "docs"
		}
	}
	return ""
}

// complexityFromBeads has no native size field. We use a conservative
// 2-bucket heuristic: empty body → low (title-only triage tickets are usually
// trivial); any non-empty body → medium. We deliberately avoid emitting "high"
// from beads to minimise label noise — the proxy is too weak to discriminate
// medium from high reliably.
func complexityFromBeads(r beadsRecord) string {
	if strings.TrimSpace(r.Body) == "" {
		return "low"
	}
	return "medium"
}
