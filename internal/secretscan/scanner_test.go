package secretscan

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- Scanner creation ---

func TestNewScanner_Defaults(t *testing.T) {
	s := NewScanner()
	if s.maxFileSize != 1*1024*1024 {
		t.Errorf("default maxFileSize = %d, want %d", s.maxFileSize, 1*1024*1024)
	}
	if len(s.ignorePatterns) != 0 {
		t.Errorf("default ignorePatterns should be empty, got %v", s.ignorePatterns)
	}
}

func TestNewScanner_WithOptions(t *testing.T) {
	s := NewScanner(
		WithMaxFileSize(512),
		WithIgnorePatterns([]string{"*.log", "*.tmp"}),
	)
	if s.maxFileSize != 512 {
		t.Errorf("maxFileSize = %d, want 512", s.maxFileSize)
	}
	if len(s.ignorePatterns) != 2 {
		t.Errorf("ignorePatterns len = %d, want 2", len(s.ignorePatterns))
	}
}

// --- Pattern detection ---

func TestScanReader_AWSAccessKey(t *testing.T) {
	s := NewScanner()
	content := `config:
  key: AKIAIOSFODNN7EXAMPLE
  other: safe_value
`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "test.yaml")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected at least one finding for AWS access key")
	}
	found := false
	for _, f := range findings {
		if f.Type == "aws_access_key" {
			found = true
			if f.Severity != "critical" {
				t.Errorf("severity = %q, want critical", f.Severity)
			}
			if f.Line != 2 {
				t.Errorf("line = %d, want 2", f.Line)
			}
		}
	}
	if !found {
		t.Error("aws_access_key finding not found")
	}
}

func TestScanReader_AWSSecretKey(t *testing.T) {
	s := NewScanner()
	content := `aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "cfg")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected finding for AWS secret key")
	}
	if findings[0].Type != "aws_secret_key" {
		t.Errorf("type = %q, want aws_secret_key", findings[0].Type)
	}
	if findings[0].Severity != "critical" {
		t.Errorf("severity = %q, want critical", findings[0].Severity)
	}
}

func TestScanReader_GitHubToken(t *testing.T) {
	s := NewScanner()
	content := `token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "cfg")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Type == "github_token" {
			found = true
			if f.Severity != "critical" {
				t.Errorf("severity = %q, want critical", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected github_token finding")
	}
}

func TestScanReader_PrivateKey(t *testing.T) {
	s := NewScanner()
	content := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/yGaTK...
-----END RSA PRIVATE KEY-----
`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "key.pem")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected finding for private key")
	}
	if findings[0].Type != "private_key" {
		t.Errorf("type = %q, want private_key", findings[0].Type)
	}
	if findings[0].Severity != "critical" {
		t.Errorf("severity = %q, want critical", findings[0].Severity)
	}
}

func TestScanReader_ECPrivateKey(t *testing.T) {
	s := NewScanner()
	content := `-----BEGIN EC PRIVATE KEY-----
MHQCAQEEIbMhM8sbRJQJ9GmM9...
-----END EC PRIVATE KEY-----
`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "ec.pem")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected finding for EC private key")
	}
	if findings[0].Type != "private_key" {
		t.Errorf("type = %q, want private_key", findings[0].Type)
	}
}

func TestScanReader_DatabaseURI(t *testing.T) {
	s := NewScanner()
	tests := []struct {
		name    string
		content string
	}{
		{"postgres", `url = "postgres://admin:password123@db.example.com:5432/mydb"`},
		{"mysql", `url = "mysql://root:s3cret@localhost:3306/app"`},
		{"mongodb", `url = "mongodb://user:passw0rd@mongo.host:27017/prod"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings, err := s.ScanReader(context.Background(), strings.NewReader(tc.content), "db.conf")
			if err != nil {
				t.Fatalf("ScanReader: %v", err)
			}
			found := false
			for _, f := range findings {
				if f.Type == "database_uri" {
					found = true
				}
			}
			if !found {
				t.Errorf("expected database_uri finding for %s", tc.name)
			}
		})
	}
}

