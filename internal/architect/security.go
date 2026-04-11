package architect

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// SecretMatch represents a detected secret in content.
type SecretMatch struct {
	Type     string `json:"type"`
	Position int    `json:"position"`
	Length   int    `json:"length"`
}

// secretPattern pairs a compiled regex with a type label and length hint.
type secretPattern struct {
	re     *regexp.Regexp
	typ    string
	length int // fixed match length (0 = use match length)
}

// SecurityFilter sanitizes a CodebaseProfile before sending to an LLM.
// It detects secrets, scrubs PII (internal package names, usernames),
// and enforces the --allow-external-llm policy.
type SecurityFilter struct {
	// patterns are compiled regexes for detecting secrets.
	patterns []secretPattern

	// AllowExternalLLM must be explicitly set to true to send data to cloud LLMs.
	AllowExternalLLM bool
}

// NewSecurityFilter returns a SecurityFilter with common secret patterns.
func NewSecurityFilter() *SecurityFilter {
	// TODO(Phase 2): Embed gitleaks ruleset as compiled Go regexps from embedded
	// gitleaks.toml config. Current Phase 1 approach uses 9 hardcoded patterns
	// plus HighEntropyCheck as catch-all. See spec Section 2.
	return &SecurityFilter{
		patterns: []secretPattern{
			{re: regexp.MustCompile(`AKIA[0-9A-Z]{16}`), typ: "aws_key", length: 20},
			{re: regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`), typ: "github_token", length: 40},
			{re: regexp.MustCompile(`-----BEGIN (RSA |EC )?PRIVATE KEY-----`), typ: "private_key"},
			{re: regexp.MustCompile(`sk-[0-9a-zA-Z]{48}`), typ: "openai_key"},
			{re: regexp.MustCompile(`(?i)(password|passwd)\s*[:=]\s*"[^"]+"`), typ: "password_assignment"},
			// C1: Additional secret patterns per security spec.
			{re: regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24,}`), typ: "stripe_live_key"},
			{re: regexp.MustCompile(`eyJ[A-Za-z0-9-_]{20,}\.eyJ[A-Za-z0-9-_]{20,}`), typ: "jwt_token"},
			{re: regexp.MustCompile(`xox[baprs]-[0-9]{10,}-[0-9]{10,}-[0-9a-zA-Z]{24,}`), typ: "slack_token"},
			{re: regexp.MustCompile(`//[^/@\s]+:[^/@\s]+@`), typ: "connection_string_credentials"},
		},
		AllowExternalLLM: false,
	}
}

// ExternalLLMAllowed returns whether sending data to external LLMs is permitted.
func (sf *SecurityFilter) ExternalLLMAllowed() bool {
	return sf.AllowExternalLLM
}

// ScanForSecrets detects secret patterns in content and returns matches.
func (sf *SecurityFilter) ScanForSecrets(content string) []SecretMatch {
	var matches []SecretMatch
	for _, p := range sf.patterns {
		locs := p.re.FindAllStringIndex(content, -1)
		for _, loc := range locs {
			length := loc[1] - loc[0]
			if p.length > 0 {
				length = p.length
			}
			matches = append(matches, SecretMatch{
				Type:     p.typ,
				Position: loc[0],
				Length:   length,
			})
		}
	}
	return matches
}

// reUserPath matches /Users/<username>/ paths (macOS).
var reUserPath = regexp.MustCompile(`/Users/[^/]+/`)

// reWindowsUserPath matches C:\Users\<username>\ paths (Windows).
var reWindowsUserPath = regexp.MustCompile(`C:\\Users\\[^\\]+`)

// reInternalPkg matches Java-style internal package names (com.xxx.yyy...).
var reInternalPkg = regexp.MustCompile(`\bcom\.\w+(?:\.\w+)+`)

