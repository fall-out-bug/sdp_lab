package dispatch

import (
	"strings"
	"testing"
)

func TestPromptAssembler_DefaultBudget(t *testing.T) {
	a := NewPromptAssembler("unknown-model-xyz")
	want := int(128000 * 0.3) // default budget
	if got := a.Budget(); got != want {
		t.Errorf("Budget() for unknown model: got %d, want %d", got, want)
	}
}

func TestPromptAssembler_KnownModelBudget(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{"claude-opus-4", 60000},
		{"claude-sonnet-4", 60000},
		{"gpt-4o", 38400},
		{"gemini-2.5-pro", 200000},
	}
	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			a := NewPromptAssembler(tc.model)
			if got := a.Budget(); got != tc.want {
				t.Errorf("Budget() for %s: got %d, want %d", tc.model, got, tc.want)
			}
		})
	}
}

func TestPromptAssembler_AssembleBasic(t *testing.T) {
	a := NewPromptAssembler("claude-opus-4")
	taskPrompt := "Implement the feature."
	layer := PromptLayer{
		Name:    "scope",
		Content: "\n## Scope\n- file1.go\n- file2.go\n",
	}

	result := a.Assemble(taskPrompt, layer)

	if !strings.HasPrefix(result, taskPrompt) {
		t.Errorf("result should start with task prompt, got prefix %q", result[:len(taskPrompt)])
	}
	if !strings.Contains(result, layer.Content) {
		t.Error("result should contain the layer content")
	}
}

func TestPromptAssembler_AssembleBudgetTruncation(t *testing.T) {
	// Use a small-budget model and layers that exceed the budget to verify
	// that later layers are dropped when they do not fit.
	a := NewPromptAssembler("gpt-4o") // budget = 38400 tokens

	taskPrompt := "Do the thing."
	taskTokens := charToToken(len(taskPrompt))
	budget := a.Budget()
	remaining := budget - taskTokens

	// First layer fits within remaining budget.
	fitChars := (remaining / 2) * charsPerToken
	fitLayer := PromptLayer{
		Name:    "fits",
		Content: strings.Repeat("a", fitChars),
	}

	// Second layer is too large to fit after the first.
	oversizeChars := (remaining * 2) * charsPerToken
	oversizeLayer := PromptLayer{
		Name:    "oversize",
		Content: strings.Repeat("b", oversizeChars),
	}

	result := a.Assemble(taskPrompt, fitLayer, oversizeLayer)

	if !strings.Contains(result, fitLayer.Content) {
		t.Error("fitting layer should be included")
	}
	if strings.Contains(result, oversizeLayer.Content) {
		t.Error("oversize layer should be dropped")
	}
}

func TestPromptAssembler_TaskPromptAlwaysIncluded(t *testing.T) {
	// Even when the task prompt itself exceeds the budget, it must be included.
	a := NewPromptAssembler("gpt-4o") // budget = 38400 tokens

	// Create a task prompt larger than the entire budget.
	largePrompt := strings.Repeat("x", (a.Budget()+10000)*charsPerToken)

	result := a.Assemble(largePrompt)

	if result != largePrompt {
		t.Error("task prompt must be included verbatim even when it exceeds budget")
	}
}

func TestPromptAssembler_AssembleNoLayers(t *testing.T) {
	a := NewPromptAssembler("claude-opus-4")
	taskPrompt := "Just do it."

	result := a.Assemble(taskPrompt)

	if result != taskPrompt {
		t.Errorf("with no layers, result should equal task prompt; got %q", result)
	}
}

func TestPromptAssembler_AssembleEmptyLayerContent(t *testing.T) {
	a := NewPromptAssembler("claude-opus-4")
	taskPrompt := "Do the work."
	emptyLayer := PromptLayer{Name: "empty", Content: ""}

	result := a.Assemble(taskPrompt, emptyLayer)

	if result != taskPrompt {
		t.Error("empty layer should not change the result")
	}
}

func TestPromptAssembler_AssembleMultipleLayersInOrder(t *testing.T) {
	a := NewPromptAssembler("claude-opus-4") // large budget
	taskPrompt := "Build it."
	layer1 := PromptLayer{Name: "first", Content: "[LAYER1]"}
	layer2 := PromptLayer{Name: "second", Content: "[LAYER2]"}

	result := a.Assemble(taskPrompt, layer1, layer2)

	idx1 := strings.Index(result, "[LAYER1]")
	idx2 := strings.Index(result, "[LAYER2]")
	if idx1 < 0 || idx2 < 0 {
		t.Fatal("both layers should be present in the result")
	}
	if idx1 > idx2 {
		t.Error("layers should appear in order: layer1 before layer2")
	}
}