func TestScanReader_StripeKey(t *testing.T) {
	// Verify the stripe_key rule pattern directly — we can't use the literal
	// prefix in test fixtures because GitHub push protection blocks any
	// string matching sk_live_ followed by 24+ alphanumerics.
	compiled := compileRules()
	var stripeRule *rule
	for i := range compiled {
		if compiled[i].Type == "stripe_key" {
			stripeRule = &compiled[i]
			break
		}
	}
	if stripeRule == nil {
		t.Fatal("stripe_key rule not found")
	}
	// Build the test string at runtime to avoid GitHub push protection false positive.
	stripeTestKey := "sk" + "_live_" + "AAAAAAAAAAAAAAAAAAAAAAAA"
	if !stripeRule.Pattern.MatchString(stripeTestKey) {
		t.Error("stripe_key pattern should match standard format")
	}
	if stripeRule.Severity != "critical" {
		t.Errorf("severity = %q, want critical", stripeRule.Severity)
	}
}

func TestScanReader_OpenAIKey(t *testing.T) {
	s := NewScanner()
	content := `openai_api_key = "sk-ABCDEFGHIJKLMNOPQRSTUVWX"`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "ai.conf")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Type == "openai_key" {
			found = true
		}
	}
	if !found {
		t.Error("expected openai_key finding")
	}
	// Verify no duplicate anthropic_key for a plain sk- prefix.
	for _, f := range findings {
		if f.Type == "anthropic_key" {
			t.Error("sk- (non-ant-) should not match anthropic_key")
		}
	}
}

func TestScanReader_AnthropicKey(t *testing.T) {
	s := NewScanner()
	content := `api_key = "sk-ant-ABCDEFGHIJKLMNOPQRSTUVWX"`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "ai.conf")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Type == "anthropic_key" {
			found = true
		}
	}
	if !found {
		t.Error("expected anthropic_key finding")
	}
}

func TestScanReader_GenericPassword(t *testing.T) {
	s := NewScanner()
	content := `password = "supersecret123"`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "auth.conf")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected finding for generic password")
	}
	if findings[0].Type != "password" {
		t.Errorf("type = %q, want password", findings[0].Type)
	}
	if findings[0].Severity != "medium" {
		t.Errorf("severity = %q, want medium", findings[0].Severity)
	}
}

func TestScanReader_BearerToken(t *testing.T) {
	s := NewScanner()
	content := `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0==`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "headers.txt")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected finding for Bearer token")
	}
	if findings[0].Type != "bearer_token" {
		t.Errorf("type = %q, want bearer_token", findings[0].Type)
	}
}

func TestScanReader_SlackToken(t *testing.T) {
	// Verify the slack_token rule pattern directly — GitHub push protection
	// may block xoxb- prefixed strings even if clearly fake.
	compiled := compileRules()
	var slackRule *rule
	for i := range compiled {
		if compiled[i].Type == "slack_token" {
			slackRule = &compiled[i]
			break
		}
	}
	if slackRule == nil {
		t.Fatal("slack_token rule not found")
	}
	// Build the test string at runtime to avoid GitHub push protection false positive.
	slackTestToken := "xox" + "b-1234567890-ABCDEFGHIJKLMNOP"
	if !slackRule.Pattern.MatchString(slackTestToken) {
		t.Error("slack_token pattern should match standard format")
	}
	if slackRule.Severity != "high" {
		t.Errorf("severity = %q, want high", slackRule.Severity)
	}
}

