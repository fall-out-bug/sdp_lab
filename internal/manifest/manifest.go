package manifest

type Harness string

const (
	HarnessClaudeCode Harness = "claude-code"
	HarnessOpenCode   Harness = "opencode"
	HarnessCodex      Harness = "codex"
	HarnessCursor     Harness = "cursor"
)

type Manifest struct {
	Version    string      `yaml:"version"`
	SDPVersion string      `yaml:"sdp_version"`
	Harnesses  []Harness   `yaml:"harnesses,omitempty"`
	Skills     []Skill     `yaml:"skills,omitempty"`
	Commands   []Command   `yaml:"commands,omitempty"`
	Agents     []Agent     `yaml:"agents,omitempty"`
	Hooks      []Hook      `yaml:"hooks,omitempty"`
	MCPServers []MCPServer `yaml:"mcp_servers,omitempty"`
}

type Skill struct {
	Name          string    `yaml:"name"`
	Path          string    `yaml:"path"`
	Version       string    `yaml:"version,omitempty"`
	Harnesses     []Harness `yaml:"harnesses,omitempty"`
	Compatibility []Harness `yaml:"compatibility,omitempty"`
	Summary       string    `yaml:"summary,omitempty"`
}

type Command struct {
	Name        string                `yaml:"name"`
	Path        string                `yaml:"path"`
	Type        string                `yaml:"type,omitempty"`
	Harnesses   []Harness             `yaml:"harnesses,omitempty"`
	Dispatch    map[Harness]string    `yaml:"dispatch,omitempty"`
	ParityNotes string                `yaml:"parity_notes,omitempty"`
	Summary     string                `yaml:"summary,omitempty"`
}

type Agent struct {
	Name             string    `yaml:"name"`
	Role             string    `yaml:"role,omitempty"`
	SystemPromptPath string    `yaml:"system_prompt_path"`
	Harnesses        []Harness `yaml:"harnesses,omitempty"`
	Summary          string    `yaml:"summary,omitempty"`
}

type Hook struct {
	Event     string    `yaml:"event"`
	Script    string    `yaml:"script"`
	Harnesses []Harness `yaml:"harnesses,omitempty"`
	Summary   string    `yaml:"summary,omitempty"`
}

type MCPServer struct {
	Name      string    `yaml:"name"`
	URL       string    `yaml:"url,omitempty"`
	Scopes    []string  `yaml:"scopes,omitempty"`
	Optional  bool      `yaml:"optional,omitempty"`
	Harnesses []Harness `yaml:"harnesses,omitempty"`
	Summary   string    `yaml:"summary,omitempty"`
}
