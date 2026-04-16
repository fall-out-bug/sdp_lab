package spec

import (
	"go/ast"
	"go/token"
	"strings"
)

// resolveGroupPrefix detects Group("...") and PathPrefix("...").Subrouter()
// patterns and returns the fully-composed prefix by resolving parent prefixes.
func resolveGroupPrefix(expr ast.Expr, prefixes map[string]string) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	parentPfx := prefixForExpr(sel.X, prefixes)
	switch sel.Sel.Name {
	case "Group":
		if len(call.Args) >= 1 {
			return parentPfx + extractStringLit(call.Args[0])
		}
	case "Subrouter":
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			if innerSel, ok := inner.Fun.(*ast.SelectorExpr); ok &&
				innerSel.Sel.Name == "PathPrefix" {
				if len(inner.Args) >= 1 {
					return parentPfx + extractStringLit(inner.Args[0])
				}
			}
		}
	}
	return ""
}

// prefixForExpr resolves the prefix for an AST expression by looking up
// identifier names in the prefix map.
func prefixForExpr(expr ast.Expr, prefixes map[string]string) string {
	if id, ok := expr.(*ast.Ident); ok {
		return prefixes[id.Name]
	}
	if call, ok := expr.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			return prefixForExpr(sel.X, prefixes)
		}
	}
	return ""
}

// collectChainedMethods walks the AST to find .Methods("GET") chains
// from gorilla/mux HandleFunc calls.
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

func extractStringLit(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	return strings.Trim(lit.Value, `"`)
}

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

func prefixForReceiver(sel *ast.SelectorExpr, prefixes map[string]string) string {
	if id, ok := sel.X.(*ast.Ident); ok {
		return prefixes[id.Name]
	}
	return ""
}
