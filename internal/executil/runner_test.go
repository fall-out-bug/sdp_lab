package executil

import (
	"context"
	"testing"
)

func TestDefaultRunner_Output(t *testing.T) {
	out, err := DefaultRunner.Output(context.Background(), "", "echo", "hello")
	if err != nil {
		t.Fatalf("Output: %v", err)
	}
	if s := string(out); s != "hello\n" && s != "hello\r\n" {
		t.Errorf("Output = %q, want hello with newline", s)
	}
}

func TestDefaultRunner_Run(t *testing.T) {
	err := DefaultRunner.Run(context.Background(), "", "true")
	if err != nil {
		t.Fatalf("Run true: %v", err)
	}
}
