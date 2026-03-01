package workstream

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateProtocolSuccess(t *testing.T) {
	root := makeProject(t,
		"F100",
		"00-100-01.md",
		`---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Something
`)

	report, err := ValidateProtocol(root, true, true)
	if err != nil {
		t.Fatalf("ValidateProtocol error: %v", err)
	}
	if report.HasErrors() {
		t.Fatalf("expected no errors, got: %+v", report.Issues)
	}
}

func TestValidateProtocolMissingBeads(t *testing.T) {
	root := makeProject(t,
		"F101",
		"00-101-01.md",
		`---
ws_id: 00-101-01
feature_id: F101
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-101-01: Example

## Acceptance Criteria

- [ ] Something
`)

	report, err := ValidateProtocol(root, false, true)
	if err != nil {
		t.Fatalf("ValidateProtocol error: %v", err)
	}
	if !report.HasErrors() {
		t.Fatalf("expected errors, got: %+v", report.Issues)
	}
}

func TestValidateProtocolMissingACCheckboxes(t *testing.T) {
	root := makeProject(t,
		"F102",
		"00-102-01.md",
		`---
ws_id: 00-102-01
feature_id: F102
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-102-01: Example

## Beads

- sdplab-1: Example

## Acceptance Criteria

- Something without checkbox
`)

	report, err := ValidateProtocol(root, false, true)
	if err != nil {
		t.Fatalf("ValidateProtocol error: %v", err)
	}
	if !report.HasErrors() {
		t.Fatalf("expected errors, got: %+v", report.Issues)
	}
}

func TestValidateProtocolStrictBeadsRejectsPlaceholder(t *testing.T) {
	root := makeProject(t,
		"F103",
		"00-103-01.md",
		`---
ws_id: 00-103-01
feature_id: F103
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-103-01: Example

## Beads

- sdplab-XX: placeholder

## Acceptance Criteria

- [ ] Something
`)

	report, err := ValidateProtocol(root, true, true)
	if err != nil {
		t.Fatalf("ValidateProtocol error: %v", err)
	}
	if !report.HasErrors() {
		t.Fatalf("expected errors in strict mode, got: %+v", report.Issues)
	}
}

func TestValidateProtocolLegacyDefaultWarningOnly(t *testing.T) {
	root := makeProject(t,
		"F104",
		"00-104-01.md",
		`# Legacy file without protocol sections
`)

	report, err := ValidateProtocol(root, false, false)
	if err != nil {
		t.Fatalf("ValidateProtocol error: %v", err)
	}
	if report.HasErrors() {
		t.Fatalf("expected no errors in non-strict mode, got: %+v", report.Issues)
	}
	if len(report.Issues) == 0 {
		t.Fatal("expected warning for legacy format")
	}
}

func TestValidateProtocolLegacyStrictStillWarning(t *testing.T) {
	root := makeProject(t,
		"F059",
		"00-059-01.md",
		`---
ws_id: 00-059-01
feature_id: F059
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-059-01: Legacy Example

## Acceptance Criteria

- [ ] Something
`)

	report, err := ValidateProtocol(root, false, true)
	if err != nil {
		t.Fatalf("ValidateProtocol error: %v", err)
	}
	if report.HasErrors() {
		t.Fatalf("expected legacy strict findings as warnings, got: %+v", report.Issues)
	}
}

func makeProject(t *testing.T, featureID, wsFile, wsContent string) string {
	t.Helper()
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	index := "# Workstream Index\n\n| Feature | Description | Workstreams | Status |\n|---------|-------------|-------------|--------|\n| **" + featureID + "** | Example | 00-" + featureID[1:] + "-01 | Backlog |\n\n## Workstream Status\n\n| WS | Feature | Title | Status |\n|----|---------|-------|--------|\n| 00-" + featureID[1:] + "-01 | " + featureID + " | Example | Backlog |\n"
	roadmap := "# Roadmap\n\n- **" + featureID + "** — Example feature\n"

	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), index)
	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), roadmap)
	write(t, filepath.Join(root, "docs", "workstreams", "backlog", wsFile), wsContent)
	return root
}

func mkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

func write(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}
