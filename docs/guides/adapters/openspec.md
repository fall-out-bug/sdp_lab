# OpenSpec Integration Guide

> **Status:** Beta (import-only, read-only)
> **Last Updated:** 2026-04-26
> **Feature:** F072-05

## Overview

SDP provides optional import support for OpenSpec change folders, allowing teams that use OpenSpec for planning to feed their artifacts into SDP's governed execution workflow without rewriting them by hand.

**Key Principle:** OpenSpec can help SDP decide *what should happen next*. SDP remains responsible for proving *what actually happened*.

---

## Integration Model

### Import-Only (Phase 1)

- **Reads:** OpenSpec change folders (`proposal.md`, `spec.md`, `design.md`, `tasks.md`)
- **Writes:** SDP planning payload and planner DAG
- **Does Not Mutate:** Beads, workstreams, or execution state

### Target Architecture

```
OpenSpec change folder (optional input)
        |
        v
  OpenSpec import adapter
        |
        v
  SDP planning payload / planner DAG
        |
        +--> next-step contract + instructions contract
        |
        v
  SDP workstreams + Beads + evidence + policy
        |
        v
       PR with proof
```

---

## Supported Artifacts

### 1. proposal.md (Required)

The proposal file defines the feature goal and metadata.

**Format:**

```markdown
# Feature Title

Feature description.

Priority: high
Labels: feature,enhancement
```

**Mapping:**

