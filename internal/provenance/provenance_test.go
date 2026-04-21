package provenance

import (
	"os"
	"testing"
	"time"
)

func TestProvenance_FormatTrailer(t *testing.T) {
	tests := []struct {
		name string
		p    Provenance
		want string
	}{
		{
			name: "full agent attribution",
			p: Provenance{
				GeneratedBy: "agent",
				Model:       "claude-opus-4-7",
				SessionID:   "session-abc123",
				Harness:     "claude-code",
				Timestamp:   time.Now(),
			},
			want: "AI-Attribution: agent/claude-opus-4-7/session-abc123 (claude-code)",
		},
		{
			name: "agent without harness",
			p: Provenance{
				GeneratedBy: "agent",
				Model:       "gpt-4",
				SessionID:   "sess-456",
				Timestamp:   time.Now(),
			},
			want: "AI-Attribution: agent/gpt-4/sess-456",
		},
		{
			name: "agent with model only",
			p: Provenance{
				GeneratedBy: "agent",
				Model:       "claude-sonnet",
				Timestamp:   time.Now(),
			},
			want: "AI-Attribution: agent/claude-sonnet",
		},
		{
			name: "human attribution",
			p: Provenance{
				GeneratedBy: "human",
				Timestamp:   time.Now(),
			},
			want: "AI-Attribution: human",
		},
		{
			name: "hybrid attribution",
			p: Provenance{
				GeneratedBy: "hybrid",
				Model:       "claude-opus-4-7",
				SessionID:   "session-xyz",
				Harness:     "cursor",
				Timestamp:   time.Now(),
			},
			want: "AI-Attribution: hybrid/claude-opus-4-7/session-xyz (cursor)",
		},
		{
			name: "empty fields",
			p: Provenance{
				Timestamp: time.Now(),
			},
			want: "AI-Attribution: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.FormatTrailer(); got != tt.want {
				t.Errorf("FormatTrailer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseTrailer(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    Provenance
		wantErr bool
	}{
		{
			name: "full agent attribution with harness",
			line: "AI-Attribution: agent/claude-opus-4-7/session-abc123 (claude-code)",
			want: Provenance{
				GeneratedBy: "agent",
				Model:       "claude-opus-4-7",
				SessionID:   "session-abc123",
				Harness:     "claude-code",
			},
			wantErr: false,
		},
		{
			name: "agent without harness",
			line: "AI-Attribution: agent/gpt-4/sess-456",
			want: Provenance{
				GeneratedBy: "agent",
				Model:       "gpt-4",
				SessionID:   "sess-456",
				Harness:     "",
			},
			wantErr: false,
		},
		{
			name: "human attribution",
			line: "AI-Attribution: human",
			want: Provenance{
				GeneratedBy: "human",
				Model:       "",
				SessionID:   "",
				Harness:     "",
			},
			wantErr: false,
		},
		{
			name: "hybrid with harness",
			line: "AI-Attribution: hybrid/claude-opus-4-7/session-xyz (cursor)",
			want: Provenance{
				GeneratedBy: "hybrid",
				Model:       "claude-opus-4-7",
				SessionID:   "session-xyz",
				Harness:     "cursor",
			},
			wantErr: false,
		},
		{
			name: "missing prefix",
			line: "agent/claude-opus-4-7/session-abc123",
			want: Provenance{
				GeneratedBy: "agent",
				Model:       "claude-opus-4-7",
				SessionID:   "session-abc123",
			},
			wantErr: true,
		},
		{
			name: "empty value",
			line: "AI-Attribution: ",
			want: Provenance{
				GeneratedBy: "",
				Model:       "",
				SessionID:   "",
				Harness:     "",
			},
			wantErr: false,
		},
		{
			name:    "agent only with harness",
			line:    "AI-Attribution: agent (codex)",
			want: Provenance{
				GeneratedBy: "agent",
				Model:       "",
				SessionID:   "",
				Harness:     "codex",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTrailer(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTrailer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.GeneratedBy != tt.want.GeneratedBy {
					t.Errorf("ParseTrailer() GeneratedBy = %q, want %q", got.GeneratedBy, tt.want.GeneratedBy)
				}
				if got.Model != tt.want.Model {
					t.Errorf("ParseTrailer() Model = %q, want %q", got.Model, tt.want.Model)
				}
				if got.SessionID != tt.want.SessionID {
					t.Errorf("ParseTrailer() SessionID = %q, want %q", got.SessionID, tt.want.SessionID)
				}
				if got.Harness != tt.want.Harness {
					t.Errorf("ParseTrailer() Harness = %q, want %q", got.Harness, tt.want.Harness)
				}
			}
		})
	}
}

func TestProvenance_RoundTrip(t *testing.T) {
	original := Provenance{
		GeneratedBy: "agent",
		Model:       "claude-opus-4-7",
		SessionID:   "session-roundtrip-test",
		Harness:     "claude-code",
		Timestamp:   time.Now(),
	}

	trailer := original.FormatTrailer()
	parsed, err := ParseTrailer(trailer)
	if err != nil {
		t.Fatalf("ParseTrailer() error = %v", err)
	}

	if parsed.GeneratedBy != original.GeneratedBy {
		t.Errorf("RoundTrip GeneratedBy = %q, want %q", parsed.GeneratedBy, original.GeneratedBy)
	}
	if parsed.Model != original.Model {
		t.Errorf("RoundTrip Model = %q, want %q", parsed.Model, original.Model)
	}
	if parsed.SessionID != original.SessionID {
		t.Errorf("RoundTrip SessionID = %q, want %q", parsed.SessionID, original.SessionID)
	}
	if parsed.Harness != original.Harness {
		t.Errorf("RoundTrip Harness = %q, want %q", parsed.Harness, original.Harness)
	}
}

func TestFromEnv(t *testing.T) {
	// Save original env vars
	origHarness := os.Getenv("SDP_HARNESS")
	origModel := os.Getenv("SDP_MODEL")
	origSessionID := os.Getenv("SDP_SESSION_ID")

	// Clean up after test
	defer func() {
		if origHarness == "" {
			os.Unsetenv("SDP_HARNESS")
		} else {
			os.Setenv("SDP_HARNESS", origHarness)
		}
		if origModel == "" {
			os.Unsetenv("SDP_MODEL")
		} else {
			os.Setenv("SDP_MODEL", origModel)
		}
		if origSessionID == "" {
			os.Unsetenv("SDP_SESSION_ID")
		} else {
			os.Setenv("SDP_SESSION_ID", origSessionID)
		}
	}()

	tests := []struct {
		name      string
		harness   string
		model     string
		sessionID string
		want      Provenance
	}{
		{
			name:      "full agent environment",
			harness:   "claude-code",
			model:     "claude-opus-4-7",
			sessionID: "session-env-test",
			want: Provenance{
				GeneratedBy: "agent",
				Harness:     "claude-code",
				Model:       "claude-opus-4-7",
				SessionID:   "session-env-test",
			},
		},
		{
			name:      "partial environment",
			harness:   "codex",
			model:     "gpt-4",
			sessionID: "",
			want: Provenance{
				GeneratedBy: "agent",
				Harness:     "codex",
				Model:       "gpt-4",
				SessionID:   "",
			},
		},
		{
			name:      "no environment variables",
			harness:   "",
			model:     "",
			sessionID: "",
			want: Provenance{
				GeneratedBy: "human",
				Harness:     "",
				Model:       "",
				SessionID:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment
			if tt.harness == "" {
				os.Unsetenv("SDP_HARNESS")
			} else {
				os.Setenv("SDP_HARNESS", tt.harness)
			}
			if tt.model == "" {
				os.Unsetenv("SDP_MODEL")
			} else {
				os.Setenv("SDP_MODEL", tt.model)
			}
			if tt.sessionID == "" {
				os.Unsetenv("SDP_SESSION_ID")
			} else {
				os.Setenv("SDP_SESSION_ID", tt.sessionID)
			}

			got := FromEnv()

			if got.GeneratedBy != tt.want.GeneratedBy {
				t.Errorf("FromEnv() GeneratedBy = %q, want %q", got.GeneratedBy, tt.want.GeneratedBy)
			}
			if got.Harness != tt.want.Harness {
				t.Errorf("FromEnv() Harness = %q, want %q", got.Harness, tt.want.Harness)
			}
			if got.Model != tt.want.Model {
				t.Errorf("FromEnv() Model = %q, want %q", got.Model, tt.want.Model)
			}
			if got.SessionID != tt.want.SessionID {
				t.Errorf("FromEnv() SessionID = %q, want %q", got.SessionID, tt.want.SessionID)
			}
			if got.Timestamp.IsZero() {
				t.Error("FromEnv() Timestamp should not be zero")
			}
		})
	}
}

func TestProvenance_EdgeCases(t *testing.T) {
	t.Run("empty fields in FormatTrailer", func(t *testing.T) {
		p := Provenance{
			GeneratedBy: "",
			Model:       "",
			SessionID:   "",
			Harness:     "",
			Timestamp:   time.Now(),
		}
		got := p.FormatTrailer()
		want := "AI-Attribution: "
		if got != want {
			t.Errorf("FormatTrailer() with empty fields = %q, want %q", got, want)
		}
	})

	t.Run("parse with extra spaces", func(t *testing.T) {
		line := "AI-Attribution:  agent / claude-opus-4-7 / session-123  ( claude-code )"
		got, err := ParseTrailer(line)
		if err != nil {
			t.Fatalf("ParseTrailer() error = %v", err)
		}
		// Parser should handle extra spaces reasonably
		if got.GeneratedBy != "agent" && got.GeneratedBy != "agent " {
			t.Errorf("ParseTrailer() GeneratedBy = %q, want agent or 'agent '", got.GeneratedBy)
		}
	})

	t.Run("unknown harness value", func(t *testing.T) {
		p := Provenance{
			GeneratedBy: "agent",
			Harness:     "unknown-harness",
			Timestamp:   time.Now(),
		}
		got := p.FormatTrailer()
		want := "AI-Attribution: agent (unknown-harness)"
		if got != want {
			t.Errorf("FormatTrailer() with unknown harness = %q, want %q", got, want)
		}
	})

	t.Run("parse without model but with session", func(t *testing.T) {
		line := "AI-Attribution: agent//session-123"
		got, err := ParseTrailer(line)
		if err != nil {
			t.Fatalf("ParseTrailer() error = %v", err)
		}
		if got.GeneratedBy != "agent" {
			t.Errorf("ParseTrailer() GeneratedBy = %q, want agent", got.GeneratedBy)
		}
		// Empty model in the middle should result in empty or session as model
		// depending on parsing - this documents current behavior
	})
}
