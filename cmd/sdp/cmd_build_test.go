package main

import (
	"testing"
)

func TestReorderFlagsFirst(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "flag after positional",
			args: []string{"add auth", "--dry-run"},
			want: []string{"--dry-run", "add auth"},
		},
		{
			name: "flags before positional",
			args: []string{"--dry-run", "add auth"},
			want: []string{"--dry-run", "add auth"},
		},
		{
			name: "flag with space-separated value after positional",
			args: []string{"add auth", "--sandbox", "docker", "--dry-run"},
			want: []string{"--sandbox", "docker", "--dry-run", "add auth"},
		},
		{
			name: "flag with equals after positional",
			args: []string{"add auth", "--sandbox=docker"},
			want: []string{"--sandbox=docker", "add auth"},
		},
		{
			name: "no flags",
			args: []string{"add auth"},
			want: []string{"add auth"},
		},
		{
			name: "multiple flags mixed (bool flag grabs next arg as value)",
			args: []string{"--strict", "fix bug", "--sandbox", "none", "--local"},
			want: []string{"--strict", "fix bug", "--sandbox", "none", "--local"},
		},
		{
			name: "empty args",
			args: []string{},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reorderFlagsFirst(tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("reorderFlagsFirst(%v) = %v, want %v", tt.args, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want[%d] = %q", i, got[i], i, tt.want[i])
				}
			}
		})
	}
}
