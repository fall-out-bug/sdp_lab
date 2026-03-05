// Package guard provides scope enforcement and permission management.
package guard

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// PermissionAction represents the OMO permission action type.
type PermissionAction string

const (
	// ActionAllow auto-approves the operation.
	ActionAllow PermissionAction = "allow"
	// ActionDeny blocks the operation.
	ActionDeny PermissionAction = "deny"
	// ActionAsk prompts for approval via boundary gate.
	ActionAsk PermissionAction = "ask"
)

// PermissionRule represents a single permission rule.
type PermissionRule struct {
	// Pattern is a glob or regex pattern to match against tool/file paths.
	Pattern string `json:"pattern"`
	
	// Action is the permission action to take.
	Action PermissionAction `json:"action"`
	
	// IsRegex indicates if pattern is a regex (default: glob).
	IsRegex bool `json:"is_regex,omitempty"`
	
	// Reason is an optional explanation for the rule.
	Reason string `json:"reason,omitempty"`
	
	// compiledRegex is the compiled regex pattern (cached).
	compiledRegex *regexp.Regexp
}

// PermissionConfig represents the OMO permission configuration.
type PermissionConfig struct {
	// Rules are the permission rules in order of precedence.
	Rules []PermissionRule `json:"rules"`
	
	// DefaultAction is the action when no rule matches.
	DefaultAction PermissionAction `json:"default_action"`
	
	// AuditLog is the path to the audit log file.
	AuditLog string `json:"audit_log,omitempty"`
}

// PermissionBridge bridges OMO permissions to SDP guard.
type PermissionBridge struct {
	mu     sync.RWMutex
	config *PermissionConfig
	
	// onAsk is called when an "ask" action is triggered.
	onAsk func(ctx context.Context, req PermissionRequest) (PermissionAction, error)
	
	// auditLogger writes permission decisions.
	auditLogger *slog.Logger
	auditFile   *os.File
}

// PermissionRequest represents a permission check request.
type PermissionRequest struct {
	// ToolName is the name of the tool being called.
	ToolName string `json:"tool_name"`
	
	// FilePath is the file path being accessed (if applicable).
	FilePath string `json:"file_path,omitempty"`
	
	// Command is the command being executed (if Bash tool).
	Command string `json:"command,omitempty"`
	
	// SessionID is the session making the request.
	SessionID string `json:"session_id"`
	
	// WorkstreamID is the active workstream.
	WorkstreamID string `json:"workstream_id"`
	
	// Timestamp is when the request was made.
	Timestamp time.Time `json:"timestamp"`
}

// PermissionDecision represents the result of a permission check.
type PermissionDecision struct {
	// Action is the decided action.
	Action PermissionAction `json:"action"`
	
	// Rule is the matching rule (if any).
	Rule *PermissionRule `json:"rule,omitempty"`
	
	// Reason explains the decision.
	Reason string `json:"reason"`
	
	// Timestamp is when the decision was made.
	Timestamp time.Time `json:"timestamp"`
}

// NewPermissionBridge creates a new permission bridge.
func NewPermissionBridge(config *PermissionConfig) (*PermissionBridge, error) {
	pb := &PermissionBridge{
		config: config,
	}
	
	if config == nil {
		pb.config = &PermissionConfig{
			DefaultAction: ActionAsk,
		}
	}
	
	// Compile regex patterns
	for i := range pb.config.Rules {
		if pb.config.Rules[i].IsRegex {
			compiled, err := regexp.Compile(pb.config.Rules[i].Pattern)
			if err != nil {
				return nil, fmt.Errorf("compile regex %q: %w", pb.config.Rules[i].Pattern, err)
			}
			pb.config.Rules[i].compiledRegex = compiled
		}
	}
	
	// Setup audit logger
	if pb.config.AuditLog != "" {
		f, err := os.OpenFile(pb.config.AuditLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("open audit log: %w", err)
		}
		pb.auditFile = f
		pb.auditLogger = slog.New(slog.NewJSONHandler(f, nil))
	}
	
	return pb, nil
}

// SetOnAsk sets the callback for "ask" actions.
func (pb *PermissionBridge) SetOnAsk(fn func(ctx context.Context, req PermissionRequest) (PermissionAction, error)) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.onAsk = fn
}