// Sanitize removes secrets and PII from a CodebaseProfile.
// Returns a sanitized copy safe for LLM consumption and a SecretsFound report.
// All string fields are scrubbed; structural data (counts, metrics) pass through.
func (sf *SecurityFilter) Sanitize(profile *CodebaseProfile) (*CodebaseProfile, *SecretsFound) {
	result := &CodebaseProfile{
		Name:    profile.Name,
		Summary: sf.sanitizeString(profile.Summary),
	}

	secrets := &SecretsFound{Redacted: true}
	typeSet := make(map[string]bool)

	// Helper to scan and count secret matches.
	countSecrets := func(s string) {
		for _, m := range sf.ScanForSecrets(s) {
			secrets.Count++
			typeSet[m.Type] = true
		}
	}

	// Scan original profile fields for secrets before sanitization.
	countSecrets(profile.Summary)
	for _, tl := range profile.FileTree.TopLevel {
		countSecrets(tl)
	}
	for _, m := range profile.Dependencies.Manifests {
		countSecrets(m.Path)
	}
	for _, f := range profile.Files {
		countSecrets(f)
	}
	for k, v := range profile.Metadata {
		countSecrets(k)
		countSecrets(v)
	}

	// Collect unique types.
	for t := range typeSet {
		secrets.Types = append(secrets.Types, t)
	}

	// --- FileTree (top-level names may contain user paths) ---
	result.FileTree = profile.FileTree // counts are safe
	result.FileTree.TopLevel = sf.sanitizeStrings(profile.FileTree.TopLevel)

	// --- Dependencies ---
	result.Dependencies = profile.Dependencies // manifest paths are relative
	result.Dependencies.Manifests = sf.sanitizeManifests(profile.Dependencies.Manifests)

	// --- ImportGraph (package paths may leak usernames) ---
	result.ImportGraph = profile.ImportGraph
	result.ImportGraph.Clusters = sf.sanitizeClusters(profile.ImportGraph.Clusters)
	result.ImportGraph.CircularDependencies = sf.sanitizeCircularDeps(profile.ImportGraph.CircularDependencies)

	// --- Infra ---
	result.Infra = profile.Infra
	result.Infra.Deployment.Evidence = sf.sanitizeStrings(profile.Infra.Deployment.Evidence)
	result.Infra.Resources = sf.sanitizeResources(profile.Infra.Resources)

	// --- SQL ---
	if profile.SQLAnalysis != nil {
		sqlCopy := *profile.SQLAnalysis
		result.SQLAnalysis = &sqlCopy
	}

	// --- Git (scrub contributor names/emails) ---
	if profile.GitAnalysis != nil {
		gitCopy := *profile.GitAnalysis
		gitCopy.TopContributors = nil // PII: scrub contributor identities
		gitCopy.Ownership = nil       // PII: scrub contributor-to-dir mapping
		result.GitAnalysis = &gitCopy
	}

	// --- Specs ---
	result.Specs = sf.sanitizeSpecs(profile.Specs)

	// --- Metrics (safe — just numbers) ---
	result.Metrics = profile.Metrics

	// --- Files ---
	if profile.Files != nil {
		result.Files = make(map[string]string, len(profile.Files))
		for path, content := range profile.Files {
			sanitizedPath := sf.sanitizeString(path)
			result.Files[sanitizedPath] = sf.sanitizeString(content)
		}
	}

	// --- Metadata ---
	if profile.Metadata != nil {
		result.Metadata = make(map[string]string, len(profile.Metadata))
		for k, v := range profile.Metadata {
			result.Metadata[k] = sf.sanitizeString(v)
		}
	}

	return result, secrets
}

// --- Sanitize helpers ---

func (sf *SecurityFilter) sanitizeStrings(ss []string) []string {
	if ss == nil {
		return nil
	}
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = sf.sanitizeString(s)
	}
	return out
}

func (sf *SecurityFilter) sanitizeManifests(ms []ManifestInfo) []ManifestInfo {
	if ms == nil {
		return nil
	}
	out := make([]ManifestInfo, len(ms))
	for i, m := range ms {
		out[i] = ManifestInfo{
			Path:      sf.sanitizeString(m.Path),
			Language:  m.Language,
			DepsCount: m.DepsCount,
		}
	}
	return out
}

func (sf *SecurityFilter) sanitizeClusters(cs []ImportCluster) []ImportCluster {
	if cs == nil {
		return nil
	}
	out := make([]ImportCluster, len(cs))
	for i, c := range cs {
		out[i] = ImportCluster{
			ID:            c.ID,
			Packages:      sf.sanitizeStrings(c.Packages),
			InternalEdges: c.InternalEdges,
			ExternalEdges: c.ExternalEdges,
		}
	}
	return out
}

func (sf *SecurityFilter) sanitizeCircularDeps(cd []CircularDep) []CircularDep {
	if cd == nil {
		return nil
	}
	out := make([]CircularDep, len(cd))
	for i, d := range cd {
		out[i] = CircularDep{
			A:        sf.sanitizeString(d.A),
			B:        sf.sanitizeString(d.B),
			EdgeType: d.EdgeType,
		}
	}
	return out
}

func (sf *SecurityFilter) sanitizeResources(rs []ResourceInfo) []ResourceInfo {
	if rs == nil {
		return nil
	}
	out := make([]ResourceInfo, len(rs))
	for i, r := range rs {
		out[i] = ResourceInfo{
			Type:     r.Type,
			Name:     r.Name,
			Provider: r.Provider,
			Source:   sf.sanitizeString(r.Source),
		}
	}
	return out
}

func (sf *SecurityFilter) sanitizeSpecs(ss []SpecArtifact) []SpecArtifact {
	if ss == nil {
		return nil
	}
	out := make([]SpecArtifact, len(ss))
	for i, s := range ss {
		out[i] = SpecArtifact{
			Kind:    s.Kind,
			Path:    sf.sanitizeString(s.Path),
			Version: s.Version,
		}
	}
	return out
}

