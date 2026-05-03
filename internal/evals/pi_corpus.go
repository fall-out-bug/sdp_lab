package evals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// --- Schema ---

// validPICorpusSchemaJSON is the JSON Schema for the prompt-injection corpus.
// It captures case id, class, surface, trust source, allowed/forbidden tools,
// expected behavior, forbidden behavior, authoritative layer, and evidence expectation.
const validPICorpusSchemaJSON = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://sdp.dev/schema/prompt-injection-corpus/v1",
  "title": "Prompt Injection Corpus Case",
  "description": "Machine-readable prompt-injection corpus contract for F164 static validation",
  "type": "object",
  "required": ["id", "class", "surface", "untrusted_source", "allowed_tools", "expected_behavior", "evidence_expectation", "authoritative_layer", "risk"],
  "properties": {
	"id": {
	  "type": "string",
	  "minLength": 1,
	  "description": "Unique case identifier (e.g. PI-001)"
    },
    "class": {
      "type": "string",
      "enum": ["direct_override", "role_play_jailbreak", "prompt_extraction", "repo_indirect", "pr_diff_indirect", "ci_log_indirect", "beads_poisoning", "workstream_poisoning", "evidence_forgery", "cross_agent_handoff", "mcp_resource_injection", "context_poisoning", "supply_chain", "external_doc_injection", "qa_uat_override", "mcp_tool_description_poisoning", "multi_vector_pr_handoff", "benign_control"],
      "description": "Attack or control class"
    },
	"surface": {
	  "type": "string",
	  "minLength": 1,
	  "description": "Target SDP surface (e.g. review_skill, prompts/agents/security.md)"
    },
	"untrusted_source": {
	  "type": "string",
	  "minLength": 1,
	  "description": "Source of untrusted content (e.g. user_message, repo_text, ci_log)"
    },
    "trusted_instruction": {
      "type": "string",
      "description": "Summary of the trusted instruction being protected"
    },
    "attack_summary": {
      "type": "string",
      "description": "Brief description of the attack scenario"
    },
    "payload_safety": {
      "type": "string",
      "enum": ["sanitized", "mock", "not_applicable"],
      "description": "Whether the test payload is safe for storage"
    },
    "allowed_tools": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Allowlist of tools permitted for this case. Any tool not listed is denied."
    },
    "forbidden_tools": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Explicit list of tools denied for this case"
    },
    "expected_behavior": {
      "type": "array",
      "minItems": 1,
      "items": { "type": "string" },
      "description": "List of expected safe behaviors"
    },
    "forbidden_behavior": {
      "type": "array",
      "items": { "type": "string" },
      "description": "List of forbidden behaviors"
    },
	"evidence_expectation": {
	  "type": "string",
	  "minLength": 1,
	  "description": "Description of what evidence the case requires"
    },
    "authoritative_layer": {
      "type": "string",
      "enum": ["static", "mock_trace", "live_provider", "human_review"],
      "description": "Which evaluation layer is authoritative for pass/fail"
    },
    "risk": {
      "type": "string",
      "enum": ["critical", "major", "minor"],
      "description": "Risk severity level"
    }
  }
}`

// PICase represents a single prompt-injection corpus test case.
type PICase struct {
	ID                  string   `yaml:"id" json:"id"`
	Class               string   `yaml:"class" json:"class"`
	Surface             string   `yaml:"surface" json:"surface"`
	UntrustedSource     string   `yaml:"untrusted_source" json:"untrusted_source"`
	TrustedInstruction  string   `yaml:"trusted_instruction,omitempty" json:"trusted_instruction,omitempty"`
	AttackSummary       string   `yaml:"attack_summary,omitempty" json:"attack_summary,omitempty"`
	PayloadSafety       string   `yaml:"payload_safety,omitempty" json:"payload_safety,omitempty"`
	AllowedTools        []string `yaml:"allowed_tools" json:"allowed_tools"`
	ForbiddenTools      []string `yaml:"forbidden_tools,omitempty" json:"forbidden_tools,omitempty"`
	ExpectedBehavior    []string `yaml:"expected_behavior" json:"expected_behavior"`
	ForbiddenBehavior   []string `yaml:"forbidden_behavior,omitempty" json:"forbidden_behavior,omitempty"`
	EvidenceExpectation string   `yaml:"evidence_expectation" json:"evidence_expectation"`
	AuthoritativeLayer  string   `yaml:"authoritative_layer" json:"authoritative_layer"`
	Risk                string   `yaml:"risk" json:"risk"`
}

// --- Provenance Envelope (minimum fields from F164-02 ownership) ---

// SourceType distinguishes the origin of evidence.
type SourceType string

const (
	SourceTypeToolResult     SourceType = "tool_result"
	SourceTypeHumanApproval  SourceType = "human_approval"
	SourceTypePolicyDecision SourceType = "policy_decision"
	SourceTypeModelJudgment  SourceType = "model_judgment"
)

// ArtifactRef references an artifact with type and path.
type ArtifactRef struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// ProvenanceEnvelope carries minimum provenance fields for evidence gates.
// Owned by F164-02; blocks F164-03.
type ProvenanceEnvelope struct {
	SourceType  SourceType  `json:"source_type"`
	SourceID    string      `json:"source_id"`
	ArtifactRef ArtifactRef `json:"artifact_ref"`
	ContentHash string      `json:"content_hash,omitempty"`
	TrustLabel  TrustLabel  `json:"trust_label"`
	CreatedBy   string      `json:"created_by"`
}

// TrustLabel classifies the trust level of content.
type TrustLabel string

const (
	TrustLabelTrustedInstruction    TrustLabel = "trusted_instruction"
	TrustLabelUntrustedContent      TrustLabel = "untrusted_content"
	TrustLabelDeterministicEvidence TrustLabel = "deterministic_evidence"
	TrustLabelJudgment              TrustLabel = "judgment"
)

// IsValidTrustLabel returns true if the label is a recognized trust label.
func IsValidTrustLabel(label string) bool {
	switch TrustLabel(label) {
	case TrustLabelTrustedInstruction, TrustLabelUntrustedContent, TrustLabelDeterministicEvidence, TrustLabelJudgment:
		return true
	default:
		return false
	}
}

// IsValidAuthoritativeLayer returns true if the layer is a recognized authoritative layer.
func IsValidAuthoritativeLayer(layer string) bool {
	switch layer {
	case "static", "mock_trace", "live_provider", "human_review":
		return true
	default:
		return false
	}
}

// IsValidRiskLevel returns true if the level is a recognized risk level.
func IsValidRiskLevel(level string) bool {
	switch level {
	case "critical", "major", "minor":
		return true
	default:
		return false
	}
}

// --- Attack Class Display Names ---

// AttackClassDisplayNames maps attack class identifiers to human-readable names.
var AttackClassDisplayNames = map[string]string{
	"direct_override":                "Direct Override",
	"role_play_jailbreak":            "Role-Play Jailbreak",
	"prompt_extraction":              "Prompt Extraction",
	"repo_indirect":                  "Indirect Repo Injection",
	"pr_diff_indirect":               "PR Diff Injection",
	"ci_log_indirect":                "CI/Log Injection",
	"beads_poisoning":                "Beads Workstream Poisoning",
	"workstream_poisoning":           "Workstream Poisoning",
	"evidence_forgery":               "Evidence Forgery",
	"cross_agent_handoff":            "Cross-Agent Handoff Poisoning",
	"mcp_resource_injection":         "MCP Resource Injection",
	"context_poisoning":              "Context/Index Poisoning",
	"supply_chain":                   "Prompt Bundle Supply Chain",
	"external_doc_injection":         "External Doc Injection",
	"qa_uat_override":                "QA/UAT Override",
	"mcp_tool_description_poisoning": "MCP Tool-Description Poisoning",
	"multi_vector_pr_handoff":        "Multi-Vector PR/Handoff Attack",
	"benign_control":                 "Benign Control Case",
	"keyword_block_only":             "Keyword-Block-Only Classification (FAIL fixture)",
}

// --- PromptOps Schema Extension Decision ---

// PI corpus uses its own dedicated schema rather than extending promptops-check.
// Rationale: PI corpus cases have fundamentally different fields (attack class,
// trust source, allowed/forbidden tools) vs PromptOps checks (check_id, status, note).
// Promptops-check.schema.json remains for downstream tool consumption (e.g. review verdict output).
// Both schemas coexist: PI corpus schema for static validation, promptops-check for review output.

// --- Validator ---

// PICorpusValidator validates prompt-injection corpus cases against the schema.
type PICorpusValidator struct {
	schemaPath string
	casesDir   string
}

// NewPICorpusValidator creates a new PI corpus validator.
func NewPICorpusValidator(schemaPath, casesDir string) *PICorpusValidator {
	return &PICorpusValidator{
		schemaPath: schemaPath,
		casesDir:   casesDir,
	}
}

// ValidateResult holds the outcome of validating one corpus case or corpus-level rule.
type ValidateResult struct {
	Case   string
	Pass   bool
	Reason string
}

// ValidateAll validates all YAML case files in the cases directory.
// Returns per-case results plus corpus-level results (benign control enforcement, supply-chain enforcement).
func (v *PICorpusValidator) ValidateAll() []ValidateResult {
	var results []ValidateResult

	// Load schema
	schemaData, err := os.ReadFile(v.schemaPath)
	if err != nil {
		return []ValidateResult{{Case: "schema", Pass: false, Reason: fmt.Sprintf("cannot read schema: %v", err)}}
	}

	schema, err := compilePICorpusSchema(schemaData)
	if err != nil {
		return []ValidateResult{{Case: "schema", Pass: false, Reason: err.Error()}}
	}

	// Load cases
	pattern := filepath.Join(v.casesDir, "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return []ValidateResult{{Case: "corpus", Pass: false, Reason: fmt.Sprintf("glob cases: %v", err)}}
	}

	var cases []PICase
	var caseResults []ValidateResult
	for _, p := range matches {
		data, err := os.ReadFile(p)
		if err != nil {
			caseResults = append(caseResults, ValidateResult{Case: filepath.Base(p), Pass: false, Reason: fmt.Sprintf("read file: %v", err)})
			continue
		}
		var raw any
		if err := yaml.Unmarshal(data, &raw); err != nil {
			caseResults = append(caseResults, ValidateResult{Case: filepath.Base(p), Pass: false, Reason: fmt.Sprintf("YAML parse: %v", err)})
			continue
		}
		normalized, err := yamlValueToJSON(raw)
		if err != nil {
			caseResults = append(caseResults, ValidateResult{Case: filepath.Base(p), Pass: false, Reason: fmt.Sprintf("YAML normalize: %v", err)})
			continue
		}
		caseID := piCaseID(filepath.Base(p), normalized)
		if err := schema.Validate(normalized); err != nil {
			reason := fmt.Sprintf("schema validation: %v", err)
			if piCaseClass(normalized) == "keyword_block_only" {
				reason = "keyword_block_only is not a valid attack class"
			}
			caseResults = append(caseResults, ValidateResult{Case: caseID, Pass: false, Reason: reason})
			continue
		}
		dataJSON, err := json.Marshal(normalized)
		if err != nil {
			caseResults = append(caseResults, ValidateResult{Case: caseID, Pass: false, Reason: fmt.Sprintf("JSON marshal: %v", err)})
			continue
		}
		var c PICase
		if err := json.Unmarshal(dataJSON, &c); err != nil {
			caseResults = append(caseResults, ValidateResult{Case: caseID, Pass: false, Reason: fmt.Sprintf("decode case: %v", err)})
			continue
		}
		cases = append(cases, c)
		caseResults = append(caseResults, v.validateCase(c, normalized))
	}

	// Corpus-level: at least one benign control case required
	hasBenign := false
	for _, c := range cases {
		if c.Class == "benign_control" {
			hasBenign = true
			break
		}
	}
	if !hasBenign && len(cases) > 0 {
		caseResults = append(caseResults, ValidateResult{Case: "corpus", Pass: false, Reason: "corpus requires at least one benign_control case"})
	}

	// Corpus-level: at least one supply-chain case required
	hasSupplyChain := false
	for _, c := range cases {
		if c.Class == "supply_chain" {
			hasSupplyChain = true
			break
		}
	}
	if !hasSupplyChain && len(cases) > 0 {
		caseResults = append(caseResults, ValidateResult{Case: "corpus", Pass: false, Reason: "corpus requires at least one supply_chain case"})
	}

	return append(results, caseResults...)
}

// validateCase checks a single PICase against the schema requirements.
func (v *PICorpusValidator) validateCase(c PICase, normalized any) ValidateResult {
	var failures []string

	fields, _ := normalized.(map[string]any)
	if _, ok := fields["allowed_tools"]; !ok {
		failures = append(failures, "missing allowed_tools")
	}

	if c.ID == "" {
		failures = append(failures, "missing id")
	}
	if c.Class == "" {
		failures = append(failures, "missing class")
	}
	if c.Surface == "" {
		failures = append(failures, "missing surface")
	}
	if c.UntrustedSource == "" {
		failures = append(failures, "missing untrusted_source (trust field required)")
	}
	if c.EvidenceExpectation == "" {
		failures = append(failures, "missing evidence_expectation")
	}
	if c.AuthoritativeLayer == "" {
		failures = append(failures, "missing authoritative_layer")
	}
	if c.Risk == "" {
		failures = append(failures, "missing risk")
	}
	if len(c.ExpectedBehavior) == 0 {
		failures = append(failures, "expected_behavior must have at least one entry")
	}

	validClasses := map[string]bool{
		"direct_override":                true,
		"role_play_jailbreak":            true,
		"prompt_extraction":              true,
		"repo_indirect":                  true,
		"pr_diff_indirect":               true,
		"ci_log_indirect":                true,
		"beads_poisoning":                true,
		"workstream_poisoning":           true,
		"evidence_forgery":               true,
		"cross_agent_handoff":            true,
		"mcp_resource_injection":         true,
		"context_poisoning":              true,
		"supply_chain":                   true,
		"external_doc_injection":         true,
		"qa_uat_override":                true,
		"mcp_tool_description_poisoning": true,
		"multi_vector_pr_handoff":        true,
		"benign_control":                 true,
	}
	if c.Class != "" && !validClasses[c.Class] {
		failures = append(failures, fmt.Sprintf("invalid class %q (keyword_block_only is not a valid attack class)", c.Class))
	}
	if c.AuthoritativeLayer != "" && !IsValidAuthoritativeLayer(c.AuthoritativeLayer) {
		failures = append(failures, fmt.Sprintf("invalid authoritative_layer %q", c.AuthoritativeLayer))
	}
	if c.Risk != "" && !IsValidRiskLevel(c.Risk) {
		failures = append(failures, fmt.Sprintf("invalid risk level %q", c.Risk))
	}

	if len(failures) > 0 {
		return ValidateResult{Case: c.ID, Pass: false, Reason: strings.Join(failures, "; ")}
	}
	return ValidateResult{Case: c.ID, Pass: true}
}

func compilePICorpusSchema(schemaData []byte) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("prompt-injection-corpus.schema.json", bytes.NewReader(schemaData)); err != nil {
		return nil, fmt.Errorf("register schema: %v", err)
	}
	schema, err := compiler.Compile("prompt-injection-corpus.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile schema: %v", err)
	}
	return schema, nil
}

func yamlValueToJSON(in any) (any, error) {
	switch v := in.(type) {
	case map[string]any:
		m := make(map[string]any, len(v))
		for k, val := range v {
			normalized, err := yamlValueToJSON(val)
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
			normalized, err := yamlValueToJSON(val)
			if err != nil {
				return nil, err
			}
			m[ks] = normalized
		}
		return m, nil
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			normalized, err := yamlValueToJSON(val)
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

func piCaseID(fallback string, normalized any) string {
	m, ok := normalized.(map[string]any)
	if !ok {
		return fallback
	}
	id, ok := m["id"].(string)
	if !ok || id == "" {
		return fallback
	}
	return id
}

func piCaseClass(normalized any) string {
	m, ok := normalized.(map[string]any)
	if !ok {
		return ""
	}
	class, _ := m["class"].(string)
	return class
}
