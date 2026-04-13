package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type fakeBDIssue struct {
	ID        string
	Status    string
	Priority  int
	CreatedAt string
	Assignee  string
}

func installFakeBD(t *testing.T, issues ...fakeBDIssue) string {
	t.Helper()

	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir fake bd state: %v", err)
	}
	for _, issue := range issues {
		payload := fmt.Sprintf("%s|%s|%d|%s|%s\n", issue.ID, issue.Status, issue.Priority, issue.CreatedAt, issue.Assignee)
		if err := os.WriteFile(filepath.Join(stateDir, issue.ID+".txt"), []byte(payload), 0o644); err != nil {
			t.Fatalf("write fake bd issue %s: %v", issue.ID, err)
		}
	}

	script := `#!/bin/sh
set -eu

STATE_DIR="$(dirname "$0")/state"

load_issue() {
  file="$STATE_DIR/$1.txt"
  [ -f "$file" ] || return 1
  IFS='|' read -r id status priority created_at assignee < "$file"
}

save_issue() {
  printf '%s|%s|%s|%s|%s\n' "$1" "$2" "$3" "$4" "$5" > "$STATE_DIR/$1.txt"
}

print_issue() {
  printf '{"id":"%s","status":"%s","priority":%s,"created_at":"%s","assignee":"%s"}' \
    "$id" "$status" "$priority" "$created_at" "$assignee"
}

cmd="${1-}"
case "$cmd" in
  show)
    shift
    first=1
    printf '['
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "--json" ]; then
        break
      fi
      load_issue "$1"
      if [ "$first" -eq 0 ]; then
        printf ','
      fi
      first=0
      print_issue
      shift
    done
    printf ']'
    ;;
	update)
	    issue_id="${2-}"
	    shift 2
	    load_issue "$issue_id"
	    if [ "${1-}" = "--claim" ]; then
	      status="in_progress"
	      assignee="Test Agent"
	      save_issue "$id" "$status" "$priority" "$created_at" "$assignee"
	      printf '[]'
	      exit 0
	    fi
	    if [ "${1-}" = "--status" ] && [ "${3-}" = "-a" ] && [ "${4-}" = "" ] && [ "${5-}" = "--json" ]; then
	      status="${2-}"
	      assignee=""
	      save_issue "$id" "$status" "$priority" "$created_at" "$assignee"
	      printf '[]'
	      exit 0
	    fi
	    if [ "${1-}" = "-a" ] && [ "${2-}" = "" ] && [ "${3-}" = "--json" ]; then
	      assignee=""
	      save_issue "$id" "$status" "$priority" "$created_at" "$assignee"
	      printf '[]'
	      exit 0
	    fi
	    echo "unsupported fake bd update: $*" >&2
	    exit 2
	    ;;
  *)
    echo "unsupported fake bd command: $*" >&2
    exit 2
    ;;
esac
`

	path := filepath.Join(root, "bd")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}
	return path
}
