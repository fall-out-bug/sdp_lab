package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/adapters"
	"github.com/fall-out-bug/sdp_lab/internal/manifest"
)

//go:embed templates/sdp.manifest.template.yaml
var embeddedManifestTemplate []byte

// knownHarnesses is the canonical ordered list of supported harnesses.
var knownHarnesses = []string{"claude-code", "opencode", "codex", "cursor", "pi"}

// harnessDirs maps each harness name to the directory it installs into.
var harnessDirs = map[string]string{
	"claude-code": ".claude",
	"opencode":    ".opencode",
	"codex":       ".codex",
	"cursor":      ".cursor",
	"pi":          ".pi",
}

// sdpLock is the structure written to sdp.lock.
type sdpLock struct {
	SDPVersion      string `json:"sdp_version"`
	ManifestVersion string `json:"manifest_version"`
	GeneratedAt     string `json:"generated_at"`
}

// parseInitFlag parses a flag arg that may be in "--flag value" or "--flag=value"
// form. It returns the flag name (with leading "--" stripped), value, and whether
// the value was embedded (=form). For non-value flags (--update, -h) it returns
// just the trimmed name.
func parseInitFlag(arg string) (name, value string, hasEq bool) {
	if strings.HasPrefix(arg, "--") {
		arg = arg[2:]
	} else if strings.HasPrefix(arg, "-") {
		return arg[1:], "", false
	}
	if idx := strings.IndexByte(arg, '='); idx != -1 {
		return arg[:idx], arg[idx+1:], true
	}
	return arg, "", false
}

// runInit implements `sdp init`.
func runInit(args []string) int {
	harnessFlag := "all"
	targetFlag := "."
	manifestFlag := ""
	update := false

	for i := 0; i < len(args); i++ {
		name, val, hasEq := parseInitFlag(args[i])
		switch name {
		case "harness":
			if !hasEq {
				if i+1 >= len(args) {
					fmt.Fprintln(os.Stderr, "error: --harness requires a value")
					return 2
				}
				val = args[i+1]
				i++
			}
			harnessFlag = val
		case "target":
			if !hasEq {
				if i+1 >= len(args) {
					fmt.Fprintln(os.Stderr, "error: --target requires a value")
					return 2
				}
				val = args[i+1]
				i++
			}
			targetFlag = val
		case "manifest":
			if !hasEq {
				if i+1 >= len(args) {
					fmt.Fprintln(os.Stderr, "error: --manifest requires a value")
					return 2
				}
				val = args[i+1]
				i++
			}
			manifestFlag = val
		case "update":
			update = true
		case "h", "help":
			printInitHelp()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "error: unknown flag: %s\n", args[i])
			return 2
		}
	}

	// Resolve target to absolute path.
	target, err := filepath.Abs(targetFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: resolve target %q: %v\n", targetFlag, err)
		return 1
	}

	// Ensure target directory exists.
	if err := os.MkdirAll(target, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: create target %q: %v\n", target, err)
		return 1
	}

	// Determine which harnesses to install.
	harnesses, err := resolveHarnesses(harnessFlag, target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if len(harnesses) == 0 {
		fmt.Fprintln(os.Stderr, "warning: no harnesses selected, nothing to install")
		return 0
	}

	// Resolve manifest: load from --manifest flag, or write embedded template.
	manifestPath := filepath.Join(target, "sdp.manifest.yaml")
	if manifestFlag != "" {
		if !filepath.IsAbs(manifestFlag) {
			manifestFlag, _ = filepath.Abs(manifestFlag)
		}
		manifestPath = manifestFlag
	} else {
		// Write embedded template if sdp.manifest.yaml does not yet exist.
		if _, serr := os.Stat(manifestPath); os.IsNotExist(serr) {
			if err := os.WriteFile(manifestPath, embeddedManifestTemplate, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error: write manifest template: %v\n", err)
				return 1
			}
			fmt.Fprintf(os.Stderr, "created %s\n", manifestPath)
		} else if update {
			fmt.Fprintf(os.Stderr, "warning: sdp.manifest.yaml already exists — skipping overwrite (--update mode)\n")
		}
	}

	// Load the manifest against the target repo. The installer copies the
	// canonical prompt sources before invoking init, so generated adapters can
	// embed real bodies instead of placeholder shells.
	m, warnings, sdpVersion, manifestVersion, loadErr := loadManifestForInit(manifestPath, target)
	if loadErr != nil {
		fmt.Fprintf(os.Stderr, "error: manifest load failed: %v\n", loadErr)
		return 1
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	// Generate adapter files for the manifest as written. The --harness flag
	// controls which live harness directories get installed below; it must not
	// rewrite a user's sdp.manifest.yaml or narrow future generation scope.
	// `.sdp/generated` remains a full manifest-derived cache so doctor checks are
	// stable without dropping comments or hand-edited formatting.
	generated, err := adapters.Generate(m, target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: generate adapters: %v\n", err)
		return 1
	}

	// Write generated files into .sdp/generated so `sdp doctor adapters` can
	// compare the same canonical output later.
	generatedRoot := filepath.Join(target, defaultGeneratedOutDir)
	for rel, content := range generated {
		dest := filepath.Join(generatedRoot, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: mkdir %s: %v\n", filepath.Dir(dest), err)
			return 1
		}
		if err := os.WriteFile(dest, content, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error: write %s: %v\n", dest, err)
			return 1
		}
	}

	// Write adapter files into target.
	written := 0
	for _, h := range harnesses {
		harnessDir := harnessDirs[h]
		if harnessDir == "" {
			continue
		}
		for rel, content := range generated {
			// Only write files that belong to this harness directory.
			if !strings.HasPrefix(rel, harnessDir+"/") {
				continue
			}
			dest := filepath.Join(target, rel)
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "error: mkdir %s: %v\n", filepath.Dir(dest), err)
				return 1
			}
			if err := os.WriteFile(dest, content, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error: write %s: %v\n", dest, err)
				return 1
			}
			written++
		}
		// Ensure the harness root directory exists even if no files were generated.
		harnessRoot := filepath.Join(target, harnessDir)
		if err := os.MkdirAll(harnessRoot, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: mkdir %s: %v\n", harnessRoot, err)
			return 1
		}
	}

	// Write sdp.lock.
	lockPath := filepath.Join(target, "sdp.lock")
	lock := sdpLock{
		SDPVersion:      sdpVersion,
		ManifestVersion: manifestVersion,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	lockBytes, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: marshal sdp.lock: %v\n", err)
		return 1
	}
	if err := os.WriteFile(lockPath, append(lockBytes, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write sdp.lock: %v\n", err)
		return 1
	}

	fmt.Printf("installed for %s, %d adapter files written, target: %s\n",
		strings.Join(harnesses, ","), written, target)
	return 0
}