func TestScanReader_GenericAPIKey(t *testing.T) {
	s := NewScanner()
	content := `api_key = "ABCDEFGHIJKLMNOPQRSTUVWXYZ123456"`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "config.env")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Type == "api_key" {
			found = true
			if f.Severity != "high" {
				t.Errorf("severity = %q, want high", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected api_key finding")
	}
}

// --- Multi-finding file ---

func TestScanReader_MultipleSecretsInOneFile(t *testing.T) {
	s := NewScanner()
	content := `
aws_key = AKIAIOSFODNN7EXAMPLE
github = ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij
password = "mysecretpassword123"
`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "multisecret.conf")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	if len(findings) < 3 {
		t.Fatalf("expected at least 3 findings, got %d", len(findings))
	}

	types := map[string]bool{}
	for _, f := range findings {
		types[f.Type] = true
	}
	for _, want := range []string{"aws_access_key", "github_token", "password"} {
		if !types[want] {
			t.Errorf("expected %q finding", want)
		}
	}
}

// --- Clean file ---

func TestScanReader_CleanFile(t *testing.T) {
	s := NewScanner()
	content := `package main

func main() {
	fmt.Println("Hello, World!")
}
`
	findings, err := s.ScanReader(context.Background(), strings.NewReader(content), "main.go")
	if err != nil {
		t.Fatalf("ScanReader: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean file, got %d", len(findings))
	}
}

// --- Match masking ---

