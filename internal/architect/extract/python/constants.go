package python

var skipDirs = map[string]bool{
	"venv":          true,
	".venv":         true,
	"env":           true,
	"__pycache__":   true,
	"node_modules":  true,
	".git":          true,
	".tox":          true,
	".mypy_cache":   true,
	".pytest_cache": true,
	".eggs":         true,
	"*.egg-info":    true,
	"dist":          true,
	"build":         true,
	".nox":          true,
}

var testDirNames = map[string]bool{
	"tests":   true,
	"test":    true,
	"spec":    true,
	"specs":   true,
	"_tests":  true,
	"_test":   true,
	"testing": true,
}
