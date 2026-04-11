package strataudit

import (
	"context"
	"testing"
)

func TestExtractorRegistry_Routing(t *testing.T) {
	cfg := &Config{} // No external command — built-in only
	registry := NewExtractorRegistry(cfg)

	tests := []struct {
		ext      string
		wantOK   bool
		wantName string
	}{
		{".txt", true, "text"},
		{".md", true, "text"},
		{".markdown", true, "text"},
		{".pdf", true, "pdf"},
		{".docx", true, "docx"},
		{".pptx", false, ""},  // no bridge configured
		{".doc", false, ""},   // no bridge configured
		{".exe", false, ""},
		{"", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := registry.CanHandle(tt.ext)
			if got != tt.wantOK {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.ext, got, tt.wantOK)
			}
		})
	}
}

func TestTextExtractor(t *testing.T) {
	ext := &TextExtractor{}
	if !ext.CanHandle(".txt") {
		t.Error("should handle .txt")
	}
	if ext.CanHandle(".pdf") {
		t.Error("should not handle .pdf")
	}
	text, err := ext.Extract(context.Background(), "test.txt", []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("got %q, want 'hello'", text)
	}
}

func TestExtOnly(t *testing.T) {
	tests := []struct {
		path, want string
	}{
		{"file.txt", ".txt"},
		{"/path/to/file.pdf", ".pdf"},
		{"noext", ""},
		{"file.tar.gz", ".gz"},
	}
	for _, tt := range tests {
		got := extOnly(tt.path)
		if got != tt.want {
			t.Errorf("extOnly(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
