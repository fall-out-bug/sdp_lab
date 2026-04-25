package manifest

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaJSON []byte

// SchemaJSON returns the embedded JSON Schema bytes. Public so the `sdp
// manifest schema` subcommand can print it for downstream consumers.
func SchemaJSON() []byte {
	out := make([]byte, len(schemaJSON))
	copy(out, schemaJSON)
	return out
}

// LoadResult bundles parsed manifest with non-fatal warnings (e.g. orphan path
// references) collected during the existence pass.
type LoadResult struct {
	Manifest *Manifest
	Warnings []string
}

// Load reads and validates a manifest from disk. The repo root is used to
// resolve `path`/`script`/`system_prompt_path` fields for the existence pass.
// A missing referenced file is a fatal error; orphan files (present in repo
// but not in manifest) are out of scope here and handled by `sdp doctor`.
func Load(manifestPath, repoRoot string) (*LoadResult, error) {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", manifestPath, err)
	}
	return Parse(raw, repoRoot)
}

// Parse runs the same validation as Load against in-memory bytes. Tests use
// this to avoid filesystem fixtures for schema-only checks; pass an empty
// repoRoot to skip the existence pass.
func Parse(raw []byte, repoRoot string) (*LoadResult, error) {
	var generic any
	if err := yaml.Unmarshal(raw, &generic); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	jsonNormalized, err := yamlToJSON(generic)
	if err != nil {
		return nil, fmt.Errorf("normalize yaml→json: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("sdp-manifest.schema.json", strings.NewReader(string(schemaJSON))); err != nil {
		return nil, fmt.Errorf("register schema: %w", err)
	}
	schema, err := compiler.Compile("sdp-manifest.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	if err := schema.Validate(jsonNormalized); err != nil {
		return nil, formatSchemaError(err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	result := &LoadResult{Manifest: &manifest}
	if repoRoot != "" {
		if err := checkPaths(&manifest, repoRoot, result); err != nil {
			return nil, err
		}
		if err := checkUniqueness(&manifest); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func yamlToJSON(in any) (any, error) {
	switch v := in.(type) {
	case map[string]any:
		m := make(map[string]any, len(v))
		for k, val := range v {
			normalized, err := yamlToJSON(val)
			if err != nil {
				return nil, err
			}
			m[k] = normalized
		}
		return m, nil
	case map[any]any:
		m := make(map[string]any, len(v))
		for k, val := range v {
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string map key %v", k)
			}
			normalized, err := yamlToJSON(val)
			if err != nil {
				return nil, err
			}
			m[ks] = normalized
		}
		return m, nil
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			normalized, err := yamlToJSON(val)
			if err != nil {
				return nil, err
			}
			out[i] = normalized
		}
		return out, nil
	default:
		return v, nil
	}
}

func formatSchemaError(err error) error {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return fmt.Errorf("manifest schema validation: %w", err)
	}
	out, marshalErr := json.MarshalIndent(ve.DetailedOutput(), "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("manifest schema validation: %s", ve.Error())
	}
	return fmt.Errorf("manifest schema validation failed:\n%s", string(out))
}

func checkPaths(m *Manifest, repoRoot string, result *LoadResult) error {
	var missing []string
	check := func(field, p string) {
		if p == "" {
			return
		}
		full := filepath.Join(repoRoot, p)
		if _, err := os.Stat(full); err != nil {
			missing = append(missing, fmt.Sprintf("%s: %s", field, p))
		}
	}
	for _, s := range m.Skills {
		check(fmt.Sprintf("skills[%s].path", s.Name), s.Path)
	}
	for _, c := range m.Commands {
		check(fmt.Sprintf("commands[%s].path", c.Name), c.Path)
	}
	for _, a := range m.Agents {
		check(fmt.Sprintf("agents[%s].system_prompt_path", a.Name), a.SystemPromptPath)
	}
	for _, h := range m.Hooks {
		check(fmt.Sprintf("hooks[%s].script", h.Event), h.Script)
	}
	if len(missing) > 0 {
		return fmt.Errorf("manifest references missing paths:\n  - %s", strings.Join(missing, "\n  - "))
	}
	_ = result
	return nil
}

func checkUniqueness(m *Manifest) error {
	seen := func(label string, names []string) error {
		dup := map[string]int{}
		for _, n := range names {
			dup[n]++
		}
		var bad []string
		for n, c := range dup {
			if c > 1 {
				bad = append(bad, fmt.Sprintf("%s: %s (×%d)", label, n, c))
			}
		}
		if len(bad) > 0 {
			return fmt.Errorf("duplicate names: %s", strings.Join(bad, "; "))
		}
		return nil
	}
	skillNames := make([]string, len(m.Skills))
	for i, s := range m.Skills {
		skillNames[i] = s.Name
	}
	if err := seen("skill", skillNames); err != nil {
		return err
	}
	cmdNames := make([]string, len(m.Commands))
	for i, c := range m.Commands {
		cmdNames[i] = c.Name
	}
	if err := seen("command", cmdNames); err != nil {
		return err
	}
	agentNames := make([]string, len(m.Agents))
	for i, a := range m.Agents {
		agentNames[i] = a.Name
	}
	if err := seen("agent", agentNames); err != nil {
		return err
	}
	mcpNames := make([]string, len(m.MCPServers))
	for i, s := range m.MCPServers {
		mcpNames[i] = s.Name
	}
	return seen("mcp_server", mcpNames)
}
