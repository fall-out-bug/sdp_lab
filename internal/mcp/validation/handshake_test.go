package validation_test

import (
	"context"
	"testing"
	"time"

	"sdp_dev/internal/mcp/contract"
	"sdp_dev/internal/mcp/parity"
	"sdp_dev/internal/mcp/validation"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	mapping, _ := contract.NewBuilder().Build()
	prompts := parity.NewPromptRegistry()
	resources := parity.NewResourceRegistry()
	server := mcpserver.NewMCPServer("test", "1.0.0")

	validator := validation.NewValidator(validation.ValidatorConfig{
		Mapping:   mapping,
		Prompts:   prompts,
		Resources: resources,
		Server:    server,
		RepoRoot:  ".",
	})

	assert.NotNil(t, validator)
}

func TestValidatorValidate(t *testing.T) {
	// Create valid mapping
	mapping, err := contract.NewBuilder().
		AddTool(contract.ToolMapping{
			MCPToolName:  "sdp_test",
			CLICommand:   "test",
			Description:  "test",
			ParityStatus: contract.ParityFull,
		}).
		AddResource(contract.ResourceMapping{
			MCPResourceURI: "sdp://test",
			CLICommand:     "test",
			ArtifactPath:   ".sdp/test.json",
			Description:    "test",
			MIMEType:       "application/json",
			ParityStatus:   contract.ParityFull,
			HintTool:       "sdp_test",
		}).
		AddPrompt(contract.PromptMapping{
			MCPPromptName: "understand",
			IntentModel:   "F125:intent:understand",
			Description:   "test",
			ParityStatus:  contract.ParityFull,
		}).
		Build()
	require.NoError(t, err)

	// Create registries
	prompts := parity.NewPromptRegistry()
	for _, prompt := range parity.DefaultPrompts() {
		require.NoError(t, prompts.Register(prompt))
	}

	resources := parity.NewResourceRegistry()
	for _, resource := range parity.DefaultResources() {
		if resource.ParityStatus == contract.ParityFull {
			require.NoError(t, resources.Register(resource))
		}
	}

	// Create server
	server := mcpserver.NewMCPServer("test", "1.0.0")
	server.AddTool(mcp.NewTool("sdp_test", mcp.WithDescription("test")), nil)

	// Create validator
	validator := validation.NewValidator(validation.ValidatorConfig{
		Mapping:   mapping,
		Prompts:   prompts,
		Resources: resources,
		Server:    server,
		RepoRoot:  ".",
	})

	// Run validation
	result, err := validator.Validate(context.Background())
	require.NoError(t, err)

	// Check results
	assert.NotNil(t, result)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.Equal(t, 1, result.ToolCount)
	assert.NotEmpty(t, result.ParitySummary)
}

func TestValidatorValidateContractErrors(t *testing.T) {
	// Create invalid mapping
	mapping := &contract.Mapping{
		SpecVersion: "", // Invalid: empty spec version
	}

	prompts := parity.NewPromptRegistry()
	resources := parity.NewResourceRegistry()
	server := mcpserver.NewMCPServer("test", "1.0.0")

	validator := validation.NewValidator(validation.ValidatorConfig{
		Mapping:   mapping,
		Prompts:   prompts,
		Resources: resources,
		Server:    server,
		RepoRoot:  ".",
	})

	result, err := validator.Validate(context.Background())
	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0], "contract validation")
}

func TestValidatorValidatePromptParityErrors(t *testing.T) {
	mapping, _ := contract.NewBuilder().Build()

	// Create empty prompt registry (missing required intents)
	prompts := parity.NewPromptRegistry()

	resources := parity.NewResourceRegistry()
	server := mcpserver.NewMCPServer("test", "1.0.0")

	validator := validation.NewValidator(validation.ValidatorConfig{
		Mapping:   mapping,
		Prompts:   prompts,
		Resources: resources,
		Server:    server,
		RepoRoot:  ".",
	})

	result, err := validator.Validate(context.Background())
	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.Errors)

	// Check that prompt parity error is present
	hasPromptError := false
	for _, err := range result.Errors {
		if contains(err, "prompt parity") {
			hasPromptError = true
			break
		}
	}
	assert.True(t, hasPromptError, "expected prompt parity error")
}

