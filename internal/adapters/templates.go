package adapters

import (
	"embed"
	"text/template"
)

//go:embed templates/claude-code/*.tmpl templates/opencode/*.tmpl templates/codex/*.tmpl templates/cursor/*.tmpl
var templateFS embed.FS

// mustParse reads a template from the embedded FS and parses it.
// Panics if the file is missing or has a parse error (programming error).
func mustParse(name, path string) *template.Template {
	raw, err := templateFS.ReadFile(path)
	if err != nil {
		panic("adapters: missing embedded template " + path + ": " + err.Error())
	}
	t, err := template.New(name).Parse(string(raw))
	if err != nil {
		panic("adapters: parse template " + path + ": " + err.Error())
	}
	return t
}
