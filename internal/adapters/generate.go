// Package adapters generates per-harness adapter files from an SDP manifest.
// Output is a map of relative path → file contents; the caller decides where
// to write them (typically .sdp/generated/).
package adapters

import (
	"bytes"
	"sort"
	"text/template"

	"sdp_dev/internal/manifest"
)

var (
	tmplClaudeCommand = mustParse("claude-command", "templates/claude-code/command.tmpl")
	tmplClaudeAgent   = mustParse("claude-agent", "templates/claude-code/agent.tmpl")
	tmplOpenCodeAgent = mustParse("opencode-agent", "templates/opencode/agent.tmpl")
	tmplOpenCodeSkill = mustParse("opencode-skill", "templates/opencode/skill.tmpl")
	tmplCodexSkill    = mustParse("codex-skill", "templates/codex/skill.tmpl")
	tmplCursorCommand = mustParse("cursor-command", "templates/cursor/command.tmpl")
)

// Generate renders adapter files for all harnesses declared in the manifest.
// Returns a deterministic map of relative output path → file contents.
// Relative paths are anchored at the repo root (e.g. ".sdp/generated/.claude/commands/build.md").
func Generate(m *manifest.Manifest) (map[string][]byte, error) {
	out := make(map[string][]byte)

	allHarnesses := m.Harnesses
	if len(allHarnesses) == 0 {
		allHarnesses = []manifest.Harness{
			manifest.HarnessClaudeCode,
			manifest.HarnessOpenCode,
			manifest.HarnessCodex,
			manifest.HarnessCursor,
		}
	}
	harnessEnabled := make(map[manifest.Harness]bool, len(allHarnesses))
	for _, h := range allHarnesses {
		harnessEnabled[h] = true
	}

	if err := generateClaudeCode(m, harnessEnabled, out); err != nil {
		return nil, err
	}
	if err := generateOpenCode(m, harnessEnabled, out); err != nil {
		return nil, err
	}
	if err := generateCodex(m, harnessEnabled, out); err != nil {
		return nil, err
	}
	if err := generateCursor(m, harnessEnabled, out); err != nil {
		return nil, err
	}

	return out, nil
}

// itemHarnesses returns the effective harness set for an item: if the item's
// own Harnesses list is empty, it inherits all harnesses enabled in the manifest.
func itemHarnesses(declared []manifest.Harness, manifestEnabled map[manifest.Harness]bool) map[manifest.Harness]bool {
	if len(declared) == 0 {
		// empty list = "all manifest harnesses"
		return manifestEnabled
	}
	out := make(map[manifest.Harness]bool, len(declared))
	for _, h := range declared {
		if manifestEnabled[h] {
			out[h] = true
		}
	}
	return out
}

func render(t *template.Template, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// generateClaudeCode emits .claude/commands/<name>.md and .claude/agents/<name>.md
func generateClaudeCode(m *manifest.Manifest, enabled map[manifest.Harness]bool, out map[string][]byte) error {
	if !enabled[manifest.HarnessClaudeCode] {
		return nil
	}

	// Sort commands for determinism
	cmds := make([]manifest.Command, len(m.Commands))
	copy(cmds, m.Commands)
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })

	for _, c := range cmds {
		ih := itemHarnesses(c.Harnesses, enabled)
		if !ih[manifest.HarnessClaudeCode] {
			continue
		}
		data, err := render(tmplClaudeCommand, c)
		if err != nil {
			return err
		}
		out[".claude/commands/"+c.Name+".md"] = data
	}

	// Sort agents for determinism
	agents := make([]manifest.Agent, len(m.Agents))
	copy(agents, m.Agents)
	sort.Slice(agents, func(i, j int) bool { return agents[i].Name < agents[j].Name })

	for _, a := range agents {
		ih := itemHarnesses(a.Harnesses, enabled)
		if !ih[manifest.HarnessClaudeCode] {
			continue
		}
		data, err := render(tmplClaudeAgent, a)
		if err != nil {
			return err
		}
		out[".claude/agents/"+a.Name+".md"] = data
	}
	return nil
}

// generateOpenCode emits .opencode/agent/<name>.json and .opencode/skill/<name>.md
func generateOpenCode(m *manifest.Manifest, enabled map[manifest.Harness]bool, out map[string][]byte) error {
	if !enabled[manifest.HarnessOpenCode] {
		return nil
	}

	agents := make([]manifest.Agent, len(m.Agents))
	copy(agents, m.Agents)
	sort.Slice(agents, func(i, j int) bool { return agents[i].Name < agents[j].Name })

	for _, a := range agents {
		ih := itemHarnesses(a.Harnesses, enabled)
		if !ih[manifest.HarnessOpenCode] {
			continue
		}
		data, err := render(tmplOpenCodeAgent, a)
		if err != nil {
			return err
		}
		out[".opencode/agent/"+a.Name+".json"] = data
	}

	skills := make([]manifest.Skill, len(m.Skills))
	copy(skills, m.Skills)
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	for _, s := range skills {
		ih := itemHarnesses(s.Harnesses, enabled)
		if !ih[manifest.HarnessOpenCode] {
			continue
		}
		data, err := render(tmplOpenCodeSkill, s)
		if err != nil {
			return err
		}
		out[".opencode/skill/"+s.Name+".md"] = data
	}
	return nil
}

// generateCodex emits .codex/skills/<name>.md
func generateCodex(m *manifest.Manifest, enabled map[manifest.Harness]bool, out map[string][]byte) error {
	if !enabled[manifest.HarnessCodex] {
		return nil
	}

	skills := make([]manifest.Skill, len(m.Skills))
	copy(skills, m.Skills)
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	for _, s := range skills {
		ih := itemHarnesses(s.Harnesses, enabled)
		if !ih[manifest.HarnessCodex] {
			continue
		}
		data, err := render(tmplCodexSkill, s)
		if err != nil {
			return err
		}
		out[".codex/skills/"+s.Name+".md"] = data
	}
	return nil
}

// generateCursor emits .cursor/rules/<name>.mdc
func generateCursor(m *manifest.Manifest, enabled map[manifest.Harness]bool, out map[string][]byte) error {
	if !enabled[manifest.HarnessCursor] {
		return nil
	}

	cmds := make([]manifest.Command, len(m.Commands))
	copy(cmds, m.Commands)
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })

	for _, c := range cmds {
		ih := itemHarnesses(c.Harnesses, enabled)
		if !ih[manifest.HarnessCursor] {
			continue
		}
		data, err := render(tmplCursorCommand, c)
		if err != nil {
			return err
		}
		out[".cursor/rules/"+c.Name+".mdc"] = data
	}
	return nil
}