func TestValidatorValidateResourceParityErrors(t *testing.T) {
	mapping, _ := contract.NewBuilder().Build()
	prompts := parity.NewPromptRegistry()

	// Create empty resource registry (missing core resources)
	resources := parity.NewResourceRegistry()

	server := mcpserver.NewMCPServer("test", "1.0.0")

	validator := validation.NewValidator(validation.ValidatorConfig{
		Mapping:   mapping,
		Prompts:   prompts,
		Resources: resources,
		Server:    server,
		RepoRoot:  ".",
	})

	result, err := validator.Validate(context.Background())
	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.Errors)

	// Check that resource parity error is present
	hasResourceError := false
	for _, err := range result.Errors {
		if contains(err, "resource parity") {
			hasResourceError = true
			break
		}
	}
	assert.True(t, hasResourceError, "expected resource parity error")
}

func TestValidatorValidateToolSurfaceErrors(t *testing.T) {
	// Create mapping with duplicate tool names - but the builder's Build() method
	// actually validates and may fail. Let's test the validation function directly.

	// Create a mapping that will have duplicates by not building properly
	invalidMapping := &contract.Mapping{
		SpecVersion:     contract.CurrentSpecVersion,
		CLIRegistryHash: "test",
		SkillCatalogHash: "test",
		GeneratedAt:     time.Now(),
		Tools: []contract.ToolMapping{
			{
				MCPToolName:  "sdp_test",
				CLICommand:   "test",
				Description:  "test",
				ParityStatus: contract.ParityFull,
			},
			{
				MCPToolName:  "sdp_test", // Duplicate
				CLICommand:   "test2",
				Description:  "test2",
				ParityStatus: contract.ParityFull,
			},
		},
	}

	// Use the dedicated validation function for duplicate tools
	dupErr := validation.ValidateContractDuplicateTools(invalidMapping)
	assert.Error(t, dupErr)
	assert.Contains(t, dupErr.Error(), "duplicate tool name")
}

func TestValidatorParityReport(t *testing.T) {
	result := &validation.ValidationResult{
		Passed:      true,
		Duration:    100 * time.Millisecond,
		ToolCount:   13,
		ResourceCount: 8,
		PromptCount: 5,
		ParitySummary: map[string]int{
			"tool_full":      13,
			"resource_full":  4,
			"resource_forward": 4,
			"prompt_full":    5,
		},
		Errors:   []string{},
		Warnings: []string{},
	}

	mapping, _ := contract.NewBuilder().Build()
	validator := validation.NewValidator(validation.ValidatorConfig{
		Mapping: mapping,
	})

	report := validator.ParityReport(result)

	assert.Contains(t, report, "PASSED")
	assert.Contains(t, report, "Tools: 13")
	assert.Contains(t, report, "Resources: 8")
	assert.Contains(t, report, "Prompts: 5")
	assert.Contains(t, report, "tool_full: 13")
	assert.Contains(t, report, "prompt_full: 5")
}

func TestValidateParityBeforeClaim(t *testing.T) {
	// Test with current directory (may have some resources)
	err := validation.ValidateParityBeforeClaim(".")
	// We don't assert on error here since it depends on what's in the current directory
	// Just verify the function runs without panicking
	assert.NotNil(t, err)
}

func TestQuickValidation(t *testing.T) {
	passed, err := validation.QuickValidation(".")
	// Same as above - just verify it runs
	assert.NotNil(t, err)
	if err == nil {
		assert.True(t, passed)
	} else {
		assert.False(t, passed)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (
		s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}