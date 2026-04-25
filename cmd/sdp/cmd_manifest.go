package main

import (
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/manifest"
)

const defaultManifestPath = "sdp.manifest.yaml"

func runManifest(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp manifest <validate|schema> [flags]")
		os.Exit(2)
	}
	switch args[0] {
	case "validate":
		os.Exit(runManifestValidate(args[1:]))
	case "schema":
		runManifestSchema()
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
