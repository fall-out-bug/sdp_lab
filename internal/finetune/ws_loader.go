package finetune

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// wsFrontMatter mirrors the YAML header of docs/workstreams/backlog/*.md.
type wsFrontMatter struct {
	WSID      string   `yaml:"ws_id"`
	FeatureID string   `yaml:"feature_id"`
	Status    string   `yaml:"status"`
	Priority  string   `yaml:"priority"`
	Size      string   `yaml:"size"`
	DependsOn []string `yaml:"depends_on"`
}

// LoadWorkstreams walks dir and returns one Sample per parseable ws file with
// a complete derived label. Files with missing fields are skipped and
// reported in the returned skipReasons map (for builder diagnostics).
func LoadWorkstreams(dir string) (samples []Sample, skipped map[string]int, err error) {
	skipped = map[string]int{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("finetune: read ws dir %s: %w", dir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		s, reason := loadOneWS(path)
		if reason != "" {
			skipped[reason]++
			continue
		}
		samples = append(samples, s)
	}
	return samples, skipped, nil
}

func loadOneWS(path string) (Sample, string) {
	f, err := os.Open(path)
	if err != nil {
		return Sample{}, "open_error"
	}
	defer f.Close()

	fmYAML, body, ok := splitFrontMatter(f)
	if !ok {
		return Sample{}, "no_frontmatter"
	}

	var fm wsFrontMatter
	if err := yaml.Unmarshal([]byte(fmYAML), &fm); err != nil {
		return Sample{}, "yaml_parse_error"
	}

	complexity := DeriveComplexity(fm.Size)
	risk := DeriveRiskFromPriority(fm.Priority)
	if complexity == "" || risk == "" {
		return Sample{}, "missing_size_or_priority"
	}

	title, goal := extractTitleAndGoal(body)
	if title == "" {
		return Sample{}, "no_title"
	}
	taskType := DeriveTaskType(title, goal)
	if taskType == "" {
		return Sample{}, "task_type_unresolved"
	}

	label := Label{Complexity: complexity, TaskType: taskType, Risk: risk}
	user := buildUserPrompt(title, goal)
	assistant := mustMarshalLabel(label)

	return Sample{
		Messages: []Message{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: user},
			{Role: "assistant", Content: assistant},
		},
		Meta: SampleMeta{
			Source:   "ws",
			SourceID: fm.WSID,
			Real:     true,
			Label:    label,
			InputKey: hashStr(user),
		},
	}, ""
}

// splitFrontMatter reads --- ... --- block from the top of the file. Returns
// the YAML payload (without delimiters), the remaining body, and ok=true on
// success.
func splitFrontMatter(f *os.File) (string, string, bool) {
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var fmLines, bodyLines []string
	state := 0 // 0: before first ---, 1: inside, 2: after closing ---
	for sc.Scan() {
		line := sc.Text()
		switch state {
		case 0:
			if strings.TrimSpace(line) == "---" {
				state = 1
				continue
			}
			return "", "", false // no frontmatter
		case 1:
			if strings.TrimSpace(line) == "---" {
				state = 2
				continue
			}
			fmLines = append(fmLines, line)
		case 2:
			bodyLines = append(bodyLines, line)
		}
	}
	if state != 2 {
		return "", "", false
	}
	return strings.Join(fmLines, "\n"), strings.Join(bodyLines, "\n"), true
}

// extractTitleAndGoal pulls the first H1 and ALL non-empty content under
// `## Goal` until the next `## ` heading (or end-of-body). Multi-paragraph
// goals are preserved (paragraphs joined with " "; blank lines collapse).
func extractTitleAndGoal(body string) (title, goal string) {
	lines := strings.Split(body, "\n")
	var goalBuf []string
	inGoal := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if title == "" && strings.HasPrefix(trim, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(trim, "#"))
			continue
		}
		if strings.HasPrefix(trim, "## ") {
			if inGoal {
				break // next H2 ends the Goal section
			}
			if strings.EqualFold(trim, "## Goal") {
				inGoal = true
			}
			continue
		}
		if inGoal && trim != "" {
			goalBuf = append(goalBuf, trim)
		}
		// Blank lines inside Goal section are ignored — we keep collecting
		// until the next H2 or end of body.
	}
	goal = strings.Join(goalBuf, " ")
	return title, goal
}

func buildUserPrompt(title, goal string) string {
	if goal == "" {
		return "Title: " + title
	}
	return "Title: " + title + "\nGoal: " + goal
}
