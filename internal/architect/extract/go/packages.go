// Package golang implements go/packages import graph builder and utilities.
package golang

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// addNode inserts a PackageNode into the map (idempotent).
func addNode(m map[string]*PackageNode, pkg *packages.Package, modPath string) {
	if _, ok := m[pkg.PkgPath]; ok {
		return
	}
	rel := strings.TrimPrefix(pkg.PkgPath, modPath+"/")
	cluster := ""
	if idx := strings.LastIndex(rel, "/"); idx > 0 {
		cluster = rel[:idx]
	}
	m[pkg.PkgPath] = &PackageNode{
		ImportPath:  pkg.PkgPath,
		Dir:         dirOf(pkg),
		Name:        pkg.Name,
		Cluster:     cluster,
		IsGenerated: isGenerated(pkg),
	}
}

func dirOf(pkg *packages.Package) string {
	if len(pkg.GoFiles) > 0 {
		return filepath.Dir(pkg.GoFiles[0])
	}
	return ""
}

// isGenerated returns true when every Go file in the package matches a
// generated-file pattern (*.pb.go, *.gen.go, etc.).
func isGenerated(pkg *packages.Package) bool {
	if len(pkg.GoFiles) == 0 && len(pkg.OtherFiles) == 0 {
		return false
	}
	for _, f := range pkg.GoFiles {
		base := filepath.Base(f)
		if !isGeneratedFile(base) {
			return false
		}
	}
	return len(pkg.GoFiles) > 0
}

// isGeneratedFile returns true if the filename appears to be generated.
func isGeneratedFile(filename string) bool {
	generatedSuffixes := []string{
		".pb.go", ".gen.go", ".generated.go", ".mock.go", ".mock_.go",
		".db.go", ".tbls.go", ".wire.go", ".wire_gen.go", ".deepcopy.go",
		".facade.go", ".inject.go",
	}
	for _, suffix := range generatedSuffixes {
		if strings.HasSuffix(filename, suffix) {
			return true
		}
	}

	base := strings.ToLower(filename)
	patterns := []string{
		"zz_generated", "generated", "wire", "mock", "facade", "deepcopy", "tbls",
	}
	for _, pattern := range patterns {
		if strings.Contains(base, pattern) {
			return true
		}
	}

	return false
}

// detectDeployUnits scans the cmd/ directory for deployable binaries.
func detectDeployUnits(rootDir string) []DeployUnit {
	cmdDir := filepath.Join(rootDir, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return nil
	}

	var units []DeployUnit
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		unitPath := filepath.Join(cmdDir, e.Name())
		if !hasGoFiles(unitPath) {
			continue
		}
		hasMain, pkgName := checkMainPackage(unitPath)
		units = append(units, DeployUnit{
			Name:        e.Name(),
			Path:        filepath.ToSlash(filepath.Join("cmd", e.Name())),
			HasMain:     hasMain,
			PackageName: pkgName,
		})
	}
	return units
}

// checkMainPackage checks if a directory contains a main package.
func checkMainPackage(dir string) (bool, string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, ""
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			content := string(data)
			for _, line := range strings.Split(content, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "package ") {
					pkgName := strings.TrimSpace(strings.TrimPrefix(line, "package"))
					return pkgName == "main", pkgName
				}
			}
		}
	}
	return false, ""
}

// hasGoFiles returns true if the directory contains .go files.
func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			return true
		}
	}
	return false
}

// isInternal returns true when pkgPath belongs to the same module.
func isInternal(pkgPath, modPath string) bool {
	return pkgPath == modPath || strings.HasPrefix(pkgPath, modPath+"/")
}

// readModulePath extracts the module path from go.mod.
func readModulePath(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", os.ErrNotExist
}
