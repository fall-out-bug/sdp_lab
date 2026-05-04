package llmguard

import (
	"encoding/base64"
	"regexp"
	"strings"
)

// rule defines a single detection rule.
type rule struct {
	Type     FindingType
	Severity FindingSeverity
	Pattern  *regexp.Regexp
}

// Scanner detects secrets and policy violations in text.
type Scanner struct {
	rules            []rule
	maxInputBytes    int
	maxDecodedBytes  int
	strictBudgetMode bool
}

// NewScanner creates a Scanner with default detection rules.
func NewScanner(maxInputBytes, maxDecodedBytes int, strictBudgetMode bool) *Scanner {
	s := &Scanner{
		maxInputBytes:    maxInputBytes,
		maxDecodedBytes:  maxDecodedBytes,
		strictBudgetMode: strictBudgetMode,
	}
	s.rules = s.defaultRules()
	return s
}

// NewScannerFromPolicy creates a Scanner from a Policy.
func NewScannerFromPolicy(p Policy) *Scanner {
	return NewScanner(p.MaxInputBytes, p.MaxDecodedBytes, p.StrictBudgetMode)
}

func (*Scanner) defaultRules() []rule {
	return []rule{
		// OpenAI keys: sk-proj-... and sk-... (not sk-ant- which is Anthropic)
		{FindingOpenAIKey, SeverityHigh, regexp.MustCompile(`sk-proj-[A-Za-z0-9_-]{20,}`)},
		{FindingOpenAIKey, SeverityHigh, regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`)},

		// GitHub tokens
		{FindingGitHubToken, SeverityHigh, regexp.MustCompile(`ghp_[A-Za-z0-9]{36,}`)},

		// AWS access keys
		{FindingAWSKey, SeverityHigh, regexp.MustCompile(`AKIA[A-Z0-9]{16}`)},

		// Bearer tokens (generic)
		{FindingBearerToken, SeverityHigh, regexp.MustCompile(`Bearer\s+[A-Za-z0-9_\-.~+/]+=*`)},

		// Email addresses
		{FindingEmail, SeverityLow, regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)},

		// Phone numbers (various formats)
		{FindingPhone, SeverityLow, regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)},

		// Card-like numbers (13-19 digits with optional spaces/dashes)
		{FindingCard, SeverityHigh, regexp.MustCompile(`\b(\d[\s-]?){12,18}\d\b`)},
	}
}

// Scan scans text and returns findings and traces.
func (s *Scanner) Scan(text string) ScanResult {
	if len(text) > s.maxInputBytes {
		return ScanResult{BudgetExceeded: true}
	}

	var result ScanResult

	// Phase 1: raw scan
	rawFindings, rawTraces := s.scanRaw(text)
	result.Findings = append(result.Findings, rawFindings...)
	result.Traces = append(result.Traces, rawTraces...)

	// Phase 2: base64 decode scan (only add findings not already found in raw)
	b64Findings, b64Traces := s.scanBase64(text)
	for _, f := range b64Findings {
		if !spanOverlaps(f.SpanStart, f.SpanEnd, result.Findings) {
			result.Findings = append(result.Findings, f)
		}
	}
	result.Traces = append(result.Traces, b64Traces...)

	// Phase 3: split-join scan for adjacent fragments (only add if not already found)
	splitFindings, splitTraces := s.scanSplitJoined(text)
	for _, f := range splitFindings {
		if !spanOverlaps(f.SpanStart, f.SpanEnd, result.Findings) {
			result.Findings = append(result.Findings, f)
		}
	}
	result.Traces = append(result.Traces, splitTraces...)

	return result
}

// ScanOutput scans model output for suspicious content.
func (s *Scanner) ScanOutput(text string) ScanResult {
	if len(text) > s.maxInputBytes {
		return ScanResult{BudgetExceeded: true}
	}

	var result ScanResult

	// Check for generated secrets using input rules on output
	rawFindings, rawTraces := s.scanRaw(text)
	result.Findings = append(result.Findings, rawFindings...)
	result.Traces = append(result.Traces, rawTraces...)

	// Check for prompt/system prompt disclosure
	disclosureFindings := s.checkPromptDisclosure(text)
	result.Findings = append(result.Findings, disclosureFindings...)

	// Check for suspicious URLs
	urlFindings := s.checkSuspiciousURLs(text)
	result.Findings = append(result.Findings, urlFindings...)

	// Check for shell commands
	shellFindings := s.checkShellCommands(text)
	result.Findings = append(result.Findings, shellFindings...)

	return result
}

func (s *Scanner) scanRaw(text string) ([]Finding, []ScanTrace) {
	var findings []Finding
	var traces []ScanTrace

	for _, r := range s.rules {
		locs := r.Pattern.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			matched := text[loc[0]:loc[1]]
			// For card-like numbers, validate with Luhn
			if r.Type == FindingCard {
				digits := stripNonDigits(matched)
				if !luhnValid(digits) {
					traces = append(traces, ScanTrace{
						Mode:            ScanModeRaw,
						Matched:         false,
						RedactedExcerpt: shortExcerpt(matched, 20),
					})
					continue
				}
			}
			findings = append(findings, Finding{
				Type:      r.Type,
				Severity:  r.Severity,
				SpanStart: loc[0],
				SpanEnd:   loc[1],
				Redacted:  replaceMatch(text, loc[0], loc[1], RedactedPlaceholder(r.Type)),
				ScanMode:  ScanModeRaw,
			})
			traces = append(traces, ScanTrace{
				Mode:            ScanModeRaw,
				CandidatesTried: 1,
				Matched:         true,
				RedactedExcerpt: RedactedPlaceholder(r.Type),
			})
		}
	}

	return findings, traces
}

func (s *Scanner) scanBase64(text string) ([]Finding, []ScanTrace) {
	var findings []Finding
	var traces []ScanTrace

	// Find potential base64 strings (at least 40 chars, padded)
	b64Pattern := regexp.MustCompile(`[A-Za-z0-9+/]{40,}={0,2}`)
	candidates := b64Pattern.FindAllStringIndex(text, -1)

	for _, loc := range candidates {
		candidate := text[loc[0]:loc[1]]
		decoded, err := base64.StdEncoding.DecodeString(candidate)
		if err != nil {
			// Try URL-safe encoding
			decoded, err = base64.URLEncoding.DecodeString(candidate)
			if err != nil {
				continue
			}
		}

		if len(decoded) > s.maxDecodedBytes {
			traces = append(traces, ScanTrace{
				Mode:            ScanModeBase64Decoded,
				CandidatesTried: 1,
				Matched:         false,
				RedactedExcerpt: "[decoded exceeds budget]",
			})
			continue
		}

		decodedStr := string(decoded)

		// Scan decoded content with input rules
		for _, r := range s.rules {
			if r.Pattern.MatchString(decodedStr) {
				findings = append(findings, Finding{
					Type:      r.Type,
					Severity:  r.Severity,
					SpanStart: loc[0],
					SpanEnd:   loc[1],
					Redacted:  replaceMatch(text, loc[0], loc[1], RedactedPlaceholder(r.Type)),
					ScanMode:  ScanModeBase64Decoded,
				})
				traces = append(traces, ScanTrace{
					Mode:            ScanModeBase64Decoded,
					CandidatesTried: 1,
					Matched:         true,
					RedactedExcerpt: RedactedPlaceholder(r.Type),
				})
			}
		}
	}

	return findings, traces
}

// scanSplitJoined detects secrets split into adjacent fragments in the same message.
// "Simple adjacent" means fragments in original order with ≤16 ASCII non-alphanumeric
// separator bytes between them.
func (s *Scanner) scanSplitJoined(text string) ([]Finding, []ScanTrace) {
	var findings []Finding
	var traces []ScanTrace

	// Look for prefix fragments like "sk-" or "sk-proj-" followed by the rest
	prefixPatterns := []struct {
		prefix   string
		fullRule *regexp.Regexp
		ruleType FindingType
	}{
		{"sk-proj-", regexp.MustCompile(`sk-proj-[A-Za-z0-9_-]{20,}`), FindingOpenAIKey},
		{"sk-", regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`), FindingOpenAIKey},
		{"ghp_", regexp.MustCompile(`ghp_[A-Za-z0-9]{36,}`), FindingGitHubToken},
		{"AKIA", regexp.MustCompile(`AKIA[A-Z0-9]{16}`), FindingAWSKey},
	}

	for _, pp := range prefixPatterns {
		// Find all occurrences of the prefix
		prefixLocs := findAllSubstring(text, pp.prefix)
		for _, pLoc := range prefixLocs {
			afterPrefix := text[pLoc+len(pp.prefix):]
			sepCount := 0
			contStart := -1
			for i, ch := range []byte(afterPrefix) {
				if isAlphanumeric(ch) || ch == '_' || ch == '-' {
					contStart = i
					break
				}
				if !isAlphanumeric(ch) {
					sepCount++
					if sepCount > 16 {
						break
					}
				}
			}
			if contStart < 0 || sepCount > 16 {
				continue
			}

			end := findAlphaRunEnd(afterPrefix, contStart)
			if end <= contStart {
				continue
			}
			continuation := afterPrefix[contStart:end]
			joinedCandidate := pp.prefix + continuation
			if !pp.fullRule.MatchString(joinedCandidate) {
				continue
			}

			absStart := pLoc
			absEnd := pLoc + len(pp.prefix) + end
			findings = append(findings, Finding{
				Type:      pp.ruleType,
				Severity:  SeverityHigh,
				SpanStart: absStart,
				SpanEnd:   absEnd,
				Redacted:  replaceMatch(text, absStart, absEnd, RedactedPlaceholder(pp.ruleType)),
				ScanMode:  ScanModeSplitJoined,
			})
			traces = append(traces, ScanTrace{
				Mode:            ScanModeSplitJoined,
				CandidatesTried: 1,
				Matched:         true,
				RedactedExcerpt: RedactedPlaceholder(pp.ruleType),
			})
		}
	}

	return findings, traces
}