// Check checks permission for a request.
func (pb *PermissionBridge) Check(ctx context.Context, req PermissionRequest) (*PermissionDecision, error) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	
	// Ensure timestamp is set
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}
	
	// Build match target (file path or command)
	target := req.FilePath
	if target == "" {
		target = req.Command
	}
	
	// Find matching rule
	for i := range pb.config.Rules {
		rule := &pb.config.Rules[i]
		if pb.matches(rule, req.ToolName, target) {
			decision := &PermissionDecision{
				Action:    rule.Action,
				Rule:      rule,
				Reason:    rule.Reason,
				Timestamp: time.Now(),
			}
			
			// Handle "ask" action via callback
			if rule.Action == ActionAsk && pb.onAsk != nil {
				action, err := pb.onAsk(ctx, req)
				if err != nil {
					return nil, err
				}
				decision.Action = action
				decision.Reason = fmt.Sprintf("ask escalated to %s", action)
			}
			
			// Log decision
			pb.logDecision(req, decision)
			
			return decision, nil
		}
	}
	
	// No rule matched, use default
	decision := &PermissionDecision{
		Action:    pb.config.DefaultAction,
		Reason:    "no matching rule",
		Timestamp: time.Now(),
	}
	
	// Handle default "ask"
	if pb.config.DefaultAction == ActionAsk && pb.onAsk != nil {
		action, err := pb.onAsk(ctx, req)
		if err != nil {
			return nil, err
		}
		decision.Action = action
		decision.Reason = fmt.Sprintf("default ask escalated to %s", action)
	}
	
	// Log decision
	pb.logDecision(req, decision)
	
	return decision, nil
}

// matches checks if a rule matches the given tool and target.
func (pb *PermissionBridge) matches(rule *PermissionRule, toolName, target string) bool {
	// Check tool name (always glob match)
	if rule.Pattern == "*" || rule.Pattern == "" {
		return true
	}
	
	// Check if pattern includes tool prefix (e.g., "edit:*")
	if strings.Contains(rule.Pattern, ":") {
		parts := strings.SplitN(rule.Pattern, ":", 2)
		if len(parts) == 2 {
			toolPattern := parts[0]
			targetPattern := parts[1]
			
			// Match tool name
			if !pb.matchPattern(toolPattern, false, toolName) {
				return false
			}
			
			// Match target
			return pb.matchPattern(targetPattern, rule.IsRegex, target)
		}
	}
	
	// Match against target only
	return pb.matchPattern(rule.Pattern, rule.IsRegex, target)
}

// matchPattern matches a pattern against a target.
func (pb *PermissionBridge) matchPattern(pattern string, isRegex bool, target string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}
	
	if isRegex {
		// Find the compiled regex
		for _, rule := range pb.config.Rules {
			if rule.Pattern == pattern && rule.compiledRegex != nil {
				return rule.compiledRegex.MatchString(target)
			}
		}
		return false
	}
	
	// Glob matching
	matched, err := filepath.Match(pattern, target)
	if err != nil {
		return false
	}
	return matched
}

// logDecision logs a permission decision.
func (pb *PermissionBridge) logDecision(req PermissionRequest, decision *PermissionDecision) {
	if pb.auditLogger == nil {
		return
	}
	
	pb.auditLogger.Info("permission_decision",
		"tool", req.ToolName,
		"file", req.FilePath,
		"command", req.Command,
		"session", req.SessionID,
		"workstream", req.WorkstreamID,
		"action", decision.Action,
		"reason", decision.Reason,
		"timestamp", decision.Timestamp,
	)
}

// Close closes the audit log file.
func (pb *PermissionBridge) Close() error {
	if pb.auditFile != nil {
		return pb.auditFile.Close()
	}
	return nil
}

// DefaultPermissionConfig returns a sensible default permission config.
func DefaultPermissionConfig() *PermissionConfig {
	return &PermissionConfig{
		Rules: []PermissionRule{
			// Allow read operations
			{Pattern: "read:*", Action: ActionAllow, Reason: "read operations are safe"},
			// Allow common safe files
			{Pattern: "*.md", Action: ActionAllow, Reason: "markdown files are safe"},
			{Pattern: "*.json", Action: ActionAllow, Reason: "json files are safe"},
			// Ask for write operations
			{Pattern: "edit:*", Action: ActionAsk, Reason: "write operations need approval"},
			{Pattern: "write:*", Action: ActionAsk, Reason: "write operations need approval"},
			// Deny dangerous commands
			{Pattern: "bash:rm -rf /*", Action: ActionDeny, Reason: "dangerous command"},
			{Pattern: "bash:*sudo*", Action: ActionDeny, Reason: "sudo requires approval"},
		},
		DefaultAction: ActionAsk,
	}
}
