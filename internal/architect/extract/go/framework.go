// Package golang implements framework detection.
package golang

import (
	"sort"
	"strings"
)

// FrameworkSignature defines a detectable Go framework.
type FrameworkSignature struct {
	Name       string
	ImportPath string
	Confidence float64
	Evidence   string
	Category   string
}

var frameworkSignatures = []FrameworkSignature{
	{Name: "Gin", ImportPath: "github.com/gin-gonic/gin", Confidence: 0.95, Evidence: "gin import detected", Category: "web"},
	{Name: "Echo", ImportPath: "github.com/labstack/echo/v4", Confidence: 0.95, Evidence: "echo v4 import detected", Category: "web"},
	{Name: "Chi", ImportPath: "github.com/go-chi/chi/v5", Confidence: 0.95, Evidence: "chi v5 import detected", Category: "web"},
	{Name: "gRPC", ImportPath: "google.golang.org/grpc", Confidence: 0.90, Evidence: "grpc import detected", Category: "grpc"},
	{Name: "stdlib HTTP", ImportPath: "net/http", Confidence: 0.70, Evidence: "net/http import detected", Category: "web"},
	{Name: "Cobra", ImportPath: "github.com/spf13/cobra", Confidence: 0.90, Evidence: "cobra CLI import detected", Category: "cli"},
	{Name: "testify", ImportPath: "github.com/stretchr/testify", Confidence: 0.90, Evidence: "testify import detected", Category: "testing"},
}

// detectFrameworks scans external imports for known Go framework signals.
func detectFrameworks(externalImports map[string]struct{}) []DetectedFramework {
	var frameworks []DetectedFramework
	seen := make(map[string]bool)

	for imp := range externalImports {
		for _, sig := range frameworkSignatures {
			if seen[sig.Name] {
				continue
			}
			if imp == sig.ImportPath || strings.HasPrefix(imp, sig.ImportPath+"/") {
				frameworks = append(frameworks, DetectedFramework{
					Name:       sig.Name,
					ImportPath: imp,
					Confidence: sig.Confidence,
					Evidence:   sig.Evidence,
				})
				seen[sig.Name] = true
			}
		}
	}

	sort.Slice(frameworks, func(i, j int) bool {
		return frameworks[i].Name < frameworks[j].Name
	})
	return frameworks
}
