package harnesscfg_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"sdp_dev/internal/harnesscfg"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func loadSchema(t *testing.T) *harnesscfg.Schema {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectRoot(t), "schema", "harness-config-manifest.schema.json"))
	if err != nil {
		t.Fatalf("schema file missing: %v", err)
	}
	s, err := harnesscfg.ParseSchema(data)
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}
	return s
}

func TestValidManifest(t *testing.T) {
	s := loadSchema(t)
	m := harnesscfg.Manifest{
		Version:        "1.0.0",
		LifecycleStage: harnesscfg.StageGreenfieldStr,
		Harnesses:      []harnesscfg.Harness{{Name: "claude-code", ConfigFile: "CLAUDE.md"}},
	}
	if err := harnesscfg.Validate(s, m); err != nil {
		t.Errorf("valid manifest should not error: %v", err)
	}
}

func TestMissingVersion(t *testing.T) {
	s := loadSchema(t)
	m := harnesscfg.Manifest{
		LifecycleStage: harnesscfg.StageBrownfieldNewStr,
		Harnesses:      []harnesscfg.Harness{{Name: "codex-cli", ConfigFile: "AGENTS.md"}},
	}
	if err := harnesscfg.Validate(s, m); err == nil {
		t.Error("expected error for missing version, got nil")
	}
}

func TestInvalidLifecycleStage(t *testing.T) {
	s := loadSchema(t)
	m := harnesscfg.Manifest{
		Version:        "1.0.0",
		LifecycleStage: "legacy",
		Harnesses:      []harnesscfg.Harness{{Name: "claude-code", ConfigFile: "CLAUDE.md"}},
	}
	if err := harnesscfg.Validate(s, m); err == nil {
		t.Error("expected error for invalid lifecycle_stage, got nil")
	}
}

func TestEmptyHarnesses(t *testing.T) {
	s := loadSchema(t)
	m := harnesscfg.Manifest{
		Version:        "1.0.0",
		LifecycleStage: harnesscfg.StageBrownfieldMatureStr,
		Harnesses:      []harnesscfg.Harness{},
	}
	if err := harnesscfg.Validate(s, m); err == nil {
		t.Error("expected error for empty harnesses, got nil")
	}
}

func TestInvalidHarnessName(t *testing.T) {
	s := loadSchema(t)
	m := harnesscfg.Manifest{
		Version:        "1.0.0",
		LifecycleStage: harnesscfg.StageGreenfieldStr,
		Harnesses:      []harnesscfg.Harness{{Name: "unknown-harness", ConfigFile: "X.md"}},
	}
	if err := harnesscfg.Validate(s, m); err == nil {
		t.Error("expected error for unknown harness name, got nil")
	}
}

func TestAllLifecycleStages(t *testing.T) {
	s := loadSchema(t)
	for _, stage := range []string{
		harnesscfg.StageGreenfieldStr,
		harnesscfg.StageBrownfieldNewStr,
		harnesscfg.StageBrownfieldMatureStr,
	} {
		m := harnesscfg.Manifest{
			Version:        "2.0.0",
			LifecycleStage: stage,
			Harnesses:      []harnesscfg.Harness{{Name: "cursor", ConfigFile: ".cursor/rules/project.mdc"}},
		}
		if err := harnesscfg.Validate(s, m); err != nil {
			t.Errorf("stage %q should be valid: %v", stage, err)
		}
	}
}

func TestSchemaHasVersionField(t *testing.T) {
	s := loadSchema(t)
	if !s.HasField("version") {
		t.Error("schema must declare a 'version' field for drift detection")
	}
}

func TestParseSchemaInvalidJSON(t *testing.T) {
	_, err := harnesscfg.ParseSchema([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestValidateMissingConfigFile(t *testing.T) {
	s := loadSchema(t)
	m := harnesscfg.Manifest{
		Version:        "1.0.0",
		LifecycleStage: harnesscfg.StageGreenfieldStr,
		Harnesses:      []harnesscfg.Harness{{Name: "claude-code", ConfigFile: ""}},
	}
	if err := harnesscfg.Validate(s, m); err == nil {
		t.Error("expected error for empty config_file, got nil")
	}
}

func TestValidateInvalidSemver(t *testing.T) {
	s := loadSchema(t)
	m := harnesscfg.Manifest{
		Version:        "not-semver",
		LifecycleStage: harnesscfg.StageGreenfieldStr,
		Harnesses:      []harnesscfg.Harness{{Name: "cursor", ConfigFile: ".cursor/rules/project.mdc"}},
	}
	if err := harnesscfg.Validate(s, m); err == nil {
		t.Error("expected error for invalid semver version, got nil")
	}
}

func TestValidateOptionalFields(t *testing.T) {
	s := loadSchema(t)
	m := harnesscfg.Manifest{
		Version:        "1.0.0",
		LifecycleStage: harnesscfg.StageBrownfieldMatureStr,
		Language:       "go",
		RulesFile:      "docs/reference/go-patterns.md",
		Harnesses: []harnesscfg.Harness{
			{Name: "zed", ConfigFile: ".zed/settings.json"},
			{Name: "warp", ConfigFile: ".warp/rules.md"},
		},
	}
	if err := harnesscfg.Validate(s, m); err != nil {
		t.Errorf("manifest with optional fields should be valid: %v", err)
	}
}
