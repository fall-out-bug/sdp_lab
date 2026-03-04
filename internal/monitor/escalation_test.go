package monitor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestIsValidNotifyCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "valid bd command",
			cmd:      "bd create task123",
			expected: true,
		},
		{
			name:     "valid notify command",
			cmd:      "notify send test@example.com",
			expected: true,
		},
		{
			name:     "valid echo command",
			cmd:      "echo hello",
			expected: true,
		},
		{
			name:     "reject shell binary",
			cmd:      "/bin/sh -c echo test",
			expected: false,
		},
		{
			name:     "empty command",
			cmd:      "",
			expected: false,
		},
		{
			name:     "invalid command",
			cmd:      "rm -rf /",
			expected: false,
		},
		{
			name:     "shell injection attempt",
			cmd:      "bd create; rm -rf /",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidNotifyCommand(tt.cmd)
			if result != tt.expected {
				t.Errorf("isValidNotifyCommand(%q) = %v; want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}

func TestParseNotifyCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected []string
		wantErr  bool
	}{
		{
			name:     "simple command",
			cmd:      "bd create task123",
			expected: []string{"bd", "create", "task123"},
			wantErr:  false,
		},
		{
			name:     "command with quoted arg",
			cmd:      "notify send \"test message\"",
			expected: []string{"notify", "send", "test message"},
			wantErr:  false,
		},
		{
			name:     "empty command",
			cmd:      "",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseNotifyCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNotifyCommand(%q) error = %v; want error: %v", tt.cmd, err, tt.wantErr)
			}
			if err == nil {
				if len(result) != len(tt.expected) {
					t.Errorf("parseNotifyCommand(%q) returned %d parts; want %d", tt.cmd, len(result), len(tt.expected))
				}
				limit := len(result)
				if len(tt.expected) < limit {
					limit = len(tt.expected)
				}
				for i := 0; i < limit; i++ {
					part := result[i]
					if part != tt.expected[i] {
						t.Errorf("parseNotifyCommand(%q) result[%d] = %q; want %q", tt.cmd, i, part, tt.expected[i])
					}
				}
			}
		})
	}
}

func TestNewEscalationHandler(t *testing.T) {
	cfg := EscalationConfig{
		CreateWisp:    false,
		NotifyCommand: "notify test@example.com",
	}

	eh := NewEscalationHandler(cfg)

	if eh == nil {
		t.Fatal("NewEscalationHandler returned nil")
	}

	if eh.createWisp != false {
		t.Errorf("createWisp = %v; want false", eh.createWisp)
	}

	if eh.notifyCmd != "notify test@example.com" {
		t.Errorf("notifyCmd = %q; want %q", eh.notifyCmd, "notify test@example.com")
	}
}

func TestEscalationHandler_Escalate(t *testing.T) {
	t.Run("calls onEscalate when set", func(t *testing.T) {
		onEscalateCalled := false
		cfg := EscalationConfig{
			CreateWisp:    false,
			NotifyCommand: "notify test@example.com",
			OnEscalate: func(sessionID string, lastEvent time.Time) {
				onEscalateCalled = true
			},
		}

		eh := NewEscalationHandler(cfg)
		_ = eh.Escalate(context.Background(), "session-123", time.Now())

		if !onEscalateCalled {
			t.Error("onEscalate was not called")
		}
	})
}

func TestEscalationHandler_SafeCommands(t *testing.T) {
	t.Run("rejects invalid commands", func(t *testing.T) {
		cfg := EscalationConfig{
			CreateWisp:    false,
			NotifyCommand: "rm -rf /tmp", // Invalid command
			OnEscalate:    func(sessionID string, lastEvent time.Time) {},
		}

		eh := NewEscalationHandler(cfg)

		// This should fail because "rm" is not in whitelist
		err := eh.Escalate(context.Background(), "session-123", time.Now())

		if err == nil {
			t.Error("expected error for invalid notify command, got nil")
		}

		// Check that error message mentions invalid command
		if err != nil && !strings.Contains(err.Error(), "invalid notify command") {
			t.Errorf("error message should mention invalid command, got: %v", err)
		}
	})
}
