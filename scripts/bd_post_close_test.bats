#!/usr/bin/env bats
# bd_post_close_test.bats — Tests for post-`bd close` auto-sync hook
#
# Requires: bats (Bash Automated Testing System)
#   Install: brew install bats-core   (macOS)
#   Install: npm install -g bats      (cross-platform)
#   Run:     cd scripts && bats bd_post_close_test.bats

# ---------------------------------------------------------------------------
# Test fixtures
# ---------------------------------------------------------------------------

setup() {
  # Create a temp directory that simulates the repo structure
  TEST_ROOT="$(mktemp -d)"
  export REPO_ROOT="$TEST_ROOT"

  # Create workstream directories
  mkdir -p "$TEST_ROOT/docs/workstreams/backlog"
  mkdir -p "$TEST_ROOT/docs/workstreams/done"

  # Path to the script under test
  SCRIPTUnderTest="$BATS_TEST_DIRNAME/bd_post_close.sh"
}

teardown() {
  [ -d "$TEST_ROOT" ] && rm -rf "$TEST_ROOT"
}

# ---------------------------------------------------------------------------
# Helper: create a mock workstream file in backlog/
# ---------------------------------------------------------------------------

create_ws_file() {
  local filename="$1"
  local ws_id="$2"
  local feature_id="$3"
  local status="${4:-backlog}"
  shift 4
  local beads="$*"

  cat > "${TEST_ROOT}/docs/workstreams/backlog/${filename}" <<WSFILE
---
ws_id: ${ws_id}
feature_id: ${feature_id}
status: ${status}
---
# ${ws_id}: Test Workstream

## Goal
Test goal.

## Beads
${beads}
WSFILE
}

# ---------------------------------------------------------------------------
# Helper: create a minimal INDEX.md
# ---------------------------------------------------------------------------

create_index() {
  cat > "${TEST_ROOT}/docs/workstreams/INDEX.md" <<'IDXFILE'
# Workstream Index

## Features

### F999: Test Feature

| Feature | Description | Workstreams | Status | Priority |
|---------|-------------|-------------|--------|----------|
| **F999** | Test Feature | 00-999-01 ... 00-999-03 | Backlog | P1 |

### F999: Test Feature Detail

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-999-01 | F999 | First WS | Backlog |
| 00-999-02 | F999 | Second WS | Backlog |
| 00-999-03 | F999 | Third WS | Backlog |
IDXFILE
}

# ---------------------------------------------------------------------------
# Test 1: Normal close — bead ID matches a workstream
# ---------------------------------------------------------------------------

@test "normal close: bead matches workstream → file moves to done/" {
  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"
  create_ws_file "00-999-02.md" "00-999-02" "F999" "backlog" "- sdplab-def2"
  create_index

  # Run hook with bead ID that matches 00-999-01
  run bash "$SCRIPTUnderTest" sdplab-abc1

  [ "$status" -eq 0 ]
  # File should have moved from backlog to done
  [ ! -f "${TEST_ROOT}/docs/workstreams/backlog/00-999-01.md" ]
  [ -f "${TEST_ROOT}/docs/workstreams/done/00-999-01.md" ]
  # Other file should remain in backlog
  [ -f "${TEST_ROOT}/docs/workstreams/backlog/00-999-02.md" ]
}

# ---------------------------------------------------------------------------
# Test 2: Close without workstream match
# ---------------------------------------------------------------------------

@test "close without match: no files moved" {
  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"
  create_index

  # Run hook with a bead ID that doesn't match any workstream
  run bash "$SCRIPTUnderTest" sdplab-zzz9

  [ "$status" -eq 0 ]
  # Original file should still be in backlog
  [ -f "${TEST_ROOT}/docs/workstreams/backlog/00-999-01.md" ]
  # Done directory should be empty
  [ -z "$(ls "${TEST_ROOT}/docs/workstreams/done/" 2>/dev/null)" ]
}

# ---------------------------------------------------------------------------
# Test 3: Multiple bead IDs close
# ---------------------------------------------------------------------------

@test "multiple closes: several bead IDs → correct files moved" {
  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"
  create_ws_file "00-999-02.md" "00-999-02" "F999" "backlog" "- sdplab-def2"
  create_ws_file "00-999-03.md" "00-999-03" "F999" "backlog" "- sdplab-ghi3"
  create_index

  # Close two beads at once
  run bash "$SCRIPTUnderTest" sdplab-abc1 sdplab-def2

  [ "$status" -eq 0 ]
  # Both matched files should move
  [ ! -f "${TEST_ROOT}/docs/workstreams/backlog/00-999-01.md" ]
  [ ! -f "${TEST_ROOT}/docs/workstreams/backlog/00-999-02.md" ]
  [ -f "${TEST_ROOT}/docs/workstreams/done/00-999-01.md" ]
  [ -f "${TEST_ROOT}/docs/workstreams/done/00-999-02.md" ]
  # Third file stays
  [ -f "${TEST_ROOT}/docs/workstreams/backlog/00-999-03.md" ]
}

