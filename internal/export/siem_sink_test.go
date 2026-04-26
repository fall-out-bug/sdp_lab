package export

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// --- SIEMRecord tests ---

func TestSIEMRecordSerialization(t *testing.T) {
	rec := SIEMRecord{
		Timestamp:   time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		EventType:   "evidence.gate.evaluated",
		Source:      "sdp-lab",
		Severity:    "info",
		TenantID:    "tenant-1",
		Actor:       "operator-1",
		Resource:    "workstream-1",
		Action:      "gate.evaluate",
		Outcome:     "allowed",
		Details:     map[string]interface{}{"discrepancies": 0},
		Checksum:    "sha256:abc123",
		Signature:   "sigstore:xyz789",
		RunID:       "run-001",
		FeatureID:   "F074",
		WorkstreamID: "00-074-02",
	}

	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed SIEMRecord
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if parsed.EventType != "evidence.gate.evaluated" {
		t.Errorf("expected evidence.gate.evaluated, got %s", parsed.EventType)
	}
	if parsed.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", parsed.TenantID)
	}
	if parsed.FeatureID != "F074" {
		t.Errorf("expected F074, got %s", parsed.FeatureID)
	}
}

// --- SyslogSink tests ---

func TestSyslogSinkWrite(t *testing.T) {
	var buf bytes.Buffer
	sink := NewSyslogSink(&buf)

	rec := SIEMRecord{
		Timestamp: time.Now().UTC(),
		EventType: "audit.access",
		Source:    "sdp-lab",
		Severity:  "info",
		Actor:     "user-1",
		Action:    "workstream.read",
		Outcome:   "allowed",
	}

	err := sink.Write(context.Background(), rec)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected output in buffer")
	}

	// Verify it's valid JSON (structured syslog payload)
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["event_type"] != "audit.access" {
		t.Errorf("expected audit.access, got %v", parsed["event_type"])
	}
}

func TestSyslogSinkWriteMultiple(t *testing.T) {
	var buf bytes.Buffer
	sink := NewSyslogSink(&buf)

	for i := 0; i < 5; i++ {
		rec := SIEMRecord{
			Timestamp: time.Now().UTC(),
			EventType: fmt.Sprintf("event.%d", i),
			Source:    "sdp-lab",
			Severity:  "info",
		}
		if err := sink.Write(context.Background(), rec); err != nil {
			t.Fatalf("write %d failed: %v", i, err)
		}
	}

	// Each record should be on its own line
	lines := bytes.Count(buf.Bytes(), []byte("\n"))
	if lines != 5 {
		t.Errorf("expected 5 lines, got %d", lines)
	}
}

// --- HTTPSink tests ---

func TestHTTPSinkPost(t *testing.T) {
	var received []SIEMRecord
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		var rec SIEMRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			t.Errorf("decode failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		received = append(received, rec)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink := NewHTTPSink(server.URL, "test-token")

	rec := SIEMRecord{
		Timestamp: time.Now().UTC(),
		EventType: "evidence.export",
		Source:    "sdp-lab",
		Severity:  "info",
		Actor:     "system",
		Action:    "export.siem",
		Outcome:   "success",
	}

	err := sink.Write(context.Background(), rec)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 record, got %d", len(received))
	}
	if received[0].EventType != "evidence.export" {
		t.Errorf("expected evidence.export, got %s", received[0].EventType)
	}
}

func TestHTTPSinkAuthHeader(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink := NewHTTPSink(server.URL, "Bearer my-token")

	rec := SIEMRecord{Timestamp: time.Now().UTC(), EventType: "test"}
	_ = sink.Write(context.Background(), rec)

	if authHeader != "Bearer my-token" {
		t.Errorf("expected 'Bearer my-token', got '%s'", authHeader)
	}
}

func TestHTTPSinkRetryOnError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink := NewHTTPSink(server.URL, "", WithMaxRetries(3))

	rec := SIEMRecord{Timestamp: time.Now().UTC(), EventType: "test.retry"}
	err := sink.Write(context.Background(), rec)

	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestHTTPSinkRetryExhausted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sink := NewHTTPSink(server.URL, "", WithMaxRetries(2), WithRetryDelay(0))

	rec := SIEMRecord{Timestamp: time.Now().UTC(), EventType: "test.fail"}
	err := sink.Write(context.Background(), rec)

	if err == nil {
		t.Error("expected error after retries exhausted")
	}
}

// --- Checksum and Signature tests ---

func TestComputeChecksum(t *testing.T) {
	rec := SIEMRecord{
		Timestamp: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		EventType: "audit.access",
		Source:    "sdp-lab",
		Actor:     "user-1",
		Action:    "read",
		Outcome:   "allowed",
	}

	checksum := rec.ComputeChecksum()
	if checksum == "" {
		t.Error("expected non-empty checksum")
	}

	// Verify it's a valid SHA-256 hex string
	parts := splitChecksum(checksum)
	if len(parts) != 2 || parts[0] != "sha256" {
		t.Errorf("expected sha256:<hex>, got %s", checksum)
	}
	_, err := hex.DecodeString(parts[1])
	if err != nil {
		t.Errorf("invalid hex in checksum: %v", err)
	}
}

