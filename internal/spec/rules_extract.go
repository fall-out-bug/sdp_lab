package spec

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// ExtractBusinessRules parses a Go file for validation tags, guard clauses, and error constants.
func ExtractBusinessRules(filePath string) ([]ValidationRule, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("spec: resolve path: %w", err)
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, abs, nil, parser.ParseComments)
	if err != nil {
		return nil, nil
	}
	rel := filepath.Base(filePath)
	var rules []ValidationRule
	ast.Inspect(node, func(n ast.Node) bool {
		if f, ok := n.(*ast.Field); ok && f.Tag != nil {
			rules = append(rules, tagRules(f.Tag.Value, f, rel)...)
			return false
		}
		return true
	})
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			rules = append(rules, guards(fn, fset, rel)...)
			return false
		}
		return true
	})
	ast.Inspect(node, func(n ast.Node) bool {
		if g, ok := n.(*ast.GenDecl); ok && g.Tok == token.VAR {
			rules = append(rules, errConsts(g, fset, rel)...)
			return false
		}
		return true
	})
	return rules, nil
}

func tagRules(tagStr string, field *ast.Field, rel string) []ValidationRule {
	var rules []ValidationRule
	tag := strings.Trim(tagStr, "`")
	for _, key := range []string{"validate", "binding"} {
		val := tagVal(tag, key)
		if val == "" {
			continue
		}
		name := fName(field)
		rules = append(rules, ValidationRule{
			Category: "validation_tag",
			Description: fmt.Sprintf("field %q has %s constraints: %s", name, key, val),
			Enforcement: key, Location: rel, Field: name,
			Constraints: parseCS(val),
		})
	}
	return rules
}

func tagVal(tag, key string) string {
	n := key + `:"`
	i := strings.Index(tag, n)
	if i < 0 {
		return ""
	}
	s := i + len(n)
	e := strings.Index(tag[s:], `"`)
	if e < 0 {
		return ""
	}
	return tag[s : s+e]
}

func parseCS(val string) []Constraint {
	var cs []Constraint
	for _, p := range strings.Split(val, ",") {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		c := Constraint{Type: kv[0]}
		if len(kv) == 2 {
			c.Value = kv[1]
		}
		cs = append(cs, c)
	}
	return cs
}

func fName(f *ast.Field) string {
	if len(f.Names) > 0 {
		return f.Names[0].Name
	}
	if id, ok := f.Type.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func guards(fn *ast.FuncDecl, fset *token.FileSet, rel string) []ValidationRule {
	if fn.Body == nil {
		return nil
	}
	var rules []ValidationRule
	for _, s := range fn.Body.List {
		ifStmt, ok := s.(*ast.IfStmt)
		if !ok || !hasRet(ifStmt.Body) {
			continue
		}
		pos := fset.Position(ifStmt.If)
		rules = append(rules, ValidationRule{
			Category: "guard_clause", Description: condStr(ifStmt.Cond),
			Enforcement: "runtime", Location: fmt.Sprintf("%s:%d", rel, pos.Line),
		})
	}
	return rules
}

func hasRet(b *ast.BlockStmt) bool {
	for _, s := range b.List {
		if _, ok := s.(*ast.ReturnStmt); ok {
			return true
		}
	}
	return false
}

func condStr(c ast.Expr) string {
	if bin, ok := c.(*ast.BinaryExpr); ok {
		return fmt.Sprintf("guard: %s %s %s", exprS(bin.X), bin.Op, exprS(bin.Y))
	}
	if u, ok := c.(*ast.UnaryExpr); ok {
		return fmt.Sprintf("guard: %s%s", u.Op, exprS(u.X))
	}
	return "guard: conditional check"
}

func exprS(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprS(v.X) + "." + v.Sel.Name
	case *ast.BinaryExpr:
		return exprS(v.X) + " " + v.Op.String() + " " + exprS(v.Y)
	case *ast.BasicLit:
		return v.Value
	case *ast.CallExpr:
		return exprS(v.Fun) + "(...)"
	case *ast.UnaryExpr:
		return v.Op.String() + exprS(v.X)
	}
	return "?"
}

func errConsts(gen *ast.GenDecl, fset *token.FileSet, rel string) []ValidationRule {
	var rules []ValidationRule
	for _, sp := range gen.Specs {
		vs, ok := sp.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range vs.Names {
			if !strings.HasPrefix(name.Name, "Err") || i >= len(vs.Values) {
				continue
			}
			if msg := errNewMsg(vs.Values[i]); msg != "" {
				pos := fset.Position(name.Pos())
				rules = append(rules, ValidationRule{
					Category: "error_constant", Description: msg,
					Enforcement: "constant", Location: fmt.Sprintf("%s:%d", rel, pos.Line),
					Field: name.Name,
				})
			}
		}
	}
	return rules
}

func errNewMsg(e ast.Expr) string {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "New" || len(c.Args) < 1 {
		return ""
	}
	return extractStringLit(c.Args[0])
}
