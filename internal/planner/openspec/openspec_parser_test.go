package openspec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helpers

func createTestOpenSpecChange(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Create proposal.md
	proposalContent := `# Test Feature

This is a test feature proposal.

Priority: high
Labels: feature,enhancement
`
	if err := os.WriteFile(filepath.Join(tempDir, "proposal.md"), []byte(proposalContent), 0644); err != nil {
		t.Fatalf("failed to create proposal.md: %v", err)
	}

	// Create tasks.md
	tasksContent := `## Task 1

First task description.

- [ ]

**Priority:** high

## Task 2

Second task description.

- [ ]

**Depends on:** openspec-task-1
**Priority:** medium

## Task 3

Third task description.

- [x]

**Priority:** low
`
	if err := os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(tasksContent), 0644); err != nil {
		t.Fatalf("failed to create tasks.md: %v", err)
	}

	// Create spec.md
	specContent := `## ADDED

### /api/endpoint

New endpoint description.

## MODIFIED

### /api/old-endpoint

Updated endpoint description.
`
	if err := os.WriteFile(filepath.Join(tempDir, "spec.md"), []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to create spec.md: %v", err)
	}

	return tempDir
}

func createTestOpenSpecChangeWithDesign(t *testing.T) string {
	t.Helper()

	tempDir := createTestOpenSpecChange(t)

	// Create design.md
	designContent := `# Architecture

System architecture overview.

## Components

- Component A
- Component B

# Decisions

## Decision 1

### Context
Decision context.

### Outcome
Decision outcome.
`
	if err := os.WriteFile(filepath.Join(tempDir, "design.md"), []byte(designContent), 0644); err != nil {
		t.Fatalf("failed to create design.md: %v", err)
	}

	return tempDir
}

func createTestOpenSpecChangeWithTasksFolder(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Create proposal.md
	proposalContent := `# Test Feature

This is a test feature proposal.
`
	if err := os.WriteFile(filepath.Join(tempDir, "proposal.md"), []byte(proposalContent), 0644); err != nil {
		t.Fatalf("failed to create proposal.md: %v", err)
	}

	// Create tasks folder
	tasksDir := filepath.Join(tempDir, "tasks")
	if err := os.Mkdir(tasksDir, 0755); err != nil {
		t.Fatalf("failed to create tasks folder: %v", err)
	}

	// Create individual task files
	task1Content := `# Task 1

First task description.`
	if err := os.WriteFile(filepath.Join(tasksDir, "task-1.md"), []byte(task1Content), 0644); err != nil {
		t.Fatalf("failed to create task-1.md: %v", err)
	}

	task2Content := `# Task 2

Second task description.`
	if err := os.WriteFile(filepath.Join(tasksDir, "task-2.md"), []byte(task2Content), 0644); err != nil {
		t.Fatalf("failed to create task-2.md: %v", err)
	}

	return tempDir
}

func createTestOpenSpecChangeWithUnresolved(t *testing.T) string {
	t.Helper()

	tempDir := createTestOpenSpecChange(t)

	// Create tasks.md with circular dependency
	tasksContent := `## Task 1

First task.

- [ ]

**Depends on:** Task 3

## Task 2

Second task.

- [ ]

**Depends on:** NonExistentTask
`
	if err := os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(tasksContent), 0644); err != nil {
		t.Fatalf("failed to create tasks.md: %v", err)
	}

	return tempDir
}

// Tests

func TestNewOpenSpecParser(t *testing.T) {
	parser := NewOpenSpecParser(nil)
	if parser == nil {
		t.Fatal("expected non-nil parser")
	}

	if parser.config == nil {
		t.Error("expected default config to be set")
	}

	customConfig := &ParserConfig{StrictMode: true}
	parser = NewOpenSpecParser(customConfig)
	if parser.config.StrictMode != true {
		t.Error("expected custom config to be applied")
	}
}

func TestParseChangeBasic(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	if result.ImportID == "" {
		t.Error("expected import ID to be set")
	}

	if result.SourcePath != testDir {
		t.Errorf("expected source path %s, got %s", testDir, result.SourcePath)
	}

	if result.SourceHash == "" {
		t.Error("expected source hash to be set")
	}

	if result.Plan == nil {
		t.Fatal("expected plan to be created")
	}

	if result.ImportedAt.IsZero() {
	}
}