func (s *Scanner) checkPromptDisclosure(text string) []Finding {
	var findings []Finding
	// Patterns suggesting the model is disclosing its system prompt
	patterns := []struct {
		pattern *regexp.Regexp
		label   string
	}{
		{regexp.MustCompile(`(?i)my\s+system\s+prompt\s+(?:is|says|states|tells?\s+me)`), "system prompt disclosure attempt"},
		{regexp.MustCompile(`(?i)I\s+was\s+instructed\s+to`), "instruction disclosure"},
		{regexp.MustCompile(`(?i)my\s+instructions?\s+(?:are|state|say|tell)\s+`), "instruction disclosure"},
	}

	for _, p := range patterns {
		locs := p.pattern.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			findings = append(findings, Finding{
				Type:      FindingPromptDisclosure,
				Severity:  SeverityLow,
				SpanStart: loc[0],
				SpanEnd:   loc[1],
				Redacted:  replaceMatch(text, loc[0], loc[1], RedactedPlaceholder(FindingPromptDisclosure)),
				ScanMode:  ScanModeRaw,
			})
		}
	}

	return findings
}

func (s *Scanner) checkSuspiciousURLs(text string) []Finding {
	var findings []Finding
	// Look for suspicious URLs (data exfiltration patterns)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`https?://[^\s]+\?[A-Za-z_]+=[^\s&]*[A-Za-z0-9+/]{20,}`),
		regexp.MustCompile(`https?://[^\s]*(?:webhook|callback|exfil|leak|capture)[^\s]*`),
	}

	for _, p := range patterns {
		locs := p.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			findings = append(findings, Finding{
				Type:      FindingSuspiciousURL,
				Severity:  SeverityLow,
				SpanStart: loc[0],
				SpanEnd:   loc[1],
				Redacted:  replaceMatch(text, loc[0], loc[1], RedactedPlaceholder(FindingSuspiciousURL)),
				ScanMode:  ScanModeRaw,
			})
		}
	}

	return findings
}

