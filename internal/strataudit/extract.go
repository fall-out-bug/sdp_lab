package strataudit

import (
	"bytes"
	"fmt"

	"github.com/ledongthuc/pdf"
)

func extractPDFWithLedongthuc(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pdf reader: %w", err)
	}

	n := reader.NumPage()

	var buf bytes.Buffer
	for i := 1; i <= n; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // skip pages that fail
		}
		buf.WriteString(text)
		buf.WriteByte('\n')
	}

	result := buf.String()
	if len(result) < 10 {
		return "", fmt.Errorf("pdf: no readable text extracted")
	}
	return result, nil
}

func extractDOCXBasic(data []byte) (string, error) {
	// DOCX is a ZIP containing word/document.xml
	// Minimal extraction: unzip and strip XML tags
	// For v1, we use a simple approach without external DOCX library
	return extractDOCXFromZIP(data)
}

func extractPPTXBasic(data []byte) (string, error) {
	return extractPPTXFromZIP(data)
}
