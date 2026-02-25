package orchestrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/executil"
)

// gitLSFiles returns tracked files for scope checks. Reserved for future use.
func gitLSFiles(projectRoot string) (map[string]bool, error) { //nolint:unused
	out, err := executil.DefaultRunner.Output(context.Background(), projectRoot, "git", "ls-files")
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			m[line] = true
		}
	}
	return m, nil
}

func gitStatusPorcelain(projectRoot string) (string, error) {
	out, err := executil.DefaultRunner.Output(context.Background(), projectRoot, "git", "status", "--porcelain")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func bdShow(projectRoot, beadsID string) (string, error) {
	out, err := executil.DefaultRunner.Output(context.Background(), projectRoot, "bd", "show", beadsID)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func wsIDToBeadsID(projectRoot, wsID string) string {
	mappingPath := filepath.Join(projectRoot, ".beads-sdp-mapping.jsonl")
	data, err := os.ReadFile(mappingPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, `"sdp_id":"`+wsID+`"`) {
			if idx := strings.Index(line, `"beads_id":"`); idx >= 0 {
				rest := line[idx+12:]
				if end := strings.Index(rest, `"`); end >= 0 {
					return rest[:end]
				}
			}
		}
	}
	return ""
}