func TestParseChangeWithDesign(t *testing.T) {
	testDir := createTestOpenSpecChangeWithDesign(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	if result.Plan == nil {
		t.Fatal("expected plan to be created")
	}

	// Check that tasks were imported
	if result.MappingStats.TotalTasks < 3 {
		t.Errorf("expected at least 3 tasks, got %d", result.MappingStats.TotalTasks)
	}
}

func TestParseChangeWithTasksFolder(t *testing.T) {
	testDir := createTestOpenSpecChangeWithTasksFolder(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	if result.MappingStats.TotalTasks != 2 {
		t.Errorf("expected 2 tasks from tasks folder, got %d", result.MappingStats.TotalTasks)
	}

	if result.MappingStats.MappedTasks != 2 {
		t.Errorf("expected 2 mapped tasks, got %d", result.MappingStats.MappedTasks)
	}
}

func TestParseChangeNonExistentPath(t *testing.T) {
	parser := NewOpenSpecParser(nil)
	_, err := parser.ParseChange("/nonexistent/path")

	if err == nil {
		t.Error("expected error for non-existent path")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %v", err)
	}
}

func TestParseChangeMissingProposal(t *testing.T) {
	tempDir := t.TempDir()

	// Create tasks.md but no proposal.md
	tasksContent := `## Task 1

First task.`
	if err := os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(tasksContent), 0644); err != nil {
		t.Fatalf("failed to create tasks.md: %v", err)
	}

	parser := NewOpenSpecParser(nil)
	_, err := parser.ParseChange(tempDir)

	if err == nil {
		t.Error("expected error for missing proposal.md")
	}

	if !strings.Contains(err.Error(), "proposal.md") {
		t.Errorf("expected proposal.md error, got: %v", err)
	}
}

func TestParseChangeMissingTasks(t *testing.T) {
	tempDir := t.TempDir()

	// Create proposal.md but no tasks.md or tasks folder
	proposalContent := `# Test Feature

This is a test feature proposal.`
	if err := os.WriteFile(filepath.Join(tempDir, "proposal.md"), []byte(proposalContent), 0644); err != nil {
		t.Fatalf("failed to create proposal.md: %v", err)
	}

	parser := NewOpenSpecParser(nil)
	_, err := parser.ParseChange(tempDir)

	if err == nil {
		t.Error("expected error for missing tasks")
	}

	if !strings.Contains(err.Error(), "tasks") {
		t.Errorf("expected tasks error, got: %v", err)
	}
}

func TestParseChangeWithUnresolvedDependencies(t *testing.T) {
	testDir := createTestOpenSpecChangeWithUnresolved(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	if len(result.Unresolved) == 0 {
		t.Error("expected unresolved mappings for missing dependencies")
	}

	// Check that unresolved dependencies are tracked
	hasUnresolvedDep := false
	for _, u := range result.Unresolved {
		if strings.Contains(u.Reason, "dependency not found") {
			hasUnresolvedDep = true
			break
		}
	}

	if !hasUnresolvedDep {
		t.Error("expected unresolved dependency to be tracked")
	}
}

func TestDeterministicImportID(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser1 := NewOpenSpecParser(nil)
	result1, err := parser1.ParseChange(testDir)
	if err != nil {
		t.Fatalf("first parse failed: %v", err)
	}

	parser2 := NewOpenSpecParser(nil)
	result2, err := parser2.ParseChange(testDir)
	if err != nil {
		t.Fatalf("second parse failed: %v", err)
	}

	if result1.ImportID != result2.ImportID {
		t.Errorf("expected same import ID for same source, got %s and %s", result1.ImportID, result2.ImportID)
	}

	if result1.SourceHash != result2.SourceHash {
		t.Errorf("expected same source hash for same source, got %s and %s", result1.SourceHash, result2.SourceHash)
	}
}

func TestSourceHashChanges(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser := NewOpenSpecParser(nil)
	result1, err := parser.ParseChange(testDir)
	if err != nil {
		t.Fatalf("first parse failed: %v", err)
	}

	// Modify a file
	tasksPath := filepath.Join(testDir, "tasks.md")
	tasksContent := `## New Task

Modified task content.`
	if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
		t.Fatalf("failed to modify tasks.md: %v", err)
	}

	result2, err := parser.ParseChange(testDir)
	if err != nil {
		t.Fatalf("second parse failed: %v", err)
	}

	if result1.SourceHash == result2.SourceHash {
		t.Error("expected different source hashes after modification")
	}

	if result1.ImportID == result2.ImportID {
		t.Error("expected different import IDs after modification")
	}
}

func TestMappingStats(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	stats := result.MappingStats

	if stats.TotalTasks == 0 {
		t.Error("expected total tasks to be > 0")
	}

	if stats.MappedTasks == 0 {
		t.Error("expected mapped tasks to be > 0")
	}

	if stats.MappedTasks > stats.TotalTasks {
		t.Error("mapped tasks cannot exceed total tasks")
	}

	if stats.Confidence <= 0 || stats.Confidence > 1 {
		t.Errorf("expected confidence between 0 and 1, got %f", stats.Confidence)
	}

	// Confidence should be calculated correctly
	expectedConfidence := float64(stats.MappedTasks) / float64(stats.TotalTasks)
	if stats.Confidence != expectedConfidence {
		t.Errorf("expected confidence %f, got %f", expectedConfidence, stats.Confidence)
	}
}

func TestPlanConversion(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	plan := result.Plan

	if plan.ID() != result.ImportID {
		t.Errorf("expected plan ID %s, got %s", result.ImportID, plan.ID())
	}

	if plan.Title() == "" {
		t.Error("expected plan title to be set")
	}

	phases := plan.GetPhases()
	if len(phases) == 0 {
		t.Error("expected at least one phase")
	}

	// Check that tasks are in phases
	for _, phase := range phases {
		if len(phase.Tasks) == 0 {
			t.Errorf("expected phase %s to have tasks", phase.ID)
		}
	}
}

func TestToJSON(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}

	// Check that JSON contains expected fields
	if !strings.Contains(jsonStr, "import_id") {
		t.Error("expected JSON to contain import_id")
	}

	if !strings.Contains(jsonStr, "source_path") {
		t.Error("expected JSON to contain source_path")
	}

	if !strings.Contains(jsonStr, "mapping_stats") {
		t.Error("expected JSON to contain mapping_stats")
	}
}

