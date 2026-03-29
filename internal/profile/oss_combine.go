package profile

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Tool struct {
	Name        string
	Description string
	CheckCmd    []string
	InstallHint string
}

type Profile interface {
	Name() string
	Description() string
	Tools() []Tool
	Directories() []string
	ConfigFiles() map[string]string
	Validate() error
	Provision() error
	Rollback() error
}

type OSSCombineProfile struct {
	projectRoot string
	dryRun      bool
}

func NewOSSCombineProfile(projectRoot string, dryRun bool) *OSSCombineProfile {
	return &OSSCombineProfile{
		projectRoot: projectRoot,
		dryRun:      dryRun,
	}
}

func (p *OSSCombineProfile) Name() string {
	return "oss-combine"
}

func (p *OSSCombineProfile) Description() string {
	return "Open-source integration environment with OMO, Beads, and Gas Town"
}

func (p *OSSCombineProfile) Tools() []Tool {
	return []Tool{
		{
			Name:        "git",
			Description: "Version control",
			CheckCmd:    []string{"git", "--version"},
			InstallHint: "Install git: https://git-scm.com/downloads",
		},
		{
			Name:        "gh",
			Description: "GitHub CLI for issue/PR operations",
			CheckCmd:    []string{"gh", "--version"},
			InstallHint: "Install gh CLI: https://cli.github.com/",
		},
		{
			Name:        "bd",
			Description: "Beads issue tracker",
			CheckCmd:    []string{"bd", "--version"},
			InstallHint: "Install beads: go install github.com/your-org/beads@latest",
		},
		{
			Name:        "go",
			Description: "Go runtime for SDP tools",
			CheckCmd:    []string{"go", "version"},
			InstallHint: "Install Go: https://go.dev/doc/install",
		},
		{
			Name:        "jq",
			Description: "JSON processor for evidence validation",
			CheckCmd:    []string{"jq", "--version"},
			InstallHint: "Install jq: https://stedolan.github.io/jq/download/",
		},
		{
			Name:        "opa",
			Description: "Open Policy Agent for governance",
			CheckCmd:    []string{"opa", "version"},
			InstallHint: "Install OPA: https://www.openpolicyagent.org/docs/latest/#running-opa",
		},
	}
}

func (p *OSSCombineProfile) Directories() []string {
	return []string{
		".sdp",
		".sdp/evidence",
		".sdp/checkpoints",
		".sdp/findings",
		".sdp/sessions",
		".sdp/traces",
		"docs/workstreams/backlog",
		"docs/plans",
		"schema/contracts",
		"schema/findings",
	}
}

func (p *OSSCombineProfile) ConfigFiles() map[string]string {
	return map[string]string{
		".sdp/config.yaml": `# SDP Configuration
version: "1.0"
profile: oss-combine

beads:
  prefix: sdplab-
  db: .beads/beads.db

github:
  findings_artifact: sdp-findings
  workflows:
    - CI
    - Protocol Validation

contracts:
  orchestration_event: schema/contracts/orchestration-event.schema.json
  runtime_decision: schema/contracts/runtime-decision.schema.json

findings:
  protocol: schema/findings/protocol-findings.schema.json
  docs: schema/findings/docs-findings.schema.json
`,
		".gitignore.sdp": `# SDP generated files
.sdp/evidence/*.json
.sdp/checkpoints/*.json
.sdp/findings/*.json
.sdp/sessions/
.sdp/traces/
.beads/
`,
	}
}

func (p *OSSCombineProfile) Validate() error {
	var missing []Tool

	for _, tool := range p.Tools() {
		if !p.checkTool(tool) {
			missing = append(missing, tool)
		}
	}

	if len(missing) > 0 {
		var msgs []string
		msgs = append(msgs, "Missing required tools:")
		for _, t := range missing {
			msgs = append(msgs, fmt.Sprintf("  - %s: %s", t.Name, t.InstallHint))
		}
		return fmt.Errorf("%s", strings.Join(msgs, "\n"))
	}

	return nil
}

func (p *OSSCombineProfile) checkTool(tool Tool) bool {
	cmd := exec.CommandContext(context.Background(), tool.CheckCmd[0], tool.CheckCmd[1:]...)
	return cmd.Run() == nil
}

func (p *OSSCombineProfile) Provision() error {
	fmt.Printf("Provisioning OSS Combine profile in: %s\n\n", p.projectRoot)

	if err := p.Validate(); err != nil {
		return err
	}

	fmt.Println("✓ All required tools available")

	fmt.Println("\nCreating directories...")
	for _, dir := range p.Directories() {
		path := filepath.Join(p.projectRoot, dir)
		if p.dryRun {
			fmt.Printf("  [DRY-RUN] Would create: %s\n", path)
			continue
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", path, err)
		}
		fmt.Printf("  ✓ %s\n", dir)
	}

	fmt.Println("\nCreating config files...")
	for name, content := range p.ConfigFiles() {
		path := filepath.Join(p.projectRoot, name)
		if p.dryRun {
			fmt.Printf("  [DRY-RUN] Would create: %s (%d bytes)\n", path, len(content))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("create parent dir for %s: %w", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("create file %s: %w", path, err)
		}
		fmt.Printf("  ✓ %s\n", name)
	}

	fmt.Println("\n✓ OSS Combine environment provisioned successfully!")
	return nil
}

func (p *OSSCombineProfile) Rollback() error {
	fmt.Printf("Rolling back OSS Combine profile in: %s\n\n", p.projectRoot)

	for name := range p.ConfigFiles() {
		path := filepath.Join(p.projectRoot, name)
		if p.dryRun {
			fmt.Printf("  [DRY-RUN] Would remove: %s\n", path)
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Printf("  ✗ Failed to remove %s: %v\n", path, err)
		} else if err == nil {
			fmt.Printf("  ✓ Removed: %s\n", name)
		}
	}

	sdpDir := filepath.Join(p.projectRoot, ".sdp")
	if p.dryRun {
		fmt.Printf("  [DRY-RUN] Would remove: %s\n", sdpDir)
	} else {
		if err := os.RemoveAll(sdpDir); err != nil {
			fmt.Printf("  ✗ Failed to remove .sdp: %v\n", err)
		} else {
			fmt.Printf("  ✓ Removed: .sdp/\n")
		}
	}

	fmt.Println("\n✓ Rollback complete!")
	return nil
}

func GetProfile(name string, projectRoot string, dryRun bool) (Profile, error) {
	switch name {
	case "oss-combine":
		return NewOSSCombineProfile(projectRoot, dryRun), nil
	default:
		return nil, fmt.Errorf("unknown profile: %s (available: oss-combine)", name)
	}
}

func AvailableProfiles() []string {
	return []string{"oss-combine"}
}
