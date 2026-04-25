package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"sdp_dev/internal/adapters"
	"sdp_dev/internal/manifest"
)

const (
	defaultGeneratedOutDir = ".sdp/generated"
)

func runGenerateAdapters(args []string) int {
	manifestPath := defaultManifestPath
	outDir := defaultGeneratedOutDir
	modeCheck := false
	modeDiff := false
	modeWrite := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--manifest":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --manifest requires a value")
				return 2
			}
			manifestPath = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --out requires a value")
				return 2
			}
			outDir = args[i+1]
			i++
		case "--check":
			modeCheck = true
		case "--diff":
			modeDiff = true
		case "--write":
			modeWrite = true
		case "-h", "--help":
			fmt.Println("usage: sdp generate-adapters [--manifest <path>] [--out <dir>] [--check|--write|--diff]")
			fmt.Println()
			fmt.Println("  --manifest <path>  Path to sdp.manifest.yaml (default: sdp.manifest.yaml)")
			fmt.Println("  --out <dir>        Output directory (default: .sdp/generated)")
			fmt.Println("  --write            Write generated files to --out (default mode)")
			fmt.Println("  --check            Fail if on-disk files differ from generated (CI gate)")
			fmt.Println("  --diff             Print diff of generated vs on-disk without writing")
			return 0
		default:
			fmt.Fprintf(os.Stderr, "error: unknown flag: %s\n", args[i])
			return 2
		}
	}

	// Default mode is --write
	if !modeCheck && !modeDiff && !modeWrite {
		modeWrite = true
	}

	if !filepath.IsAbs(manifestPath) {
		abs, err := filepath.Abs(manifestPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: resolve manifest path: %v\n", err)
			return 1
		}
		manifestPath = abs
	}

	repoRoot := filepath.Dir(manifestPath)

	res, err := manifest.Load(manifestPath, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load manifest: %v\n", err)
		return 1
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	generated, err := adapters.Generate(res.Manifest, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: generate adapters: %v\n", err)
		return 1
	}

	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(repoRoot, outDir)
	}

	switch {
	case modeWrite:
		return doWrite(generated, outDir)
	case modeCheck:
		return doCheck(generated, outDir)
	case modeDiff:
		return doDiff(generated, outDir)
	}
	return 0
}

func doWrite(generated map[string][]byte, outDir string) int {
	// Sort for deterministic output
	paths := sortedKeys(generated)
	written := 0
	for _, rel := range paths {
		dest := filepath.Join(outDir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: mkdir %s: %v\n", filepath.Dir(dest), err)
			return 1
		}
		if err := os.WriteFile(dest, generated[rel], 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error: write %s: %v\n", dest, err)
			return 1
		}
		written++
	}
	fmt.Fprintf(os.Stderr, "wrote %d adapter files to %s\n", written, outDir)
	return 0
}

func doCheck(generated map[string][]byte, outDir string) int {
	diffs := collectDiffs(generated, outDir)
	if len(diffs) > 0 {
		for _, d := range diffs {
			fmt.Fprintf(os.Stderr, "drift: %s\n", d)
		}
		fmt.Fprintf(os.Stderr, "error: %d adapter file(s) out of date — re-run `sdp generate-adapters --write`\n", len(diffs))
		return 1
	}
	fmt.Fprintf(os.Stderr, "ok: %d adapter files up to date in %s\n", len(generated), outDir)
	return 0
}

func doDiff(generated map[string][]byte, outDir string) int {
	diffs := collectDiffs(generated, outDir)
	if len(diffs) > 0 {
		for _, d := range diffs {
			fmt.Println(d)
		}
		return 1
	}
	fmt.Fprintln(os.Stderr, "ok: no diff")
	return 0
}

// collectDiffs returns a list of paths that differ or are missing on disk.
func collectDiffs(generated map[string][]byte, outDir string) []string {
	var diffs []string
	for _, rel := range sortedKeys(generated) {
		dest := filepath.Join(outDir, rel)
		existing, err := os.ReadFile(dest)
		if err != nil {
			diffs = append(diffs, rel+": missing")
			continue
		}
		if string(existing) != string(generated[rel]) {
			diffs = append(diffs, rel+": content differs")
		}
	}
	return diffs
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
