package typescript

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Import regexes — compiled once.
var (
	reESImport      = regexp.MustCompile(`import\s+(?:.*?)\s+from\s+['"]([^'"]+)['"]`)
	reSideEffect    = regexp.MustCompile(`^import\s+['"]([^'"]+)['"]\s*;?\s*$`)
	reCommonJS      = regexp.MustCompile(`(?:const|let|var)\s+\w+\s*=\s*require\(\s*['"]([^'"]+)['"]\s*\)`)
	reReExport      = regexp.MustCompile(`export\s+(?:.*?)\s+from\s+['"]([^'"]+)['"]`)
	reDynamicImport = regexp.MustCompile(`import\(\s*['"]([^'"]+)['"]\s*\)`)
)

// extractFileImports reads a single source file and returns import edges and
// a set of external specifiers.
func extractFileImports(path, relPath, rootDir string) ([]TSImportEdge, map[string]struct{}) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	var edges []TSImportEdge
	externals := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments.
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		type match struct {
			specifier string
			kind      TSImportKind
		}

		var matches []match

		// Re-export: export ... from "X"
		if m := reReExport.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportReExport})
		}

		// ES module: import X from "Y"
		if m := reESImport.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportESModule})
		}

		// Side-effect: import "X"
		if m := reSideEffect.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportSideEffect})
		}

		// Dynamic import: import("X")
		if m := reDynamicImport.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportDynamic})
		}

		// CommonJS: const X = require("Y")
		if m := reCommonJS.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportCommonJS})
		}

		for _, mt := range matches {
			spec := mt.specifier
			resolved := resolveSpecifier(spec, relPath, rootDir)
			isExternal := !isLocalSpecifier(spec)

			edge := TSImportEdge{
				From:     relPath,
				To:       resolved,
				Kind:     mt.kind,
				Line:     lineNo,
				Resolved: !isExternal,
			}

			if isExternal {
				externals[spec] = struct{}{}
				// For external packages, use the specifier as-is for the target.
				edge.To = spec
			}

			edges = append(edges, edge)
		}
	}

	return edges, externals
}

// isLocalSpecifier returns true if the import specifier refers to a local file.
func isLocalSpecifier(spec string) bool {
	return strings.HasPrefix(spec, "./") ||
		strings.HasPrefix(spec, "../") ||
		strings.HasPrefix(spec, "/")
}

// resolveSpecifier attempts to resolve an import specifier to a relative path.
func resolveSpecifier(spec, fromRelPath, rootDir string) string {
	if !isLocalSpecifier(spec) {
		// Package specifier: return the package name (first segment, possibly scoped).
		parts := strings.SplitN(spec, "/", -1)
		if strings.HasPrefix(spec, "@") && len(parts) >= 2 {
			return parts[0] + "/" + parts[1] // @scope/package
		}
		return parts[0]
	}

	// Local import: resolve relative to the importing file's directory.
	fromDir := filepath.Dir(fromRelPath)
	resolved := filepath.Join(fromDir, spec)
	// Clean the path.
	resolved = filepath.Clean(resolved)
	return resolved
}

// isBarrelFile checks if a file is a barrel file (index.ts/js) that contains
// re-export statements.
func isBarrelFile(absPath, relPath string) bool {
	base := filepath.Base(relPath)
	if !barrelFileNames[base] {
		return false
	}
	// Verify that the file contains at least one re-export.
	data, err := os.ReadFile(absPath)
	if err != nil {
		return false
	}
	return reReExport.Match(data)
}