- `# Title` → SDP Plan title
- Description → SDP Plan goal
- `Priority:` → Task priorities (if tasks don't override)

### 2. tasks.md or tasks/ (Required)

Task definitions can be in a single `tasks.md` file or individual files in a `tasks/` folder.

**Single File Format:**

```markdown
## Task 1

Task description.

- [ ]

**Priority:** high
**Estimate:** 2h

## Task 2

Task description.

- [ ]

**Depends on:** openspec-task-1
**Priority:** medium

## Task 3

Task description.

- [x]

**Priority:** low
```

**Folder Format:**

```
tasks/
  task-1.md
  task-2.md
  task-3.md
```

**Mapping:**

- `## Title` → SDP Task title
- Description → SDP Task description
- `- [x]` → `completed` status
- `- [ ]` → `pending` status
- `**Depends on:**` → SDP Task dependencies
- `**Priority:**` → SDP Task priority (0=highest, 3=lowest)
- `**Estimate:**` → SDP Task estimated hours

### 3. spec.md (Optional)

Specification delta file with ADDED/MODIFIED/REMOVED sections.

**Format:**

```markdown
## ADDED

### /api/endpoint

New endpoint description.

## MODIFIED

### /api/old-endpoint

Updated endpoint description.

## REMOVED

### /api/deprecated-endpoint

Removed endpoint description.
```

**Mapping:** Stored in import result for reference. Does not generate tasks directly in Phase 1.

### 4. design.md (Optional)

Design documentation with architecture and decisions.

**Format:**

```markdown
# Architecture

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
```

**Mapping:** Stored in import result for reference. Does not generate tasks directly in Phase 1.

---

## Usage

### Basic Import

```go
package main

import (
    "fmt"
    "github.com/fall-out-bug/sdp_lab/internal/planner/openspec"
)

func main() {
    parser := openspec.NewOpenSpecParser(nil)
    result, err := parser.ParseChange("/path/to/openspec/change")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Import ID: %s\n", result.ImportID)
    fmt.Printf("Tasks mapped: %d/%d\n", result.MappingStats.MappedTasks, result.MappingStats.TotalTasks)

    // Validate import quality
    if err := result.ValidateImport(0.7); err != nil {
        fmt.Printf("Import validation failed: %v\n", err)
    }

    // Access SDP plan
    plan := result.Plan
    graph := result.GetPlanGraph()
}
```

### CLI Usage (Future)

```bash
# Import OpenSpec change into SDP planning payload
sdp import openspec /path/to/openspec/change

# Import with validation threshold
sdp import openspec /path/to/openspec/change --min-confidence 0.8

# Generate JSON contract
sdp import openspec /path/to/openspec/change --format json
```

### Strict Mode

Enable strict mode to fail on optional artifact parsing errors:

```go
config := &openspec.ParserConfig{
    StrictMode:    true,
    MinConfidence: 0.8,
}
parser := openspec.NewOpenSpecParser(config)
```

---

## Deterministic Behavior

### Source Hashing

Same OpenSpec change folder produces same import ID and source hash:

```go
// First import
result1, _ := parser.ParseChange("/path/to/change")

// Second import (unchanged source)
result2, _ := parser.ParseChange("/path/to/change")

assert(result1.ImportID == result2.ImportID)
assert(result1.SourceHash == result2.SourceHash)
```

### ID Generation

Import IDs are deterministic SHA-256 hashes of the source folder:

```
importID = "openspec-import-{first-8-bytes-of-sha256(sourceHash)}"
```

This allows:
- Idempotent imports
- Change detection (different import ID = source changed)
- Reproducible audits

---

## Unresolved Mappings

Import surfaces ambiguous or unsupported mappings explicitly instead of silently guessing:

### Example Unresolved Items

```go
for _, unresolved := range result.Unresolved {
    fmt.Printf("Source: %s\n", unresolved.Source)
    fmt.Printf("Reason: %s\n", unresolved.Reason)
    fmt.Printf("Confidence: %.2f\n", unresolved.Confidence)
    fmt.Printf("Suggestion: %s\n", unresolved.Suggestion)
}
```

### Common Unresolved Cases

1. **Missing Dependencies:** Task depends on non-existent task
2. **Unknown Status:** Task status doesn't match known values
3. **Invalid Priority:** Priority value not in P0-P3 range
4. **Malformed Estimates:** Estimate cannot be parsed as hours

---

## Mapping Statistics

Import result includes detailed statistics:

```go
stats := result.MappingStats

fmt.Printf("Total tasks: %d\n", stats.TotalTasks)
fmt.Printf("Mapped tasks: %d\n", stats.MappedTasks)
fmt.Printf("Unresolved tasks: %d\n", stats.UnresolvedTasks)
fmt.Printf("Total dependencies: %d\n", stats.TotalDeps)
fmt.Printf("Mapped dependencies: %d\n", stats.MappedDeps)
fmt.Printf("Confidence: %.2f\n", stats.Confidence)
```

### Confidence Calculation

```
confidence = mappedTasks / totalTasks
```

Use confidence to decide if import quality is acceptable:

```go
if result.MappingStats.Confidence < 0.8 {
    return fmt.Errorf("import confidence too low: %.2f", result.MappingStats.Confidence)
}
```

---

## Validation

### Import Quality Check

```go
// Validate with minimum confidence threshold
err := result.ValidateImport(0.7)
if err != nil {
    // Handle validation errors
    if strings.Contains(err.Error(), "below threshold") {
        fmt.Println("Import quality below threshold")
    }
    if strings.Contains(err.Error(), "no tasks were successfully mapped") {
        fmt.Println("No tasks could be imported")
    }
    if strings.Contains(err.Error(), "unresolved mappings") {
        fmt.Println("Import has unresolved items - check Unresolved field")
    }
}
```

### Plan Validation

Imported plans are validated automatically:

- Tasks must have valid IDs and titles
- Dependencies must reference existing tasks
- Plan must have at least one phase
- Phases must reference valid tasks

---

## Metadata Preservation

Original OpenSpec source information is preserved in task metadata:

```go
for _, task := range plan.GetReadyTasks() {
    source := task.Metadata["openspec_source"]    // Original file
    path := task.Metadata["openspec_import"]      // Import path
}
```

---

## Error Handling

### Common Errors

1. **proposal.md not found**
   - **Cause:** Required proposal file missing
   - **Fix:** Create proposal.md in OpenSpec change folder

2. **No tasks.md or tasks/ folder found**
   - **Cause:** No task definitions found
   - **Fix:** Create tasks.md file or tasks/ folder

3. **Plan validation failed**
   - **Cause:** Circular dependencies or invalid references
   - **Fix:** Check task dependencies in source

4. **Import confidence below threshold**
   - **Cause:** Too many unmapped tasks or dependencies
   - **Fix:** Review unresolved mappings and fix source

---

## Best Practices

### 1. Review Before Execution

Import creates a planning payload, not execution state. Always review before applying:

```go
result, _ := parser.ParseChange("/path/to/change")

// Review plan
plan := result.Plan
fmt.Printf("Plan: %s\n", plan.Title())
fmt.Printf("Tasks: %d\n", len(plan.GetReadyTasks()))

// Review unresolved
for _, u := range result.Unresolved {
    fmt.Printf("Unresolved: %s - %s\n", u.Source, u.Reason)
}
```

### 2. Set Confidence Thresholds

Use confidence thresholds to catch low-quality imports:

```go
// High-value work: require 90% confidence
result, _ := parser.ParseChange(path)
if err := result.ValidateImport(0.9); err != nil {
    return fmt.Errorf("import quality insufficient for production")
}
```

### 3. Handle Unresolved Explicitly

Never ignore unresolved mappings:

```go
if len(result.Unresolved) > 0 {
    log.Warn("Import has unresolved mappings")
    for _, u := range result.Unresolved {
        log.Warnf("  - %s: %s", u.Source, u.Reason)
    }
    // Decide: proceed with warnings or fail?
}
```

### 4. Use Deterministic IDs

Rely on import ID for change detection:

```go
currentImportID := result.ImportID
storedImportID := getStoredImportID()

if currentImportID != storedImportID {
    fmt.Println("OpenSpec source has changed - re-import needed")
}
```

---

## Limitations (Phase 1)

### Not Supported

- **Export:** SDP → OpenSpec (no bidirectional sync)
- **Direct Beads Mutation:** Import creates planning payload only
- **Automatic Workstream Creation:** Requires human review
- **Complex Estimate Parsing:** Only simple "Xh" format supported
- **Multi-file Proposals:** Only single proposal.md supported
- **Nested Task Folders:** Only flat tasks/ folder supported

### Future Work (Phase 2)

- Generate draft workstream proposals from imported payload
- Export SDP execution status back to OpenSpec format
- Support for more complex estimate formats
- Nested task folder structures
- Automatic dependency inference

---

## Integration with SDP Workflow

### Example: Import → Review → Execute

```go
// 1. Import OpenSpec change
parser := import.NewOpenSpecParser(nil)
result, err := parser.ParseChange("/path/to/openspec/change")
if err != nil {
    return fmt.Errorf("import failed: %w", err)
}

// 2. Validate quality
if err := result.ValidateImport(0.8); err != nil {
    return fmt.Errorf("import quality check failed: %w", err)
}

// 3. Review plan
plan := result.Plan
graph := result.GetPlanGraph()

// 4. Convert to workstream (human-reviewed)
// In Phase 1, this is a manual step
// In Phase 2, draft workstream generation will be supported

// 5. Execute with SDP governance
// workflow.Execute(plan)
```

---

## Reference Implementation

See `internal/planner/openspec/openspec_parser.go` for the complete implementation.

### Key Types

- `OpenSpecParser` - Main parser
- `OpenSpecChange` - Parsed OpenSpec artifacts
- `ImportResult` - Normalized SDP planning payload
- `MappingStats` - Import quality metrics
- `UnresolvedMapping` - Ambiguous or unsupported mappings

---

## Support

For issues or questions about OpenSpec integration:

1. Check this guide for common patterns
2. Review `internal/planner/openspec/openspec_parser_test.go` for examples
3. Consult `docs/archive/plans/2026-03-08-openspec-integration-plan.md` for design context

---

## Related Documentation

- [OpenSpec Integration Plan](../../archive/plans/2026-03-08-openspec-integration-plan.md)
- [Planner Graph Documentation](../../reference/project-map.md#planner)
- [Workstream Documentation](../../workstreams/INDEX.md)
