package roles

import (
	"context"
	"strings"
	"testing"

	"sdp_dev/internal/agent"
	"sdp_dev/internal/federation"
)

func TestGet(t *testing.T) {
	if Get("analyst") == nil {
		t.Error("Get(analyst) should return strategy")
	}
	if Get("coder") == nil {
		t.Error("Get(coder) should return strategy")
	}
	if Get("unknown-role-xyz") != nil {
		t.Error("Get(unknown) should return nil")
	}
}

func TestExecute_unknownRole(t *testing.T) {
	ctx := context.Background()
	input := TaskInput{
		FederatedTask: federation.FederatedTask{ProjectID: "p1"},
		Ctx:           &agent.Context{},
	}
	_, err := Execute(ctx, "unknown-role-xyz", input)
	if err == nil {
		t.Fatal("Execute(unknown) should return error")
	}
	if !strings.Contains(err.Error(), "unknown role") {
		t.Errorf("error should mention unknown role: %v", err)
	}
}
