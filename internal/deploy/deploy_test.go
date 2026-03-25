package deploy

import (
	"encoding/json"
	"context"
	"testing"
	"time"
)

func TestShortHash(t *testing.T) {
	if shortHash("abcdef1234567890") != "abcdef123456" {
		t.Error("wrong short hash")
	}
	if shortHash("abc") != "abc" {
		t.Error("short hash should not truncate short strings")
	}
}

func TestParseTime(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	parsed := parseTime(now)
	if parsed.IsZero() {
		t.Error("failed to parse valid time")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("/home/user/my.project")
	if cfg.ComposeStaging != "/home/user/my.project/docker-compose.staging.yml" {
		t.Error("wrong staging path")
	}
	if cfg.ComposeProd != "/home/user/my.project/docker-compose.yml" {
		t.Error("wrong prod path")
	}
	if cfg.ProjectName != "my-project" {
		t.Error("wrong project name")
	}
}

func TestResult_JSON(t *testing.T) {
	r := Result{
		Phase:    "staging",
		Target:   "staging",
		ImageTag: "test:latest",
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Phase != "staging" {
		t.Error("phase mismatch")
	}
}

func TestHealthCheckResult(t *testing.T) {
	h := HealthCheckResult{
		Passed:  true,
		Checks:  []string{"running"},
		Minutes: 5.0,
	}
	if !h.Passed {
		t.Error("should be passed")
	}
}

func TestSmokeTestResult(t *testing.T) {
	s := TestResult{
		Passed:   true,
		ExitCode: 0,
		Output:   "all tests passed",
	}
	if !s.Passed {
		t.Error("should be passed")
	}
}

func TestContainerInfo(t *testing.T) {
	c := ContainerInfo{
		Name:  "app",
		ID:    "abc123",
		Image: "test:latest",
		Status: "running",
		Ports: "80:80",
	}
	if c.Name != "app" {
		t.Error("name mismatch")
	}
}

func TestNilConfig(t *testing.T) {
	_, err := Staging(context.TODO(), nil, "hash")
	if err == nil {
		t.Error("expected error for nil config")
	}
	_, err = Production(context.TODO(), nil, "tag")
	if err == nil {
		t.Error("expected error for nil config")
	}
	_, err = Rollback(context.TODO(), nil, "tag")
	if err == nil {
		t.Error("expected error for nil config")
	}
}
