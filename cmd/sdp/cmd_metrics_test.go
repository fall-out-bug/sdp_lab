package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"github.com/fall-out-bug/sdp_lab/internal/metrics"
)

func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestRenderTextEmptyPeriod(t *testing.T) {
	report := metrics.MetricsReport{
		Version:  "1.0.0",
		RepoPath: "/test",
	}
	output := captureOutput(func() {
		renderText(&report)
	})
	if strings.Contains(output, "0001") {
		t.Errorf("renderText should not render zero-value Period dates, got:\n%s", output)
	}
}

func TestRenderMarkdownEmptyPeriod(t *testing.T) {
	report := metrics.MetricsReport{
		Version:  "1.0.0",
		RepoPath: "/test",
	}
	output := captureOutput(func() {
		renderMarkdown(&report)
	})
	if strings.Contains(output, "0001") {
		t.Errorf("renderMarkdown should not render zero-value Period dates, got:\n%s", output)
	}
}
