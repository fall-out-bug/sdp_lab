package strataudit

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BridgeExtractor calls external commands (textutil/libreoffice) to convert documents.
type BridgeExtractor struct {
	command    string
	args       []string
	extensions map[string]bool
	timeout    time.Duration
	stdout     bool // true = read stdout, false = read output file
}

// ExtractorsConfig configures external document converters.
type ExtractorsConfig struct {
	ExternalCommand string   `yaml:"external_command"`
	Extensions      []string `yaml:"extensions"`
}

// NewBridgeExtractor creates a bridge extractor from config.
func NewBridgeExtractor(cfg ExtractorsConfig) (*BridgeExtractor, error) {
	cmd := cfg.ExternalCommand
	if cmd == "" {
		cmd = "textutil"
	}

	// Check command exists
	if _, err := exec.LookPath(cmd); err != nil {
		return nil, fmt.Errorf("external command %q not found: %w", cmd, err)
	}

	exts := cfg.Extensions
	if len(exts) == 0 {
		exts = []string{".pptx", ".doc", ".rtf", ".xls", ".xlsx", ".odt", ".odp"}
	}

	extMap := make(map[string]bool, len(exts))
	for _, e := range exts {
		e = strings.ToLower(e)
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extMap[e] = true
	}

	be := &BridgeExtractor{
		command:    cmd,
		extensions: extMap,
		timeout:    180 * time.Second,
	}

	// Determine command mode
	switch cmd {
	case "textutil":
		be.stdout = true
		be.args = []string{"-convert", "txt", "-stdout"}
	case "libreoffice":
		be.stdout = false
		be.args = []string{"--headless", "--convert-to", "txt"}
	default:
		be.stdout = true
	}

	return be, nil
}

func (b *BridgeExtractor) Name() string { return "bridge:" + b.command }

func (b *BridgeExtractor) CanHandle(ext string) bool {
	return b.extensions[ext]
}

func (b *BridgeExtractor) Extract(ctx context.Context, path string, _ []byte) (string, error) {
	// Sanitize path: only use basename
	base := filepath.Base(path)
	if base == "." || base == "/" || strings.Contains(base, "..") {
		return "", fmt.Errorf("bridge: invalid path %q", path)
	}

	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	if b.stdout {
		return b.extractStdout(ctx, path)
	}
	return b.extractFile(ctx, path)
}

func (b *BridgeExtractor) extractStdout(ctx context.Context, path string) (string, error) {
	args := append(b.args, path)
	cmd := exec.CommandContext(ctx, b.command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("bridge %s: %w (%s)", b.command, err, strings.TrimSpace(stderr.String()))
	}

	text := stdout.String()
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("bridge %s: empty output for %s", b.command, filepath.Base(path))
	}
	return text, nil
}

func (b *BridgeExtractor) extractFile(ctx context.Context, path string) (string, error) {
	// libreoffice --headless --convert-to txt writes to current directory
	tmpDir, err := os.MkdirTemp("", "strataudit-bridge-*")
	if err != nil {
		return "", fmt.Errorf("bridge tmpdir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	args := append(b.args, path)
	cmd := exec.CommandContext(ctx, b.command, args...)
	cmd.Dir = tmpDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("bridge %s: %w (%s)", b.command, err, strings.TrimSpace(stderr.String()))
	}

	// Find the output .txt file
	base := filepath.Base(path)
	txtName := strings.TrimSuffix(base, filepath.Ext(base)) + ".txt"
	outPath := filepath.Join(tmpDir, txtName)

	data, err := os.ReadFile(outPath)
	if err != nil {
		return "", fmt.Errorf("bridge read output: %w", err)
	}

	text := string(data)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("bridge %s: empty output for %s", b.command, base)
	}
	return text, nil
}
