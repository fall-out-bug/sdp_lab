package evidence

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	intotoDSSEPayloadType = "application/vnd.in-toto+json"
	githubOIDCAudience    = "sigstore"
)

type Signer struct {
	httpClient    *http.Client
	lookPath      func(file string) (string, error)
	runCommand    func(name string, args ...string) *exec.Cmd
	newTUFClient  func() (*tuf.Client, error)
	newNow        func() time.Time
	strictVerify  bool
	cosignEnabled bool
}

type dsseEnvelopeJSON struct {
	PayloadType string              `json:"payloadType"`
	Payload     string              `json:"payload"`
	Signatures  []dsseSignatureJSON `json:"signatures"`
}

type dsseSignatureJSON struct {
	KeyID string `json:"keyid,omitempty"`
	Sig   string `json:"sig"`
}

func NewSigner() *Signer {
	return &Signer{
		httpClient: &http.Client{Timeout: 20 * time.Second},
		lookPath:   exec.LookPath,
		runCommand: exec.Command,
		newTUFClient: func() (*tuf.Client, error) {
			return tuf.New(tuf.DefaultOptions())
		},
		newNow:        func() time.Time { return time.Now() },
		strictVerify:  os.Getenv("SDP_SIGSTORE_STRICT_VERIFY") == "1",
		cosignEnabled: true,
	}
}

func (s *Signer) SignAttestation(stmt CodingWorkflowStatement) ([]byte, error) {
	if s == nil {
		s = NewSigner()
	}

	payload, err := json.Marshal(stmt)
	if err != nil {
		return nil, fmt.Errorf("marshal attestation statement: %w", err)
	}

	idToken, hasOIDC, err := s.resolveOIDCToken(context.Background())
	if err != nil {
		return nil, err
	}

	if hasOIDC {
		signed, signErr := s.signWithSigstoreBundle(context.Background(), payload, idToken)
		if signErr == nil {
			return signed, nil
		}
		if s.isGitHubActions() {
			return nil, fmt.Errorf("keyless CI signing failed: %w", signErr)
		}
	}

	if s.cosignEnabled {
		if signed, cosignErr := s.signWithCosignBundle(payload); cosignErr == nil {
			return signed, nil
		}
	}

	return marshalUnsignedEnvelope(payload)
}

func (s *Signer) VerifyAttestation(signed []byte) (CodingWorkflowStatement, error) {
	if s == nil {
		s = NewSigner()
	}

	stmt, err := tryParseStatement(signed)
	if err == nil {
		return stmt, nil
	}

	b, bundleErr := unmarshalSigstoreBundle(signed)
	if bundleErr != nil {
		return CodingWorkflowStatement{}, fmt.Errorf("decode signed attestation: %w", err)
	}

	env := b.GetDsseEnvelope()
	if env == nil {
		return CodingWorkflowStatement{}, errors.New("sigstore bundle missing DSSE envelope")
	}

	if unmarshalErr := json.Unmarshal(env.Payload, &stmt); unmarshalErr != nil {
		return CodingWorkflowStatement{}, fmt.Errorf("unmarshal in-toto payload from DSSE envelope: %w", unmarshalErr)
	}

	if len(env.Signatures) == 0 {
		return stmt, nil
	}

	verifyErr := s.verifyBundle(b)
	if verifyErr != nil {
		if s.strictVerify {
			return CodingWorkflowStatement{}, verifyErr
		}
		return stmt, nil
	}

	return stmt, nil
}

func (s *Signer) verifyBundle(b *bundle.Bundle) error {
	tufClient, err := s.newTUFClient()
	if err != nil {
		return fmt.Errorf("initialize sigstore TUF client: %w", err)
	}

	trustedRoot, err := root.GetTrustedRoot(tufClient)
	if err != nil {
		return fmt.Errorf("load sigstore trusted root: %w", err)
	}

	verifier, err := verify.NewVerifier(trustedRoot, verify.WithTransparencyLog(1), verify.WithCurrentTime())
	if err != nil {
		return fmt.Errorf("create verifier: %w", err)
	}

	_, err = verifier.Verify(b, verify.NewPolicy(verify.WithoutArtifactUnsafe(), verify.WithoutIdentitiesUnsafe()))
	if err != nil {
		return fmt.Errorf("verify sigstore bundle: %w", err)
	}

	return nil
}

