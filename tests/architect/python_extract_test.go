package architect_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
	"github.com/fall-out-bug/sdp_lab/internal/architect/extract"
)

// helper to create a temp dir with files.
func setupTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func hasDep(deps []architect.Dependency, name, kind string) bool {
	for _, d := range deps {
		if d.Name == name && d.Kind == kind {
			return true
		}
	}
	return false
}

func hasDepBySource(deps []architect.Dependency, name, source string) bool {
	for _, d := range deps {
		if d.Name == name && d.Source == source {
			return true
		}
	}
	return false
}

func hasFramework(fws []architect.Framework, name string) bool {
	for _, fw := range fws {
		if fw.Name == name {
			return true
		}
	}
	return false
}

// TestPythonExtractor_BasicImports verifies absolute and from-import extraction,
// stdlib vs third-party classification, and triple-quote/comment skipping.
func TestPythonExtractor_BasicImports(t *testing.T) {
	root := setupTree(t, map[string]string{
		"app.py": `import os
import json
import flask
from requests import get
# import should_be_skipped
"""
import inside_triple_quote
"""
import after_string
`,
	})

	ext := &extract.PythonExtractor{}
	res, err := ext.Extract(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	if res.Language != "python" {
		t.Errorf("Language = %q, want python", res.Language)
	}
	if res.ExtractionMethod != "regex" {
		t.Errorf("ExtractionMethod = %q, want regex", res.ExtractionMethod)
	}
	if res.AccuracyEstimate != 0.55 {
		t.Errorf("AccuracyEstimate = %f, want 0.55", res.AccuracyEstimate)
	}
	if res.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", res.FileCount)
	}

	// stdlib
	if !hasDep(res.Dependencies, "os", "stdlib") {
		t.Error("missing stdlib dep: os")
	}
	if !hasDep(res.Dependencies, "json", "stdlib") {
		t.Error("missing stdlib dep: json")
	}

	// third-party
	if !hasDep(res.Dependencies, "flask", "third-party") {
		t.Error("missing third-party dep: flask")
	}
	if !hasDep(res.Dependencies, "requests", "third-party") {
		t.Error("missing third-party dep: requests")
	}

	// should NOT contain skipped imports
	if hasDep(res.Dependencies, "should_be_skipped", "third-party") {
		t.Error("comment import should_be_skipped should not be extracted")
	}
	if hasDep(res.Dependencies, "inside_triple_quote", "third-party") {
		t.Error("triple-quote import inside_triple_quote should not be extracted")
	}

	// after_string should be present — it's after the closing triple quote
	if !hasDep(res.Dependencies, "after_string", "third-party") {
		t.Error("import after_string should be extracted (it follows a closed triple-quote)")
	}
}

