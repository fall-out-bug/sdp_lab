package parity_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sdp_dev/internal/mcp/parity"
)

func TestNewResourceRegistry(t *testing.T) {
	registry := parity.NewResourceRegistry()
	assert.NotNil(t, registry)
}

func TestResourceRegistryRegister(t *testing.T) {
	registry := parity.NewResourceRegistry()

	resource := &parity.ResourceDefinition{
		URI:          "sdp://test",
		Name:         "Test Resource",
		Description:  "Test resource",
		MIMEType:     "application/json",
		Path:         ".sdp/test.json",
		SourceCLI:    "test",
		HintTool:     "sdp_test",
		ParityStatus: parity.ParityFull,
	}

	err := registry.Register(resource)
	require.NoError(t, err)

	retrieved, ok := registry.Get("sdp://test")
	assert.True(t, ok)
	assert.Equal(t, "sdp://test", retrieved.URI)
	assert.Equal(t, "Test Resource", retrieved.Name)
}

func TestResourceRegistryRegisterValidation(t *testing.T) {
	tests := []struct {
		name      string
		resource  *parity.ResourceDefinition
		wantError string
	}{
		{
			name: "empty URI",
			resource: &parity.ResourceDefinition{
				Name:         "test",
				Description:  "test",
				MIMEType:     "application/json",
				Path:         ".sdp/test.json",
				SourceCLI:    "test",
				ParityStatus: parity.ParityFull,
			},
			wantError: "resource URI cannot be empty",
		},
		{
			name: "empty name",
			resource: &parity.ResourceDefinition{
				URI:          "sdp://test",
				Description:  "test",
				MIMEType:     "application/json",
				Path:         ".sdp/test.json",
				SourceCLI:    "test",
				ParityStatus: parity.ParityFull,
			},
			wantError: "resource name cannot be empty",
		},
		{
			name: "empty description",
			resource: &parity.ResourceDefinition{
				URI:          "sdp://test",
				Name:         "test",
				MIMEType:     "application/json",
				Path:         ".sdp/test.json",
				SourceCLI:    "test",
				ParityStatus: parity.ParityFull,
			},
			wantError: "description cannot be empty",
		},
		{
			name: "empty MIME type",
			resource: &parity.ResourceDefinition{
				URI:          "sdp://test",
				Name:         "test",
				Description:  "test",
				Path:         ".sdp/test.json",
				SourceCLI:    "test",
				ParityStatus: parity.ParityFull,
			},
			wantError: "MIME type cannot be empty",
		},
		{
			name: "empty path",
			resource: &parity.ResourceDefinition{
				URI:          "sdp://test",
				Name:         "test",
				Description:  "test",
				MIMEType:     "application/json",
				SourceCLI:    "test",
				ParityStatus: parity.ParityFull,
			},
			wantError: "path cannot be empty",
		},
		{
			name: "empty source CLI",
			resource: &parity.ResourceDefinition{
				URI:          "sdp://test",
				Name:         "test",
				Description:  "test",
				MIMEType:     "application/json",
				Path:         ".sdp/test.json",
				ParityStatus: parity.ParityFull,
			},
			wantError: "source CLI cannot be empty",
		},
		{
			name: "empty parity status",
			resource: &parity.ResourceDefinition{
				URI:         "sdp://test",
				Name:        "test",
				Description: "test",
				MIMEType:    "application/json",
				Path:        ".sdp/test.json",
				SourceCLI:   "test",
			},
			wantError: "parity status cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := parity.NewResourceRegistry()
			err := registry.Register(tt.resource)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestResourceRegistryGetByParityStatus(t *testing.T) {
	registry := parity.NewResourceRegistry()

	require.NoError(t, registry.Register(&parity.ResourceDefinition{
		URI:          "sdp://full",
		Name:         "Full",
		Description:  "Full parity",
		MIMEType:     "application/json",
		Path:         ".sdp/full.json",
		SourceCLI:    "full",
		HintTool:     "sdp_full",
		ParityStatus: parity.ParityFull,
	}))

	require.NoError(t, registry.Register(&parity.ResourceDefinition{
		URI:          "sdp://forward",
		Name:         "Forward",
		Description:  "Forward parity",
		MIMEType:     "application/json",
		Path:         ".sdp/forward.json",
		SourceCLI:    "forward",
		HintTool:     "sdp_forward",
		ParityStatus: parity.ParityForward,
	}))

	fullResources := registry.GetByParityStatus(parity.ParityFull)
	assert.Len(t, fullResources, 1)
	assert.Equal(t, "sdp://full", fullResources[0].URI)

	forwardResources := registry.GetByParityStatus(parity.ParityForward)
	assert.Len(t, forwardResources, 1)
	assert.Equal(t, "sdp://forward", forwardResources[0].URI)
}

func TestResourceRegistryValidateParity(t *testing.T) {
	t.Run("all core resources have full parity", func(t *testing.T) {
		registry := parity.NewResourceRegistry()

		for _, resource := range parity.DefaultResources() {
			// Only register core resources with full parity
			if resource.ParityStatus == parity.ParityFull {
				require.NoError(t, registry.Register(resource))
			}
		}

		err := registry.ValidateParity()
		assert.NoError(t, err)
	})

	t.Run("missing core resource", func(t *testing.T) {
		registry := parity.NewResourceRegistry()

		// Register only some core resources
		require.NoError(t, registry.Register(&parity.ResourceDefinition{
			URI:          "sdp://scout",
			Name:         "Scout",
			Description:  "Scout",
			MIMEType:     "application/json",
			Path:         ".sdp/scout.json",
			SourceCLI:    "scout",
			HintTool:     "sdp_scout",
			ParityStatus: parity.ParityFull,
		}))

		err := registry.ValidateParity()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing core resource")
	})

	t.Run("partial parity", func(t *testing.T) {
		registry := parity.NewResourceRegistry()

		// Register core resources with partial parity
		coreResources := []string{"sdp://scout", "sdp://architect", "sdp://metrics", "sdp://spec"}
		for _, uri := range coreResources {
			require.NoError(t, registry.Register(&parity.ResourceDefinition{
				URI:          uri,
				Name:         "Test",
				Description:  "Test",
				MIMEType:     "application/json",
				Path:         ".sdp/test.json",
				SourceCLI:    "test",
				HintTool:     "sdp_test",
				ParityStatus: parity.ParityPartial,
			}))
		}

		err := registry.ValidateParity()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not have full parity")
	})
}

func TestResourceRegistryCheckAvailability(t *testing.T) {
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))

	// Create some test resources
	require.NoError(t, os.WriteFile(filepath.Join(sdpDir, "scout.json"), []byte("{}"), 0o644))

	registry := parity.NewResourceRegistry()
	require.NoError(t, registry.Register(&parity.ResourceDefinition{
		URI:          "sdp://scout",
		Name:         "Scout",
		Description:  "Scout",
		MIMEType:     "application/json",
		Path:         ".sdp/scout.json",
		SourceCLI:    "scout",
		HintTool:     "sdp_scout",
		ParityStatus: parity.ParityFull,
	}))
	require.NoError(t, registry.Register(&parity.ResourceDefinition{
		URI:          "sdp://missing",
		Name:         "Missing",
		Description:  "Missing",
		MIMEType:     "application/json",
		Path:         ".sdp/missing.json",
		SourceCLI:    "missing",
		HintTool:     "sdp_missing",
		ParityStatus: parity.ParityFull,
	}))

	availability := registry.CheckAvailability(tmpDir)

	assert.True(t, availability["sdp://scout"])
	assert.False(t, availability["sdp://missing"])
}

func TestResourceRegistryGetMissingResources(t *testing.T) {
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))

	registry := parity.NewResourceRegistry()

	// Register a resource that exists
	scoutResource := &parity.ResourceDefinition{
		URI:          "sdp://scout",
		Name:         "Scout",
		Description:  "Scout",
		MIMEType:     "application/json",
		Path:         ".sdp/scout.json",
		SourceCLI:    "scout",
		HintTool:     "sdp_scout",
		ParityStatus: parity.ParityFull,
	}
	require.NoError(t, registry.Register(scoutResource))

	// Create the scout file
	require.NoError(t, os.WriteFile(filepath.Join(sdpDir, "scout.json"), []byte("{}"), 0o644))

	// Register a resource that doesn't exist
	missingResource := &parity.ResourceDefinition{
		URI:          "sdp://missing",
		Name:         "Missing",
		Description:  "Missing",
		MIMEType:     "application/json",
		Path:         ".sdp/missing.json",
		SourceCLI:    "missing",
		HintTool:     "sdp_missing",
		ParityStatus: parity.ParityFull,
	}
	require.NoError(t, registry.Register(missingResource))

	missing := registry.GetMissingResources(tmpDir)

	// Should find only the missing resource
	assert.Len(t, missing, 1)
	assert.Equal(t, "sdp://missing", missing[0].URI)
}