# ---------------------------------------------------------------------------
# Test 4: Dry-run mode
# ---------------------------------------------------------------------------

@test "dry-run mode: lists changes without applying" {
  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"
  create_index

  BD_POST_CLOSE_DRY_RUN=1 run bash "$SCRIPTUnderTest" sdplab-abc1

  [ "$status" -eq 0 ]
  # Output should mention DRY-RUN
  echo "$output" | grep -q "DRY-RUN"
  # File should NOT have moved
  [ -f "${TEST_ROOT}/docs/workstreams/backlog/00-999-01.md" ]
  [ ! -f "${TEST_ROOT}/docs/workstreams/done/00-999-01.md" ]
}

# ---------------------------------------------------------------------------
# Test 5: INDEX.md detail table update
# ---------------------------------------------------------------------------

@test "INDEX.md detail table: status updated from Backlog to Done" {
  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"
  create_index

  run bash "$SCRIPTUnderTest" sdplab-abc1

  [ "$status" -eq 0 ]
  # Check INDEX.md has "Done" for the moved workstream
  grep -q "| 00-999-01 | F999 | First WS | Done |" "${TEST_ROOT}/docs/workstreams/INDEX.md"
}

# ---------------------------------------------------------------------------
# Test 6: Frontmatter status updated in moved file
# ---------------------------------------------------------------------------

@test "frontmatter: status changed to done in moved file" {
  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"

  run bash "$SCRIPTUnderTest" sdplab-abc1

  [ "$status" -eq 0 ]
  # Moved file should have status: done in frontmatter
  grep -q "^status: done" "${TEST_ROOT}/docs/workstreams/done/00-999-01.md"
}

# ---------------------------------------------------------------------------
# Test 7: Stdin input (piped bd close output)
# ---------------------------------------------------------------------------

@test "stdin input: bead IDs parsed from piped output" {
  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"

  # Simulate bd close output piped to the script
  run bash -c "echo 'Closed issue sdplab-abc1 successfully' | REPO_ROOT=$TEST_ROOT bash '$SCRIPTUnderTest'"

  [ "$status" -eq 0 ]
  [ -f "${TEST_ROOT}/docs/workstreams/done/00-999-01.md" ]
}

# ---------------------------------------------------------------------------
# Test 8: Workstream with multiple beads — match on any
# ---------------------------------------------------------------------------

@test "multi-bead workstream: matches on any bead ID" {
  cat > "${TEST_ROOT}/docs/workstreams/backlog/00-999-01.md" <<'WSFILE'
---
ws_id: 00-999-01
feature_id: F999
status: backlog
---
# 00-999-01: Multi-Bead WS

## Beads
- sdplab-aaa1
- sdplab-bbb2
WSFILE

  run bash "$SCRIPTUnderTest" sdplab-bbb2

  [ "$status" -eq 0 ]
  [ -f "${TEST_ROOT}/docs/workstreams/done/00-999-01.md" ]
}

# ---------------------------------------------------------------------------
# Test 9: No input — graceful no-op
# ---------------------------------------------------------------------------

@test "no input: exits 0 with no changes" {
  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"

  run bash "$SCRIPTUnderTest"

  [ "$status" -eq 0 ]
  echo "$output" | grep -q "No bead IDs"
  [ -f "${TEST_ROOT}/docs/workstreams/backlog/00-999-01.md" ]
}

# ---------------------------------------------------------------------------
# Test 10: Feature summary status → Done when all workstreams closed
# ---------------------------------------------------------------------------

@test "feature summary: status changes to Done when all WS closed" {
  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"
  create_ws_file "00-999-02.md" "00-999-02" "F999" "backlog" "- sdplab-def2"
  create_ws_file "00-999-03.md" "00-999-03" "F999" "backlog" "- sdplab-ghi3"
  create_index

  # Close all three
  run bash "$SCRIPTUnderTest" sdplab-abc1 sdplab-def2 sdplab-ghi3

  [ "$status" -eq 0 ]
  # Feature summary row should show Done
  grep -q "| \*\*F999\*\* |.*| Done |" "${TEST_ROOT}/docs/workstreams/INDEX.md"
}

# ---------------------------------------------------------------------------
# Test 11: done/ directory created if absent
# ---------------------------------------------------------------------------

@test "done dir: created automatically if absent" {
  # Remove done/ directory
  rmdir "${TEST_ROOT}/docs/workstreams/done" 2>/dev/null || true

  create_ws_file "00-999-01.md" "00-999-01" "F999" "backlog" "- sdplab-abc1"

  run bash "$SCRIPTUnderTest" sdplab-abc1

  [ "$status" -eq 0 ]
  [ -d "${TEST_ROOT}/docs/workstreams/done" ]
  [ -f "${TEST_ROOT}/docs/workstreams/done/00-999-01.md" ]
}