func (s *Signer) signWithSigstoreBundle(ctx context.Context, payload []byte, idToken string) ([]byte, error) {
	if idToken == "" {
		return nil, errors.New("missing OIDC token for keyless signing")
	}

	tufClient, err := s.newTUFClient()
	if err != nil {
		return nil, fmt.Errorf("initialize sigstore TUF client: %w", err)
	}

	signingConfig, err := root.GetSigningConfig(tufClient)
	if err != nil {
		return nil, fmt.Errorf("load sigstore signing config: %w", err)
	}

	trustedRoot, err := root.GetTrustedRoot(tufClient)
	if err != nil {
		return nil, fmt.Errorf("load sigstore trusted root: %w", err)
	}

	keypair, err := sign.NewEphemeralKeypair(nil)
	if err != nil {
		return nil, fmt.Errorf("create ephemeral keypair: %w", err)
	}

	fulcioService, err := root.SelectService(
		signingConfig.FulcioCertificateAuthorityURLs(),
		sign.FulcioAPIVersions,
		s.newNow(),
	)
	if err != nil {
		return nil, fmt.Errorf("select Fulcio service: %w", err)
	}

	rekorServices, err := root.SelectServices(
		signingConfig.RekorLogURLs(),
		signingConfig.RekorLogURLsConfig(),
		sign.RekorAPIVersions,
		s.newNow(),
	)
	if err != nil {
		return nil, fmt.Errorf("select Rekor service: %w", err)
	}

	if len(rekorServices) == 0 {
		return nil, errors.New("signing config returned no active Rekor services")
	}

	bundleOpts := sign.BundleOptions{
		Context: ctx,
		CertificateProvider: sign.NewFulcio(&sign.FulcioOptions{
			BaseURL: fulcioService.URL,
			Timeout: 30 * time.Second,
			Retries: 1,
		}),
		CertificateProviderOptions: &sign.CertificateProviderOptions{IDToken: idToken},
		TrustedRoot:                trustedRoot,
	}

	for _, rekorService := range rekorServices {
		bundleOpts.TransparencyLogs = append(bundleOpts.TransparencyLogs, sign.NewRekor(&sign.RekorOptions{
			BaseURL: rekorService.URL,
			Timeout: 90 * time.Second,
			Retries: 1,
			Version: rekorService.MajorAPIVersion,
		}))
	}

	b, err := sign.Bundle(&sign.DSSEData{Data: payload, PayloadType: intotoDSSEPayloadType}, keypair, bundleOpts)
	if err != nil {
		return nil, fmt.Errorf("sign DSSE attestation with sigstore: %w", err)
	}

	out, err := protojson.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("marshal sigstore bundle: %w", err)
	}

	return out, nil
}

