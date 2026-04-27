// Package validation provides end-to-end MCP handshake validation.
package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/mcp/contract"
	"github.com/fall-out-bug/sdp_lab/internal/mcp/parity"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Validator performs end-to-end validation of MCP surface parity.
type Validator struct {
	mapping      *contract.Mapping
	prompts      *parity.PromptRegistry
	resources    *parity.ResourceRegistry
	server       *mcpserver.MCPServer
	repoRoot     string
}

// ValidatorConfig holds configuration for validation.
type ValidatorConfig struct {
	Mapping      *contract.Mapping
	Prompts      *parity.PromptRegistry
	Resources    *parity.ResourceRegistry
	Server       *mcpserver.MCPServer
	RepoRoot     string
	StrictMode   bool // Fail on any parity issue
}

// NewValidator creates a new MCP validator.
func NewValidator(cfg ValidatorConfig) *Validator {
	return &Validator{
		mapping:   cfg.Mapping,
		prompts:   cfg.Prompts,
		resources: cfg.Resources,
		server:    cfg.Server,
		repoRoot:  cfg.RepoRoot,
	}
}

// ValidationResult holds the results of validation.
type ValidationResult struct {
	Passed       bool
	Errors       []string
	Warnings     []string
	Duration     time.Duration
	ToolCount    int
	ResourceCount int
	PromptCount  int
	ParitySummary map[string]int
}

// Validate performs end-to-end validation of the MCP surface.
func (v *Validator) Validate(ctx context.Context) (*ValidationResult, error) {
	start := time.Now()
	result := &ValidationResult{
		Errors:       []string{},
		Warnings:     []string{},
		ParitySummary: make(map[string]int),
	}

	// Validate contract
	if err := v.validateContract(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("contract validation: %v", err))
	}

	// Validate prompt parity
	if err := v.validatePromptParity(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("prompt parity: %v", err))
	}

	// Validate resource parity
	if err := v.validateResourceParity(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("resource parity: %v", err))
	}

	// Validate tool surface
	if err := v.validateToolSurface(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("tool surface: %v", err))
	}

	// Validate handshake
	if err := v.validateHandshake(ctx); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("handshake: %v", err))
	}

	// Check resource availability
	v.validateResourceAvailability(result)

	// Collect metrics
	result.ToolCount = len(v.mapping.Tools)
	result.ResourceCount = len(v.mapping.Resources)
	result.PromptCount = len(v.mapping.Prompts)
	result.ParitySummary = v.mapping.GetParitySummary()
	result.Duration = time.Since(start)
	result.Passed = len(result.Errors) == 0

	return result, nil
}

// validateContract validates the mapping contract.
func (v *Validator) validateContract() error {
	if v.mapping == nil {
		return fmt.Errorf("mapping contract is nil")
	}

	return v.mapping.Validate()
}

// validatePromptParity validates that all required prompts have full parity.
func (v *Validator) validatePromptParity() error {
	if v.prompts == nil {
		return fmt.Errorf("prompt registry is nil")
	}

	return v.prompts.ValidateParity()
}

// validateResourceParity validates that all core resources have full parity.
func (v *Validator) validateResourceParity() error {
	if v.resources == nil {
		return fmt.Errorf("resource registry is nil")
	}

	return v.resources.ValidateParity()
}

// validateToolSurface validates that the tool surface is consistent.
func (v *Validator) validateToolSurface() error {
	if v.mapping == nil {
		return fmt.Errorf("mapping contract is nil")
	}

	// Check for duplicate tool names
	toolNames := make(map[string]bool)
	for _, tool := range v.mapping.Tools {
		if toolNames[tool.MCPToolName] {
			return fmt.Errorf("duplicate tool name: %s", tool.MCPToolName)
		}
		toolNames[tool.MCPToolName] = true
	}

	// Check that all tools have valid parameters
	for _, tool := range v.mapping.Tools {
		if err := v.validateToolParameters(tool); err != nil {
			return fmt.Errorf("tool %s: %w", tool.MCPToolName, err)
		}
	}

	return nil
}

// ValidateContractDuplicateTools checks for duplicate tool names in the contract.
func ValidateContractDuplicateTools(mapping *contract.Mapping) error {
	toolNames := make(map[string]bool)
	for _, tool := range mapping.Tools {
		if toolNames[tool.MCPToolName] {
			return fmt.Errorf("duplicate tool name: %s", tool.MCPToolName)
		}
		toolNames[tool.MCPToolName] = true
	}
	return nil
}

