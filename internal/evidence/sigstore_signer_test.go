package evidence

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/sigstore/sigstore-go/pkg/tuf"
)

func TestSignAttestationFallsBackToUnsignedDSSE(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
	t.Setenv("SIGSTORE_ID_TOKEN", "")

	s := newTestSigner()
	s.lookPath = func(string) (string, error) {
		return "", exec.ErrNotFound
	}

	stmt := testStatement()
	signed, err := s.SignAttestation(stmt)
	if err != nil {
		t.Fatalf("SignAttestation returned error: %v", err)
	}

	var env dsseEnvelopeJSON
	if err := json.Unmarshal(signed, &env); err != nil {
		t.Fatalf("signed payload is not DSSE JSON: %v", err)
	}

	if env.PayloadType != intotoDSSEPayloadType {
		t.Fatalf("payloadType = %q, want %q", env.PayloadType, intotoDSSEPayloadType)
	}
	if len(env.Signatures) != 0 {
		t.Fatalf("expected no signatures in unsigned fallback, got %d", len(env.Signatures))
	}

	payload, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		t.Fatalf("decode envelope payload: %v", err)
	}

	var got CodingWorkflowStatement
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("decode statement from envelope: %v", err)
	}
	if got.Predicate.Provenance.RunID != stmt.Predicate.Provenance.RunID {
		t.Fatalf("unexpected run_id: got %q want %q", got.Predicate.Provenance.RunID, stmt.Predicate.Provenance.RunID)
	}
}

func TestVerifyAttestationSupportsLegacyUnsignedStatement(t *testing.T) {
	stmt := testStatement()
	b, err := json.Marshal(stmt)
	if err != nil {
		t.Fatalf("marshal statement: %v", err)
	}

	got, err := newTestSigner().VerifyAttestation(b)
	if err != nil {
		t.Fatalf("VerifyAttestation returned error: %v", err)
	}

	if got.Type != StatementType {
		t.Fatalf("unexpected statement type: %q", got.Type)
	}
	if got.Predicate.Provenance.RunID != stmt.Predicate.Provenance.RunID {
		t.Fatalf("unexpected run_id: got %q want %q", got.Predicate.Provenance.RunID, stmt.Predicate.Provenance.RunID)
	}
}

func TestSignAttestationCIKeylessFailureReturnsError(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "test-request-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"value":"oidc-token"}`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", server.URL)

	s := newTestSigner()
	s.httpClient = server.Client()
	s.newTUFClient = func() (*tuf.Client, error) {
		return nil, errors.New("boom")
	}
	s.lookPath = func(string) (string, error) {
		return "", exec.ErrNotFound
	}

	_, err := s.SignAttestation(testStatement())
	if err == nil {
		t.Fatal("expected CI keyless signing failure to return error")
	}
}

func TestFetchGitHubOIDCTokenAddsAudience(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("audience"); got != githubOIDCAudience {
			t.Fatalf("audience query = %q, want %q", got, githubOIDCAudience)
		}
		if got := r.Header.Get("Authorization"); got != "bearer req-token" {
			t.Fatalf("authorization header = %q", got)
		}
		_, _ = w.Write([]byte(`{"value":"issued-token"}`))
	}))
	t.Cleanup(server.Close)

	s := newTestSigner()
	s.httpClient = server.Client()

	tok, err := s.fetchGitHubOIDCToken(t.Context(), server.URL, "req-token", githubOIDCAudience)
	if err != nil {
		t.Fatalf("fetchGitHubOIDCToken returned error: %v", err)
	}
	if tok != "issued-token" {
		t.Fatalf("token = %q, want %q", tok, "issued-token")
	}
}

func TestVerifyAttestationRejectsInvalidPayload(t *testing.T) {
	_, err := newTestSigner().VerifyAttestation([]byte(`{"not":"attestation"}`))
	if err == nil {
		t.Fatal("expected error for invalid attestation payload")
	}
}

func newTestSigner() *Signer {
	s := NewSigner()
	s.newNow = func() time.Time {
		return time.Unix(1_700_000_000, 0).UTC()
	}
	return s
}

func testStatement() CodingWorkflowStatement {
	//nolint:staticcheck // compatibility with current statement constructor expecting in-toto v0 Subject
	return NewStatement([]intoto.Subject{{Name: "test", Digest: map[string]string{"sha256": "abc123"}}}, CodingWorkflowPredicate{
		Intent:       Intent{IssueID: "sdp_dev-test", Trigger: "unit-test", RiskClass: "low"},
		Plan:         Plan{Workstreams: []string{"00-001-01"}, OrderingRationale: "test"},
		Execution:    Execution{Branch: "feature/test", ChangedFiles: []string{"internal/evidence/sigstore_signer.go"}},
		Verification: Verification{Tests: []GateResult{{Name: "go-test", Status: "pass"}}},
		Boundary:     Boundary{Compliance: BoundaryCompliance{OK: true, Reason: "test"}},
		Provenance:   Provenance{RunID: "run-test-1", CapturedAt: "2026-01-01T00:00:00Z", Orchestrator: "test", Runtime: "local"},
	})
}
