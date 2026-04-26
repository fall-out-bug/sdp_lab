package finetune

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ValidationResult is the outcome of validating a single JSONL file.
type ValidationResult struct {
	Path        string
	LineCount   int
	Errors      []ValidationError
	UniqueUsers int
}

// ValidationError points at one bad line.
type ValidationError struct {
	Line   int
	Reason string
}

const (
	minUserChars = 20
	maxUserChars = 4000
)

// ValidJSONL streams path and checks every line.
//
// Rules:
//   - parses as JSON object with `messages` array of length 3
//   - roles are exactly system, user, assistant in that order
//   - no empty content
//   - user content within [minUserChars, maxUserChars]
//   - assistant content parses as a Label with all three enum values valid
//   - no duplicate user content within the file
func ValidJSONL(path string) (ValidationResult, error) {
	res := ValidationResult{Path: path}
	f, err := os.Open(path)
	if err != nil {
		return res, fmt.Errorf("validate: open %s: %w", path, err)
	}
	defer f.Close()

	seen := map[string]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		if strings.TrimSpace(text) == "" {
			continue
		}
		res.LineCount++

		var s Sample
		if err := json.Unmarshal([]byte(text), &s); err != nil {
			res.Errors = append(res.Errors, ValidationError{line, "invalid JSON: " + err.Error()})
			continue
		}
		if reason := validateSample(&s); reason != "" {
			res.Errors = append(res.Errors, ValidationError{line, reason})
			continue
		}
		userContent := s.Messages[1].Content
		if prev, dup := seen[userContent]; dup {
			res.Errors = append(res.Errors, ValidationError{line, fmt.Sprintf("duplicate user content (also at line %d)", prev)})
			continue
		}
		seen[userContent] = line
	}
	if err := sc.Err(); err != nil {
		return res, fmt.Errorf("validate: scan: %w", err)
	}
	res.UniqueUsers = len(seen)
	return res, nil
}

func validateSample(s *Sample) string {
	if len(s.Messages) != 3 {
		return fmt.Sprintf("expected 3 messages, got %d", len(s.Messages))
	}
	wantRoles := []string{"system", "user", "assistant"}
	for i, m := range s.Messages {
		if m.Role != wantRoles[i] {
			return fmt.Sprintf("message[%d] role=%q want %q", i, m.Role, wantRoles[i])
		}
		if strings.TrimSpace(m.Content) == "" {
			return fmt.Sprintf("message[%d] empty content", i)
		}
	}
	user := s.Messages[1].Content
	if len(user) < minUserChars {
		return fmt.Sprintf("user content too short (%d < %d)", len(user), minUserChars)
	}
	if len(user) > maxUserChars {
		return fmt.Sprintf("user content too long (%d > %d)", len(user), maxUserChars)
	}
	var label Label
	if err := json.Unmarshal([]byte(s.Messages[2].Content), &label); err != nil {
		return "assistant content is not valid Label JSON: " + err.Error()
	}
	if !validComplexity(label.Complexity) {
		return "invalid complexity: " + label.Complexity
	}
	if !validTaskType(label.TaskType) {
		return "invalid task_type: " + label.TaskType
	}
	if !validRisk(label.Risk) {
		return "invalid risk: " + label.Risk
	}
	return ""
}

func validComplexity(v string) bool {
	return v == "low" || v == "medium" || v == "high"
}

func validTaskType(v string) bool {
	switch v {
	case "feature", "bugfix", "refactor", "test", "docs":
		return true
	}
	return false
}

func validRisk(v string) bool { return v == "low" || v == "high" }