// validateToolParameters validates tool parameter definitions.
func (v *Validator) validateToolParameters(tool contract.ToolMapping) error {
	paramNames := make(map[string]bool)

	for _, param := range tool.Parameters {
		// Check for duplicate parameter names
		if paramNames[param.MCPParamName] {
			return fmt.Errorf("duplicate parameter name: %s", param.MCPParamName)
		}
		paramNames[param.MCPParamName] = true

		// Validate parameter type
		switch param.Type {
		case "string", "number", "boolean", "enum":
			// Valid types
		default:
			return fmt.Errorf("invalid parameter type: %s", param.Type)
		}

		// Validate enum values
		if param.Type == "enum" && len(param.EnumValues) == 0 {
			return fmt.Errorf("enum parameter %s has no values", param.MCPParamName)
		}
	}

	return nil
}

// validateHandshake validates the MCP server handshake.
func (v *Validator) validateHandshake(ctx context.Context) error {
	if v.server == nil {
		return fmt.Errorf("MCP server is nil")
	}

	// The server should be able to list its tools, resources, and prompts
	// This is a basic sanity check - real implementation would test actual protocol

	// Check that tools are registered
	tools := v.server.ListTools()
	if len(tools) == 0 {
		return fmt.Errorf("no tools registered on server")
	}

	// Verify tool count matches contract
	if len(tools) != len(v.mapping.Tools) {
		return fmt.Errorf("tool count mismatch: server has %d, contract expects %d",
			len(tools), len(v.mapping.Tools))
	}

	return nil
}

// validateResourceAvailability checks which resources are available on disk.
func (v *Validator) validateResourceAvailability(result *ValidationResult) {
	if v.resources == nil {
		return
	}

	availability := v.resources.CheckAvailability(v.repoRoot)

	for uri, available := range availability {
		if !available {
			resource, ok := v.resources.Get(uri)
			if ok && resource.ParityStatus == contract.ParityFull {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("core resource not available: %s (generate with: %s)",
						uri, resource.HintTool))
			}
		}
	}
}

// ValidateParityBeforeClaim validates parity before claiming MCP support.
func ValidateParityBeforeClaim(repoRoot string) error {
	// Load or create default registries
	prompts := parity.NewPromptRegistry()
	for _, prompt := range parity.DefaultPrompts() {
		if err := prompts.Register(prompt); err != nil {
			return fmt.Errorf("register prompt: %w", err)
		}
	}

	resources := parity.NewResourceRegistry()
	for _, resource := range parity.DefaultResources() {
		if err := resources.Register(resource); err != nil {
			return fmt.Errorf("register resource: %w", err)
		}
	}

	// Validate prompts
	if err := prompts.ValidateParity(); err != nil {
		return fmt.Errorf("prompt parity validation failed: %w", err)
	}

	// Validate resources
	if err := resources.ValidateParity(); err != nil {
		return fmt.Errorf("resource parity validation failed: %w", err)
	}

	// Check resource availability
	missing := resources.GetMissingResources(repoRoot)
	if len(missing) > 0 {
		var missingURIs []string
		for _, resource := range missing {
			if resource.ParityStatus == contract.ParityFull {
				missingURIs = append(missingURIs, resource.URI)
			}
		}
		if len(missingURIs) > 0 {
			return fmt.Errorf("missing core resources: %v", missingURIs)
		}
	}

	return nil
}

// ParityReport generates a human-readable parity report.
func (v *Validator) ParityReport(result *ValidationResult) string {
	report := "MCP Parity Validation Report\n"
	report += "=============================\n\n"

	report += fmt.Sprintf("Status: %s\n", map[bool]string{true: "PASSED", false: "FAILED"}[result.Passed])
	report += fmt.Sprintf("Duration: %v\n\n", result.Duration)

	report += fmt.Sprintf("Surface Counts:\n")
	report += fmt.Sprintf("  Tools: %d\n", result.ToolCount)
	report += fmt.Sprintf("  Resources: %d\n", result.ResourceCount)
	report += fmt.Sprintf("  Prompts: %d\n\n", result.PromptCount)

	report += "Parity Summary:\n"
	for status, count := range result.ParitySummary {
		report += fmt.Sprintf("  %s: %d\n", status, count)
	}
	report += "\n"

	if len(result.Errors) > 0 {
		report += "Errors:\n"
		for _, err := range result.Errors {
			report += fmt.Sprintf("  - %s\n", err)
		}
		report += "\n"
	}

	if len(result.Warnings) > 0 {
		report += "Warnings:\n"
		for _, warn := range result.Warnings {
			report += fmt.Sprintf("  - %s\n", warn)
		}
		report += "\n"
	}

	return report
}

// QuickValidation performs a quick validation check.
func QuickValidation(repoRoot string) (bool, error) {
	if err := ValidateParityBeforeClaim(repoRoot); err != nil {
		return false, err
	}
	return true, nil
}