package federation

import (
	"testing"

	"sdp_dev/internal/registry"
)

func TestNewBridge(t *testing.T) {
	b := NewBridge(BridgeConfig{
		ProjectID: "p1",
		WorkDir:   t.TempDir(),
		Labels:   []string{"autonomy"},
		Limit:    5,
	})
	if b == nil {
		t.Fatal("NewBridge returned nil")
	}
}

func TestNewBridge_defaultLimit(t *testing.T) {
	b := NewBridge(BridgeConfig{ProjectID: "p2", WorkDir: "/tmp", Limit: 0})
	if b == nil {
		t.Fatal("NewBridge returned nil")
	}
}

func TestNewBridge_withStore(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	b := NewBridge(BridgeConfig{
		ProjectID: "p3",
		WorkDir:   dir,
		Store:     store,
	})
	if b == nil {
		t.Error("NewBridge returned nil")
	}
}