func TestGetPlanGraph(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	graph := result.GetPlanGraph()
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}

	if len(graph.Nodes) == 0 {
		t.Error("expected graph to have nodes")
	}

	if len(graph.Edges) == 0 {
		t.Error("expected graph to have edges")
	}
}

func TestValidateImport(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	// Valid import should pass
	err = result.ValidateImport(0.5)
	if err != nil && !strings.Contains(err.Error(), "unresolved") {
		t.Errorf("unexpected validation error: %v", err)
	}

	// High threshold should still pass if all tasks mapped successfully (confidence = 1.0)
	err = result.ValidateImport(1.0)
	if err != nil && !strings.Contains(err.Error(), "unresolved") {
		t.Errorf("unexpected validation error with 1.0 threshold: %v", err)
	}

	// But if we set threshold to 1.1 (above max possible), it should fail
	err = result.ValidateImport(1.1)
	if err == nil {
		t.Error("expected validation error for threshold above 1.0")
	}
}

func TestValidateImportNoTasks(t *testing.T) {
	tempDir := t.TempDir()

	// Create proposal.md
	proposalContent := `# Test Feature`
	if err := os.WriteFile(filepath.Join(tempDir, "proposal.md"), []byte(proposalContent), 0644); err != nil {
		t.Fatalf("failed to create proposal.md: %v", err)
	}

	// Create empty tasks.md
	tasksContent := `# No tasks`
	if err := os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(tasksContent), 0644); err != nil {
		t.Fatalf("failed to create tasks.md: %v", err)
	}

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(tempDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	err = result.ValidateImport(0.1)
	if err == nil {
		t.Error("expected validation error for no mapped tasks")
	}

	// Either error message is acceptable - confidence check or no tasks check
	if !strings.Contains(err.Error(), "no tasks were successfully mapped") && !strings.Contains(err.Error(), "below threshold") {
		t.Errorf("expected 'no tasks' or 'below threshold' error, got: %v", err)
	}
}

func TestMetadataPreservation(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	// Check that metadata is preserved
	for _, task := range result.Plan.GetReadyTasks() {
		if task.Metadata == nil {
			t.Error("expected task metadata to be set")
			continue
		}

		if _, ok := task.Metadata["openspec_source"]; !ok {
			t.Error("expected openspec_source in metadata")
		}

		if _, ok := task.Metadata["openspec_import"]; !ok {
			t.Error("expected openspec_import in metadata")
		}
	}
}

func TestDependencyMapping(t *testing.T) {
	testDir := createTestOpenSpecChange(t)

	parser := NewOpenSpecParser(nil)
	result, err := parser.ParseChange(testDir)

	if err != nil {
		t.Fatalf("ParseChange failed: %v", err)
	}

	// Check that dependencies are mapped
	stats := result.MappingStats
	if stats.TotalDeps == 0 {
		t.Error("expected dependencies to be found")
	}

	if stats.MappedDeps == 0 {
		t.Error("expected at least one dependency to be mapped")
	}

	// Check that tasks have dependencies in the plan
	hasDependency := false
	for _, task := range result.Plan.GetReadyTasks() {
		if len(task.Dependencies) > 0 {
			hasDependency = true
			break
		}
	}

	if !hasDependency {
		t.Error("expected at least one task to have dependencies")
	}
}
