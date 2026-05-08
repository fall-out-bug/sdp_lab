package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerVerifiesRepoLocalCLI(t *testing.T) {
	script := readRepoFile(t, "scripts/install.sh")
	for _, want := range []string{
		`"$LOCAL_SDP" manifest validate --help`,
		`"$LOCAL_SDP" scout --help`,
		`"$LOCAL_SDP" manifest validate --manifest "$TARGET_ABS/sdp.manifest.yaml" --repo-root "$TARGET_ABS"`,
		`"$LOCAL_SDP" doctor adapters --manifest "$TARGET_ABS/sdp.manifest.yaml" --out "$TARGET_ABS/.sdp/generated"`,
		"repo-local CLI verified",
		"current shell resolves 'sdp' to:",
	} {
		t.Run(want, func(t *testing.T) {
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
		for _, want := range []string{
			"./.sdp/bin/sdp manifest validate",
			"./.sdp/bin/sdp doctor adapters",
			"command -v sdp",
		} {
			t.Run(path+" "+want, func(t *testing.T) {
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
		for _, want := range []string{
			"./.sdp/bin/sdp scout --format text .",
			"./.sdp/bin/sdp metrics --format markdown .",
			"./.sdp/bin/sdp index build --format text .",
			"./.sdp/bin/sdp spec --format text .",
		} {
			t.Run(path+" "+want, func(t *testing.T) {
				if !strings.Contains(doc, want) {
					t.Fatalf("%s should contain %q", path, want)
				}
			})
		}
	}
}

func TestOnboardingDocsUsePublicInstallerSurface(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs/QUICKSTART.md",
		"docs/runbooks/onboarding-downstream-repo.md",
		"scripts/install.sh",
	} {
		t.Run(path, func(t *testing.T) {
			doc := readRepoFile(t, path)
			if !strings.Contains(doc, "raw.githubusercontent.com/fall-out-bug/sdp/main/scripts/install.sh") {
				t.Fatalf("%s should reference the public sdp installer URL", path)
			}
			if strings.Contains(doc, "raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh") {
				t.Fatalf("%s should not send downstream users to the lab installer URL", path)
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
