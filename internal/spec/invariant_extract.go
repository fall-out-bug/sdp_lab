package spec

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ExtractInvariants scans a directory for Go invariant patterns.
func ExtractInvariants(root string) (Invariants, error) {
	var inv Invariants
	abs, err := filepath.Abs(root)
	if err != nil {
		return inv, fmt.Errorf("spec: resolve path: %w", err)
	}
	_ = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		ts, tys, cs, as := extractFileInvariants(path)
		inv.Database = append(inv.Database, ts...)
		inv.TypeSystem = append(inv.TypeSystem, tys...)
		inv.Concurrency = append(inv.Concurrency, cs...)
		inv.Architectural = append(inv.Architectural, as...)
		return nil
	})
	inv.Total = len(inv.Database) + len(inv.TypeSystem) + len(inv.Concurrency) + len(inv.Architectural)
	return inv, nil
}

func extractFileInvariants(filePath string) (db []DBInvariant, ty []TypeInvariant, co []ConcInvariant, ar []ArchInvariant) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return
	}
	rel := filepath.Base(filePath)
	ast.Inspect(node, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.GenDecl:
			ar = append(ar, interfaceCompliance(v, fset, rel)...)
		case *ast.FuncDecl:
			if v.Body != nil {
				co = append(co, mutexGuards(v, fset, rel)...)
				ty = append(ty, typeAssertions(v, fset, rel)...)
				db = append(db, contextDeadlines(v, fset, rel)...)
			}
		}
		return true
	})
	ar = append(ar, buildConstraints(node, rel)...)
	return
}

// interfaceCompliance finds "var _ Interface = (*Impl)(nil)" patterns.
func interfaceCompliance(gen *ast.GenDecl, fset *token.FileSet, rel string) []ArchInvariant {
	if gen.Tok != token.VAR {
		return nil
	}
	var out []ArchInvariant
	for _, sp := range gen.Specs {
		vs, ok := sp.(*ast.ValueSpec)
		if !ok || len(vs.Names) == 0 || len(vs.Values) == 0 || vs.Names[0].Name != "_" {
			continue
		}
		iface := exprS(vs.Type)
		if call, ok := vs.Values[0].(*ast.CallExpr); ok && iface != "" {
			impl := exprS(call.Fun)
			if impl != "" {
				pos := fset.Position(vs.Pos())
				out = append(out, ArchInvariant{
					Category: "interface_compliance",
					Detail:   fmt.Sprintf("%s must implement %s", impl, iface),
					Location: fmt.Sprintf("%s:%d", rel, pos.Line),
				})
			}
		}
	}
	return out
}

// mutexGuards finds sync.Mutex Lock/Unlock patterns in functions.
func mutexGuards(fn *ast.FuncDecl, fset *token.FileSet, rel string) []ConcInvariant {
	var out []ConcInvariant
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}
		if sel.Sel.Name == "Lock" || sel.Sel.Name == "RLock" {
			pos := fset.Position(call.Lparen)
			out = append(out, ConcInvariant{
				Category: "mutex_guard",
				Detail:   fmt.Sprintf("mutex %s at %s", sel.Sel.Name, exprS(sel.X)),
				Location: fmt.Sprintf("%s:%d", rel, pos.Line),
			})
		}
		return true
	})
	return out
}

// typeAssertions finds type assertion patterns like v.(string).
func typeAssertions(fn *ast.FuncDecl, fset *token.FileSet, rel string) []TypeInvariant {
	var out []TypeInvariant
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ta, ok := n.(*ast.TypeAssertExpr)
		if !ok || ta.Type == nil {
			return true
		}
		pos := fset.Position(ta.Lparen)
		out = append(out, TypeInvariant{
			Category: "type_assertion",
			Detail:   fmt.Sprintf("assert to %s", exprS(ta.Type)),
			Location: fmt.Sprintf("%s:%d", rel, pos.Line),
		})
		return true
	})
	return out
}

// contextDeadlines finds context.WithTimeout/WithDeadline calls.
func contextDeadlines(fn *ast.FuncDecl, fset *token.FileSet, rel string) []DBInvariant {
	var out []DBInvariant
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}
		if sel.Sel.Name == "WithTimeout" || sel.Sel.Name == "WithDeadline" {
			dur := ""
			if len(call.Args) >= 2 {
				dur = exprS(call.Args[1])
			}
			pos := fset.Position(call.Lparen)
			out = append(out, DBInvariant{
				Constraint: "operation_timeout",
				Detail:     fmt.Sprintf("context.%s(%s)", sel.Sel.Name, dur),
				Location:   fmt.Sprintf("%s:%d", rel, pos.Line),
			})
		}
		return true
	})
	return out
}

// buildConstraints finds //go:build and // +build comments.
func buildConstraints(node *ast.File, rel string) []ArchInvariant {
	var out []ArchInvariant
	for _, cg := range node.Comments {
		for _, c := range cg.List {
			txt := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if strings.HasPrefix(txt, "go:build ") || strings.HasPrefix(txt, "+build ") {
				detail := strings.TrimSpace(strings.TrimPrefix(txt, "go:build "))
				detail = strings.TrimSpace(strings.TrimPrefix(detail, "+build "))
				out = append(out, ArchInvariant{Category: "build_constraint", Detail: detail, Location: rel})
			}
		}
	}
	return out
}
