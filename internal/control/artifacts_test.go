package control

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileArtifactStore_StoreAndLoad(t *testing.T) {
	tmp := t.TempDir()
	store := NewFileArtifactStore(tmp)

	ctx := context.Background()
	ref, err := store.Store(ctx, "F082", ArtifactRef{
		Type: ArtifactDispatchPacket,
		Path: "F082/dispatch.json",
	}, []byte(`{"task_id":"F082"}`))
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if ref.Hash == "" {
		t.Error("expected non-empty hash")
	}

	data, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if string(data) != `{"task_id":"F082"}` {
		t.Errorf("unexpected data: %s", string(data))
	}
}

func TestFileArtifactStore_List(t *testing.T) {
	tmp := t.TempDir()
	store := NewFileArtifactStore(tmp)
	ctx := context.Background()

	// Store two artifacts
	_, _ = store.Store(ctx, "F082", ArtifactRef{Type: ArtifactDispatchPacket, Path: "F082/d1.json"}, []byte("a"))
	_, _ = store.Store(ctx, "F082", ArtifactRef{Type: ArtifactEvidence, Path: "F082/e1.json"}, []byte("b"))

	refs, err := store.List(ctx, "F082")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("expected 2 artifacts, got %d", len(refs))
	}
}

func TestFileArtifactStore_Delete(t *testing.T) {
	tmp := t.TempDir()
	store := NewFileArtifactStore(tmp)
	ctx := context.Background()

	ref, _ := store.Store(ctx, "F082", ArtifactRef{Type: ArtifactDispatchPacket, Path: "F082/to-delete.json"}, []byte("x"))
	if err := store.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(ref.Path); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestFileArtifactStore_ListEmpty(t *testing.T) {
	tmp := t.TempDir()
	store := NewFileArtifactStore(tmp)

	refs, err := store.List(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 artifacts, got %d", len(refs))
	}
}

func TestFileArtifactStore_StoreHashesContent(t *testing.T) {
	tmp := t.TempDir()
	store := NewFileArtifactStore(tmp)

	ref1, _ := store.Store(context.Background(), "F099", ArtifactRef{Type: ArtifactContract, Path: "c.json"}, []byte("content"))
	ref2, _ := store.Store(context.Background(), "F099", ArtifactRef{Type: ArtifactContract, Path: "c.json"}, []byte("content"))

	if ref1.Hash != ref2.Hash {
		t.Error("same content should produce same hash")
	}

	ref3, _ := store.Store(context.Background(), "F099", ArtifactRef{Type: ArtifactContract, Path: "c.json"}, []byte("different"))
	if ref1.Hash == ref3.Hash {
		t.Error("different content should produce different hash")
	}
}

func TestFileArtifactStore_StoreCreatesDirectories(t *testing.T) {
	tmp := t.TempDir()
	store := NewFileArtifactStore(tmp)

	ref, _ := store.Store(context.Background(), "F099", ArtifactRef{
		Type: ArtifactEvidence,
		Path: "build/evidence.json",
	}, []byte("{}"))

	expectedDir := filepath.Join(tmp, ".sdp", "artifacts", "F099", "evidence")
	info, err := os.Stat(expectedDir)
	if err != nil || !info.IsDir() {
		t.Errorf("expected directory %s to exist", expectedDir)
	}
	if ref.Size != 2 {
		t.Errorf("expected size 2, got %d", ref.Size)
	}
}
