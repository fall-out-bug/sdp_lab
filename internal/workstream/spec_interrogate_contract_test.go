package workstream

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecInterrogateDocumentsSocraticPiProtocol(t *testing.T) {
	root := filepath.Join("..", "..")
	paths := []string{
		"prompts/skills/spec-interrogate/SKILL.md",
		".agents/skills/spec-interrogate.md",
		".agents/skills/spec-interrogate/SKILL.md",
	}

	required := []string{
		"## Rubrics",
		"critic",
		"MUST NOT propose solutions",
		"judge",
		"provider rotation",
		"zai/glm-5.1",
		"kimi-coding/k2p6",
		"minimax/MiniMax-M2.7",
		"--no-tools --no-context-files --no-session",
		"not a gate",
		"critic_provider",
		"judge_provider",
	}

	for _, relPath := range paths {
		contentBytes, err := os.ReadFile(filepath.Join(root, relPath))
		if err != nil {
			t.Fatalf("read %s: %v", relPath, err)
		}
		content := string(contentBytes)
		for _, marker := range required {
			if !strings.Contains(content, marker) {
				t.Errorf("%s missing %q", relPath, marker)
			}
		}
	}
}
