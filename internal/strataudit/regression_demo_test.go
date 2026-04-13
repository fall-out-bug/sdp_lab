package strataudit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunRegressionDemo_WritesOfflineArtifacts(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "demo-report")

	run, err := RunRegressionDemo(context.Background(), outDir)
	if err != nil {
		t.Fatalf("RunRegressionDemo: %v", err)
	}

	if run.OutputDir != outDir {
		t.Fatalf("OutputDir = %q, want %q", run.OutputDir, outDir)
	}
	if run.Result.Link.TracesCreated != 1 {
		t.Fatalf("TracesCreated = %d, want 1", run.Result.Link.TracesCreated)
	}
	if run.Result.Extract.RejectedEntities != 1 {
		t.Fatalf("RejectedEntities = %d, want 1", run.Result.Extract.RejectedEntities)
	}

	for _, path := range []string{
		run.ReportHTMLPath,
		run.ReportJSONPath,
		run.ReportCompat,
		run.DiagnosticsPath,
		run.DatabasePath,
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
}
