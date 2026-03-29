package dispatch

import (
	"context"
	"errors"
	"testing"
)

func TestShellBeadsReader_ReadBead(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    *BeadContext
		wantErr bool
	}{
		{
			name: "full bd show output",
			output: `[{
				"id": "sdplab-3t5p",
				"title": "Unexport ~40 internal-only symbols",
				"description": "Why this issue exists...",
				"notes": "See PR #42",
				"status": "open",
				"priority": 3,
				"issue_type": "task",
				"labels": ["refactor", "cleanup"],
				"dependencies": [
					{"id": "sdplab-abc1", "dependency_type": "depends_on"},
					{"id": "sdplab-abc2", "dependency_type": "depends_on"},
					{"id": "sdplab-xyz1", "dependency_type": "blocks"}
				]
			}]`,
			want: &BeadContext{
				ID:          "sdplab-3t5p",
				Title:       "Unexport ~40 internal-only symbols",
				Description: "Why this issue exists...",
				Status:      "open",
				DependsOn:   []string{"sdplab-abc1", "sdplab-abc2"},
				BlockedBy:   []string{"sdplab-xyz1"},
				Labels:      []string{"refactor", "cleanup"},
				Priority:    3,
				Type:        "task",
				Notes:       "See PR #42",
			},
		},
		{
			name: "minimal bd show output",
			output: `[{
				"id": "sdplab-min1",
				"title": "Minimal issue",
				"status": "open"
			}]`,
			want: &BeadContext{
				ID:        "sdplab-min1",
				Title:     "Minimal issue",
				Status:    "open",
				DependsOn: nil,
				BlockedBy: nil,
				Labels:    nil,
			},
		},
		{
			name: "no dependencies at all",
			output: `[{
				"id": "sdplab-nodeps",
				"title": "No deps issue",
				"status": "in_progress",
				"priority": 2,
				"issue_type": "bug",
				"labels": ["urgent"]
			}]`,
			want: &BeadContext{
				ID:        "sdplab-nodeps",
				Title:     "No deps issue",
				Status:    "in_progress",
				DependsOn: nil,
				BlockedBy: nil,
				Labels:    []string{"urgent"},
				Priority:  2,
				Type:      "bug",
			},
		},
		{
			name:    "empty array",
			output:  `[]`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			output:  `not json at all`,
			wantErr: true,
		},
		{
			name:    "bd show returns error",
			output:  "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockOutput := tc.output
			mockErr := error(nil)
			if tc.name == "bd show returns error" {
				mockErr = errors.New("bd: exit status 1")
			}

			reader := &ShellBeadsReader{
				ProjectRoot: "/tmp/test-project",
				BdShow: func(_ context.Context, _, _ string) (string, error) {
					return mockOutput, mockErr
				},
			}

			got, err := reader.ReadBead(context.Background(), "sdplab-test")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.ID != tc.want.ID {
				t.Errorf("ID: got %q, want %q", got.ID, tc.want.ID)
			}
			if got.Title != tc.want.Title {
				t.Errorf("Title: got %q, want %q", got.Title, tc.want.Title)
			}
			if got.Description != tc.want.Description {
				t.Errorf("Description: got %q, want %q", got.Description, tc.want.Description)
			}
			if got.Status != tc.want.Status {
				t.Errorf("Status: got %q, want %q", got.Status, tc.want.Status)
			}
			if got.Priority != tc.want.Priority {
				t.Errorf("Priority: got %d, want %d", got.Priority, tc.want.Priority)
			}
			if got.Type != tc.want.Type {
				t.Errorf("Type: got %q, want %q", got.Type, tc.want.Type)
			}
			if got.Notes != tc.want.Notes {
				t.Errorf("Notes: got %q, want %q", got.Notes, tc.want.Notes)
			}
			assertStringSlice(t, "DependsOn", got.DependsOn, tc.want.DependsOn)
			assertStringSlice(t, "BlockedBy", got.BlockedBy, tc.want.BlockedBy)
			assertStringSlice(t, "Labels", got.Labels, tc.want.Labels)
		})
	}
}

func TestShellBeadsReader_ReadDependencies(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    []string
		wantErr bool
	}{
		{
			name: "extracts depends_on dependencies",
			output: `[{
				"id": "sdplab-dep1",
				"title": "Has deps",
				"status": "open",
				"dependencies": [
					{"id": "sdplab-a1", "dependency_type": "depends_on"},
					{"id": "sdplab-a2", "dependency_type": "depends_on"},
					{"id": "sdplab-b1", "dependency_type": "blocks"}
				]
			}]`,
			want: []string{"sdplab-a1", "sdplab-a2"},
		},
		{
			name: "no dependencies returns empty slice",
			output: `[{
				"id": "sdplab-nodep",
				"title": "No deps",
				"status": "open"
			}]`,
			want: []string{},
		},
		{
			name: "only blocks dependencies returns empty depends_on",
			output: `[{
				"id": "sdplab-onlyblocks",
				"title": "Only blockers",
				"status": "open",
				"dependencies": [
					{"id": "sdplab-x1", "dependency_type": "blocks"}
				]
			}]`,
			want: []string{},
		},
		{
			name:    "bd show error propagates",
			output:  "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockOutput := tc.output
			mockErr := error(nil)
			if tc.name == "bd show error propagates" {
				mockErr = errors.New("bd: not found")
			}

			reader := &ShellBeadsReader{
				ProjectRoot: "/tmp/test-project",
				BdShow: func(_ context.Context, _, _ string) (string, error) {
					return mockOutput, mockErr
				},
			}

			got, err := reader.ReadDependencies(context.Background(), "sdplab-test")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertStringSlice(t, "dependencies", got, tc.want)
		})
	}
}

func TestShellBeadsReader_NilBdShow(t *testing.T) {
	reader := &ShellBeadsReader{
		ProjectRoot: "/tmp/test",
		BdShow:      nil,
	}

	_, err := reader.ReadBead(context.Background(), "sdplab-test")
	if err == nil {
		t.Fatal("expected error when BdShow is nil, got nil")
	}
}

func TestNewShellBeadsReader(t *testing.T) {
	reader := NewShellBeadsReader("/some/root")
	if reader.ProjectRoot != "/some/root" {
		t.Errorf("ProjectRoot: got %q, want %q", reader.ProjectRoot, "/some/root")
	}
	if reader.BdShow == nil {
		t.Error("BdShow should not be nil after NewShellBeadsReader")
	}
}

func TestParseCommaList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty string", input: "", want: nil},
		{name: "none marker", input: "(none)", want: nil},
		{name: "single value", input: "refactor", want: []string{"refactor"}},
		{name: "comma separated", input: "refactor, cleanup, go", want: []string{"refactor", "cleanup", "go"}},
		{name: "extra spaces", input: "  a , b , c  ", want: []string{"a", "b", "c"}},
		{name: "trailing comma", input: "a,b,", want: []string{"a", "b"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCommaList(tc.input)
			assertStringSlice(t, "result", got, tc.want)
		})
	}
}

// assertStringSlice compares two string slices for equality.
func assertStringSlice(t *testing.T, field string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s length: got %d (%v), want %d (%v)", field, len(got), got, len(want), want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %q, want %q", field, i, got[i], want[i])
		}
	}
}
