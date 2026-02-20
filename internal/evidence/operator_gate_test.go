package evidence

import "testing"

func TestValidateRoleLogOK(t *testing.T) {
	log := `noise line
{"run_id":"run-1","role":"analyst","status":"ok","summary":"done","artifacts":[{"id":"a1"}]}
more noise`
	res := ValidateRoleLog("analyst", "run-1", log)
	if !res.OK {
		t.Fatalf("expected ok, got %+v", res)
	}
}

func TestValidateRoleLogProviderError(t *testing.T) {
	log := `ProviderModelNotFoundError: Model not found: zai/glm-5.`
	res := ValidateRoleLog("coder", "run-1", log)
	if res.OK {
		t.Fatalf("expected failure for provider error")
	}
}

func TestValidateRoleLogConnectivityError(t *testing.T) {
	log := `Error: Unable to connect. Is the computer able to access the url?`
	res := ValidateRoleLog("coder", "run-1", log)
	if res.OK {
		t.Fatalf("expected failure for connectivity error")
	}
}

func TestValidateRoleLogRoleMismatch(t *testing.T) {
	log := `{"run_id":"run-1","role":"analyst","status":"ok","summary":"done","artifacts":[]}`
	res := ValidateRoleLog("reviewer", "run-1", log)
	if res.OK {
		t.Fatalf("expected role mismatch failure")
	}
}
