package spec

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// ExtractGoRoutes parses a single Go file and extracts HTTP route registrations.
func ExtractGoRoutes(filePath string) ([]Endpoint, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("spec: resolve path: %w", err)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, abs, nil, parser.ParseComments)
	if err != nil {
		return nil, nil
	}

	var endpoints []Endpoint
	relPath := filepath.Base(filePath)

	// Collect gorilla method chains: map from line to method
	chainedMethods := collectChainedMethods(node, fset)

	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Framework method calls: r.Get, r.GET, e.POST, etc.
		if method, found := isHTTPMethodCall(sel); found && len(call.Args) >= 1 {
			path := extractStringLit(call.Args[0])
			handler := extractIdentName(call.Args, 1)
			if path != "" {
				pos := fset.Position(call.Lparen)
				endpoints = append(endpoints, Endpoint{
					Method:     method,
					Path:       path,
					Handler:    handler,
					SourceFile: relPath,
					SourceLine: pos.Line,
				})
			}
		}

		// HandleFunc / Handle (stdlib and gorilla base)
		if sel.Sel != nil && (sel.Sel.Name == "HandleFunc" || sel.Sel.Name == "Handle") {
			if len(call.Args) >= 1 {
				path := extractStringLit(call.Args[0])
				handler := extractIdentName(call.Args, 1)
				if path != "" {
					pos := fset.Position(call.Lparen)
					method := ""
					// Check for chained .Methods() on same line
					if m, ok := chainedMethods[pos.Line]; ok {
						method = m
					}
					endpoints = append(endpoints, Endpoint{
						Method:     method,
						Path:       path,
						Handler:    handler,
						SourceFile: relPath,
						SourceLine: pos.Line,
					})
				}
			}
		}

		return true
	})

	return endpoints, nil
}

// collectChainedMethods walks the AST to find .Methods("GET") chains
// from gorilla/mux HandleFunc calls. Returns a map of HandleFunc line -> method.
func collectChainedMethods(node ast.Node, fset *token.FileSet) map[int]string {
	result := make(map[int]string)

	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Methods" {
			return true
		}
		if len(call.Args) < 1 {
			return true
		}

		method := extractStringLit(call.Args[0])
		if method == "" {
			return true
		}

		// Walk the X chain to find the HandleFunc call
		handleFuncLine := findHandleFuncLine(sel.X, fset)
		if handleFuncLine > 0 {
			result[handleFuncLine] = strings.ToUpper(method)
		}

		return true
	})

	return result
}

// findHandleFuncLine walks a chain of selector/call expressions to find
// the HandleFunc/Handle call and returns its line number.
func findHandleFuncLine(expr ast.Expr, fset *token.FileSet) int {
	for {
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			return 0
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return 0
		}
		if sel.Sel != nil && (sel.Sel.Name == "HandleFunc" || sel.Sel.Name == "Handle") {
			return fset.Position(call.Lparen).Line
		}
		// Continue walking the chain (e.g., .Subrouter() chains)
		expr = sel.X
	}
}

// isHTTPMethodCall returns the HTTP method if the selector is a method registration.
func isHTTPMethodCall(sel *ast.SelectorExpr) (string, bool) {
	if sel.Sel == nil {
		return "", false
	}
	name := sel.Sel.Name
	upper := strings.ToUpper(name)
	switch upper {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		return upper, true
	}
	return "", false
}

// extractStringLit extracts a string literal from an AST expression.
func extractStringLit(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	return strings.Trim(lit.Value, `"`)
}

// extractIdentName extracts a function/identifier name from call arguments.
func extractIdentName(args []ast.Expr, idx int) string {
	if len(args) <= idx {
		return ""
	}
	switch arg := args[idx].(type) {
	case *ast.Ident:
		return arg.Name
	case *ast.SelectorExpr:
		return arg.Sel.Name
	}
	return ""
}
