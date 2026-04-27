package evidence

const ContractVersion = "v1.0.0"

// Attester defines the v1 contract for creating and validating attestations.
// This is an aspirational unified interface — no single existing type implements all four methods.
// The contract defines the v1 surface that consumers should target; existing code uses
// package-level functions that will be refactored to implement this interface.
type Attester interface {
	// Attest generates a CodingWorkflowStatement from CI/CD observations.
	// The opts parameter contains runtime configuration like branch, PR info, and repo root.
	Attest(opts AutoAttestOptions) (CodingWorkflowStatement, error)

	// Validate checks that a statement meets schema and business rules.
	// If requirePRURL is true, the statement must include a PR URL in trace.pr_url.
	Validate(stmt CodingWorkflowStatement, requirePRURL bool) Result

	// Sign creates a cryptographic signature over the statement using Sigstore.
	// Returns a DSSE envelope or Sigstore bundle. If signing fails, may return
	// an unsigned envelope.
	Sign(stmt CodingWorkflowStatement) ([]byte, error)

	// Verify extracts and validates a statement from a signed envelope.
	// If strict verification is enabled, cryptographic signatures are validated.
	// Returns the extracted statement or an error if verification fails.
	Verify(signed []byte) (CodingWorkflowStatement, error)
}

// DiscrepancyDetector defines the v1 contract for comparing agent and CI attestations.
// Implementations detect drift between claimed and observed evidence.
type DiscrepancyDetector interface {
	// Compare analyzes agent and CI attestations for the same runID.
	// Takes paths to agent and CI directories containing attestation files.
	// Returns a report listing any discrepancies with severity ratings.
	// Coverage thresholds and file scope limits are configurable via compareOptions.
	Compare(runID string, agentDir, ciDir string) (DiscrepancyReport, error)
}

// Inspector defines the v1 contract for validating and formatting evidence files.
// Implementations support both in-toto attestations and legacy evidence formats.
type Inspector interface {
	// Inspect reads an evidence file, validates it, and returns a formatted summary.
	// If requirePRURL is true, validation fails for statements missing trace.pr_url.
	// Returns formatted summary text, validation result, and any error.
	Inspect(path string, requirePRURL bool) (string, Result, error)
}

// TraceValidator defines the v1 contract for validating trace event chains.
// Implementations check phase completeness and temporal consistency.
type TraceValidator interface {
	// ValidateChain validates trace events contain required phases and detects gaps.
	// Maps to ValidateTraceChain function.
	// Returns validation result with missing phases and detected time gaps.
	ValidateChain(events []TraceEvent) TraceValidationResult
}

// IngestContract documents the v1 ingestion contract.
//
// Idempotency: Evidence ingestion is idempotent by runID+subject digest.
// Re-ingesting the same (runID, subject) combination replaces the previous
// evidence file without side effects.
//
// At-most-once: Cryptographic signing operations are at-most-once per runID.
// Attempting to sign the same runID twice should be rejected or detected as
// a duplicate. This prevents split-brain scenarios where multiple actors claim
// the same run.
//
// Collision handling: If two agents attempt to ingest for the same runID,
// the last write wins based on filesystem timestamp. Implementations should
// validate that runID is unique per workspace before signing.
type IngestContract struct {
	// RunID uniquely identifies this execution across all evidence
	RunID string

	// SubjectDigest is the SHA-256 digest of the statement subject (usually commit SHA)
	SubjectDigest string

	// Timestamp of ingestion, used for conflict resolution
	Timestamp string
}

// RenderContract documents the v1 rendering contract.
//
// Serialization format: All CodingWorkflowStatement payloads MUST be serialized
// as canonical JSON (json.Marshal with indentation). Binary formats are not
// supported in v1.
//
// Statement type header: Every attestation MUST include the in-toto StatementHeader
// with Type = "https://in-toto.io/Statement/v1" and PredicateType set to the
// CodingWorkflow predicate type. Legacy formats without these headers are accepted
// for backward compatibility but will not be emitted.
//
// Digest algorithm: SHA-256 is the only supported hash algorithm for subject
// digests, prompt_hash, context_source hashes, and payload digests. Other algorithms
// are rejected during validation.
//
// Rendered output: The primary rendering format is JSON files on disk at
// .sdp/evidence/{prefix}{runID}.json. Console output uses human-readable
// text summaries. HTML and other formats are not supported in v1.
type RenderContract struct {
	// Format is always "json" for v1
	Format string

	// Canonical JSON is required for reproducibility
	CanonicalJSON bool

	// StatementHeader fields must be present
	StatementHeaderRequired bool
}
