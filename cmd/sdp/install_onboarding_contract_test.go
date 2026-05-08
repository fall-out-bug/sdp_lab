package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerVerifiesRepoLocalCLI(t *testing.T) {
	script := readRepoFile(t, "scripts/install.sh")
	for i, want := range []string{
		`"$LOCAL_SDP" manifest validate --help`,
		`"$LOCAL_SDP" scout --help`,
		`"$LOCAL_SDP" manifest validate --manifest "$TARGET_ABS/sdp.manifest.yaml" --repo-root "$TARGET_ABS"`,
		`"$LOCAL_SDP" doctor adapters --manifest "$TARGET_ABS/sdp.manifest.yaml" --out "$TARGET_ABS/.sdp/generated"`,
		"repo-local CLI verified",
		"current shell resolves 'sdp' to:",
	} {
		t.Run(fmt.Sprintf("installer-check-%02d", i+1), func(t *testing.T) {
			if !strings.Contains(script, want) {
				t.Fatalf("installer should contain %q", want)
			}
		})
	}
}

func TestOnboardingDocsPreferRepoLocalCLIUntilPathIsTrusted(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs/QUICKSTART.md",
		"docs/runbooks/onboarding-downstream-repo.md",
	} {
		doc := readRepoFile(t, path)
		for i, want := range []string{
			"./.sdp/bin/sdp manifest validate",
			"./.sdp/bin/sdp doctor adapters",
			"command -v sdp",
		} {
			t.Run(fmt.Sprintf("%s/local-cli-%02d", path, i+1), func(t *testing.T) {
				if !strings.Contains(doc, want) {
					t.Fatalf("%s should contain %q", path, want)
				}
			})
		}
	}

	for _, path := range []string{
		"docs/QUICKSTART.md",
		"docs/runbooks/onboarding-downstream-repo.md",
		"docs/reference/product-surface.md",
	} {
		doc := readRepoFile(t, path)
		for i, want := range []string{
			"./.sdp/bin/sdp scout --format text .",
			"./.sdp/bin/sdp metrics --format markdown .",
			"./.sdp/bin/sdp index build --format text .",
			"./.sdp/bin/sdp spec --format text .",
		} {
			t.Run(fmt.Sprintf("%s/toolbox-%02d", path, i+1), func(t *testing.T) {
				if !strings.Contains(doc, want) {
					t.Fatalf("%s should contain %q", path, want)
				}
			})
		}
	}
}

func TestOnboardingDocsUseSDPLabInstallerSurface(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs/QUICKSTART.md",
		"docs/runbooks/onboarding-downstream-repo.md",
		"scripts/install.sh",
	} {
		t.Run(path, func(t *testing.T) {
			doc := readRepoFile(t, path)
			if strings.Contains(doc, "raw.githubusercontent.com/fall-out-bug/sdp/main/scripts/install.sh") {
				t.Fatalf("%s should not make sdp_lab onboarding depend on the public sdp mirror", path)
			}
			if !strings.Contains(doc, "raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh") {
				t.Fatalf("%s should reference the sdp_lab installer URL", path)
			}
		})
	}
}

func TestOnboardingDocsDeclareAllHarnesses(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs/QUICKSTART.md",
		"docs/runbooks/onboarding-downstream-repo.md",
		"docs/reference/product-surface.md",
		"docs/reference/project-map.md",
	} {
		t.Run(path, func(t *testing.T) {
			doc := readRepoFile(t, path)
			staleHarnessLists := []string{
				"Claude Code, OpenCode, Codex, and Cursor",
				"Claude Code, OpenCode, Codex, or Cursor",
			}
			for _, stale := range staleHarnessLists {
				if strings.Contains(doc, stale) {
					t.Fatalf("%s should include Pi when naming the supported harness set", path)
				}
			}
			if !strings.Contains(doc, "Pi") && !strings.Contains(doc, ".pi/") {
				t.Fatalf("%s should mention the Pi harness surface", path)
			}
		})
	}
}

func TestUserFacingDocsDoNotPointInstallToPublicSDPRelease(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs/QUICKSTART.md",
		"docs/runbooks/onboarding-downstream-repo.md",
	} {
		t.Run(path, func(t *testing.T) {
			doc := readRepoFile(t, path)
			if strings.Contains(doc, "github.com/fall-out-bug/sdp/releases/download") {
				t.Fatalf("%s should not make sdp_lab install or release examples depend on public sdp releases", path)
			}
		})
	}
}

func TestCustomizeDocsSeparateGeneratedCacheFromLiveInstall(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs/QUICKSTART.md",
		"docs/runbooks/onboarding-downstream-repo.md",
	} {
		t.Run(path, func(t *testing.T) {
			doc := readRepoFile(t, path)
			if !strings.Contains(doc, "./.sdp/bin/sdp generate-adapters --write") {
				t.Fatalf("%s should document generated-cache refresh", path)
			}
			if !strings.Contains(doc, "./.sdp/bin/sdp init --update") {
				t.Fatalf("%s should document live harness adapter refresh via init --update", path)
			}
		})
	}
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	data, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}
