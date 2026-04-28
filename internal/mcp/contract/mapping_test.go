package contract_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/mcp/contract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	builder := contract.NewBuilder()
	assert.NotNil(t, builder)
}

func TestBuilderWithRegistrySnapshot(t *testing.T) {
	snapshot := &contract.RegistrySnapshot{
		Commands: []contract.CommandEntry{
			{
				Name: "scout",
				Flags: []contract.FlagEntry{
					{Name: "format", Type: "string"},
				},
				Source: "cmd/sdp/cmd_scout.go",
			},
		},
	}

	builder := contract.NewBuilder().WithRegistrySnapshot(snapshot)
	mapping, err := builder.Build()
	require.NoError(t, err)
	assert.NotEmpty(t, mapping.CLIRegistryHash)
}

func TestBuilderWithSkillSnapshot(t *testing.T) {
	snapshot := &contract.SkillCatalogSnapshot{
		Intents: []contract.IntentEntry{
			{
				Name:        "understand",
				Description: "Understand codebase",
				Arguments: []contract.ArgumentMapping{
					{Name: "depth", Description: "Depth of analysis"},
				},
			},
		},
	}

	builder := contract.NewBuilder().WithSkillSnapshot(snapshot)
	mapping, err := builder.Build()
	require.NoError(t, err)
	assert.NotEmpty(t, mapping.SkillCatalogHash)
}

func TestBuilderAddTool(t *testing.T) {
	tool := contract.ToolMapping{
		MCPToolName:  "sdp_scout",
		CLICommand:   "scout",
		Description:  "Quick codebase reconnaissance",
		ParityStatus: contract.ParityFull,
		Parameters: []contract.ParameterMapping{
			{
				MCPParamName: "path",
				CLIFlag:      "--path",
				Type:         "string",
			},
		},
	}

	mapping, err := contract.NewBuilder().
		AddTool(tool).
		Build()

	require.NoError(t, err)
	assert.Len(t, mapping.Tools, 1)
	assert.Equal(t, "sdp_scout", mapping.Tools[0].MCPToolName)
}

func TestBuilderAddResource(t *testing.T) {
	resource := contract.ResourceMapping{
		MCPResourceURI: "sdp://scout",
		CLICommand:     "scout",
		ArtifactPath:   ".sdp/scout.json",
		Description:    "Scout report",
		MIMEType:       "application/json",
		ParityStatus:   contract.ParityFull,
		HintTool:       "sdp_scout",
	}

	mapping, err := contract.NewBuilder().
		AddResource(resource).
		Build()

	require.NoError(t, err)
	assert.Len(t, mapping.Resources, 1)
	assert.Equal(t, "sdp://scout", mapping.Resources[0].MCPResourceURI)
}

func TestBuilderAddPrompt(t *testing.T) {
	prompt := contract.PromptMapping{
		MCPPromptName: "understand",
		IntentModel:   "F125:intent:understand",
		Description:   "Understand codebase context",
		Arguments: []contract.ArgumentMapping{
			{Name: "depth", Description: "Analysis depth"},
		},
		ResourcesUsed: []string{"sdp://scout", "sdp://architect"},
		ParityStatus:  contract.ParityFull,
	}

	mapping, err := contract.NewBuilder().
		AddPrompt(prompt).
		Build()

	require.NoError(t, err)
	assert.Len(t, mapping.Prompts, 1)
	assert.Equal(t, "understand", mapping.Prompts[0].MCPPromptName)
}

