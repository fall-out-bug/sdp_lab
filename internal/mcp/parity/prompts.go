// Package parity provides prompt and resource parity alignment with F125 intent model.
package parity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IntentModel represents the F125 intent model categories.
type IntentModel string

const (
	// IntentUnderstand represents the understand intent.
	IntentUnderstand IntentModel = "F125:intent:understand"
	// IntentBuild represents the build intent.
	IntentBuild IntentModel = "F125:intent:build"
	// IntentFix represents the fix intent.
	IntentFix IntentModel = "F125:intent:fix"
	// IntentReview represents the review intent.
	IntentReview IntentModel = "F125:intent:review"
	// IntentOperate represents the operate intent.
	IntentOperate IntentModel = "F125:intent:operate"
)

// PromptDefinition defines an MCP prompt aligned with the intent model.
type PromptDefinition struct {
	Name          string                 `json:"name"`
	IntentModel   IntentModel            `json:"intent_model"`
	Description   string                 `json:"description"`
	Arguments     []PromptArgument       `json:"arguments"`
	Resources     []string               `json:"resources"`
	Template      string                 `json:"template"`
	ParityStatus  ParityStatus           `json:"parity_status"`
	SkillFiles    []string               `json:"skill_files"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PromptArgument defines a prompt argument.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// ParityStatus represents the parity status of a surface.
type ParityStatus string

const (
	// ParityFull indicates complete parity.
	ParityFull ParityStatus = "full"
	// ParityPartial indicates partial parity.
	ParityPartial ParityStatus = "partial"
	// ParityDeprecated indicates deprecated surface.
	ParityDeprecated ParityStatus = "deprecated"
	// ParityForward indicates forward-compatible surface.
	ParityForward ParityStatus = "forward"
)

// PromptRegistry manages prompt definitions aligned with the intent model.
type PromptRegistry struct {
	prompts map[string]*PromptDefinition
}

// NewPromptRegistry creates a new prompt registry.
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		prompts: make(map[string]*PromptDefinition),
	}
}

// Register registers a prompt definition.
func (r *PromptRegistry) Register(prompt *PromptDefinition) error {
	if prompt.Name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}
	if prompt.IntentModel == "" {
		return fmt.Errorf("intent model cannot be empty")
	}
	if prompt.Description == "" {
		return fmt.Errorf("description cannot be empty")
	}

	r.prompts[prompt.Name] = prompt
	return nil
}

// Get retrieves a prompt by name.
func (r *PromptRegistry) Get(name string) (*PromptDefinition, bool) {
	prompt, ok := r.prompts[name]
	return prompt, ok
}

// List returns all registered prompts.
func (r *PromptRegistry) List() []*PromptDefinition {
	prompts := make([]*PromptDefinition, 0, len(r.prompts))
	for _, prompt := range r.prompts {
		prompts = append(prompts, prompt)
	}
	return prompts
}

// GetByIntentModel returns prompts for a specific intent model.
func (r *PromptRegistry) GetByIntentModel(model IntentModel) []*PromptDefinition {
	var result []*PromptDefinition
	for _, prompt := range r.prompts {
		if prompt.IntentModel == model {
			result = append(result, prompt)
		}
	}
	return result
}

// ValidateParity validates that all intents have full parity.
func (r *PromptRegistry) ValidateParity() error {
	requiredIntents := []IntentModel{
		IntentUnderstand,
		IntentBuild,
		IntentFix,
		IntentReview,
		IntentOperate,
	}

	for _, intent := range requiredIntents {
		prompts := r.GetByIntentModel(intent)
		if len(prompts) == 0 {
			return fmt.Errorf("missing prompt for intent: %s", intent)
		}

		for _, prompt := range prompts {
			if prompt.ParityStatus != ParityFull {
				return fmt.Errorf("prompt %s for intent %s does not have full parity (status: %s)",
					prompt.Name, intent, prompt.ParityStatus)
			}
		}
	}

	return nil
}

// LoadFromSkillDirectory loads prompt definitions from skill files.
func (r *PromptRegistry) LoadFromSkillDirectory(skillDir string) error {
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return fmt.Errorf("read skill directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		skillPath := filepath.Join(skillDir, entry.Name())
		prompt, err := r.parseSkillFile(skillPath)
		if err != nil {
			// Log but don't fail - not all skill files define prompts
			continue
		}

		if prompt != nil {
			prompt.SkillFiles = append(prompt.SkillFiles, skillPath)
			if err := r.Register(prompt); err != nil {
				return fmt.Errorf("register prompt from %s: %w", skillPath, err)
			}
		}
	}

	return nil
}

// parseSkillFile parses a skill file to extract prompt definition.
func (r *PromptRegistry) parseSkillFile(path string) (*PromptDefinition, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	// Parse frontmatter and content
	lines := strings.Split(string(content), "\n")
	var frontmatter []string
	var contentStart int
	inFrontmatter := false

	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				contentStart = i + 1
				break
			}
		}
		if inFrontmatter {
			frontmatter = append(frontmatter, line)
		}
	}

	// Extract metadata from frontmatter
	metadata := make(map[string]string)
	for _, line := range frontmatter {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				metadata[key] = value
			}
		}
	}

	// Determine intent model from filename or metadata
	name := strings.TrimSuffix(filepath.Base(path), ".md")
	intentModel, ok := metadata["intent"]
	if !ok {
		// Infer intent model from name
		intentModel = r.inferIntentModel(name)
	}

	if intentModel == "" {
		return nil, nil // Not an intent-based skill
	}

	// Build prompt definition
	prompt := &PromptDefinition{
		Name:        name,
		IntentModel: IntentModel(intentModel),
		Description: metadata["description"],
		Template:    strings.Join(lines[contentStart:], "\n"),
		ParityStatus: ParityFull,
	}

	return prompt, nil
}

// inferIntentModel infers the intent model from a prompt name.
func (r *PromptRegistry) inferIntentModel(name string) string {
	switch strings.ToLower(name) {
	case "understand":
		return string(IntentUnderstand)
	case "build":
		return string(IntentBuild)
	case "fix":
		return string(IntentFix)
	case "review":
		return string(IntentReview)
	case "operate":
		return string(IntentOperate)
	default:
		return ""
	}
}

// DefaultPrompts returns the default set of prompts aligned with F125 intent model.
func DefaultPrompts() []*PromptDefinition {
	return []*PromptDefinition{
		{
			Name:        "understand",
			IntentModel: IntentUnderstand,
			Description: "Understand codebase context, architecture, and patterns",
			Arguments: []PromptArgument{
				{
					Name:        "depth",
					Description: "Analysis depth: quick, standard, or deep",
					Required:    false,
					Default:     "standard",
				},
				{
					Name:        "focus",
					Description: "Optional focus area (e.g., architecture, patterns, health)",
					Required:    false,
				},
			},
			Resources: []string{
				"sdp://scout",
				"sdp://architect",
				"sdp://metrics",
				"sdp://spec",
			},
			Template: understandTemplate(),
			ParityStatus: ParityFull,
			SkillFiles: []string{
				".agents/skills/understand.md",
			},
		},
		{
			Name:        "build",
			IntentModel: IntentBuild,
			Description: "Build and implement features with proper patterns",
			Arguments: []PromptArgument{
				{
					Name:        "description",
					Description: "Feature description",
					Required:    true,
				},
				{
					Name:        "workstream",
					Description: "Optional workstream ID",
					Required:    false,
				},
			},
			Resources: []string{
				"sdp://scout",
				"sdp://architect",
				"sdp://spec",
			},
			Template: buildTemplate(),
			ParityStatus: ParityFull,
			SkillFiles: []string{
				".agents/skills/build.md",
			},
		},
		{
			Name:        "fix",
			IntentModel: IntentFix,
			Description: "Diagnose and fix bugs systematically",
			Arguments: []PromptArgument{
				{
					Name:        "description",
					Description: "Bug description",
					Required:    true,
				},
				{
					Name:        "context",
					Description: "Additional context about the bug",
					Required:    false,
				},
			},
			Resources: []string{
				"sdp://scout",
				"sdp://metrics",
				"sdp://spec",
			},
			Template: fixTemplate(),
			ParityStatus: ParityFull,
			SkillFiles: []string{
				".agents/skills/fix.md",
			},
		},
		{
			Name:        "review",
			IntentModel: IntentReview,
			Description: "Review code changes for quality and compliance",
			Arguments: []PromptArgument{
				{
					Name:        "scope",
					Description: "Review scope: changes, file, or full",
					Required:    false,
					Default:     "changes",
				},
			},
			Resources: []string{
				"sdp://architect",
				"sdp://metrics",
				"sdp://spec",
			},
			Template: reviewTemplate(),
			ParityStatus: ParityFull,
			SkillFiles: []string{
				".agents/skills/review.md",
			},
		},
		{
			Name:        "operate",
			IntentModel: IntentOperate,
			Description: "Operate and maintain deployed systems",
			Arguments: []PromptArgument{
				{
					Name:        "task",
					Description: "Operation task description",
					Required:    true,
				},
			},
			Resources: []string{
				"sdp://metrics",
				"sdp://spec",
			},
			Template: operateTemplate(),
			ParityStatus: ParityFull,
			SkillFiles: []string{
				".agents/skills/operate.md",
			},
		},
	}
}

// Template functions for default prompts

func understandTemplate() string {
	return `# Understand Codebase Context

Analyze the codebase to understand:

1. **Architecture & Structure**
   - Key components and their relationships
   - Design patterns in use
   - Technology stack

2. **Code Quality**
   - Test coverage
   - Documentation quality
   - Code health signals

3. **Process & Patterns**
   - Development workflow
   - CI/CD setup
   - Team dynamics

## Resources
- Scout: {{.scout}}
- Architect: {{.architect}}
- Metrics: {{.metrics}}
- Spec: {{.spec}}
`
}

func buildTemplate() string {
	return `# Build Feature

Implement the following feature following project patterns:

**Description:** {{.description}}

{{if .workstream}}**Workstream:** {{.workstream}}{{end}}

## Checklist
- [ ] Follow existing patterns
- [ ] Add tests
- [ ] Update documentation
- [ ] Verify CI passes

## Resources
- Scout: {{.scout}}
- Architect: {{.architect}}
- Spec: {{.spec}}
`
}

func fixTemplate() string {
	return `# Fix Bug

Systematically diagnose and fix the reported issue:

**Bug Description:** {{.description}}

{{if .context}}**Context:** {{.context}}{{end}}

## Process
1. Reproduce the issue
2. Identify root cause
3. Implement fix
4. Add tests
5. Verify fix

## Resources
- Scout: {{.scout}}
- Metrics: {{.metrics}}
- Spec: {{.spec}}
`
}

func reviewTemplate() string {
	return `# Code Review

Review code changes for quality and compliance:

**Scope:** {{.scope}}

## Review Criteria
- Architecture alignment
- Code quality
- Test coverage
- Documentation
- Security considerations

## Resources
- Architect: {{.architect}}
- Metrics: {{.metrics}}
- Spec: {{.spec}}
`
}

func operateTemplate() string {
	return `# Operate System

Perform operational task:

**Task:** {{.task}}

## Considerations
- System health
- Performance impact
- Monitoring
- Rollback plan

## Resources
- Metrics: {{.metrics}}
- Spec: {{.spec}}
`
}