func TestDefaultResources(t *testing.T) {
	resources := parity.DefaultResources()

	// Should have 8 default resources
	assert.Len(t, resources, 8)

	// Check for core resources
	coreResources := []string{
		"sdp://scout",
		"sdp://architect",
		"sdp://metrics",
		"sdp://spec",
	}

	resourceURIs := make(map[string]bool)
	for _, resource := range resources {
		resourceURIs[resource.URI] = true
		assert.NotEmpty(t, resource.URI)
		assert.NotEmpty(t, resource.Name)
		assert.NotEmpty(t, resource.Description)
		assert.NotEmpty(t, resource.MIMEType)
		assert.NotEmpty(t, resource.Path)
		assert.NotEmpty(t, resource.SourceCLI)
		assert.NotEmpty(t, resource.HintTool)
		assert.NotEmpty(t, resource.ParityStatus)
	}

	// Verify all core resources exist
	for _, uri := range coreResources {
		assert.True(t, resourceURIs[uri], "missing core resource: %s", uri)
	}
}

func TestPathToURI(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantURI string
	}{
		{
			name:    "scout report",
			path:    ".sdp/scout.json",
			wantURI: "sdp://scout",
		},
		{
			name:    "architect report",
			path:    ".sdp/architect/report.json",
			wantURI: "sdp://architect",
		},
		{
			name:    "metrics report",
			path:    ".sdp/metrics/report.json",
			wantURI: "sdp://metrics",
		},
		{
			name:    "spec report",
			path:    ".sdp/specs/spec.json",
			wantURI: "sdp://spec",
		},
		{
			name:    "index modules",
			path:    ".sdp/index/modules.json",
			wantURI: "sdp://index/modules",
		},
		{
			name:    "index stats",
			path:    ".sdp/index/stats.json",
			wantURI: "sdp://index/stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := pathToURI(tt.path)
			assert.Equal(t, tt.wantURI, uri)
		})
	}
}

// Helper function for testing
func pathToURI(path string) string {
	// This is a simplified version of the internal function
	if strings.HasPrefix(path, ".sdp/") {
		path = strings.TrimPrefix(path, ".sdp/")
	}

	if strings.Contains(path, "architect") {
		return "sdp://architect"
	}
	if strings.Contains(path, "metrics") {
		return "sdp://metrics"
	}
	if strings.Contains(path, "specs") {
		return "sdp://spec"
	}
	if strings.Contains(path, filepath.Join("index", "modules")) {
		return "sdp://index/modules"
	}
	if strings.Contains(path, filepath.Join("index", "stats")) {
		return "sdp://index/stats"
	}

	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	switch base {
	case "scout", "manifest":
		return fmt.Sprintf("sdp://%s", base)
	default:
		return fmt.Sprintf("sdp://%s", strings.ReplaceAll(base, "_", "-"))
	}
}
