package agent

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillRegistry loads role-specific capabilities from specs/agent-skills.yaml.
type SkillRegistry struct {
	role   string
	skills []string
}

// NewSkillRegistry creates a SkillRegistry for the role.
func NewSkillRegistry(role, workDir string) *SkillRegistry {
	r := &SkillRegistry{role: role}
	_ = r.LoadFromFile(workDir)
	return r
}

// LoadFromFile loads skills from specs/agent-skills.yaml.
func (r *SkillRegistry) LoadFromFile(workDir string) error {
	path := filepath.Join(workDir, "specs", "agent-skills.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			r.skills = defaultSkillsForRole(r.role)
			return nil
		}
		return err
	}
	var cfg struct {
		Roles map[string]struct {
			Skills []string `yaml:"skills"`
		} `yaml:"roles"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	roleCfg, ok := cfg.Roles[r.role]
	if !ok {
		r.skills = defaultSkillsForRole(r.role)
		return nil
	}
	r.skills = roleCfg.Skills
	if len(r.skills) == 0 {
		r.skills = defaultSkillsForRole(r.role)
	}
	return nil
}

func defaultSkillsForRole(role string) []string {
	switch role {
	case "analyst":
		return []string{"requirement-decomposition", "risk-analysis", "dependency-mapping"}
	case "coder":
		return []string{"code-generation", "test-writing", "refactoring", "boundary-compliance"}
	case "reviewer":
		return []string{"adversarial-review", "consensus-scoring", "feedback-structuring"}
	case "retro":
		return []string{"telemetry-analysis", "pattern-detection", "improvement-proposal"}
	case "orchestrator":
		return []string{"scheduling", "lifecycle-management", "dispatch"}
	default:
		return []string{"generic-execution"}
	}
}

// Skills returns the loaded skills for this role.
func (r *SkillRegistry) Skills() []string {
	return append([]string(nil), r.skills...)
}

// HasSkill returns true if the registry has the given skill.
func (r *SkillRegistry) HasSkill(skill string) bool {
	for _, s := range r.skills {
		if strings.EqualFold(s, skill) {
			return true
		}
	}
	return false
}
