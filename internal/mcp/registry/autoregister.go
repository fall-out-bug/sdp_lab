// Package registry provides auto-registration of MCP tools from CLI registry.
package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// AutoRegister provides automatic MCP tool registration from CLI registry.
type AutoRegister struct {
	discovery *Discovery
}

// NewAutoRegister creates a new auto-registration instance.
func NewAutoRegister(binaryPath, repoRoot string) *AutoRegister {
	return &AutoRegister{
		discovery: NewDiscovery(binaryPath, repoRoot),
	}
}

// ToolGenerator generates MCP tool definitions from CLI commands.
type ToolGenerator struct {
	discovery *Discovery
}

// NewToolGenerator creates a new tool generator.
func NewToolGenerator(binaryPath, repoRoot string) *ToolGenerator {
	return &ToolGenerator{
		discovery: NewDiscovery(binaryPath, repoRoot),
	}
}

// GenerateTool generates an MCP tool definition from a CLI command.
func (g *ToolGenerator) GenerateTool(commandPath string) (mcp.Tool, error) {
	cmdInfo, err := g.discovery.DiscoverCommand(commandPath)
	if err != nil {
		return mcp.Tool{}, fmt.Errorf("discover command: %w", err)
	}

	// Generate MCP tool name: sdp_<command>
	mcpName := mcpToolName(commandPath)

	// Build tool definition with basic parameters
	tool := mcp.NewTool(mcpName, mcp.WithDescription(cmdInfo.Description))

	return tool, nil
}

// GenerateHandler generates a tool handler function for a CLI command.
func (g *ToolGenerator) GenerateHandler(commandPath string) ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// This is a placeholder - real implementation would use the executor interface
		return mcp.NewToolResultText(fmt.Sprintf("Would execute: %s", commandPath)), nil
	}
}

// ToolHandlerFunc is a function that handles an MCP tool request.
type ToolHandlerFunc func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

// RegisterAllTools automatically registers all discovered CLI commands as MCP tools.
func (a *AutoRegister) RegisterAllTools(server MCPServerInterface) error {
	commands, err := a.discovery.DiscoverAll()
	if err != nil {
		return fmt.Errorf("discover commands: %w", err)
	}

	generator := NewToolGenerator(a.discovery.binaryPath, a.discovery.repoRoot)

	for _, cmd := range commands {
		// Skip commands that shouldn't be exposed
		if skipCommand(cmd.Name) {
			continue
		}

		tool, err := generator.GenerateTool(cmd.Path)
		if err != nil {
			return fmt.Errorf("generate tool for %s: %w", cmd.Name, err)
		}

		handler := generator.GenerateHandler(cmd.Path)
		server.AddTool(tool, handler)
	}

	return nil
}

// MCPServerInterface defines the interface for adding tools to an MCP server.
type MCPServerInterface interface {
	AddTool(tool mcp.Tool, handler interface{})
}

// mcpToolName converts a CLI command path to an MCP tool name.
func mcpToolName(commandPath string) string {
	// Replace spaces with underscores and prefix with "sdp_"
	parts := strings.Fields(commandPath)
	return "sdp_" + strings.Join(parts, "_")
}

// skipCommand returns true if a command should not be exposed as an MCP tool.
func skipCommand(commandName string) bool {
	// Skip internal/debug commands
	skipList := []string{
		"help",
		"version",
		"completion",
		"__internal",
		"test",
	}

	for _, skip := range skipList {
		if strings.EqualFold(commandName, skip) {
			return true
		}
	}

	return false
}

// skipFlag returns true if a flag should not be exposed as an MCP parameter.
func skipFlag(flagName string) bool {
	// Skip flags that are not relevant for MCP exposure
	skipList := []string{
		"help",
		"h",
		"version",
		"v",
		"verbose",
		"quiet",
		"log-level",
		"output", // Handled separately for artifact persistence
		"format", // MCP always uses JSON
	}

	for _, skip := range skipList {
		if strings.EqualFold(flagName, skip) {
			return true
		}
	}

	return false
}

// convertRequestToArgs converts an MCP tool request to CLI arguments.
func convertRequestToArgs(req mcp.CallToolRequest, commandPath string) ([]string, error) {
	// Start with command path
	parts := strings.Fields(commandPath)
	args := append([]string{}, parts...)

	// Add parameters as flags
	if req.Params.Arguments != nil {
		// Type assertion for Arguments map
		argsMap, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid arguments type")
		}

		for key, value := range argsMap {
			flagName := "--" + strings.ReplaceAll(key, "_", "-")
			args = append(args, flagName)

			// Add value if not boolean
			if boolVal, ok := value.(bool); !ok || !boolVal {
				args = append(args, fmt.Sprintf("%v", value))
			}
		}
	}

	return args, nil
}