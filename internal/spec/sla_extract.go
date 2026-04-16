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

// ExtractSLAParameters scans Go files for SLA-related patterns.
func ExtractSLAParameters(root string) (SLAParameters, error) {
	var sla SLAParameters
	abs, err := filepath.Abs(root)
	if err != nil {
		return sla, fmt.Errorf("spec: resolve path: %w", err)
	}
	filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		params := extractFileSLA(path)
		sla.Timeouts = append(sla.Timeouts, params.Timeouts...)
		sla.Retries = append(sla.Retries, params.Retries...)
		sla.RateLimits = append(sla.RateLimits, params.RateLimits...)
		sla.ResourcePools = append(sla.ResourcePools, params.ResourcePools...)
		sla.HealthChecks = append(sla.HealthChecks, params.HealthChecks...)
		return nil
	})
	sla.Total = len(sla.Timeouts) + len(sla.Retries) + len(sla.RateLimits) +
		len(sla.CircuitBreakers) + len(sla.ResourcePools) + len(sla.HealthChecks)
	return sla, nil
}

func extractFileSLA(filePath string) SLAParameters {
	var sla SLAParameters
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return sla
	}
	rel := filepath.Base(filePath)
	ast.Inspect(node, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.CallExpr:
			sla.Timeouts = append(sla.Timeouts, callTimeouts(v, fset, rel)...)
			sla.RateLimits = append(sla.RateLimits, callRateLimits(v, fset, rel)...)
			sla.HealthChecks = append(sla.HealthChecks, callHealthChecks(v, rel)...)
		case *ast.CompositeLit:
			sla.Timeouts = append(sla.Timeouts, structTimeouts(v, fset, rel)...)
		case *ast.AssignStmt:
			sla.Retries = append(sla.Retries, retryFromAssign(v, fset, rel)...)
		}
		return true
	})
	return sla
}

func callTimeouts(call *ast.CallExpr, fset *token.FileSet, rel string) []SLAParam {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return nil
	}
	if sel.Sel.Name != "WithTimeout" && sel.Sel.Name != "WithDeadline" {
		return nil
	}
	dur := ""
	if len(call.Args) >= 2 {
		dur = exprS(call.Args[1])
	}
	pos := fset.Position(call.Lparen)
	return []SLAParam{{
		Category:  "timeout",
		Component: exprS(call.Fun),
		Value:     dur,
		Location:  fmt.Sprintf("%s:%d", rel, pos.Line),
	}}
}

func callRateLimits(call *ast.CallExpr, fset *token.FileSet, rel string) []SLAParam {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "NewLimiter" {
		return nil
	}
	rate, burst := "", ""
	if len(call.Args) >= 1 {
		rate = exprS(call.Args[0])
	}
	if len(call.Args) >= 2 {
		burst = exprS(call.Args[1])
	}
	pos := fset.Position(call.Lparen)
	return []SLAParam{{
		Category:  "rate_limit",
		Component: exprS(call.Fun),
		Value:     fmt.Sprintf("rate=%s burst=%s", rate, burst),
		Location:  fmt.Sprintf("%s:%d", rel, pos.Line),
	}}
}

func callHealthChecks(call *ast.CallExpr, rel string) []SLAParam {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return nil
	}
	if sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle" {
		return nil
	}
	if len(call.Args) < 1 {
		return nil
	}
	path := extractStringLit(call.Args[0])
	if !isHealthPath(path) {
		return nil
	}
	return []SLAParam{{
		Category:  "health_check",
		Component: path,
		Value:     "endpoint",
		Location:  rel,
	}}
}

func isHealthPath(p string) bool {
	for _, h := range []string{"/health", "/healthz", "/ready", "/readyz", "/alive"} {
		if p == h || strings.HasPrefix(p, h+"/") {
			return true
		}
	}
	return false
}

// structTimeouts finds Timeout/ReadTimeout/WriteTimeout fields in struct literals.
func structTimeouts(lit *ast.CompositeLit, fset *token.FileSet, rel string) []SLAParam {
	var out []SLAParam
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if key.Name == "Timeout" || key.Name == "ReadTimeout" || key.Name == "WriteTimeout" {
			pos := fset.Position(kv.Pos())
			out = append(out, SLAParam{
				Category:  "timeout",
				Component: key.Name,
				Value:     exprS(kv.Value),
				Location:  fmt.Sprintf("%s:%d", rel, pos.Line),
			})
		}
	}
	return out
}

// retryFromAssign finds "maxRetries := N" patterns.
func retryFromAssign(assign *ast.AssignStmt, fset *token.FileSet, rel string) []SLAParam {
	for i, lhs := range assign.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		low := strings.ToLower(id.Name)
		if (strings.Contains(low, "retry") || strings.Contains(low, "retries")) && strings.Contains(low, "max") {
			if i < len(assign.Rhs) {
				pos := fset.Position(assign.Pos())
				return []SLAParam{{
					Category:  "retry",
					Component: id.Name,
					Value:     exprS(assign.Rhs[i]),
					Location:  fmt.Sprintf("%s:%d", rel, pos.Line),
				}}
			}
		}
	}
	return nil
}
