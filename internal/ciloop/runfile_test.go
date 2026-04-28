package ciloop_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/ciloop"
)

func writeRunFile(t *testing.T, dir, name string) {
	t.Helper()
	content := map[string]interface{}{
		"run_id":     name,
		"feature_id": "F014",
		"events":     []interface{}{},
		"last_phase": "init",
		"last_state": "ok",
	}
	data, _ := json.Marshal(content)
	if err := os.WriteFile(filepath.Join(dir, name+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAppendRunEvent(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, dir, "oneshot-F014-20260223T000000Z")

	err := ciloop.AppendRunEvent(dir, "F014", "ci", "ok", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read back and verify event was appended.
	data, err := os.ReadFile(filepath.Join(dir, "oneshot-F014-20260223T000000Z.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rf map[string]interface{}
	if err := json.Unmarshal(data, &rf); err != nil {
		t.Fatal(err)
	}
	events, ok := rf["events"].([]interface{})
	if !ok || len(events) != 1 {
		t.Errorf("expected 1 event, got %v", rf["events"])
	}
	if rf["last_phase"] != "ci" {
		t.Errorf("expected last_phase=ci, got %v", rf["last_phase"])
	}
	if rf["last_state"] != "ok" {
		t.Errorf("expected last_state=ok, got %v", rf["last_state"])
	}
}

func TestAppendRunEventLatestFile(t *testing.T) {
	dir := t.TempDir()
	// Two run files - should pick the lexicographically latest.
	writeRunFile(t, dir, "oneshot-F014-20260223T000000Z")
	writeRunFile(t, dir, "oneshot-F014-20260223T120000Z")

	err := ciloop.AppendRunEvent(dir, "F014", "ci", "ok", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The earlier file should be untouched.
	data, err := os.ReadFile(filepath.Join(dir, "oneshot-F014-20260223T000000Z.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var rf1 map[string]interface{}
	if err := json.Unmarshal(data, &rf1); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	events1 := rf1["events"].([]interface{})
	if len(events1) != 0 {
		t.Errorf("expected 0 events in older file, got %d", len(events1))
	}

	// The later file should have the event.
	data2, err := os.ReadFile(filepath.Join(dir, "oneshot-F014-20260223T120000Z.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var rf2 map[string]interface{}
	if err := json.Unmarshal(data2, &rf2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	events2 := rf2["events"].([]interface{})
	if len(events2) != 1 {
		t.Errorf("expected 1 event in latest file, got %d", len(events2))
	}
}

func TestAppendRunEventNoRunFile(t *testing.T) {
	dir := t.TempDir()
	err := ciloop.AppendRunEvent(dir, "F999", "ci", "ok", "")
	if err == nil {
		t.Fatal("expected error when no run file exists, got nil")
	}
}

func TestAppendRunEventWithNotes(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, dir, "oneshot-F014-20260223T000000Z")

	err := ciloop.AppendRunEvent(dir, "F014", "ci", "escalated", "secrets-scan failure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "oneshot-F014-20260223T000000Z.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var rf map[string]interface{}
	if err := json.Unmarshal(data, &rf); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	events := rf["events"].([]interface{})
	ev := events[0].(map[string]interface{})
	if ev["notes"] != "secrets-scan failure" {
		t.Errorf("expected notes to be set, got %v", ev["notes"])
	}
}

// TestAppendRunEventConcurrent verifies flock prevents corruption under concurrent access.
func TestAppendRunEventConcurrent(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, dir, "oneshot-F014-20260223T000000Z")

	const concurrency = 20
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			notes := "event-" + strconv.Itoa(n)
			if err := ciloop.AppendRunEvent(dir, "F014", "ci", "ok", notes); err != nil {
				t.Errorf("AppendRunEvent: %v", err)
			}
		}(i)
	}
	wg.Wait()

	data, err := os.ReadFile(filepath.Join(dir, "oneshot-F014-20260223T000000Z.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rf map[string]interface{}
	if err := json.Unmarshal(data, &rf); err != nil {
		t.Fatalf("corrupt JSON: %v", err)
	}
	events, ok := rf["events"].([]interface{})
	if !ok {
		t.Fatalf("expected events array, got %T", rf["events"])
	}
	if len(events) != concurrency {
		t.Errorf("expected %d events, got %d (flock may be missing)", concurrency, len(events))
	}
}
