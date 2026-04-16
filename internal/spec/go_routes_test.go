package spec

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRoutes_Chi(t *testing.T) {
	endpoints := extractRoutesFromTestdata(t, "routes_chi.go")
	require.NotEmpty(t, endpoints, "should find chi routes")

	type pathMethod struct{ path, method string }
	seen := map[pathMethod]bool{}
	for _, e := range endpoints {
		seen[pathMethod{e.Path, e.Method}] = true
	}
	assert.True(t, seen[pathMethod{"/users", "GET"}])
	assert.True(t, seen[pathMethod{"/users", "POST"}])
	assert.True(t, seen[pathMethod{"/users/{id}", "PUT"}])
	assert.True(t, seen[pathMethod{"/users/{id}", "DELETE"}])

	// Nested Route() prefix composition: /admin/settings
	assert.True(t, seen[pathMethod{"/admin/settings", "GET"}],
		"chi Route() should compose /admin/settings GET")
	assert.True(t, seen[pathMethod{"/admin/settings", "POST"}],
		"chi Route() should compose /admin/settings POST")
}

func TestExtractRoutes_Gin(t *testing.T) {
	endpoints := extractRoutesFromTestdata(t, "routes_gin.go")
	require.NotEmpty(t, endpoints, "should find gin routes")

	byHandler := map[string]Endpoint{}
	for _, e := range endpoints {
		byHandler[e.Handler] = e
	}
	// Group("/api/v1") prefix composition
	assert.Equal(t, "/api/v1/health", byHandler["healthCheck"].Path,
		"gin Group() should compose /api/v1/health")
	assert.Equal(t, "/api/v1/deploy", byHandler["deployHandler"].Path,
		"gin Group() should compose /api/v1/deploy")
	// Top-level routes unchanged
	assert.Equal(t, "/ping", byHandler["pingHandler"].Path)
}

func TestExtractRoutes_Echo(t *testing.T) {
	endpoints := extractRoutesFromTestdata(t, "routes_echo.go")
	require.NotEmpty(t, endpoints, "should find echo routes")

	byHandler := map[string]Endpoint{}
	for _, e := range endpoints {
		byHandler[e.Handler] = e
	}
	// Group("/api") prefix composition
	assert.Equal(t, "/api/status", byHandler["statusHandler"].Path,
		"echo Group() should compose /api/status")
	assert.Equal(t, "/api/webhook", byHandler["webhookHandler"].Path,
		"echo Group() should compose /api/webhook")
	// Top-level routes unchanged
	assert.Equal(t, "/", byHandler["homeHandler"].Path)
}

func TestExtractRoutes_Stdlib(t *testing.T) {
	endpoints := extractRoutesFromTestdata(t, "routes_stdlib.go")
	require.NotEmpty(t, endpoints, "should find stdlib routes")

	paths := map[string]bool{}
	for _, e := range endpoints {
		paths[e.Path] = true
	}
	assert.True(t, paths["/home"], "should find /home")
	assert.True(t, paths["/legacy"], "should find /legacy")
}

func TestExtractRoutes_Gorilla(t *testing.T) {
	endpoints := extractRoutesFromTestdata(t, "routes_gorilla.go")
	require.NotEmpty(t, endpoints, "should find gorilla/mux routes")

	byHandler := map[string]Endpoint{}
	for _, e := range endpoints {
		byHandler[e.Handler] = e
	}
	// PathPrefix("/admin").Subrouter() prefix composition
	assert.Equal(t, "/admin/users", byHandler["adminListUsers"].Path,
		"gorilla Subrouter() should compose /admin/users")
	assert.Equal(t, "GET", byHandler["adminListUsers"].Method)
	assert.Equal(t, "/admin/users", byHandler["adminCreateUser"].Path,
		"gorilla Subrouter() should compose /admin/users for POST")
	assert.Equal(t, "POST", byHandler["adminCreateUser"].Method)
	// Top-level routes unchanged
	assert.Equal(t, "/products", byHandler["listProducts"].Path)
}

func TestExtractRoutes_EmptyFile(t *testing.T) {
	endpoints, err := ExtractGoRoutes(filepath.Join("testdata", "nonexistent.go"))
	assert.NoError(t, err)
	assert.Empty(t, endpoints)
}

func TestEndpointHasSourceInfo(t *testing.T) {
	endpoints := extractRoutesFromTestdata(t, "routes_chi.go")
	require.NotEmpty(t, endpoints)
	for _, e := range endpoints {
		assert.NotEmpty(t, e.SourceFile, "endpoint should have source file")
		assert.Greater(t, e.SourceLine, 0, "endpoint should have source line")
	}
}

// extractRoutesFromTestdata is a test helper that extracts routes from a testdata file.
func extractRoutesFromTestdata(t *testing.T, filename string) []Endpoint {
	t.Helper()
	path := filepath.Join("testdata", filename)
	endpoints, err := ExtractGoRoutes(path)
	require.NoError(t, err)
	return endpoints
}
