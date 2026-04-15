package scout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScaleCountsFilesAndLOC(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":       "module example.com/app\ngo 1.26\n",
		"main.go":      "package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n",
		"util.go":      "package main\n\nfunc util() {}\n",
		"main_test.go": "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n",
	}, false)

	scale := detectScale(dir, nil)

	if scale.SourceFiles != 2 {
		t.Errorf("SourceFiles = %d, want 2", scale.SourceFiles)
	}
	if scale.TestFiles != 1 {
		t.Errorf("TestFiles = %d, want 1", scale.TestFiles)
	}
	if scale.TotalLoc < 4 {
		t.Errorf("TotalLoc = %d, want >= 4", scale.TotalLoc)
	}
}

func TestScaleExcludesVendorAndGenerated(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"main.go":            "package main\nfunc main() {}\n",
		"vendor/lib.go":      "package vendor\nfunc Lib() {}\n",
		"foo.pb.go":         "// generated\npackage foo\n",
		"node_modules/a.js": "module.exports = {};\n",
	}, false)

	scale := detectScale(dir, nil)

	if scale.VendorFiles != 0 {
		t.Errorf("VendorFiles = %d, want 0", scale.VendorFiles)
	}
	if scale.SourceFiles > 1 {
		t.Errorf("SourceFiles = %d, want <= 1", scale.SourceFiles)
	}
}

func TestScaleSkipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	binContent := make([]byte, 100)
	binContent[10] = 0x00
	_ = os.WriteFile(filepath.Join(dir, "image.png"), binContent, 0o644)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)

	scale := detectScale(dir, nil)
	if scale.TotalLoc != 1 {
		t.Errorf("TotalLoc = %d, want 1 (binary skipped)", scale.TotalLoc)
	}
}