func TestBuilderBuildRequiresValidFields(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*contract.Builder)
		wantError string
	}{
		{
			name:  "empty tool name",
			setup: func(b *contract.Builder) {
				b.AddTool(contract.ToolMapping{
					MCPToolName:  "",
					CLICommand:   "scout",
					Description:  "test",
					ParityStatus: contract.ParityFull,
				})
			},
			wantError: "tool mcp_tool_name is required",
		},
		{
			name:  "empty cli command",
			setup: func(b *contract.Builder) {
				b.AddTool(contract.ToolMapping{
					MCPToolName:  "sdp_test",
					CLICommand:   "",
					Description:  "test",
					ParityStatus: contract.ParityFull,
				})
			},
			wantError: "cli_command is required",
		},
		{
			name:  "empty parity status",
			setup: func(b *contract.Builder) {
				b.AddTool(contract.ToolMapping{
					MCPToolName: "sdp_test",
					CLICommand:  "test",
					Description: "test",
				})
			},
			wantError: "parity_status is required",
		},
		{
			name:  "duplicate tool name",
			setup: func(b *contract.Builder) {
				tool := contract.ToolMapping{
					MCPToolName:  "sdp_test",
					CLICommand:   "test",
					Description:  "test",
					ParityStatus: contract.ParityFull,
				}
				b.AddTool(tool).AddTool(tool)
			},
			wantError: "duplicate tool name",
		},
		{
			name:  "resource without URI",
			setup: func(b *contract.Builder) {
				b.AddResource(contract.ResourceMapping{
					MCPResourceURI: "",
					CLICommand:     "scout",
					ArtifactPath:   ".sdp/scout.json",
					Description:    "test",
					MIMEType:       "application/json",
					ParityStatus:   contract.ParityFull,
				})
			},
			wantError: "resource mcp_resource_uri is required",
		},
		{
			name:  "prompt without name",
			setup: func(b *contract.Builder) {
				b.AddPrompt(contract.PromptMapping{
					MCPPromptName: "",
					IntentModel:   "F125:intent:test",
					Description:   "test",
					ParityStatus:  contract.ParityFull,
				})
			},
			wantError: "prompt mcp_prompt_name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := contract.NewBuilder()
			tt.setup(builder)
			_, err := builder.Build()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestMappingSaveAndLoad(t *testing.T) {
	original, err := contract.NewBuilder().
		AddTool(contract.ToolMapping{
			MCPToolName:  "sdp_scout",
			CLICommand:   "scout",
			Description:  "Quick reconnaissance",
			ParityStatus: contract.ParityFull,
		}).
		AddResource(contract.ResourceMapping{
			MCPResourceURI: "sdp://scout",
			CLICommand:     "scout",
			ArtifactPath:   ".sdp/scout.json",
			Description:    "Scout report",
			MIMEType:       "application/json",
			ParityStatus:   contract.ParityFull,
		}).
		AddPrompt(contract.PromptMapping{
			MCPPromptName: "understand",
			IntentModel:   "F125:intent:understand",
			Description:   "Understand codebase",
			ParityStatus:  contract.ParityFull,
		}).
		Build()

	require.NoError(t, err)

	// Save to temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "mapping.json")
	require.NoError(t, original.SaveToFile(path))

	// Load from file
	loaded, err := contract.LoadFromFile(path)
	require.NoError(t, err)

	// Verify content
	assert.Equal(t, original.SpecVersion, loaded.SpecVersion)
	assert.Equal(t, len(original.Tools), len(loaded.Tools))
	assert.Equal(t, len(original.Resources), len(loaded.Resources))
	assert.Equal(t, len(original.Prompts), len(loaded.Prompts))
}

func TestMappingGetToolByMCPName(t *testing.T) {
	mapping, err := contract.NewBuilder().
		AddTool(contract.ToolMapping{
			MCPToolName:  "sdp_scout",
			CLICommand:   "scout",
			Description:  "Quick reconnaissance",
			ParityStatus: contract.ParityFull,
		}).
		Build()

	require.NoError(t, err)

	tool, found := mapping.GetToolByMCPName("sdp_scout")
	assert.True(t, found)
	assert.Equal(t, "scout", tool.CLICommand)

	_, found = mapping.GetToolByMCPName("nonexistent")
	assert.False(t, found)
}

func TestMappingGetResourceByURI(t *testing.T) {
	mapping, err := contract.NewBuilder().
		AddResource(contract.ResourceMapping{
			MCPResourceURI: "sdp://scout",
			CLICommand:     "scout",
			ArtifactPath:   ".sdp/scout.json",
			Description:    "Scout report",
			MIMEType:       "application/json",
			ParityStatus:   contract.ParityFull,
		}).
		Build()

	require.NoError(t, err)

	resource, found := mapping.GetResourceByURI("sdp://scout")
	assert.True(t, found)
	assert.Equal(t, "scout", resource.CLICommand)

	_, found = mapping.GetResourceByURI("sdp://nonexistent")
	assert.False(t, found)
}

func TestMappingGetPromptByMCPName(t *testing.T) {
	mapping, err := contract.NewBuilder().
		AddPrompt(contract.PromptMapping{
			MCPPromptName: "understand",
			IntentModel:   "F125:intent:understand",
			Description:   "Understand codebase",
			ParityStatus:  contract.ParityFull,
		}).
		Build()

	require.NoError(t, err)

	prompt, found := mapping.GetPromptByMCPName("understand")
	assert.True(t, found)
	assert.Equal(t, "F125:intent:understand", prompt.IntentModel)

	_, found = mapping.GetPromptByMCPName("nonexistent")
	assert.False(t, found)
}

func TestMappingGetToolsByParityStatus(t *testing.T) {
	mapping, err := contract.NewBuilder().
		AddTool(contract.ToolMapping{
			MCPToolName:  "sdp_scout",
			CLICommand:   "scout",
			Description:  "Quick reconnaissance",
			ParityStatus: contract.ParityFull,
		}).
		AddTool(contract.ToolMapping{
			MCPToolName:  "sdp_future",
			CLICommand:   "future",
			Description:  "Future tool",
			ParityStatus: contract.ParityForward,
		}).
		Build()

	require.NoError(t, err)

	fullTools := mapping.GetToolsByParityStatus(contract.ParityFull)
	assert.Len(t, fullTools, 1)
	assert.Equal(t, "sdp_scout", fullTools[0].MCPToolName)

	forwardTools := mapping.GetToolsByParityStatus(contract.ParityForward)
	assert.Len(t, forwardTools, 1)
	assert.Equal(t, "sdp_future", forwardTools[0].MCPToolName)
}

func TestMappingValidateParity(t *testing.T) {
	mapping, err := contract.NewBuilder().
		WithRegistrySnapshot(&contract.RegistrySnapshot{
			Commands: []contract.CommandEntry{
				{Name: "scout"},
			},
		}).
		WithSkillSnapshot(&contract.SkillCatalogSnapshot{
			Intents: []contract.IntentEntry{
				{Name: "understand"},
			},
		}).
		Build()

	require.NoError(t, err)

	// Parity should match
	err = mapping.ValidateParity(mapping.CLIRegistryHash, mapping.SkillCatalogHash)
	assert.NoError(t, err)

	// Parity should fail with mismatched hashes
	err = mapping.ValidateParity("wrong", mapping.SkillCatalogHash)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CLI registry hash mismatch")
}

func TestMappingIsFullParity(t *testing.T) {
	t.Run("all full parity", func(t *testing.T) {
		mapping, err := contract.NewBuilder().
			AddTool(contract.ToolMapping{
				MCPToolName:  "sdp_scout",
				CLICommand:   "scout",
				Description:  "test",
				ParityStatus: contract.ParityFull,
			}).
			AddResource(contract.ResourceMapping{
				MCPResourceURI: "sdp://scout",
				CLICommand:     "scout",
				ArtifactPath:   ".sdp/scout.json",
				Description:    "test",
				MIMEType:       "application/json",
				ParityStatus:   contract.ParityFull,
			}).
			AddPrompt(contract.PromptMapping{
				MCPPromptName: "understand",
				IntentModel:   "F125:intent:understand",
				Description:   "test",
				ParityStatus:  contract.ParityFull,
			}).
			Build()

		require.NoError(t, err)
		assert.True(t, mapping.IsFullParity())
	})

	t.Run("partial parity", func(t *testing.T) {
		mapping, err := contract.NewBuilder().
			AddTool(contract.ToolMapping{
				MCPToolName:  "sdp_scout",
				CLICommand:   "scout",
				Description:  "test",
				ParityStatus: contract.ParityFull,
			}).
			AddTool(contract.ToolMapping{
				MCPToolName:  "sdp_future",
				CLICommand:   "future",
				Description:  "test",
				ParityStatus: contract.ParityForward,
			}).
			Build()

		require.NoError(t, err)
		assert.False(t, mapping.IsFullParity())
	})
}

func TestMappingString(t *testing.T) {
	mapping, err := contract.NewBuilder().
		AddTool(contract.ToolMapping{
			MCPToolName:  "sdp_scout",
			CLICommand:   "scout",
			Description:  "test",
			ParityStatus: contract.ParityFull,
		}).
		Build()

	require.NoError(t, err)

	str := mapping.String()
	assert.Contains(t, str, "Mapping Contract")
	assert.Contains(t, str, "Tools: 1")
	assert.Contains(t, str, "tool_full: 1")
}

func TestLoadFromFileInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")
	require.NoError(t, os.WriteFile(path, []byte("invalid json"), 0o644))

	_, err := contract.LoadFromFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal mapping")
}

func TestLoadFromFileInvalidValidation(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"spec_version": "1.0.0"}`), 0o644))

	_, err := contract.LoadFromFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validate mapping")
}

func TestMappingGeneratedAtIsSet(t *testing.T) {
	before := time.Now().UTC()
	mapping, err := contract.NewBuilder().Build()
	after := time.Now().UTC()

	require.NoError(t, err)
	assert.True(t, mapping.GeneratedAt.After(before) || mapping.GeneratedAt.Equal(before))
	assert.True(t, mapping.GeneratedAt.Before(after) || mapping.GeneratedAt.Equal(after))
}