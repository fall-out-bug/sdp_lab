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

	// chi uses .Get/.Post/.Put/.Delete calls
	methods := map[string]bool{}
	for _, e := range endpoints {
		methods[e.Method] = true
	}
	assert.True(t, methods["GET"], "should find GET routes")
	assert.True(t, methods["POST"], "should find POST routes")
	assert.True(t, methods["PUT"], "should find PUT routes")
	assert.True(t, methods["DELETE"], "should find DELETE routes")

	// Check specific path+method combinations exist
	type pathMethod struct{ path, method string }
	seen := map[pathMethod]bool{}
	for _, e := range endpoints {
		seen[pathMethod{e.Path, e.Method}] = true
	}
	assert.True(t, seen[pathMethod{"/users", "GET"}])
	assert.True(t, seen[pathMethod{"/users", "POST"}])
	assert.True(t, seen[pathMethod{"/users/{id}", "PUT"}])
}

func TestExtractRoutes_Gin(t *testing.T) {
	endpoints := extractRoutesFromTestdata(t, "routes_gin.go")
	require.NotEmpty(t, endpoints, "should find gin routes")

	methods := map[string]bool{}
	for _, e := range endpoints {
		methods[e.Method] = true
	}
	assert.True(t, methods["GET"], "should find GET routes")
	assert.True(t, methods["POST"], "should find POST routes")
	assert.True(t, methods["PUT"], "should find PUT routes")
	assert.True(t, methods["DELETE"], "should find DELETE routes")
}

func TestExtractRoutes_Echo(t *testing.T) {
	endpoints := extractRoutesFromTestdata(t, "routes_echo.go")
	require.NotEmpty(t, endpoints, "should find echo routes")

	methods := map[string]bool{}
	for _, e := range endpoints {
		methods[e.Method] = true
	}
	assert.True(t, methods["GET"], "should find GET routes")
	assert.True(t, methods["POST"], "should find POST routes")
}

func TestExtractRoutes_Stdlib(t *testing.T) {
	endpoints := extractRoutesFromTestdata(t, "routes_stdlib.go")
	require.NotEmpty(t, endpoints, "should find stdlib routes")

	// stdlib uses HandleFunc/Handle, methods default to "" or we detect the pattern
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

	methods := map[string]bool{}
	for _, e := range endpoints {
		methods[e.Method] = true
	}
	assert.True(t, methods["GET"], "should find GET routes")
	assert.True(t, methods["POST"], "should find POST routes")
	assert.True(t, methods["PUT"], "should find PUT routes")
	assert.True(t, methods["DELETE"], "should find DELETE routes")
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
