package spec

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
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

	chainedMethods := collectChainedMethods(node, fset)

	ast.Inspect(node, func(n ast.Node) bool {
		// Track short variable assignments: api := r.Group("/prefix")
		if assign, ok := n.(*ast.AssignStmt); ok {
			if len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
				if id, ok := assign.Lhs[0].(*ast.Ident); ok {
					if pfx := resolveGroupPrefix(assign.Rhs[0], prefixes); pfx != "" {
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
					return false
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

// extractEndpointsFromBlock extracts routes from a chi Route() closure body.
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
