package executil

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

func TestDefaultRunner_Output(t *testing.T) {
	out, err := GetDefaultRunner().Output(context.Background(), "", "echo", "hello")
	if err != nil {
		t.Fatalf("Output: %v", err)
	}
	if s := string(out); s != "hello\n" && s != "hello\r\n" {
		t.Errorf("Output = %q, want hello with newline", s)
	}
}

func TestDefaultRunner_Run(t *testing.T) {
	err := GetDefaultRunner().Run(context.Background(), "", "true")
	if err != nil {
		t.Fatalf("Run true: %v", err)
	}
}

func TestDefaultRunner_CombinedOutputWaitDelayBoundsPipeWait(t *testing.T) {
	oldDelay := commandWaitDelay
	commandWaitDelay = 20 * time.Millisecond
	t.Cleanup(func() { commandWaitDelay = oldDelay })

	start := time.Now()
	_, err := GetDefaultRunner().CombinedOutput(context.Background(), "", "sh", "-c", "sleep 30 &")
	if !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("CombinedOutput error = %v, want ErrWaitDelay", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("CombinedOutput was not bounded by WaitDelay; elapsed=%s", elapsed)
	}
}
