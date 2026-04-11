package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/agentloop"
)

// TestCmdNew_createSession verifies that cmdNew creates a session DB file.
func TestCmdNew_createSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)

	err := cmdNew([]string{"--session=test-session-123"})
	if err != nil {
		t.Fatalf("cmdNew: %v", err)
	}

	dbFile := filepath.Join(dir, "test-session-123.db")
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("DB file not created: %v", err)
	}
}

// TestCmdRun_noAPIKey verifies that run fails with a clear error when OPENROUTER_API_KEY is absent.
func TestCmdRun_noAPIKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("OPENROUTER_API_KEY", "")

	// Create session first
	if err := cmdNew([]string{"--session=nokey-test"}); err != nil {
		t.Fatalf("cmdNew: %v", err)
	}

	err := cmdRun([]string{"--session=nokey-test", "--prompt=hello"})
	if err == nil {
		t.Fatal("expected error when OPENROUTER_API_KEY is empty, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "API key") && !strings.Contains(errMsg, "OPENROUTER_API_KEY") {
		t.Errorf("error message %q does not mention API key requirement", errMsg)
	}
}

// TestCmdRun_terminatedSession verifies errors.Is(err, ErrHarnessTerminated) works.
func TestCmdRun_terminatedSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("OPENROUTER_API_KEY", "testkey")

	// Manually create a terminated session via the store
	path := filepath.Join(dir, "term-test.db")
	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_, err = agentloop.NewSession("term-test", store)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	// Stop the session to mark it terminated
	h, err := agentloop.RestoreHarness("term-test", "", store, nil, nil, nil)
	if err != nil {
		t.Fatalf("restore for stop: %v", err)
	}
	h.Stop(context.Background(), "")
	_ = store.Close()

	// Now attempt to run — RestoreHarness should return ErrHarnessTerminated
	err = cmdRun([]string{"--session=term-test", "--prompt=hello"})
	if err == nil {
		t.Fatal("expected error for terminated session, got nil")
	}
	if !errors.Is(err, agentloop.ErrHarnessTerminated) {
		t.Errorf("expected ErrHarnessTerminated, got: %v", err)
	}
}