// TestPythonExtractor_RelativeImports verifies that relative imports (dot-prefixed)
// are resolved to absolute module paths.
func TestPythonExtractor_RelativeImports(t *testing.T) {
	root := setupTree(t, map[string]string{
		"mypackage/__init__.py": "",
		"mypackage/core.py":    "from . import utils\n",
		"mypackage/sub/deep.py": "from .. import core\nfrom ..core import helper\n",
	})

	ext := &extract.PythonExtractor{}
	res, err := ext.Extract(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	// "from . import utils" in mypackage/core.py -> "mypackage.utils"
	if !hasDep(res.Dependencies, "mypackage.utils", "relative") {
		t.Error("expected relative dep mypackage.utils")
		for _, d := range res.Dependencies {
			t.Logf("  dep: %s (kind=%s source=%s)", d.Name, d.Kind, d.Source)
		}
	}

	// "from .. import core" in mypackage/sub/deep.py -> "mypackage.core"
	// (going up one level from mypackage/sub -> mypackage, then appending core)
	// Actually from .. is 2 dots = go up 1 level from current package (mypackage/sub)
	// -> mypackage, then import core -> mypackage.core
	found := false
	for _, d := range res.Dependencies {
		if d.Name == "mypackage.core" && d.Kind == "relative" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected relative dep mypackage.core from 'from .. import core'")
		for _, d := range res.Dependencies {
			t.Logf("  dep: %s (kind=%s source=%s)", d.Name, d.Kind, d.Source)
		}
	}
}

// TestPythonExtractor_Requirements verifies requirements.txt and pyproject.toml parsing.
func TestPythonExtractor_Requirements(t *testing.T) {
	root := setupTree(t, map[string]string{
		"requirements.txt": `flask==2.3.0
requests>=2.28.0
# this is a comment
gunicorn~=21.2

-r other-requirements.txt
`,
		"pyproject.toml": `[project]
name = "myapp"

[project.dependencies]
"celery>=5.0",
"redis>=4.0",

[tool.poetry.dependencies]
python = "^3.11"
sqlalchemy = "^2.0"
`,
	})

	ext := &extract.PythonExtractor{}
	res, err := ext.Extract(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	// requirements.txt
	if !hasDepBySource(res.Dependencies, "flask", "requirements.txt") {
		t.Error("missing requirements.txt dep: flask")
	}
	if !hasDepBySource(res.Dependencies, "requests", "requirements.txt") {
		t.Error("missing requirements.txt dep: requests")
	}
	if !hasDepBySource(res.Dependencies, "gunicorn", "requirements.txt") {
		t.Error("missing requirements.txt dep: gunicorn")
	}

	// pyproject.toml
	if !hasDepBySource(res.Dependencies, "celery", "pyproject.toml") {
		t.Error("missing pyproject.toml dep: celery")
	}
	if !hasDepBySource(res.Dependencies, "redis", "pyproject.toml") {
		t.Error("missing pyproject.toml dep: redis")
	}
	if !hasDepBySource(res.Dependencies, "sqlalchemy", "pyproject.toml") {
		t.Error("missing pyproject.toml dep: sqlalchemy")
	}

	// python itself should be skipped
	if hasDepBySource(res.Dependencies, "python", "pyproject.toml") {
		t.Error("python should not appear as a dependency")
	}
}

// TestPythonExtractor_FlaskDetection verifies Flask, FastAPI, and Django framework detection.
func TestPythonExtractor_FlaskDetection(t *testing.T) {
	root := setupTree(t, map[string]string{
		"flask_app.py": `from flask import Flask
app = Flask(__name__)

@app.route("/")
def index():
    return "hello"
`,
		"fastapi_app.py": `from fastapi import FastAPI
app = FastAPI()

@app.get("/items")
def items():
    return []
`,
		"settings.py": `INSTALLED_APPS = [
    "django.contrib.admin",
    "myapp",
]
`,
	})

	ext := &extract.PythonExtractor{}
	res, err := ext.Extract(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	if !hasFramework(res.Frameworks, "Flask") {
		t.Error("Flask framework not detected")
	}
	if !hasFramework(res.Frameworks, "FastAPI") {
		t.Error("FastAPI framework not detected")
	}
	if !hasFramework(res.Frameworks, "Django") {
		t.Error("Django framework not detected")
	}
}

// TestPythonExtractor_NoPythonFiles verifies that running against a tree with no
// .py files returns an empty (but non-nil) result without errors.
func TestPythonExtractor_NoPythonFiles(t *testing.T) {
	root := setupTree(t, map[string]string{
		"README.md":  "# Hello",
		"main.go":    "package main",
		"Makefile":   "all:\n\techo hi",
	})

	ext := &extract.PythonExtractor{}
	res, err := ext.Extract(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if res.FileCount != 0 {
		t.Errorf("FileCount = %d, want 0", res.FileCount)
	}
	if len(res.Dependencies) != 0 {
		t.Errorf("Dependencies = %d, want 0", len(res.Dependencies))
	}
	if len(res.Frameworks) != 0 {
		t.Errorf("Frameworks = %d, want 0", len(res.Frameworks))
	}
	if res.Language != "python" {
		t.Errorf("Language = %q, want python", res.Language)
	}
	if res.ExtractionMethod != "regex" {
		t.Errorf("ExtractionMethod = %q, want regex", res.ExtractionMethod)
	}
}

// TestPythonExtractor_SkipVenv ensures venv and __pycache__ dirs are skipped.
func TestPythonExtractor_SkipVenv(t *testing.T) {
	root := setupTree(t, map[string]string{
		"app.py":                     "import mylib\n",
		"venv/lib/site.py":           "import should_be_skipped\n",
		".venv/lib/site.py":          "import should_be_skipped2\n",
		"__pycache__/app.cpython.py": "import cached\n",
	})

	ext := &extract.PythonExtractor{}
	res, err := ext.Extract(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	if res.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1 (only app.py)", res.FileCount)
	}
	if !hasDep(res.Dependencies, "mylib", "third-party") {
		t.Error("missing dep: mylib")
	}
	if hasDep(res.Dependencies, "should_be_skipped", "third-party") {
		t.Error("venv dep should be skipped")
	}
	if hasDep(res.Dependencies, "should_be_skipped2", "third-party") {
		t.Error(".venv dep should be skipped")
	}
	if hasDep(res.Dependencies, "cached", "third-party") {
		t.Error("__pycache__ dep should be skipped")
	}
}

// PythonExtractor has its own Extract signature returning *architect.ExtractionResult
// rather than *architect.ProfileFragment, so it does not implement architect.Extractor.
