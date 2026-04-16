package spec

import (
	"os"
	"path/filepath"
	"strings"
)

// ExtractAPIContracts scans a directory for Go files and extracts HTTP endpoints.
func ExtractAPIContracts(dir string) (*APIContracts, error) {
	var endpoints []Endpoint

	abs, err := filepath.Abs(dir)
	if err != nil {
		return &APIContracts{}, nil
	}

	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return &APIContracts{}, nil
	}

	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileEndpoints, exErr := ExtractGoRoutes(path)
		if exErr != nil {
			return nil
		}
		endpoints = append(endpoints, fileEndpoints...)
		return nil
	})
	if err != nil {
		return &APIContracts{}, nil
	}

	return &APIContracts{
		HTTPEndpoints: endpoints,
		Total:         len(endpoints),
	}, nil
}
