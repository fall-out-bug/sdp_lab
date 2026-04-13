package strataudit

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Extractor extracts text from a file format.
type Extractor interface {
	CanHandle(ext string) bool
	Extract(ctx context.Context, path string, data []byte) (string, error)
	Name() string
}

// ExtractorRegistry routes file extensions to extractors.
type ExtractorRegistry struct {
	extractors []Extractor
}

// NewExtractorRegistry creates a registry with built-in extractors and optional bridge.
func NewExtractorRegistry(cfg *Config) *ExtractorRegistry {
	r := &ExtractorRegistry{}
	r.Register(&TextExtractor{})
	r.Register(&PDFExtractor{})
	r.Register(&PPTXExtractor{})
	r.Register(&DOCXExtractor{})
	if cfg.Extractors.ExternalCommand != "" {
		be, err := NewBridgeExtractor(cfg.Extractors)
		if err != nil {
			slog.Warn("bridge extractor disabled", "err", err)
		} else {
			r.Register(be)
			slog.Info("bridge extractor enabled", "command", cfg.Extractors.ExternalCommand,
				"extensions", cfg.Extractors.Extensions)
		}
	}
	return r
}

func (r *ExtractorRegistry) Register(e Extractor) {
	r.extractors = append(r.extractors, e)
}

func (r *ExtractorRegistry) CanHandle(ext string) bool {
	ext = strings.ToLower(ext)
	for _, e := range r.extractors {
		if e.CanHandle(ext) {
			return true
		}
	}
	return false
}

func (r *ExtractorRegistry) Extract(ctx context.Context, path string, data []byte) (string, error) {
	ext := strings.ToLower(extOnly(path))
	for _, e := range r.extractors {
		if e.CanHandle(ext) {
			text, err := e.Extract(ctx, path, data)
			if err != nil {
				return "", fmt.Errorf("%s extractor: %w", e.Name(), err)
			}
			return text, nil
		}
	}
	return "", fmt.Errorf("no extractor for %q", ext)
}

func extOnly(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx:]
	}
	return ""
}

// TextExtractor handles .txt, .md, .markdown
type TextExtractor struct{}

func (t *TextExtractor) Name() string { return "text" }
func (t *TextExtractor) CanHandle(ext string) bool {
	return ext == ".txt" || ext == ".md" || ext == ".markdown"
}
func (t *TextExtractor) Extract(_ context.Context, _ string, data []byte) (string, error) {
	return string(data), nil
}

// PDFExtractor handles .pdf
type PDFExtractor struct{}

func (p *PDFExtractor) Name() string              { return "pdf" }
func (p *PDFExtractor) CanHandle(ext string) bool { return ext == ".pdf" }
func (p *PDFExtractor) Extract(_ context.Context, _ string, data []byte) (string, error) {
	return extractPDF(data)
}

// PPTXExtractor handles .pptx
type PPTXExtractor struct{}

func (p *PPTXExtractor) Name() string              { return "pptx" }
func (p *PPTXExtractor) CanHandle(ext string) bool { return ext == ".pptx" }
func (p *PPTXExtractor) Extract(_ context.Context, _ string, data []byte) (string, error) {
	return extractPPTXBasic(data)
}

// DOCXExtractor handles .docx
type DOCXExtractor struct{}

func (d *DOCXExtractor) Name() string              { return "docx" }
func (d *DOCXExtractor) CanHandle(ext string) bool { return ext == ".docx" }
func (d *DOCXExtractor) Extract(_ context.Context, _ string, data []byte) (string, error) {
	return extractDOCX(data)
}
