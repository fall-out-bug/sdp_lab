package sdputil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteJSON_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	orig := sample{Name: "test", Count: 42}

	if err := AtomicWriteJSON(path, orig); err != nil {
		t.Fatalf("AtomicWriteJSON: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got sample
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got != orig {
		t.Fatalf("roundtrip mismatch: got %+v, want %+v", got, orig)
	}
}

func TestAtomicWriteFile_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	content := []byte("hello world")

	if err := AtomicWriteFile(path, content, 0o644); err != nil {
		t.Fatalf("AtomicWriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(got) != string(content) {
		t.Fatalf("content mismatch: got %q, want %q", got, content)
	}
}

func TestAtomicWriteFile_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "deep", "file.txt")
	content := []byte("nested")

	if err := AtomicWriteFile(path, content, 0o644); err != nil {
		t.Fatalf("AtomicWriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(got) != string(content) {
		t.Fatalf("content mismatch: got %q, want %q", got, content)
	}
}
