package export

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SIEMRecord represents a normalized audit/evidence record for SIEM export.
type SIEMRecord struct {
	Timestamp    time.Time              `json:"timestamp"`
	EventType    string                 `json:"event_type"`
	Source       string                 `json:"source"`
	Severity     string                 `json:"severity,omitempty"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	Actor        string                 `json:"actor,omitempty"`
	Resource     string                 `json:"resource,omitempty"`
	Action       string                 `json:"action,omitempty"`
	Outcome      string                 `json:"outcome,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Checksum     string                 `json:"checksum,omitempty"`
	Signature    string                 `json:"signature,omitempty"`
	RunID        string                 `json:"run_id,omitempty"`
	FeatureID    string                 `json:"feature_id,omitempty"`
	WorkstreamID string                 `json:"workstream_id,omitempty"`
}

// ComputeChecksum calculates SHA-256 over the canonical JSON of the record
// (excluding the checksum and signature fields).
func (r SIEMRecord) ComputeChecksum() string {
	r.Checksum = ""
	r.Signature = ""
	data, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return "sha256:" + sha256Hex(data)
}

// VerifyChecksum checks that the stored checksum matches the computed one.
func (r SIEMRecord) VerifyChecksum() bool {
	if r.Checksum == "" {
		return false
	}
	return r.Checksum == r.ComputeChecksum()
}

// SINK is the interface for writing SIEM records.
type SINK interface {
	Write(record SIEMRecord) error
}

// --- Syslog Sink ---

// SyslogSink writes structured JSON records to an io.Writer (stdout/file).
type SyslogSink struct {
	w io.Writer
}

// NewSyslogSink creates a sink that writes JSON lines to w.
func NewSyslogSink(w io.Writer) *SyslogSink {
	return &SyslogSink{w: w}
}

// Write serializes the record as a JSON line.
func (s *SyslogSink) Write(record SIEMRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal siem record: %w", err)
	}
	_, err = fmt.Fprintf(s.w, "%s\n", data)
	return err
}

// --- HTTP Sink ---

// HTTPSink posts records to a SIEM HTTP endpoint.
type HTTPSink struct {
	endpoint   string
	authToken  string
	client     *http.Client
	maxRetries int
	retryDelay time.Duration
}

// HTTPOption configures an HTTPSink.
type HTTPOption func(*HTTPSink)

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) HTTPOption {
	return func(s *HTTPSink) { s.maxRetries = n }
}

// WithRetryDelay sets the delay between retries.
func WithRetryDelay(d time.Duration) HTTPOption {
	return func(s *HTTPSink) { s.retryDelay = d }
}

// NewHTTPSink creates a sink that POSTs JSON to endpoint.
func NewHTTPSink(endpoint, authToken string, opts ...HTTPOption) *HTTPSink {
	s := &HTTPSink{
		endpoint:   endpoint,
		authToken:  authToken,
		client:     &http.Client{Timeout: 30 * time.Second},
		maxRetries: 3,
		retryDelay: 100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Write posts a single record to the HTTP endpoint with retries.
func (s *HTTPSink) Write(record SIEMRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal siem record: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < s.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(s.retryDelay)
		}

		req, err := http.NewRequest(http.MethodPost, s.endpoint, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if s.authToken != "" {
			req.Header.Set("Authorization", s.authToken)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http post: %w", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("http %d", resp.StatusCode)
	}

	return fmt.Errorf("exhausted %d retries: %w", s.maxRetries, lastErr)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
