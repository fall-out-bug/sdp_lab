package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunQuality_DefaultUsesMatrixOnly(t *testing.T) {
	root := t.TempDir()
	writeQualityScript(t, root, `#!/bin/sh
echo "matrix=$SDP_QUALITY_MATRIX_ONLY"
`)
	withCwd(t, root)

	var stdout, stderr bytes.Buffer
	code := runQualityWithWriters(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "matrix=1" {
		t.Fatalf("stdout = %q, want matrix=1", got)
	}
}

func TestRunQuality_FullDoesNotSetMatrixOnly(t *testing.T) {
	root := t.TempDir()
	writeQualityScript(t, root, `#!/bin/sh
echo "matrix=${SDP_QUALITY_MATRIX_ONLY:-unset}"
`)
	withCwd(t, root)

	var stdout, stderr bytes.Buffer
	code := runQualityWithWriters([]string{"--full"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "matrix=unset" {
		t.Fatalf("stdout = %q, want matrix=unset", got)
	}
}

func TestRunQuality_PropagatesScriptExitCode(t *testing.T) {
	root := t.TempDir()
	writeQualityScript(t, root, `#!/bin/sh
echo "bad" >&2
exit 7
`)
	withCwd(t, root)

	var stdout, stderr bytes.Buffer
	code := runQualityWithWriters(nil, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("exit code = %d, want 7", code)
	}
	if !strings.Contains(stderr.String(), "bad") {
		t.Fatalf("stderr = %q, want script stderr", stderr.String())
	}
}

func writeQualityScript(t *testing.T, root, body string) {
	t.Helper()
	dir := filepath.Join(root, "scripts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "quality-metrics.sh")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func withCwd(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}
