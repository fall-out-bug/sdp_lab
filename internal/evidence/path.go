package evidence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath ensures path is under baseDir. Rejects .. traversal and absolute paths outside baseDir.
// Use CWD or project root as baseDir. Returns error if path escapes base or baseDir cannot be resolved.
func ValidatePath(path, baseDir string) error {
	if baseDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("path validation: cannot get working directory: %w", err)
		}
		baseDir = wd
	}
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("path validation: invalid base dir: %w", err)
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("path validation: invalid path: %w", err)
	}
	rel, err := filepath.Rel(baseAbs, resolved)
	if err != nil {
		return fmt.Errorf("path validation: path not under base: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path validation: path escapes base directory (reject .. traversal)")
	}
	return nil
}
