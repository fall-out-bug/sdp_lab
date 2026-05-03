// Package contract provides the CLI-to-MCP mapping contract types and validation.
package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// CurrentSpecVersion is the current mapping contract specification version.
	CurrentSpecVersion = "1.0.0"

	// ParityFull indicates complete parity between CLI and MCP surface.
	ParityFull = "full"

	// ParityPartial indicates partial parity (some features not exposed).
	ParityPartial = "partial"

	// ParityDeprecated indicates deprecated surface (kept for backwards compat).
	ParityDeprecated = "deprecated"

	// ParityForward indicates forward-compatible surface (reserved for future use).
	ParityForward = "forward"

	// ToolCapabilityRead indicates a tool does not intentionally mutate external state.
	ToolCapabilityRead = "read"

	// ToolCapabilityWrite indicates a tool can mutate files, queues, issues, or other external state.
	ToolCapabilityWrite = "write"
)

// Mapping represents the complete CLI-to-MCP mapping contract.
type Mapping struct {
	SpecVersion      string            `json:"spec_version"`
	CLIRegistryHash  string            `json:"cli_registry_hash"`
	SkillCatalogHash string            `json:"skill_catalog_hash"`
	GeneratedAt      time.Time         `json:"generated_at"`
	Tools            []ToolMapping     `json:"tools"`
	Resources        []ResourceMapping `json:"resources"`
	Prompts          []PromptMapping   `json:"prompts"`
}

// ToolMapping maps a CLI command to an MCP tool.
type ToolMapping struct {
	MCPToolName    string             `json:"mcp_tool_name"`
	CLICommand     string             `json:"cli_command"`
	Description    string             `json:"description"`
	Parameters     []ParameterMapping `json:"parameters"`
	ParityStatus   string             `json:"parity_status"`
	Capability     string             `json:"capability"`
	SourceLocation string             `json:"source_location,omitempty"`
}

// ResourceMapping maps CLI output to an MCP resource.
type ResourceMapping struct {
	MCPResourceURI string `json:"mcp_resource_uri"`
	CLICommand     string `json:"cli_command"`
	ArtifactPath   string `json:"artifact_path"`
	Description    string `json:"description"`
	MIMEType       string `json:"mime_type"`
	ParityStatus   string `json:"parity_status"`
	HintTool       string `json:"hint_tool,omitempty"`
}

// PromptMapping maps a skill intent to an MCP prompt.
type PromptMapping struct {
	MCPPromptName string            `json:"mcp_prompt_name"`
	IntentModel   string            `json:"intent_model"`
	Description   string            `json:"description"`
	Arguments     []ArgumentMapping `json:"arguments"`
	ResourcesUsed []string          `json:"resources_used"`
	ParityStatus  string            `json:"parity_status"`
	SkillFiles    []string          `json:"skill_files,omitempty"`
}

// ParameterMapping maps a CLI flag to an MCP tool parameter.
type ParameterMapping struct {
	MCPParamName string      `json:"mcp_param_name"`
	CLIFlag      string      `json:"cli_flag"`
	Type         string      `json:"type"`
	Required     bool        `json:"required"`
	EnumValues   []string    `json:"enum_values,omitempty"`
	Default      interface{} `json:"default,omitempty"`
}

// ArgumentMapping maps a prompt argument to its definition.
type ArgumentMapping struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// RegistrySnapshot represents a snapshot of CLI commands for hash generation.
type RegistrySnapshot struct {
	Commands []CommandEntry `json:"commands"`
}

