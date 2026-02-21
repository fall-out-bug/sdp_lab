package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func patchSlugifyForTrim(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "decision.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	old := "\tif len(t) > 48 {\n\t\treturn t[:48]\n\t}\n\treturn t\n"
	new := "\tif len(t) > 48 {\n\t\tt = t[:48]\n\t\tt = strings.Trim(t, \"-\")\n\t}\n\tif t == \"\" {\n\t\treturn \"task\"\n\t}\n\treturn t\n"
	if !strings.Contains(content, old) {
		return errors.New("slugify block not found")
	}
	content = strings.Replace(content, old, new, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}

func addSlugifyRegressionTest(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "decision_test.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	needle := "func TestDecideCriticalEscalates(t *testing.T) {"
	insert := "func TestBuildBranchNameTrimsTrailingDashAfterTruncation(t *testing.T) {\n\ttitle := strings.Repeat(\"word-\", 20)\n\tbranch := BuildBranchName(\"id-4\", title)\n\tif strings.HasSuffix(branch, \"-\") {\n\t\tt.Fatalf(\"expected no trailing dash, got %s\", branch)\n\t}\n}\n\n"
	if strings.Contains(content, "TestBuildBranchNameTrimsTrailingDashAfterTruncation") {
		return nil
	}
	if !strings.Contains(content, needle) {
		return errors.New("test insertion point not found")
	}
	content = strings.Replace(content, needle, insert+needle, 1)
	if !strings.Contains(content, "\"strings\"") {
		content = strings.Replace(content, "import \"testing\"", "import (\n\t\"strings\"\n\t\"testing\"\n)", 1)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func patchModelChainUnknownFallback(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "model_chain.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	old := "func ResolveFallbackSequence(start string) []string {\n\tsequence := []string{start}\n\tcurrent := start\n"
	new := "func ResolveFallbackSequence(start string) []string {\n\tif start == \"\" || !AllowedModel(start) {\n\t\tstart = DefaultModel()\n\t}\n\tsequence := []string{start}\n\tcurrent := start\n"
	if strings.Contains(content, "!AllowedModel(start)") {
		return nil
	}
	if !strings.Contains(content, old) {
		return errors.New("model_chain sequence block not found")
	}
	content = strings.Replace(content, old, new, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}

func addModelChainRegressionTest(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "model_chain_test.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	if strings.Contains(content, "TestResolveFallbackSequenceUnknownStartsFromDefault") {
		return nil
	}
	needle := "func TestResolveFallbackSequence(t *testing.T) {"
	insert := "func TestResolveFallbackSequenceUnknownStartsFromDefault(t *testing.T) {\n\tseq := ResolveFallbackSequence(\"unknown-model\")\n\tif len(seq) != 3 {\n\t\tt.Fatalf(\"expected 3 steps, got %d\", len(seq))\n\t}\n\tif seq[0] != \"glm-5\" || seq[1] != \"glm-4.7\" || seq[2] != \"escalated\" {\n\t\tt.Fatalf(\"unexpected sequence: %#v\", seq)\n\t}\n}\n\n"
	if !strings.Contains(content, needle) {
		return errors.New("model_chain test insertion point not found")
	}
	content = strings.Replace(content, needle, insert+needle, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}

func patchRiskK8sHigh(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "decision.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	if strings.Contains(content, "regexp.MustCompile(`k8s`)") {
		return nil
	}
	needle := "\tregexp.MustCompile(`git`),\n"
	insert := "\tregexp.MustCompile(`git`),\n\tregexp.MustCompile(`k8s`),\n"
	if !strings.Contains(content, needle) {
		return errors.New("decision high risk pattern block not found")
	}
	content = strings.Replace(content, needle, insert, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}

func addRiskK8sRegressionTest(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "decision_test.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	if strings.Contains(content, "TestDecideK8sPathIsHighRisk") {
		return nil
	}
	needle := "func TestDecideCriticalEscalates(t *testing.T) {"
	insert := "func TestDecideK8sPathIsHighRisk(t *testing.T) {\n\tres := Decide(DecisionRequest{IssueID: \"id-k8s\", Title: \"Update worker manifests\", PreferredModel: \"glm-5\", ChangedPaths: []string{\"deploy/k8s/workers/opencode-agent.yaml\"}})\n\tif res.RiskClass != \"high\" {\n\t\tt.Fatalf(\"expected high risk, got %s\", res.RiskClass)\n\t}\n\tif res.PolicyVerdict != \"allow\" {\n\t\tt.Fatalf(\"expected allow, got %s\", res.PolicyVerdict)\n\t}\n}\n\n"
	if !strings.Contains(content, needle) {
		return errors.New("decision test insertion point not found")
	}
	content = strings.Replace(content, needle, insert+needle, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}
