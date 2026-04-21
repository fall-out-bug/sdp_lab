package golang

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func scanSourceGraph(dir, modPath string, nodeMap map[string]*PackageNode, edges *[]ImportEdge, edgeSet map[ImportEdge]struct{}, externalImports map[string]struct{}) {
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") || isGeneratedFile(entry.Name()) {
			return nil
		}

		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil
		}
		relDir, relErr := filepath.Rel(dir, filepath.Dir(path))
		if relErr != nil {
			return nil
		}
		relDir = filepath.ToSlash(relDir)
		importPath := modPath
		cluster := ""
		if relDir != "." && relDir != "" {
			importPath = modPath + "/" + relDir
			cluster = relDir
		}
		upsertScannedNode(nodeMap, importPath, filepath.Dir(path), file.Name.Name, cluster)

		for _, imp := range file.Imports {
			value, unquoteErr := strconv.Unquote(imp.Path.Value)
			if unquoteErr != nil || value == "C" {
				continue
			}
			if isInternal(value, modPath) {
				edge := ImportEdge{From: importPath, To: value}
				if _, ok := edgeSet[edge]; !ok {
					edgeSet[edge] = struct{}{}
					*edges = append(*edges, edge)
				}
				if _, ok := nodeMap[value]; !ok {
					nodeMap[value] = &PackageNode{ImportPath: value}
				}
			} else {
				externalImports[value] = struct{}{}
			}
		}
		return nil
	})
}

func upsertScannedNode(nodeMap map[string]*PackageNode, importPath, dir, name, cluster string) {
	node, ok := nodeMap[importPath]
	if !ok {
		nodeMap[importPath] = &PackageNode{
			ImportPath: importPath,
			Dir:        dir,
			Name:       name,
			Cluster:    cluster,
		}
		return
	}
	if node.Dir == "" {
		node.Dir = dir
	}
	if node.Name == "" {
		node.Name = name
	}
	if node.Cluster == "" {
		node.Cluster = cluster
	}
}
