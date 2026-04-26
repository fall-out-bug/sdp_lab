package export

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BatchSink wraps another sink and sends records in batches for backfill mode.
type BatchSink struct {
	inner     SinkWriter
	batchSize int
	httpSink  *HTTPSink
}

// NewBatchSink creates a batch sink. For HTTP sinks, it sends batches as JSON arrays.
func NewBatchSink(inner SinkWriter, batchSize int) *BatchSink {
	if batchSize <= 0 {
		batchSize = 1
	}
	s := &BatchSink{
		inner:     inner,
		batchSize: batchSize,
	}
	if hs, ok := inner.(*HTTPSink); ok {
		s.httpSink = hs
	}
	return s
}

// WriteBatch sends records in batches of batchSize.
func (s *BatchSink) WriteBatch(records []SIEMRecord) error {
	if len(records) == 0 {
		return nil
	}
	if s.httpSink != nil {
		return s.writeHTTPBatches(records)
	}
	ctx := context.Background()
	for _, rec := range records {
		if err := s.inner.Write(ctx, rec); err != nil {
			return err
		}
	}
	return nil
}

func (s *BatchSink) writeHTTPBatches(records []SIEMRecord) error {
	for i := 0; i < len(records); i += s.batchSize {
		end := i + s.batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]
		if err := s.postBatchWithRetry(batch); err != nil {
			return err
		}
	}
	return nil
}

func (s *BatchSink) postBatchWithRetry(batch []SIEMRecord) error {
	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < s.httpSink.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(s.httpSink.retryDelay)
		}

		req, err := http.NewRequest(http.MethodPost, s.httpSink.endpoint, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if s.httpSink.authToken != "" {
			req.Header.Set("Authorization", s.httpSink.authToken)
		}

		resp, err := s.httpSink.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("post batch: %w", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("batch post returned http %d", resp.StatusCode)
	}
	return fmt.Errorf("batch exhausted %d retries: %w", s.httpSink.maxRetries, lastErr)
}

// ExportBundle is a compliance-ready evidence bundle with integrity verification.
type ExportBundle struct {
	BundleID       string       `json:"bundle_id"`
	TenantID       string       `json:"tenant_id"`
	FeatureID      string       `json:"feature_id"`
	CreatedAt      time.Time    `json:"created_at"`
	RecordCount    int          `json:"record_count"`
	Records        []SIEMRecord `json:"records"`
	BundleChecksum string       `json:"bundle_checksum"`
}

// NewExportBundle creates a bundle with computed checksums for all records and the bundle.
func NewExportBundle(tenantID, featureID string, records []SIEMRecord) *ExportBundle {
	now := time.Now().UTC()
	bundleID := fmt.Sprintf("bundle-%s-%s-%d", tenantID, featureID, now.Unix())

	// Compute checksums for records that don't already have one
	for i := range records {
		if records[i].Checksum == "" {
			records[i].Checksum = records[i].ComputeChecksum()
		}
	}

	b := &ExportBundle{
		BundleID:    bundleID,
		TenantID:    tenantID,
		FeatureID:   featureID,
		CreatedAt:   now,
		RecordCount: len(records),
		Records:     records,
	}
	b.BundleChecksum = b.computeBundleChecksum()
	return b
}

// Verify checks integrity of the bundle and all its records.
func (b *ExportBundle) Verify() bool {
	if b.BundleChecksum != b.computeBundleChecksum() {
		return false
	}
	for _, rec := range b.Records {
		if !rec.VerifyChecksum() {
			return false
		}
	}
	return true
}

func (b *ExportBundle) computeBundleChecksum() string {
	h := sha256.New()
	// Include all structural fields for tamper detection
	fmt.Fprintf(h, "%s\n%s\n%s\n%s\n%d\n", b.BundleID, b.TenantID, b.FeatureID, b.CreatedAt.Format(time.RFC3339Nano), b.RecordCount)
	for _, rec := range b.Records {
		h.Write([]byte(rec.Checksum))
		h.Write([]byte{0})
	}
	return "sha256:" + sha256Hex(h.Sum(nil))
}
