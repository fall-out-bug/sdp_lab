package guard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewPermissionBridge(t *testing.T) {
	config := &PermissionConfig{
		Rules: []PermissionRule{
			{Pattern: "*.go", Action: ActionAllow},
			{Pattern: "edit:*.md", Action: ActionAsk},
		},
		DefaultAction: ActionDeny,
	}
	
	pb, err := NewPermissionBridge(config)
	if err != nil {
		t.Fatalf("NewPermissionBridge failed: %v", err)
	}
	defer pb.Close()
	
	if pb.config.DefaultAction != ActionDeny {
		t.Errorf("DefaultAction = %v, want %v", pb.config.DefaultAction, ActionDeny)
	}
}

func TestNewPermissionBridgeNilConfig(t *testing.T) {
	pb, err := NewPermissionBridge(nil)
	if err != nil {
		t.Fatalf("NewPermissionBridge failed: %v", err)
	}
	defer pb.Close()
	
	if pb.config.DefaultAction != ActionAsk {
		t.Errorf("DefaultAction = %v, want %v", pb.config.DefaultAction, ActionAsk)
	}
}

func TestNewPermissionBridgeRegex(t *testing.T) {
	config := &PermissionConfig{
		Rules: []PermissionRule{
			{Pattern: `^edit:.*\.go$`, Action: ActionAllow, IsRegex: true},
		},
	}
	
	pb, err := NewPermissionBridge(config)
	if err != nil {
		t.Fatalf("NewPermissionBridge failed: %v", err)
	}
	defer pb.Close()
	
	// Check that regex was compiled
	if pb.config.Rules[0].compiledRegex == nil {
		t.Error("regex should be compiled")
	}
}

func TestNewPermissionBridgeInvalidRegex(t *testing.T) {
	config := &PermissionConfig{
		Rules: []PermissionRule{
			{Pattern: `[invalid(`, Action: ActionAllow, IsRegex: true},
		},
	}
	
	_, err := NewPermissionBridge(config)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestCheckAllow(t *testing.T) {
	config := &PermissionConfig{
		Rules: []PermissionRule{
			{Pattern: "*.go", Action: ActionAllow},
		},
		DefaultAction: ActionDeny,
	}
	
	pb, err := NewPermissionBridge(config)
	if err != nil {
		t.Fatalf("NewPermissionBridge failed: %v", err)
	}
	defer pb.Close()
	
	decision, err := pb.Check(context.Background(), PermissionRequest{
		ToolName: "edit",
		FilePath: "test.go",
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	
	if decision.Action != ActionAllow {
		t.Errorf("Action = %v, want %v", decision.Action, ActionAllow)
	}
}

func TestCheckDeny(t *testing.T) {
	config := &PermissionConfig{
		Rules: []PermissionRule{
			{Pattern: "*.secret", Action: ActionDeny},
		},
		DefaultAction: ActionAllow,
	}
	
	pb, err := NewPermissionBridge(config)
	if err != nil {
		t.Fatalf("NewPermissionBridge failed: %v", err)
	}
	defer pb.Close()
	
	decision, err := pb.Check(context.Background(), PermissionRequest{
		ToolName: "edit",
		FilePath: "api.secret",
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	
	if decision.Action != ActionDeny {
		t.Errorf("Action = %v, want %v", decision.Action, ActionDeny)
	}
}

func TestCheckAskWithCallback(t *testing.T) {
	config := &PermissionConfig{
		Rules: []PermissionRule{
			{Pattern: "*.md", Action: ActionAsk},
		},
		DefaultAction: ActionDeny,
	}
	
	pb, err := NewPermissionBridge(config)
	if err != nil {
		t.Fatalf("NewPermissionBridge failed: %v", err)
	}
	defer pb.Close()
	
	askCalled := false
	pb.SetOnAsk(func(ctx context.Context, req PermissionRequest) (PermissionAction, error) {
		askCalled = true
		return ActionAllow, nil
	})
	
	decision, err := pb.Check(context.Background(), PermissionRequest{
		ToolName: "edit",
		FilePath: "README.md",
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	
	if !askCalled {
		t.Error("OnAsk callback should be called")
	}
	
	if decision.Action != ActionAllow {
		t.Errorf("Action = %v, want %v", decision.Action, ActionAllow)
	}
}

func TestCheckDefaultAction(t *testing.T) {
	config := &PermissionConfig{
		Rules:         []PermissionRule{},
		DefaultAction: ActionDeny,
	}
	
	pb, err := NewPermissionBridge(config)
	if err != nil {
		t.Fatalf("NewPermissionBridge failed: %v", err)
	}
	defer pb.Close()
	
	decision, err := pb.Check(context.Background(), PermissionRequest{
		ToolName: "edit",
		FilePath: "unknown.txt",
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	
	if decision.Action != ActionDeny {
		t.Errorf("Action = %v, want %v", decision.Action, ActionDeny)
	}
}

func TestCheckWithToolPrefix(t *testing.T) {
	config := &PermissionConfig{
		Rules: []PermissionRule{
			{Pattern: "edit:*.go", Action: ActionAllow},
			{Pattern: "read:*", Action: ActionAllow},
		},
		DefaultAction: ActionDeny,
	}
	
	pb, err := NewPermissionBridge(config)
	if err != nil {
		t.Fatalf("NewPermissionBridge failed: %v", err)
	}
	defer pb.Close()
	
	// Should match edit:*.go
	decision, err := pb.Check(context.Background(), PermissionRequest{
		ToolName: "edit",
		FilePath: "main.go",
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if decision.Action != ActionAllow {
		t.Errorf("Action = %v, want %v", decision.Action, ActionAllow)
	}
	
	// Should not match (bash, not edit)
	decision, err = pb.Check(context.Background(), PermissionRequest{
		ToolName: "bash",
		Command:  "main.go",
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if decision.Action != ActionDeny {
		t.Errorf("Action = %v, want %v", decision.Action, ActionDeny)
	}
}

func TestAuditLog(t *testing.T) {
	tmpDir := t.TempDir()
	auditPath := filepath.Join(tmpDir, "audit.log")
	
	config := &PermissionConfig{
		Rules: []PermissionRule{
			{Pattern: "*", Action: ActionAllow},
		},
		AuditLog: auditPath,
	}
	
	pb, err := NewPermissionBridge(config)
	if err != nil {
		t.Fatalf("NewPermissionBridge failed: %v", err)
	}
	defer pb.Close()
	
	_, err = pb.Check(context.Background(), PermissionRequest{
		ToolName:  "edit",
		FilePath:  "test.go",
		SessionID: "test-session",
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	
	// Verify audit log was written
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	
	if len(data) == 0 {
		t.Error("audit log should not be empty")
	}
}

func TestDefaultPermissionConfig(t *testing.T) {
	config := DefaultPermissionConfig()
	
	if config == nil {
		t.Fatal("DefaultPermissionConfig returned nil")
	}
	
	if len(config.Rules) == 0 {
		t.Error("DefaultPermissionConfig should have rules")
	}
	
	if config.DefaultAction != ActionAsk {
		t.Errorf("DefaultAction = %v, want %v", config.DefaultAction, ActionAsk)
	}
}

func TestMatchesGlob(t *testing.T) {
	pb, _ := NewPermissionBridge(nil)
	defer pb.Close()
	
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"*", "anything", true},
		{"*.go", "main.go", true},
		{"*.go", "test.txt", false},
		{"cmd/*", "cmd/main.go", true},
		{"cmd/*", "pkg/main.go", false},
	}
	
	for _, tt := range tests {
		got := pb.matchPattern(tt.pattern, false, tt.target)
		if got != tt.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}
