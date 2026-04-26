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
	inner      SinkWriter
	batchSize  int
	httpClient *http.Client
	httpURL    string
}

// NewBatchSink creates a batch sink. For HTTP sinks, it sends batches as JSON arrays.
func NewBatchSink(inner SinkWriter, batchSize int) *BatchSink {
	s := &BatchSink{
		inner:     inner,
		batchSize: batchSize,
	}
	if hs, ok := inner.(*HTTPSink); ok {
		s.httpClient = hs.client
		s.httpURL = hs.endpoint
	}
	return s
}

// WriteBatch sends records in batches of batchSize.
func (s *BatchSink) WriteBatch(records []SIEMRecord) error {
	if len(records) == 0 {
		return nil
	}
	if s.httpClient != nil && s.httpURL != "" {
		return s.writeHTTPBatches(records)
	}
	for _, rec := range records {
		if err := s.inner.Write(context.Background(), rec); err != nil {
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
		data, err := json.Marshal(batch)
		if err != nil {
			return fmt.Errorf("marshal batch: %w", err)
		}
		resp, err := s.httpClient.Post(s.httpURL, "application/json", bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("post batch: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("batch post returned http %d", resp.StatusCode)
		}
	}
	return nil
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
	for _, rec := range b.Records {
		h.Write([]byte(rec.Checksum))
		h.Write([]byte{0})
	}
	data := append(h.Sum(nil), []byte(b.BundleID)...)
	return "sha256:" + sha256Hex(data)
}