func TestChecksumDeterministic(t *testing.T) {
	rec := SIEMRecord{
		Timestamp: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		EventType: "audit.access",
		Source:    "sdp-lab",
		Actor:     "user-1",
	}

	c1 := rec.ComputeChecksum()
	c2 := rec.ComputeChecksum()
	if c1 != c2 {
		t.Errorf("checksums should be deterministic: %s != %s", c1, c2)
	}
}

func TestChecksumTamperDetection(t *testing.T) {
	rec := SIEMRecord{
		Timestamp: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		EventType: "audit.access",
		Source:    "sdp-lab",
		Actor:     "user-1",
	}

	c1 := rec.ComputeChecksum()
	rec.Actor = "user-2"
	c2 := rec.ComputeChecksum()

	if c1 == c2 {
		t.Error("checksum should change when record is modified")
	}
}

func TestVerifyChecksum(t *testing.T) {
	rec := SIEMRecord{
		Timestamp: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		EventType: "audit.access",
		Source:    "sdp-lab",
		Actor:     "user-1",
	}
	rec.Checksum = rec.ComputeChecksum()

	if !rec.VerifyChecksum() {
		t.Error("checksum verification should succeed for untampered record")
	}

	rec.Actor = "tampered"
	if rec.VerifyChecksum() {
		t.Error("checksum verification should fail for tampered record")
	}
}

// --- BatchSink (backfill) tests ---

func TestBatchSinkBackfill(t *testing.T) {
	var received []SIEMRecord
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var batch []SIEMRecord
		if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, batch...)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpSink := NewHTTPSink(server.URL, "")
	sink := NewBatchSink(httpSink, 2)

	records := []SIEMRecord{
		{Timestamp: time.Now().UTC(), EventType: "backfill.1", Source: "sdp-lab"},
		{Timestamp: time.Now().UTC(), EventType: "backfill.2", Source: "sdp-lab"},
		{Timestamp: time.Now().UTC(), EventType: "backfill.3", Source: "sdp-lab"},
	}

	err := sink.WriteBatch(records)
	if err != nil {
		t.Fatalf("batch write failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 3 {
		t.Errorf("expected 3 records, got %d", len(received))
	}
}

func TestBatchSinkEmpty(t *testing.T) {
	var buf bytes.Buffer
	sink := NewBatchSink(NewSyslogSink(&buf), 10)

	err := sink.WriteBatch([]SIEMRecord{})
	if err != nil {
		t.Fatalf("empty batch should succeed: %v", err)
	}
}

// --- ExportBundle tests ---

func TestExportBundleIntegrity(t *testing.T) {
	records := []SIEMRecord{
		{Timestamp: time.Now().UTC(), EventType: "event.1", Source: "sdp-lab", Actor: "a1"},
		{Timestamp: time.Now().UTC(), EventType: "event.2", Source: "sdp-lab", Actor: "a2"},
	}
	for i := range records {
		records[i].Checksum = records[i].ComputeChecksum()
	}

	bundle := NewExportBundle("tenant-1", "F074", records)

	if bundle.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", bundle.TenantID)
	}
	if bundle.RecordCount != 2 {
		t.Errorf("expected 2 records, got %d", bundle.RecordCount)
	}
	if bundle.BundleChecksum == "" {
		t.Error("expected bundle checksum")
	}
	if !bundle.Verify() {
		t.Error("bundle integrity verification failed")
	}
}

func TestExportBundleTamperDetection(t *testing.T) {
	records := []SIEMRecord{
		{Timestamp: time.Now().UTC(), EventType: "event.1", Source: "sdp-lab", Actor: "a1"},
	}
	records[0].Checksum = records[0].ComputeChecksum()

	bundle := NewExportBundle("tenant-1", "F074", records)

	// Tamper with a record
	bundle.Records[0].Actor = "tampered"

	if bundle.Verify() {
		t.Error("bundle should fail verification after tampering")
	}
}

func TestExportBundleSerialization(t *testing.T) {
	records := []SIEMRecord{
		{Timestamp: time.Now().UTC(), EventType: "event.1", Source: "sdp-lab"},
	}
	records[0].Checksum = records[0].ComputeChecksum()

	bundle := NewExportBundle("tenant-1", "F074", records)

	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored ExportBundle
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if restored.RecordCount != 1 {
		t.Errorf("expected 1 record, got %d", restored.RecordCount)
	}
}

// helper
func splitChecksum(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

// Verify SHA-256 helper exists
func TestSHA256Helper(t *testing.T) {
	data := []byte("test")
	expected := fmt.Sprintf("%x", sha256.Sum256(data))
	got := sha256Hex(data)
	if got != expected {
		t.Errorf("sha256Hex mismatch: expected %s, got %s", expected, got)
	}
}
