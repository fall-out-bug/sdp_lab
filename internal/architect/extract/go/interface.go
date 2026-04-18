// Package golang implements interface detection (stub implementation).
package golang

import (
	"golang.org/x/tools/go/packages"
)

// extractInterfaceInfo extracts interface definitions and implementations.
func extractInterfaceInfo(pkgs []*packages.Package, modPath string, nodeMap map[string]*PackageNode) {
	// Stub implementation - would need full type checking for real implementation
}