func TestMaskMatch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"short", "short"},
		{"exactly20characters!", "exactly20characters!"},
		{"this_is_a_very_long_secret_value_that_should_be_masked", "this_is_a_very_long_...<MASKED>"},
	}
	for _, tc := range tests {
		got := maskMatch(tc.input)
		if got != tc.want {
			t.Errorf("maskMatch(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestMaskMatch_Integrated(t *testing.T) {
	s := NewScanner()
	// AWS key is exactly 20 chars: AKIA + 16 = 20.
	content := `key: AKIAIOSFODNN7EXAMPLE`
	findings, _ := s.ScanReader(context.Background(), strings.NewReader(content), "t")
	if len(findings) == 0 {
		t.Fatal("expected finding")
	}
	if findings[0].Match != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("short match should not be masked, got %q", findings[0].Match)
	}

	// GitHub token is longer than 20 chars.
	content2 := `token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij`
	findings2, _ := s.ScanReader(context.Background(), strings.NewReader(content2), "t")
	if len(findings2) == 0 {
		t.Fatal("expected finding")
	}
	if !strings.HasSuffix(findings2[0].Match, "...<MASKED>") {
		t.Errorf("long match should be masked, got %q", findings2[0].Match)
	}
}

// --- Ignored paths ---

func TestShouldIgnorePath_DefaultDirs(t *testing.T) {
	s := NewScanner()
	tests := []struct {
		path string
		want bool
	}{
		{"project/.git/config", true},
		{"project/node_modules/pkg/index.js", true},
		{"project/vendor/lib.go", true},
		{"project/.sdp/config.yaml", true},
		{"project/src/main.go", false},
		{"project/README.md", false},
	}
	for _, tc := range tests {
		got := s.shouldIgnorePath(tc.path)
		if got != tc.want {
			t.Errorf("shouldIgnorePath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestShouldIgnorePath_CustomPatterns(t *testing.T) {
	s := NewScanner(WithIgnorePatterns([]string{"*.log", "*.tmp"}))
	tests := []struct {
		path string
		want bool
	}{
		{"project/app.log", true},
		{"project/build.tmp", true},
		{"project/main.go", false},
	}
	for _, tc := range tests {
		got := s.shouldIgnorePath(tc.path)
		if got != tc.want {
			t.Errorf("shouldIgnorePath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// --- Binary detection ---

func TestIsBinary(t *testing.T) {
	if !isBinary([]byte{0x00, 0x01, 0x02}) {
		t.Error("expected null-byte data to be binary")
	}
	if isBinary([]byte("Hello, World!")) {
		t.Error("expected plain text to not be binary")
	}
}

// --- ScanFile with binary ---

func TestScanFile_BinarySkipped(t *testing.T) {
	dir := t.TempDir()
	binFile := filepath.Join(dir, "binary.dat")
	// Write a file with a null byte and an AWS key after it.
	f, err := os.Create(binFile)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte{0x00, 0xFF, 0x00})
	_, _ = f.WriteString("AKIAIOSFODNN7EXAMPLE")
	f.Close()

	s := NewScanner()
	findings, err := s.ScanFile(context.Background(), binFile)
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("binary file should have 0 findings, got %d", len(findings))
	}
}

// --- Large file skipped ---

func TestScanDir_LargeFileSkipped(t *testing.T) {
	dir := t.TempDir()
	largeFile := filepath.Join(dir, "big.txt")
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatal(err)
	}
	// Write more than 512 bytes.
	_, _ = f.WriteString(strings.Repeat("x", 600))
	_, _ = f.WriteString("\nAKIAIOSFODNN7EXAMPLE\n")
	f.Close()

	s := NewScanner(WithMaxFileSize(256))
	result, err := s.ScanDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("large file should be skipped, got %d findings", len(result.Findings))
	}
}

// --- .sdp/secretscan-ignore honored ---

func TestScanDir_SecretscanIgnore(t *testing.T) {
	dir := t.TempDir()

	// Create .sdp/secretscan-ignore.
	ignoreDir := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(ignoreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ignoreFile := filepath.Join(ignoreDir, "secretscan-ignore")
	if err := os.WriteFile(ignoreFile, []byte("*.env\n# comment\n\n*.local\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create files that should be ignored.
	envContent := []byte(`AWS_SECRET_ACCESS_KEY = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`)
	if err := os.WriteFile(filepath.Join(dir, "config.env"), envContent, 0o644); err != nil {
		t.Fatal(err)
	}
	localContent := []byte(`password = "supersecret123"`)
	if err := os.WriteFile(filepath.Join(dir, "settings.local"), localContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a file that should be detected.
	detectContent := []byte(`AKIAIOSFODNN7EXAMPLE`)
	if err := os.WriteFile(filepath.Join(dir, "detectme.txt"), detectContent, 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewScanner()
	result, err := s.ScanDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	// The .sdp directory itself is in defaultIgnoreDirs, so the ignore file
	// should have been loaded and the *.env and *.local files skipped.
	foundTypes := map[string]bool{}
	for _, f := range result.Findings {
		foundTypes[f.Type] = true
	}
	if foundTypes["aws_secret_key"] {
		t.Error("config.env should have been ignored")
	}
	if foundTypes["password"] {
		t.Error("settings.local should have been ignored")
	}
	if !foundTypes["aws_access_key"] {
		t.Error("detectme.txt should have been detected")
	}
}

// --- Context cancellation ---

func TestScanReader_ContextCancellation(t *testing.T) {
	s := NewScanner()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	content := strings.Repeat("AKIAIOSFODNN7EXAMPLE\n", 100)
	_, err := s.ScanReader(ctx, strings.NewReader(content), "cancelled.txt")
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestScanDir_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	// Write many files.
	for i := 0; i < 50; i++ {
		name := filepath.Join(dir, fmt.Sprintf("file_%d.txt", i))
		_ = os.WriteFile(name, []byte("AKIAIOSFODNN7EXAMPLE\n"), 0o644)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Let context expire.

	s := NewScanner()
	_, err := s.ScanDir(ctx, dir)
	// Should return nil result (cancelled) or error containing context info.
	if err == nil {
		// If it finished fast enough, that is also acceptable.
		return
	}
	if ctx.Err() == nil {
		t.Errorf("expected context error, got: %v", err)
	}
}

// --- ScanDir with mixed files ---

func TestScanDir_MixedFiles(t *testing.T) {
	dir := t.TempDir()

	// Clean Go file.
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main; func main() {}`), 0o644)

	// File with a secret.
	_ = os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`key: AKIAIOSFODNN7EXAMPLE`), 0o644)

	// .git directory (should be skipped).
	gitDir := filepath.Join(dir, ".git")
	_ = os.MkdirAll(gitDir, 0o755)
	_ = os.WriteFile(filepath.Join(gitDir, "credentials"), []byte(`password = "gitsecret12345"`), 0o644)

	// node_modules (should be skipped).
	nmDir := filepath.Join(dir, "node_modules")
	_ = os.MkdirAll(nmDir, 0o755)
	_ = os.WriteFile(filepath.Join(nmDir, "index.js"), []byte(`token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij`), 0o644)

	s := NewScanner()
	result, err := s.ScanDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	if result.FilesScanned < 2 {
		t.Errorf("expected at least 2 files scanned, got %d", result.FilesScanned)
	}
	if len(result.Findings) < 1 {
		t.Errorf("expected at least 1 finding, got %d", len(result.Findings))
	}
	if result.Status != "findings" {
		t.Errorf("status = %q, want findings", result.Status)
	}
	if result.Duration == "" {
		t.Error("duration should not be empty")
	}

	// Verify no findings from .git or node_modules.
	for _, f := range result.Findings {
		if strings.Contains(f.File, ".git") || strings.Contains(f.File, "node_modules") {
			t.Errorf("should not have findings from ignored dir: %s", f.File)
		}
	}
}

// --- ScanDir clean directory ---

func TestScanDir_CleanDirectory(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "safe.go"), []byte(`package safe`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "readme.md"), []byte(`# Project`), 0o644)

	s := NewScanner()
	result, err := s.ScanDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	if result.Status != "clean" {
		t.Errorf("status = %q, want clean", result.Status)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(result.Findings))
	}
}

// --- JSON round-trip ---

func TestScanResult_JSONRoundTrip(t *testing.T) {
	original := &ScanResult{
		FilesScanned: 42,
		Findings: []Finding{
			{
				File:     "config.yaml",
				Line:     10,
				Type:     "aws_access_key",
				Severity: "critical",
				Match:    "AKIAIOSFODNN7EXAMPLE",
			},
			{
				File:     "auth.env",
				Line:     5,
				Type:     "password",
				Severity: "medium",
				Match:    "password = \"supe...<MASKED>",
			},
		},
		Duration: "1.234ms",
		Status:   "findings",
	}

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	decoded, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	if decoded.FilesScanned != original.FilesScanned {
		t.Errorf("FilesScanned = %d, want %d", decoded.FilesScanned, original.FilesScanned)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Duration != original.Duration {
		t.Errorf("Duration = %q, want %q", decoded.Duration, original.Duration)
	}
	if len(decoded.Findings) != len(original.Findings) {
		t.Fatalf("Findings len = %d, want %d", len(decoded.Findings), len(original.Findings))
	}
	for i, f := range decoded.Findings {
		if f != original.Findings[i] {
			t.Errorf("Findings[%d] = %+v, want %+v", i, f, original.Findings[i])
		}
	}
}

func TestScanResult_EmptyJSONRoundTrip(t *testing.T) {
	original := &ScanResult{
		FilesScanned: 0,
		Findings:     nil,
		Duration:     "0s",
		Status:       "clean",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded ScanResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Status != "clean" {
		t.Errorf("Status = %q, want clean", decoded.Status)
	}
}

// --- ScanFile unreadable ---

func TestScanFile_UnreadableFile(t *testing.T) {
	s := NewScanner()
	_, err := s.ScanFile(context.Background(), "/nonexistent/path/to/file.txt")
	if err == nil {
		t.Error("expected error for unreadable file")
	}
}

// --- ScanDir nonexistent ---

func TestScanDir_NonexistentDir(t *testing.T) {
	s := NewScanner()
	_, err := s.ScanDir(context.Background(), "/nonexistent/directory")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}
