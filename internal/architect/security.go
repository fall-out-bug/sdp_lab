package architect

import (
	"regexp"
	"strings"
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
	return &SecurityFilter{
		patterns: []secretPattern{
			{re: regexp.MustCompile(`AKIA[0-9A-Z]{16}`), typ: "aws_key", length: 20},
			{re: regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`), typ: "github_token", length: 40},
			{re: regexp.MustCompile(`-----BEGIN (RSA |EC )?PRIVATE KEY-----`), typ: "private_key"},
			{re: regexp.MustCompile(`sk-[0-9a-zA-Z]{48}`), typ: "openai_key"},
			{re: regexp.MustCompile(`(?i)(password|passwd)\s*[:=]\s*"[^"]+"`), typ: "password_assignment"},
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

// reUserPath matches /Users/<username>/ paths.
var reUserPath = regexp.MustCompile(`/Users/[^/]+/`)

// reInternalPkg matches Java-style internal package names (com.xxx.yyy...).
var reInternalPkg = regexp.MustCompile(`\bcom\.\w+(?:\.\w+)+`)

// Sanitize removes secrets and PII from a CodebaseProfile.
// Returns a sanitized copy safe for LLM consumption.
// All string fields are scrubbed; structural data (counts, metrics) pass through.
func (sf *SecurityFilter) Sanitize(profile *CodebaseProfile) *CodebaseProfile {
	result := &CodebaseProfile{
		Name:    profile.Name,
		Summary: sf.sanitizeString(profile.Summary),
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

	return result
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
			return "[REDACTED:" + p.typ + "]"
		})
	}

	// 2. Scrub user paths.
	s = reUserPath.ReplaceAllString(s, "/Users/[REDACTED]/")

	// 3. Hash internal package names.
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
