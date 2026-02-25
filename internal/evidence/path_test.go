package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePath_UnderBase(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "evidence.json")
	_ = os.WriteFile(f, []byte("{}"), 0o644)
	if err := ValidatePath(f, dir); err != nil {
		t.Errorf("path under base should be allowed: %v", err)
	}
}

func TestValidatePath_RejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	escape := filepath.Join(dir, "..", "etc", "passwd")
	err := ValidatePath(escape, dir)
	if err == nil {
		t.Error("expected error for .. traversal")
	}
	if err != nil && !strings.Contains(err.Error(), "escapes") {
		t.Errorf("error should mention escape: %v", err)
	}
}

func TestValidatePath_RejectsAbsoluteOutside(t *testing.T) {
	dir := t.TempDir()
	if err := ValidatePath("/etc/passwd", dir); err == nil {
		t.Error("expected error for absolute path outside base")
	}
}

func TestValidatePath_EmptyBaseUsesCWD(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	_ = os.Chdir(dir)
	_ = os.WriteFile("x.json", []byte("{}"), 0o644)
	if err := ValidatePath("x.json", ""); err != nil {
		t.Errorf("empty base should use CWD: %v", err)
	}
}

