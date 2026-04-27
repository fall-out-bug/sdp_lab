package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fall-out-bug/sdp_lab/internal/adapters"
	"github.com/fall-out-bug/sdp_lab/internal/cli"
	"github.com/fall-out-bug/sdp_lab/internal/manifest"
)

func runDoctor(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp doctor <control|adapters|backlog|all>")
		os.Exit(2)
	}
	switch args[0] {
	case "control":
		runDoctorControl()
	case "adapters":
		os.Exit(runDoctorAdapters(args[1:]))
	case "backlog":
		runDoctorBacklog(args[1:])
	case "all":
		runDoctorControl()
		runDoctorBacklog(nil)
		os.Exit(runDoctorAdapters(args[1:]))
	default:
		fmt.Fprintf(os.Stderr, "error: unknown doctor subcommand: %s\n", args[0])
		os.Exit(2)
	}
}

func runDoctorControl() {
	store := openStore()
	report, err := store.DoctorControl()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: doctor control: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(cli.RenderDoctorControl(report))
	if len(report.Checks) > 0 {
		os.Exit(1)
	}
}

// runDoctorAdapters checks generated adapters in .sdp/generated/ against the
// manifest and also scans live harness trees for orphan files.
// Exit 0 = clean, 1 = drift or (strict && orphans), 2 = usage error.
func runDoctorAdapters(args []string) int {
	manifestPath := defaultManifestPath
	outDir := defaultGeneratedOutDir
	strict := false

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
		case "--strict":
			strict = true
		case "-h", "--help":
			fmt.Println("usage: sdp doctor adapters [--manifest <path>] [--out <dir>] [--strict]")
			fmt.Println()
			fmt.Println("  --manifest <path>  Path to sdp.manifest.yaml (default: sdp.manifest.yaml)")
			fmt.Println("  --out <dir>        Generated output dir to check (default: .sdp/generated)")
			fmt.Println("  --strict           Treat orphan files as errors (exit 1)")
			return 0
		default:
			fmt.Fprintf(os.Stderr, "error: unknown flag: %s\n", args[i])
			return 2
		}
	}

	// Resolve manifest path.
	if !filepath.IsAbs(manifestPath) {
		abs, err := filepath.Abs(manifestPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: resolve manifest path: %v\n", err)
			return 1
		}
		manifestPath = abs
	}
	repoRoot := filepath.Dir(manifestPath)

	// Resolve outDir relative to repoRoot.
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(repoRoot, outDir)
	}

	// Load manifest.
	res, err := manifest.Load(manifestPath, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load manifest: %v\n", err)
		return 1
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	// Generate adapter map (in-memory only; no writes).
	generated, err := adapters.Generate(res.Manifest, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: generate adapters: %v\n", err)
		return 1
	}

	// Run drift + orphan check.
	result, err := adapters.CheckDrift(generated, outDir, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: check drift: %v\n", err)
		return 1
	}

	// Print report.
	fmt.Print(adapters.FormatDriftReport(result, outDir, strict))

	if len(result.Drifts) > 0 {
		return 1
	}
	if strict && len(result.Orphans) > 0 {
		return 1
	}
	return 0
}