// CommandEntry represents a single CLI command entry.
type CommandEntry struct {
	Name     string            `json:"name"`
	Flags    []FlagEntry       `json:"flags"`
	Source   string            `json:"source"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// FlagEntry represents a CLI flag definition.
type FlagEntry struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Required bool        `json:"required"`
	Default  interface{} `json:"default,omitempty"`
}

// SkillCatalogSnapshot represents a snapshot of skill intents for hash generation.
type SkillCatalogSnapshot struct {
	Intents []IntentEntry `json:"intents"`
}

// IntentEntry represents a skill intent entry.
type IntentEntry struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Arguments   []ArgumentMapping `json:"arguments"`
	Resources   []string          `json:"resources"`
	SkillFiles  []string          `json:"skill_files"`
}

// Builder builds a Mapping contract incrementally.
type Builder struct {
	mapping          *Mapping
	registrySnapshot *RegistrySnapshot
	skillSnapshot    *SkillCatalogSnapshot
}

// NewBuilder creates a new Mapping builder.
func NewBuilder() *Builder {
	return &Builder{
		mapping: &Mapping{
			SpecVersion: CurrentSpecVersion,
			GeneratedAt: time.Now().UTC(),
			Tools:       []ToolMapping{},
			Resources:   []ResourceMapping{},
			Prompts:     []PromptMapping{},
		},
		registrySnapshot: &RegistrySnapshot{
			Commands: []CommandEntry{},
		},
		skillSnapshot: &SkillCatalogSnapshot{
			Intents: []IntentEntry{},
		},
	}
}

// WithRegistrySnapshot sets the CLI registry snapshot.
func (b *Builder) WithRegistrySnapshot(snapshot *RegistrySnapshot) *Builder {
	b.registrySnapshot = snapshot
	b.mapping.CLIRegistryHash = computeHash(snapshot)
	return b
}

// WithSkillSnapshot sets the skill catalog snapshot.
func (b *Builder) WithSkillSnapshot(snapshot *SkillCatalogSnapshot) *Builder {
	b.skillSnapshot = snapshot
	b.mapping.SkillCatalogHash = computeHash(snapshot)
	return b
}

// AddTool adds a tool mapping to the contract.
func (b *Builder) AddTool(tool ToolMapping) *Builder {
	if tool.Capability == "" {
		tool.Capability = InferToolCapability(tool.MCPToolName, tool.CLICommand)
	}
	b.mapping.Tools = append(b.mapping.Tools, tool)
	return b
}

// AddResource adds a resource mapping to the contract.
func (b *Builder) AddResource(resource ResourceMapping) *Builder {
	b.mapping.Resources = append(b.mapping.Resources, resource)
	return b
}

// AddPrompt adds a prompt mapping to the contract.
func (b *Builder) AddPrompt(prompt PromptMapping) *Builder {
	b.mapping.Prompts = append(b.mapping.Prompts, prompt)
	return b
}

// Build builds and validates the mapping contract.
func (b *Builder) Build() (*Mapping, error) {
	// Set default hashes if not provided (for testing purposes)
	if b.mapping.CLIRegistryHash == "" && b.registrySnapshot != nil {
		b.mapping.CLIRegistryHash = computeHash(b.registrySnapshot)
	} else if b.mapping.CLIRegistryHash == "" {
		b.mapping.CLIRegistryHash = "test-cli-hash"
	}

	if b.mapping.SkillCatalogHash == "" && b.skillSnapshot != nil {
		b.mapping.SkillCatalogHash = computeHash(b.skillSnapshot)
	} else if b.mapping.SkillCatalogHash == "" {
		b.mapping.SkillCatalogHash = "test-skill-hash"
	}

	// Sort for deterministic serialization
	sort.Slice(b.mapping.Tools, func(i, j int) bool {
		return b.mapping.Tools[i].MCPToolName < b.mapping.Tools[j].MCPToolName
	})
	sort.Slice(b.mapping.Resources, func(i, j int) bool {
		return b.mapping.Resources[i].MCPResourceURI < b.mapping.Resources[j].MCPResourceURI
	})
	sort.Slice(b.mapping.Prompts, func(i, j int) bool {
		return b.mapping.Prompts[i].MCPPromptName < b.mapping.Prompts[j].MCPPromptName
	})

	if err := b.mapping.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return b.mapping, nil
}

// Validate validates the mapping contract.
func (m *Mapping) Validate() error {
	if m.SpecVersion == "" {
		return fmt.Errorf("spec_version is required")
	}
	if m.CLIRegistryHash == "" {
		return fmt.Errorf("cli_registry_hash is required")
	}
	if m.SkillCatalogHash == "" {
		return fmt.Errorf("skill_catalog_hash is required")
	}
	if m.GeneratedAt.IsZero() {
		return fmt.Errorf("generated_at is required")
	}

	// Validate tool names are unique and unambiguous after normalization.
	toolNames := make(map[string]bool)
	normalizedToolNames := make(map[string]string)
	for _, tool := range m.Tools {
		if tool.MCPToolName == "" {
			return fmt.Errorf("tool mcp_tool_name is required")
		}
		if tool.CLICommand == "" {
			return fmt.Errorf("tool %s: cli_command is required", tool.MCPToolName)
		}
		if tool.Description == "" {
			return fmt.Errorf("tool %s: description is required", tool.MCPToolName)
		}
		if tool.ParityStatus == "" {
			return fmt.Errorf("tool %s: parity_status is required", tool.MCPToolName)
		}
		if tool.Capability != ToolCapabilityRead && tool.Capability != ToolCapabilityWrite {
			return fmt.Errorf("tool %s: capability must be %q or %q", tool.MCPToolName, ToolCapabilityRead, ToolCapabilityWrite)
		}
		if toolNames[tool.MCPToolName] {
			return fmt.Errorf("duplicate tool name: %s", tool.MCPToolName)
		}
		toolNames[tool.MCPToolName] = true
		normalizedName := normalizeToolIdentity(tool.MCPToolName)
		if existing, ok := normalizedToolNames[normalizedName]; ok {
			return fmt.Errorf("ambiguous tool name: %s conflicts with %s", tool.MCPToolName, existing)
		}
		normalizedToolNames[normalizedName] = tool.MCPToolName
	}

	// Validate resource URIs are unique
	resourceURIs := make(map[string]bool)
	for _, resource := range m.Resources {
		if resource.MCPResourceURI == "" {
			return fmt.Errorf("resource mcp_resource_uri is required")
		}
		if resource.CLICommand == "" {
			return fmt.Errorf("resource %s: cli_command is required", resource.MCPResourceURI)
		}
		if resource.ArtifactPath == "" {
			return fmt.Errorf("resource %s: artifact_path is required", resource.MCPResourceURI)
		}
		if resource.Description == "" {
			return fmt.Errorf("resource %s: description is required", resource.MCPResourceURI)
		}
		if resource.MIMEType == "" {
			return fmt.Errorf("resource %s: mime_type is required", resource.MCPResourceURI)
		}
		if resource.ParityStatus == "" {
			return fmt.Errorf("resource %s: parity_status is required", resource.MCPResourceURI)
		}
		if resourceURIs[resource.MCPResourceURI] {
			return fmt.Errorf("duplicate resource URI: %s", resource.MCPResourceURI)
		}
		resourceURIs[resource.MCPResourceURI] = true
	}

	// Validate prompt names are unique
	promptNames := make(map[string]bool)
	for _, prompt := range m.Prompts {
		if prompt.MCPPromptName == "" {
			return fmt.Errorf("prompt mcp_prompt_name is required")
		}
		if prompt.IntentModel == "" {
			return fmt.Errorf("prompt %s: intent_model is required", prompt.MCPPromptName)
		}
		if prompt.Description == "" {
			return fmt.Errorf("prompt %s: description is required", prompt.MCPPromptName)
		}
		if prompt.ParityStatus == "" {
			return fmt.Errorf("prompt %s: parity_status is required", prompt.MCPPromptName)
		}
		if promptNames[prompt.MCPPromptName] {
			return fmt.Errorf("duplicate prompt name: %s", prompt.MCPPromptName)
		}
		promptNames[prompt.MCPPromptName] = true
	}

	return nil
}

// InferToolCapability returns the default capability for known SDP MCP tools.
func InferToolCapability(mcpToolName, cliCommand string) string {
	name := strings.ToLower(strings.TrimSpace(mcpToolName))
	command := strings.ToLower(strings.TrimSpace(cliCommand))
	if strings.Contains(name, "create") ||
		strings.Contains(name, "close") ||
		strings.Contains(name, "update") ||
		strings.Contains(name, "delete") ||
		strings.Contains(name, "build") ||
		strings.Contains(name, "bootstrap") ||
		strings.Contains(name, "write") ||
		strings.Contains(command, "create") ||
		strings.Contains(command, "close") ||
		strings.Contains(command, "update") ||
		strings.Contains(command, "delete") ||
		strings.Contains(command, "build") ||
		strings.Contains(command, "bootstrap") {
		return ToolCapabilityWrite
	}
	return ToolCapabilityRead
}

func normalizeToolIdentity(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SaveToFile saves the mapping contract to a JSON file.
func (m *Mapping) SaveToFile(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mapping: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write mapping: %w", err)
	}

	return nil
}

// LoadFromFile loads a mapping contract from a JSON file.
func LoadFromFile(path string) (*Mapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mapping: %w", err)
	}

	var mapping Mapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("unmarshal mapping: %w", err)
	}
	mapping.ApplyDefaults()

	if err := mapping.Validate(); err != nil {
		return nil, fmt.Errorf("validate mapping: %w", err)
	}

	return &mapping, nil
}

// ApplyDefaults backfills fields added after the initial contract version.
func (m *Mapping) ApplyDefaults() {
	for i := range m.Tools {
		if m.Tools[i].Capability == "" {
			m.Tools[i].Capability = InferToolCapability(m.Tools[i].MCPToolName, m.Tools[i].CLICommand)
		}
	}
}

// computeHash computes a SHA256 hash of a serializable object.
func computeHash(obj interface{}) string {
	data, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])[:16]
}

// GetToolByMCPName retrieves a tool mapping by MCP tool name.
func (m *Mapping) GetToolByMCPName(name string) (*ToolMapping, bool) {
	for _, tool := range m.Tools {
		if tool.MCPToolName == name {
			return &tool, true
		}
	}
	return nil, false
}

// GetResourceByURI retrieves a resource mapping by URI.
func (m *Mapping) GetResourceByURI(uri string) (*ResourceMapping, bool) {
	for _, resource := range m.Resources {
		if resource.MCPResourceURI == uri {
			return &resource, true
		}
	}
	return nil, false
}

// GetPromptByMCPName retrieves a prompt mapping by MCP prompt name.
func (m *Mapping) GetPromptByMCPName(name string) (*PromptMapping, bool) {
	for _, prompt := range m.Prompts {
		if prompt.MCPPromptName == name {
			return &prompt, true
		}
	}
	return nil, false
}

// GetToolsByParityStatus retrieves tools filtered by parity status.
func (m *Mapping) GetToolsByParityStatus(status string) []ToolMapping {
	var result []ToolMapping
	for _, tool := range m.Tools {
		if tool.ParityStatus == status {
			result = append(result, tool)
		}
	}
	return result
}

// GetToolsByCapability retrieves tools filtered by capability (read or write).
func (m *Mapping) GetToolsByCapability(capability string) []ToolMapping {
	var result []ToolMapping
	for _, tool := range m.Tools {
		if tool.Capability == capability {
			result = append(result, tool)
		}
	}
	return result
}

// GetWriteTools returns all write-capable tools.
func (m *Mapping) GetWriteTools() []ToolMapping {
	return m.GetToolsByCapability(ToolCapabilityWrite)
}

// GetReadTools returns all read-only tools.
func (m *Mapping) GetReadTools() []ToolMapping {
	return m.GetToolsByCapability(ToolCapabilityRead)
}

// GetResourcesByParityStatus retrieves resources filtered by parity status.
func (m *Mapping) GetResourcesByParityStatus(status string) []ResourceMapping {
	var result []ResourceMapping
	for _, resource := range m.Resources {
		if resource.ParityStatus == status {
			result = append(result, resource)
		}
	}
	return result
}

// GetPromptsByParityStatus retrieves prompts filtered by parity status.
func (m *Mapping) GetPromptsByParityStatus(status string) []PromptMapping {
	var result []PromptMapping
	for _, prompt := range m.Prompts {
		if prompt.ParityStatus == status {
			result = append(result, prompt)
		}
	}
	return result
}

// ValidateParity checks if the MCP surface matches the CLI registry and skill catalog.
func (m *Mapping) ValidateParity(currentRegistryHash, currentSkillHash string) error {
	if m.CLIRegistryHash != currentRegistryHash {
		return fmt.Errorf("CLI registry hash mismatch: contract=%s, current=%s", m.CLIRegistryHash, currentRegistryHash)
	}
	if m.SkillCatalogHash != currentSkillHash {
		return fmt.Errorf("skill catalog hash mismatch: contract=%s, current=%s", m.SkillCatalogHash, currentSkillHash)
	}
	return nil
}

// GetParitySummary returns a summary of parity status across all surfaces.
func (m *Mapping) GetParitySummary() map[string]int {
	summary := make(map[string]int)

	for _, tool := range m.Tools {
		summary["tool_"+tool.ParityStatus]++
	}
	for _, resource := range m.Resources {
		summary["resource_"+resource.ParityStatus]++
	}
	for _, prompt := range m.Prompts {
		summary["prompt_"+prompt.ParityStatus]++
	}

	return summary
}

// IsFullParity returns true if all surfaces have full parity.
func (m *Mapping) IsFullParity() bool {
	for _, tool := range m.Tools {
		if tool.ParityStatus != ParityFull {
			return false
		}
	}
	for _, resource := range m.Resources {
		if resource.ParityStatus != ParityFull {
			return false
		}
	}
	for _, prompt := range m.Prompts {
		if prompt.ParityStatus != ParityFull {
			return false
		}
	}
	return true
}

// String returns a string representation of the mapping contract.
func (m *Mapping) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Mapping Contract v%s\n", m.SpecVersion))
	sb.WriteString(fmt.Sprintf("Generated: %s\n", m.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("CLI Registry Hash: %s\n", m.CLIRegistryHash))
	sb.WriteString(fmt.Sprintf("Skill Catalog Hash: %s\n", m.SkillCatalogHash))
	sb.WriteString(fmt.Sprintf("Tools: %d\n", len(m.Tools)))
	sb.WriteString(fmt.Sprintf("Resources: %d\n", len(m.Resources)))
	sb.WriteString(fmt.Sprintf("Prompts: %d\n", len(m.Prompts)))
	sb.WriteString("Parity Summary:\n")
	for status, count := range m.GetParitySummary() {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", status, count))
	}
	return sb.String()
}