func (s *Scanner) checkShellCommands(text string) []Finding {
	var findings []Finding
	// Detect shell-like command patterns that might indicate injection
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:^|\n)\s*(?:curl|wget|bash|sh|python|perl|ruby|nc|ncat)\s+[^\n]{10,}`),
		regexp.MustCompile(`(?:^|\n)\s*(?:rm|chmod|chown|sudo|eval|exec)\s+[^\n]{5,}`),
	}

	for _, p := range patterns {
		locs := p.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			findings = append(findings, Finding{
				Type:      FindingShellCommand,
				Severity:  SeverityLow,
				SpanStart: loc[0],
				SpanEnd:   loc[1],
				Redacted:  replaceMatch(text, loc[0], loc[1], RedactedPlaceholder(FindingShellCommand)),
				ScanMode:  ScanModeRaw,
			})
		}
	}

	return findings
}

// --- helper functions ---

func replaceMatch(text string, start, end int, replacement string) string {
	return text[:start] + replacement + text[end:]
}

func shortExcerpt(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func stripNonDigits(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// luhnValid checks if a string of digits passes the Luhn algorithm.
func luhnValid(digits string) bool {
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	n := len(digits)
	sum := 0
	parity := n % 2
	for i := 0; i < n; i++ {
		d := int(digits[i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		if i%2 == parity {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}

func isAlphanumeric(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func findAlphaRunEnd(s string, start int) int {
	for i := start; i < len(s); i++ {
		if !isAlphanumeric(s[i]) && s[i] != '_' && s[i] != '-' {
			return i
		}
	}
	return len(s)
}

func findAllSubstring(s, substr string) []int {
	var locs []int
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			locs = append(locs, i)
		}
	}
	return locs
}

// spanOverlaps returns true if [start, end) overlaps with any existing finding's span.
func spanOverlaps(start, end int, existing []Finding) bool {
	for _, f := range existing {
		if start < f.SpanEnd && end > f.SpanStart {
			return true
		}
	}
	return false
}
