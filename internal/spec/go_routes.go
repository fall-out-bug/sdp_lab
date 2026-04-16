package spec

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// ExtractGoRoutes parses a single Go file and extracts HTTP route registrations,
// composing group/subrouter prefixes for chi Route(), gin/echo Group(), and
// gorilla PathPrefix().Subrouter() patterns.
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

	relPath := filepath.Base(filePath)
	return extractRoutesWithPrefixes(node, fset, relPath), nil
}

// extractRoutesWithPrefixes walks the AST, tracking variable assignments for
// group/subrouter patterns and composing full route paths.
func extractRoutesWithPrefixes(node ast.Node, fset *token.FileSet, relPath string) []Endpoint {
	prefixes := map[string]string{} // varName -> prefix
	var endpoints []Endpoint

	// Collect gorilla method chains: map from HandleFunc line to method.
	chainedMethods := collectChainedMethods(node, fset)

	// Single pass: collect prefixes and extract route registrations.
	ast.Inspect(node, func(n ast.Node) bool {
		// Track short variable assignments: api := r.Group("/prefix")
		if assign, ok := n.(*ast.AssignStmt); ok {
			if len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
				if id, ok := assign.Lhs[0].(*ast.Ident); ok {
					if pfx := extractGroupPrefix(assign.Rhs[0]); pfx != "" {
						prefixes[id.Name] = pfx
					}
				}
			}
			return true
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// chi: r.Route("/prefix", func(r chi.Router) { ... })
		if sel.Sel.Name == "Route" && len(call.Args) >= 2 {
			if pfx := extractStringLit(call.Args[0]); pfx != "" {
				if fn, ok := call.Args[1].(*ast.FuncLit); ok {
					innerPrefixes := map[string]string{}
					if len(fn.Type.Params.List) > 0 {
						for _, name := range fn.Type.Params.List[0].Names {
							innerPrefixes[name.Name] = pfx
						}
					}
					inner := extractEndpointsFromBlock(fn.Body, fset, relPath,
						innerPrefixes, chainedMethods)
					endpoints = append(endpoints, inner...)
					return false // don't double-process
				}
			}
		}

		// HTTP method calls: r.Get, r.GET, e.POST, etc.
		if method, found := isHTTPMethodCall(sel); found && len(call.Args) >= 1 {
			path := extractStringLit(call.Args[0])
			handler := extractIdentName(call.Args, 1)
			if path != "" {
				pfx := prefixForReceiver(sel, prefixes)
				pos := fset.Position(call.Lparen)
				endpoints = append(endpoints, Endpoint{
					Method: method, Path: pfx + path, Handler: handler,
					SourceFile: relPath, SourceLine: pos.Line,
				})
			}
		}

		// HandleFunc / Handle (stdlib and gorilla base)
		if sel.Sel != nil && (sel.Sel.Name == "HandleFunc" || sel.Sel.Name == "Handle") {
			if len(call.Args) >= 1 {
				path := extractStringLit(call.Args[0])
				handler := extractIdentName(call.Args, 1)
				if path != "" {
					pfx := prefixForReceiver(sel, prefixes)
					pos := fset.Position(call.Lparen)
					method := ""
					if m, ok := chainedMethods[pos.Line]; ok {
						method = m
					}
					endpoints = append(endpoints, Endpoint{
						Method: method, Path: pfx + path, Handler: handler,
						SourceFile: relPath, SourceLine: pos.Line,
					})
				}
			}
		}

		return true
	})

	return endpoints
}

// extractEndpointsFromBlock extracts route registrations from a block statement
// (used for chi Route() closure bodies), using the provided prefix map.
func extractEndpointsFromBlock(body *ast.BlockStmt, fset *token.FileSet,
	relPath string, prefixes map[string]string, chainedMethods map[int]string,
) []Endpoint {
	var endpoints []Endpoint
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if method, found := isHTTPMethodCall(sel); found && len(call.Args) >= 1 {
			path := extractStringLit(call.Args[0])
			handler := extractIdentName(call.Args, 1)
			if path != "" {
				pfx := prefixForReceiver(sel, prefixes)
				pos := fset.Position(call.Lparen)
				endpoints = append(endpoints, Endpoint{
					Method: method, Path: pfx + path, Handler: handler,
					SourceFile: relPath, SourceLine: pos.Line,
				})
			}
		}
		return true
	})
	return endpoints
}

// prefixForReceiver looks up the prefix for the receiver variable of sel.X.
func prefixForReceiver(sel *ast.SelectorExpr, prefixes map[string]string) string {
	if id, ok := sel.X.(*ast.Ident); ok {
		return prefixes[id.Name]
	}
	return ""
}

// extractGroupPrefix detects Group("...") and PathPrefix("...").Subrouter()
// patterns and returns the prefix string.
func extractGroupPrefix(expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	switch sel.Sel.Name {
	case "Group":
		if len(call.Args) >= 1 {
			return extractStringLit(call.Args[0])
		}
	case "Subrouter":
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			if innerSel, ok := inner.Fun.(*ast.SelectorExpr); ok &&
				innerSel.Sel.Name == "PathPrefix" {
				if len(inner.Args) >= 1 {
					return extractStringLit(inner.Args[0])
				}
			}
		}
	}
	return ""
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
		expr = sel.X
	}
}

// isHTTPMethodCall returns the HTTP method if the selector is a method registration.
func isHTTPMethodCall(sel *ast.SelectorExpr) (string, bool) {
	if sel.Sel == nil {
		return "", false
	}
	upper := strings.ToUpper(sel.Sel.Name)
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