// resolveHarnesses resolves the --harness flag value to a list of harness names.
func resolveHarnesses(flag, target string) ([]string, error) {
	switch flag {
	case "all":
		return knownHarnesses, nil
	case "auto":
		return detectHarnesses(target), nil
	default:
		// Comma-separated list.
		var out []string
		for _, h := range strings.Split(flag, ",") {
			h = strings.TrimSpace(h)
			if h == "" {
				continue
			}
			if !isKnownHarness(h) {
				return nil, fmt.Errorf("unknown harness %q (valid: %s)", h, strings.Join(knownHarnesses, ", "))
			}
			out = append(out, h)
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("--harness value %q resolved to no harnesses", flag)
		}
		return out, nil
	}
}

// detectHarnesses checks for existing harness directories in target.
// If none are found, returns all known harnesses.
func detectHarnesses(target string) []string {
	var found []string
	for _, h := range knownHarnesses {
		d := filepath.Join(target, harnessDirs[h])
		if _, err := os.Stat(d); err == nil {
			found = append(found, h)
		}
	}
	if len(found) == 0 {
		return knownHarnesses
	}
	return found
}

func isKnownHarness(h string) bool {
	for _, k := range knownHarnesses {
		if k == h {
			return true
		}
	}
	return false
}

// loadManifestForInit loads the manifest from path and validates referenced
// paths against repoRoot when protocol sources are present.
// Returns manifest, warnings, sdpVersion, manifestVersion, error.
func loadManifestForInit(manifestPath, repoRoot string) (*manifest.Manifest, []string, string, string, error) {
	res, err := manifest.Load(manifestPath, repoRoot)
	if err != nil {
		return nil, nil, "1.0.0", "1.0.0", err
	}
	sdpVersion := res.Manifest.SDPVersion
	if sdpVersion == "" {
		sdpVersion = "1.0.0"
	}
	manifestVersion := res.Manifest.Version
	if manifestVersion == "" {
		manifestVersion = "1.0.0"
	}
	return res.Manifest, res.Warnings, sdpVersion, manifestVersion, nil
}

func printInitHelp() {
	fmt.Println("usage: sdp init [flags]")
	fmt.Println()
	fmt.Println("One-shot SDP installer for downstream repositories.")
	fmt.Println("Writes harness adapter files and sdp.lock to the target directory.")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --harness <list>  Harnesses to install: comma-separated (claude-code,opencode,codex,cursor,pi),")
	fmt.Println("                    'all' (default), or 'auto' (detect by existing dirs)")
	fmt.Println("  --target <dir>    Target directory (default: current working directory)")
	fmt.Println("  --manifest <path> Path to manifest template (default: embedded minimal template)")
	fmt.Println("  --update          Re-run without overwriting existing user-modified manifest")
	fmt.Println("  -h, --help        Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  sdp init                          # Install all harnesses in current directory")
	fmt.Println("  sdp init --harness=claude-code    # Install only Claude Code")
	fmt.Println("  sdp init --harness=auto --target=/path/to/myrepo")
	fmt.Println("  sdp init --update                 # Re-run, keep existing manifest")
}