func (s *Signer) signWithCosignBundle(payload []byte) ([]byte, error) {
	if _, err := s.lookPath("cosign"); err != nil {
		return nil, errors.New("cosign not available")
	}

	tmpDir, err := os.MkdirTemp("", "sdp-sigstore-cosign-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir for cosign signing: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	predicatePath := filepath.Join(tmpDir, "predicate.json")
	artifactPath := filepath.Join(tmpDir, "artifact.txt")
	bundlePath := filepath.Join(tmpDir, "attestation.sigstore.json")

	if err := os.WriteFile(predicatePath, payload, 0o600); err != nil {
		return nil, fmt.Errorf("write temporary payload for cosign: %w", err)
	}
	if err := os.WriteFile(artifactPath, []byte("sdp-auto-attestation"), 0o600); err != nil {
		return nil, fmt.Errorf("write temporary artifact for cosign: %w", err)
	}

	args := []string{
		"attest-blob",
		"--yes",
		"--predicate", predicatePath,
		"--type", PredicateTypeCodingWorkflow,
		"--bundle", bundlePath,
		artifactPath,
	}
	if tok := strings.TrimSpace(os.Getenv("SIGSTORE_ID_TOKEN")); tok != "" {
		args = append(args, "--identity-token", tok)
	}
	if key := strings.TrimSpace(os.Getenv("COSIGN_KEY")); key != "" {
		args = append(args, "--key", key)
	}

	cmd := s.runCommand("cosign", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cosign attest-blob failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	b, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("read cosign bundle output: %w", err)
	}

	if _, err := unmarshalSigstoreBundle(b); err != nil {
		return nil, fmt.Errorf("parse cosign bundle output: %w", err)
	}

	return b, nil
}

func (s *Signer) resolveOIDCToken(ctx context.Context) (string, bool, error) {
	if tok := strings.TrimSpace(os.Getenv("SIGSTORE_ID_TOKEN")); tok != "" {
		return tok, true, nil
	}

	if !s.isGitHubActions() {
		return "", false, nil
	}

	requestToken := strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN"))
	requestURL := strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"))
	if requestToken == "" || requestURL == "" {
		return "", false, errors.New("github actions OIDC requested but ACTIONS_ID_TOKEN_REQUEST_TOKEN or ACTIONS_ID_TOKEN_REQUEST_URL not set")
	}

	token, err := s.fetchGitHubOIDCToken(ctx, requestURL, requestToken, githubOIDCAudience)
	if err != nil {
		return "", false, fmt.Errorf("fetch GitHub Actions OIDC token: %w", err)
	}

	return token, true, nil
}

func (s *Signer) fetchGitHubOIDCToken(ctx context.Context, requestURL, requestToken, audience string) (string, error) {
	u, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("parse ACTIONS_ID_TOKEN_REQUEST_URL: %w", err)
	}

	q := u.Query()
	q.Set("audience", audience)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create OIDC token request: %w", err)
	}
	req.Header.Set("Authorization", "bearer "+requestToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request OIDC token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read OIDC token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("OIDC token endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Value string `json:"value"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode OIDC token response: %w", err)
	}

	token := strings.TrimSpace(parsed.Value)
	if token == "" {
		token = strings.TrimSpace(parsed.Token)
	}
	if token == "" {
		return "", errors.New("OIDC token response missing token value")
	}

	return token, nil
}

func (s *Signer) isGitHubActions() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GITHUB_ACTIONS")), "true")
}

func tryParseStatement(data []byte) (CodingWorkflowStatement, error) {
	var stmt CodingWorkflowStatement
	if err := json.Unmarshal(data, &stmt); err == nil && stmt.Type == StatementType {
		return stmt, nil
	}

	var env dsseEnvelopeJSON
	if err := json.Unmarshal(data, &env); err != nil {
		return CodingWorkflowStatement{}, fmt.Errorf("parse JSON payload: %w", err)
	}

	if env.PayloadType == "" || env.Payload == "" {
		return CodingWorkflowStatement{}, errors.New("not a coding workflow statement or DSSE envelope")
	}

	payload, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return CodingWorkflowStatement{}, fmt.Errorf("decode DSSE payload: %w", err)
	}

	if err := json.Unmarshal(payload, &stmt); err != nil {
		return CodingWorkflowStatement{}, fmt.Errorf("decode statement from DSSE payload: %w", err)
	}

	if stmt.Type != StatementType {
		return CodingWorkflowStatement{}, errors.New("DSSE payload is not an in-toto statement")
	}

	return stmt, nil
}

func marshalUnsignedEnvelope(payload []byte) ([]byte, error) {
	env := dsseEnvelopeJSON{
		PayloadType: intotoDSSEPayloadType,
		Payload:     base64.StdEncoding.EncodeToString(payload),
		Signatures:  []dsseSignatureJSON{},
	}

	b, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("marshal unsigned DSSE envelope: %w", err)
	}

	return b, nil
}

func unmarshalSigstoreBundle(data []byte) (*bundle.Bundle, error) {
	var pb struct {
		MediaType string `json:"mediaType"`
	}
	if err := json.Unmarshal(data, &pb); err != nil {
		return nil, fmt.Errorf("parse candidate sigstore bundle: %w", err)
	}
	if !strings.Contains(pb.MediaType, "sigstore.bundle") {
		return nil, errors.New("payload is not a sigstore bundle")
	}

	b := &bundle.Bundle{}
	if err := b.UnmarshalJSON(data); err != nil {
		return nil, fmt.Errorf("unmarshal sigstore bundle JSON: %w", err)
	}

	return b, nil
}