// sanitizeString applies all sanitization passes to a string.
func (sf *SecurityFilter) sanitizeString(s string) string {
	// 1. Redact secrets.
	for _, p := range sf.patterns {
		s = p.re.ReplaceAllStringFunc(s, func(match string) string {
			return "[REDACTED_" + p.typ + "]"
		})
	}

	// 2. Scrub user paths (macOS).
	s = reUserPath.ReplaceAllString(s, "/Users/[REDACTED]/")

	// 3. Scrub Windows user paths.
	s = reWindowsUserPath.ReplaceAllString(s, `C:\Users\[REDACTED]`)

	// 4. Hash internal package names.
	s = reInternalPkg.ReplaceAllStringFunc(s, func(match string) string {
		parts := strings.Split(match, ".")
		if len(parts) < 3 {
			return match
		}
		// Replace with "pkg." + last segment
		return "pkg." + parts[len(parts)-1]
	})

	return s
}

// SecretsFound represents secrets detected during sanitization.
type SecretsFound struct {
	Count    int      `json:"count"`
	Types    []string `json:"types,omitempty"`
	Redacted bool     `json:"redacted"`
}

// ---------------------------------------------------------------------------
// C5: Canonical ScrubSecrets (spec signature)
// ---------------------------------------------------------------------------

// ScrubSecrets applies regex-based secret redaction using the compiled patterns
// from SecurityFilter. It returns the scrubbed text, a count of redactions per
// secret type, and any error.
//
// This is the canonical entry point used by the enrichment pipeline.
func ScrubSecrets(text string) (scrubbed string, redactionCounts map[string]int, err error) {
	sf := NewSecurityFilter()
	redactionCounts = make(map[string]int)

	// Apply each pattern and count redactions.
	for _, p := range sf.patterns {
		newText := p.re.ReplaceAllStringFunc(text, func(match string) string {
			redactionCounts[p.typ]++
			return "[REDACTED_" + p.typ + "]"
		})
		text = newText
	}

	// Scrub user paths (macOS).
	macMatches := reUserPath.FindAllString(text, -1)
	if len(macMatches) > 0 {
		redactionCounts["user_path"] += len(macMatches)
		text = reUserPath.ReplaceAllString(text, "/Users/[REDACTED]/")
	}

	// Scrub user paths (Windows).
	winMatches := reWindowsUserPath.FindAllString(text, -1)
	if len(winMatches) > 0 {
		redactionCounts["windows_user_path"] += len(winMatches)
		text = reWindowsUserPath.ReplaceAllString(text, `C:\Users\[REDACTED]`)
	}

	return text, redactionCounts, nil
}

// ---------------------------------------------------------------------------
// C4: HighEntropyCheck — Shannon entropy detection
// ---------------------------------------------------------------------------

// highEntropyAllowlist contains exact full-string regex patterns for strings
// that are high-entropy but are known non-secrets (integrity hashes, UUIDs, etc.).
// Each pattern uses full anchors (^...$).
var highEntropyAllowlist = []*regexp.Regexp{
	regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),           // UUID
	regexp.MustCompile(`^sha(256|384|512)-[A-Za-z0-9+/=]+$`),                                          // integrity hash (SRI)
	regexp.MustCompile(`^[0-9a-f]{64}$`),                                                               // SHA256 hex
	regexp.MustCompile(`^[0-9a-f]{128}$`),                                                              // SHA512 hex
	regexp.MustCompile(`^[0-9a-f]{32}$`),                                                               // MD5 hex
}

// HighEntropyCheck flags strings with Shannon entropy > 4.5 and length >= 20
// that don't match known non-secret patterns.
//
// Allowlist uses EXACT regex patterns only — no broad category allowances:
//   - UUID: ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$
//   - Integrity hashes: ^sha(256|384|512)-[A-Za-z0-9+/=]+$
//   - SHA256 hex: ^[0-9a-f]{64}$
//   - SHA512 hex: ^[0-9a-f]{128}$
//   - MD5 hex: ^[0-9a-f]{32}$
//
// Each allowlisted pattern is matched as a full-string anchor (^...$).
func HighEntropyCheck(s string, context string) bool {
	if len(s) < 20 {
		return false
	}

	// Check allowlist first.
	for _, pattern := range highEntropyAllowlist {
		if pattern.MatchString(s) {
			return false
		}
	}

	// Compute Shannon entropy.
	entropy := shannonEntropy(s)
	if entropy > 4.5 {
		return true
	}

	_ = context // context reserved for future use (e.g. contextual allowlisting)
	return false
}

