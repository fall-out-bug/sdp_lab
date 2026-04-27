package parity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fall-out-bug/sdp_lab/internal/mcp/parity"
)

func TestNewPromptRegistry(t *testing.T) {
	registry := parity.NewPromptRegistry()
	assert.NotNil(t, registry)
}

func TestPromptRegistryRegister(t *testing.T) {
	registry := parity.NewPromptRegistry()

	prompt := &parity.PromptDefinition{
		Name:         "test",
		IntentModel:  parity.IntentUnderstand,
		Description:  "Test prompt",
		ParityStatus: parity.ParityFull,
	}

	err := registry.Register(prompt)
	require.NoError(t, err)

	retrieved, ok := registry.Get("test")
	assert.True(t, ok)
	assert.Equal(t, "test", retrieved.Name)
	assert.Equal(t, parity.IntentUnderstand, retrieved.IntentModel)
}

func TestPromptRegistryRegisterValidation(t *testing.T) {
	tests := []struct {
		name      string
		prompt    *parity.PromptDefinition
		wantError string
	}{
		{
			name: "empty name",
			prompt: &parity.PromptDefinition{
				IntentModel: parity.IntentUnderstand,
				Description: "test",
			},
			wantError: "prompt name cannot be empty",
		},
		{
			name: "empty intent model",
			prompt: &parity.PromptDefinition{
				Name:        "test",
				Description: "test",
			},
			wantError: "intent model cannot be empty",
		},
		{
			name: "empty description",
			prompt: &parity.PromptDefinition{
				Name:        "test",
				IntentModel: parity.IntentUnderstand,
			},
			wantError: "description cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := parity.NewPromptRegistry()
			err := registry.Register(tt.prompt)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestPromptRegistryGetByIntentModel(t *testing.T) {
	registry := parity.NewPromptRegistry()

	require.NoError(t, registry.Register(&parity.PromptDefinition{
		Name:         "understand",
		IntentModel:  parity.IntentUnderstand,
		Description:  "Understand codebase",
		ParityStatus: parity.ParityFull,
	}))

	require.NoError(t, registry.Register(&parity.PromptDefinition{
		Name:         "build",
		IntentModel:  parity.IntentBuild,
		Description:  "Build feature",
		ParityStatus: parity.ParityFull,
	}))

	understandPrompts := registry.GetByIntentModel(parity.IntentUnderstand)
	assert.Len(t, understandPrompts, 1)
	assert.Equal(t, "understand", understandPrompts[0].Name)

	buildPrompts := registry.GetByIntentModel(parity.IntentBuild)
	assert.Len(t, buildPrompts, 1)
	assert.Equal(t, "build", buildPrompts[0].Name)
}

func TestPromptRegistryValidateParity(t *testing.T) {
	t.Run("all intents have full parity", func(t *testing.T) {
		registry := parity.NewPromptRegistry()

		for _, prompt := range parity.DefaultPrompts() {
			require.NoError(t, registry.Register(prompt))
		}

		err := registry.ValidateParity()
		assert.NoError(t, err)
	})

	t.Run("missing intent", func(t *testing.T) {
		registry := parity.NewPromptRegistry()

		require.NoError(t, registry.Register(&parity.PromptDefinition{
			Name:         "understand",
			IntentModel:  parity.IntentUnderstand,
			Description:  "test",
			ParityStatus: parity.ParityFull,
		}))

		err := registry.ValidateParity()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing prompt for intent")
	})

	t.Run("partial parity", func(t *testing.T) {
		registry := parity.NewPromptRegistry()

		for _, prompt := range parity.DefaultPrompts() {
			testPrompt := *prompt
			testPrompt.ParityStatus = parity.ParityPartial
			require.NoError(t, registry.Register(&testPrompt))
		}

		err := registry.ValidateParity()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not have full parity")
	})
}

func TestDefaultPrompts(t *testing.T) {
	prompts := parity.DefaultPrompts()

	assert.Len(t, prompts, 5)

	promptNames := make(map[string]bool)
	for _, prompt := range prompts {
		promptNames[prompt.Name] = true
		assert.NotEmpty(t, prompt.Name)
		assert.NotEmpty(t, prompt.IntentModel)
		assert.NotEmpty(t, prompt.Description)
		assert.NotEmpty(t, prompt.Resources)
		assert.Equal(t, parity.ParityFull, prompt.ParityStatus)
	}

	// Verify all required intents are present
	requiredIntents := []string{"understand", "build", "fix", "review", "operate"}
	for _, intent := range requiredIntents {
		assert.True(t, promptNames[intent], "missing prompt: %s", intent)
	}
}

func TestPromptDefinitionResources(t *testing.T) {
	prompts := parity.DefaultPrompts()

	for _, prompt := range prompts {
		assert.NotEmpty(t, prompt.Resources, "prompt %s should have resources", prompt.Name)

		// Verify resources are valid URIs
		for _, resource := range prompt.Resources {
			assert.Contains(t, resource, "sdp://", "resource %s should have sdp:// prefix", resource)
		}
	}
}
