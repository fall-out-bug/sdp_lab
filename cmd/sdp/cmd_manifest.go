package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/manifest"
)

const (
	defaultManifestPath    = "sdp.manifest.yaml"
	defaultParityMatrixDoc = "docs/reference/harness-parity-matrix.md"
)

func runManifest(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp manifest <validate|schema|parity> [flags]")
		os.Exit(2)
	}
	switch args[0] {
	case "validate":
		os.Exit(runManifestValidate(args[1:]))
	case "schema":
		runManifestSchema()
	case "parity":
		os.Exit(runManifestParity(args[1:]))
	default:
		fmt.Fprintf(os.Stderr, "error: unknown manifest subcommand: %s\n", args[0])
		os.Exit(2)
	}
}

func runManifestValidate(args []string) int {
	manifestPath := defaultManifestPath
	repoRoot := "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--manifest":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --manifest requires a value")
				return 2
			}
			manifestPath = args[i+1]
			i++
		case "--repo-root":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --repo-root requires a value")
				return 2
			}
			repoRoot = args[i+1]
			i++
		case "-h", "--help":
			fmt.Println("usage: sdp manifest validate [--manifest <path>] [--repo-root <path>]")
			return 0
		default:
			fmt.Fprintf(os.Stderr, "error: unknown flag: %s\n", args[i])
			return 2
		}
	}

	if !filepath.IsAbs(manifestPath) {
		abs, err := filepath.Abs(manifestPath)
		if err == nil {
			manifestPath = abs
		}
	}
	if !filepath.IsAbs(repoRoot) {
		abs, err := filepath.Abs(repoRoot)
		if err == nil {
			repoRoot = abs
		}
	}

	res, err := manifest.Load(manifestPath, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	fmt.Printf("ok: %s\n", manifestPath)
	fmt.Printf("  version=%s sdp_version=%s\n", res.Manifest.Version, res.Manifest.SDPVersion)
	fmt.Printf("  skills=%d commands=%d agents=%d hooks=%d mcp_servers=%d\n",
		len(res.Manifest.Skills),
		len(res.Manifest.Commands),
		len(res.Manifest.Agents),
		len(res.Manifest.Hooks),
		len(res.Manifest.MCPServers),
	)
	for _, w := range res.Warnings {
		fmt.Printf("  warning: %s\n", w)
	}
	return 0
}

func runManifestSchema() {
	if _, err := os.Stdout.Write(manifest.SchemaJSON()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runManifestParity(args []string) int {
	manifestPath := defaultManifestPath
	output := defaultParityMatrixDoc
	mode := "stdout"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--manifest":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --manifest requires a value")
				return 2
			}
			manifestPath = args[i+1]
			i++
		case "--write":
			mode = "write"
		case "--check":
			mode = "check"
		case "--output":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --output requires a value")
				return 2
			}
			output = args[i+1]
			i++
		case "-h", "--help":
			fmt.Println("usage: sdp manifest parity [--manifest <path>] [--write|--check] [--output <path>]")
			return 0
		default:
			fmt.Fprintf(os.Stderr, "error: unknown flag: %s\n", args[i])
			return 2
		}
	}

	if !filepath.IsAbs(manifestPath) {
		if abs, err := filepath.Abs(manifestPath); err == nil {
			manifestPath = abs
		}
	}
	repoRoot := filepath.Dir(manifestPath)
	res, err := manifest.Load(manifestPath, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	rendered := res.Manifest.ParityMatrix(time.Now())

	switch mode {
	case "stdout":
		_, _ = os.Stdout.WriteString(rendered)
	case "write":
		if !filepath.IsAbs(output) {
			output = filepath.Join(repoRoot, output)
		}
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: mkdir: %v\n", err)
			return 1
		}
		if err := os.WriteFile(output, []byte(rendered), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error: write: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", output)
	case "check":
		if !filepath.IsAbs(output) {
			output = filepath.Join(repoRoot, output)
		}
		existing, err := os.ReadFile(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s missing or unreadable: %v\n", output, err)
			return 1
		}
		if normalizeForCheck(string(existing)) != normalizeForCheck(rendered) {
			fmt.Fprintf(os.Stderr, "error: %s is out of date — re-run `sdp manifest parity --write`\n", output)
			return 1
		}
		fmt.Fprintf(os.Stderr, "ok: %s up to date\n", output)
	}
	return 0
}

// normalizeForCheck strips the volatile `Generated: <date>` line so that
// `--check` is stable from one day to the next without forcing a doc update.
func normalizeForCheck(s string) string {
	out := []byte{}
	for _, line := range splitLinesKeep(s) {
		if len(line) >= len("Generated:") && line[:len("Generated:")] == "Generated:" {
			continue
		}
		out = append(out, line...)
	}
	return string(out)
}

func splitLinesKeep(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
