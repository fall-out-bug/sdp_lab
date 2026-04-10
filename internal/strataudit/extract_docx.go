package strataudit

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

func extractDOCXFromZIP(data []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("docx zip: %w", err)
	}

	var buf bytes.Buffer
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("docx open document.xml: %w", err)
			}
			content, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return "", fmt.Errorf("docx read: %w", err)
			}
			buf.Write(content)
		}
	}

	if buf.Len() == 0 {
		return "", fmt.Errorf("docx: no document.xml found")
	}

	// Strip XML tags — simple regex approach
	text := stripXMLTags(buf.String())
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("docx: no text content")
	}
	return text, nil
}

var xmlTagRe = regexp.MustCompile(`<[^>]+>`)

func stripXMLTags(xmlContent string) string {
	text := xmlTagRe.ReplaceAllString(xmlContent, " ")
	// Collapse whitespace
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
