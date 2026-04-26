package finetune

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempJSONL(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "data.jsonl")
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validLine = `{"messages":[{"role":"system","content":"sysprompt long enough"},{"role":"user","content":"Title: Add new feature flag\nGoal: gate behavior"},{"role":"assistant","content":"{\"complexity\":\"low\",\"task_type\":\"feature\",\"risk\":\"low\"}"}]}`

func TestValidJSONL_Clean(t *testing.T) {
	path := writeTempJSONL(t, []string{validLine})
	res, err := ValidJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("want 0 errors, got %d: %v", len(res.Errors), res.Errors)
	}
	if res.LineCount != 1 || res.UniqueUsers != 1 {
		t.Fatalf("counts: lines=%d unique=%d", res.LineCount, res.UniqueUsers)
	}
}

func TestValidJSONL_RejectsBadSamples(t *testing.T) {
	cases := map[string]string{
		"two_messages":      `{"messages":[{"role":"system","content":"x"},{"role":"user","content":"y"}]}`,
		"wrong_roles":       `{"messages":[{"role":"system","content":"x"},{"role":"assistant","content":"y"},{"role":"user","content":"z"}]}`,
		"empty_assistant":   `{"messages":[{"role":"system","content":"sysprompt"},{"role":"user","content":"valid user content here"},{"role":"assistant","content":""}]}`,
		"bad_label_enum":    `{"messages":[{"role":"system","content":"sys"},{"role":"user","content":"valid user content here"},{"role":"assistant","content":"{\"complexity\":\"unknown\",\"task_type\":\"feature\",\"risk\":\"low\"}"}]}`,
		"non_json_assistant": `{"messages":[{"role":"system","content":"sys"},{"role":"user","content":"valid user content here"},{"role":"assistant","content":"hello"}]}`,
		"too_short_user":    `{"messages":[{"role":"system","content":"sys"},{"role":"user","content":"short"},{"role":"assistant","content":"{\"complexity\":\"low\",\"task_type\":\"feature\",\"risk\":\"low\"}"}]}`,
	}
	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTempJSONL(t, []string{line})
			res, err := ValidJSONL(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Errors) == 0 {
				t.Fatalf("expected error for %s, got none", name)
			}
		})
	}
}

func TestValidJSONL_DetectsDuplicates(t *testing.T) {
	path := writeTempJSONL(t, []string{validLine, validLine})
	res, err := ValidJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 dup error, got %d: %v", len(res.Errors), res.Errors)
	}
}
