package orchestrator

import (
	"testing"
)

func TestSelectRuntime(t *testing.T) {
	tests := []struct {
		labels []string
		want   Runtime
	}{
		{nil, RuntimeOpenCode},
		{[]string{}, RuntimeOpenCode},
		{[]string{"autonomy"}, RuntimeOpenCode},
		{[]string{"runtime:openclaw"}, RuntimeOpenClaw},
		{[]string{"autonomy", "runtime:openclaw"}, RuntimeOpenClaw},
	}
	for _, tt := range tests {
		got := SelectRuntime(tt.labels)
		if got != tt.want {
			t.Errorf("SelectRuntime(%v) = %v, want %v", tt.labels, got, tt.want)
		}
	}
}

func TestNewSSHDispatcher(t *testing.T) {
	d := NewSSHDispatcher("/tmp")
	if d == nil || d.workDir != "/tmp" {
		t.Fatalf("NewSSHDispatcher: got %+v", d)
	}
}

func TestNewInClusterDispatcher(t *testing.T) {
	d := NewInClusterDispatcher("/tmp")
	if d == nil || d.workDir != "/tmp" {
		t.Fatalf("NewInClusterDispatcher: got %+v", d)
	}
}

func TestSSHDispatcher_Dispatch_EmptyHost(t *testing.T) {
	d := NewSSHDispatcher(t.TempDir())
	err := d.Dispatch(DispatchConfig{Host: ""})
	if err == nil || err.Error() != "host required for SSH dispatch" {
		t.Errorf("expected host required error, got %v", err)
	}
}