// shannonEntropy computes the Shannon entropy of a string in bits per character.
func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]float64)
	for _, r := range s {
		freq[r]++
	}

	length := float64(len(s))
	var entropy float64
	for _, count := range freq {
		p := count / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// ---------------------------------------------------------------------------
// C3: ValidatePath and ValidatedFile — TOCTOU-safe path validation
// ---------------------------------------------------------------------------

// ValidatedFile wraps an open file descriptor after path validation.
// Exposes only Read, Seek, Close — no path-based operations.
// Implements io.ReadSeekCloser.
type ValidatedFile struct {
	fd int
}

// Read reads up to len(p) bytes from the validated file.
func (f *ValidatedFile) Read(p []byte) (n int, err error) {
	return unix.Read(f.fd, p)
}

// Seek sets the offset for the next Read on the validated file.
func (f *ValidatedFile) Seek(offset int64, whence int) (int64, error) {
	return unix.Seek(f.fd, offset, whence)
}

// Close closes the underlying file descriptor and clears the finalizer.
func (f *ValidatedFile) Close() error {
	runtime.SetFinalizer(f, nil)
	return unix.Close(f.fd)
}

// ValidatePath ensures the resolved path is within repoRoot using TOCTOU-safe
// openat() with O_NOFOLLOW. It returns an io.ReadSeekCloser — callers can only
// Read/Seek/Close, no path operations.
//
// Platform support:
//   - Linux: unix.Openat with O_NOFOLLOW, /proc/self/fd for realpath
//   - macOS: unix.Openat with O_NOFOLLOW, fcntl F_GETPATH for realpath
//   - Windows: NOT SUPPORTED — returns error.
//
// Cleanup: callers MUST defer Close(). runtime.SetFinalizer is a backup only.
func ValidatePath(rawPath, repoRoot string) (io.ReadSeekCloser, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("ValidatePath: Windows is not supported")
	}

	// Step 1: Open the repo root directory to obtain a dirfd (anchored).
	rootFd, err := unix.Open(repoRoot, unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, fmt.Errorf("ValidatePath: open repo root %q: %w", repoRoot, err)
	}
	defer unix.Close(rootFd)

	// Step 2: Clean the relative path.
	relPath := filepath.Clean(rawPath)
	// Reject absolute paths — must be relative to repoRoot.
	if filepath.IsAbs(relPath) {
		return nil, fmt.Errorf("ValidatePath: path %q is absolute; must be relative to repo root", rawPath)
	}
	// Reject path traversal attempts.
	if strings.HasPrefix(relPath, "..") {
		return nil, fmt.Errorf("ValidatePath: path %q escapes repo root", rawPath)
	}

	// Step 3: Open the file relative to rootFd with O_NOFOLLOW.
	fd, err := unix.Openat(rootFd, relPath, unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("ValidatePath: open %q relative to %q: %w", rawPath, repoRoot, err)
	}

	// Step 4: Get the real path from the open fd.
	var realPath string
	switch runtime.GOOS {
	case "linux":
		linkPath := fmt.Sprintf("/proc/self/fd/%d", fd)
		realPath, err = os.Readlink(linkPath)
		if err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("ValidatePath: readlink /proc/self/fd/%d: %w", fd, err)
		}
	case "darwin":
		// macOS: use F_GETPATH via fcntl.
		// Allocate a buffer for the path (MAXPATHLEN = 1024 on macOS).
		buf := make([]byte, 1024)
		_, _, errno := unix.Syscall(unix.SYS_FCNTL, uintptr(fd), unix.F_GETPATH, uintptr(unsafe.Pointer(&buf[0])))
		if errno != 0 {
			unix.Close(fd)
			return nil, fmt.Errorf("ValidatePath: fcntl F_GETPATH fd=%d: %w", fd, errno)
		}
		// Buffer is a C string; find the null terminator.
		n := bytes.IndexByte(buf, 0)
		if n < 0 {
			n = len(buf)
		}
		realPath = string(buf[:n])
	default:
		unix.Close(fd)
		return nil, fmt.Errorf("ValidatePath: unsupported OS %q", runtime.GOOS)
	}

	// Step 5: Resolve the repo root.
	rootResolved, err := filepath.EvalSymlinks(filepath.Clean(repoRoot))
	if err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("ValidatePath: resolve repo root %q: %w", repoRoot, err)
	}

	// Step 6: Verify the real path is within the resolved repo root.
	if !strings.HasPrefix(realPath, rootResolved+string(filepath.Separator)) && realPath != rootResolved {
		unix.Close(fd)
		return nil, fmt.Errorf("ValidatePath: path %q resolves to %q which is outside repo root %q", rawPath, realPath, rootResolved)
	}

	// Step 7: Return the ValidatedFile wrapper.
	vf := &ValidatedFile{fd: fd}
	runtime.SetFinalizer(vf, func(f *ValidatedFile) {
		unix.Close(f.fd)
	})

	return vf, nil
}

