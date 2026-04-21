package bootstrap

import (
	"strings"
	"time"
)

// renderPrinciples generates the DRAFT-PRINCIPLES.md content for a greenfield
// project. The output is deterministic: identical config always produces
// identical content.
func renderPrinciples(config GreenfieldConfig) string {
	var b strings.Builder

	b.WriteString(draftHeaderNow())
	b.Grow(512)

	b.WriteString("# DRAFT-PRINCIPLES.md\n\n")

	writeValuesSection(&b, config)
	writeArchitectureSection(&b, config)
	writeQualitySection(&b, config)
	writeGuidelinesSection(&b, config)

	return b.String()
}

// renderAgentsRules generates the AGENTS.md rules section content for a
// greenfield project. Like renderPrinciples, output is deterministic.
func renderAgentsRules(config GreenfieldConfig) string {
	var b strings.Builder

	b.WriteString(draftHeaderNow())
	b.Grow(256)

	b.WriteString("## Rules (generated)\n\n")
	b.WriteString("### Project Identity\n\n")
	b.WriteString("- Project type: " + config.ProjectType + "\n")
	b.WriteString("- Primary language: " + config.PrimaryLanguage + "\n\n")

	writeTestRulesSection(&b, config)
	writeCIRulesSection(&b, config)
	writeDeployRulesSection(&b, config)
	writeLanguageConventions(&b, config)

	return b.String()
}

// draftHeaderNow returns the DRAFT header comment with today's date.
func draftHeaderNow() string {
	return DraftHeader(time.Now().Format("2006-01-02"))
}

// --- Principles sections ---

func writeValuesSection(b *strings.Builder, config GreenfieldConfig) {
	b.WriteString("## Values\n\n")
	b.WriteString("- Simplicity over cleverness\n")
	b.WriteString("- Correctness over performance (until measured)\n")
	b.WriteString("- Explicit over implicit\n")
	b.WriteString("- Small, composable functions\n")
	b.WriteString("- Deterministic outputs from deterministic inputs\n")
	b.WriteString("\n")
}

func writeArchitectureSection(b *strings.Builder, config GreenfieldConfig) {
	b.WriteString("## Architecture\n\n")
	b.WriteString("Project type: " + config.ProjectType + "\n")
	b.WriteString("Language: " + config.PrimaryLanguage + "\n")
	b.WriteString("Deploy target: " + config.DeployTarget + "\n\n")
	b.WriteString(architecturePhilosophy(config))
	b.WriteString("\n")
}

func architecturePhilosophy(config GreenfieldConfig) string {
	switch config.ProjectType {
	case "web-service":
		return heredoc(`
			Adopt a layered architecture: handlers, services, repositories.
			Keep business logic out of transport concerns.
			Prefer dependency injection via interfaces.`)
	case "cli":
		return heredoc(`
			Structure as a single binary with subcommands.
			Separate flag parsing from business logic.
			Ensure every command is testable in isolation.`)
	case "library":
		return heredoc(`
			Export a minimal public API surface.
			Internal packages encapsulate implementation details.
			Prefer composition over inheritance.`)
	case "monorepo":
		return heredoc(`
			Each component owns its directory and tests.
			Shared code lives in a dedicated internal package.
			Changes span the repo atomically.`)
	default:
		return "Follow conventions appropriate for " + config.ProjectType + " projects."
	}
}

func writeQualitySection(b *strings.Builder, config GreenfieldConfig) {
	b.WriteString("## Quality\n\n")
	b.WriteString(qualityDescription(config.TestStrategy))
	b.WriteString("\n")
}

func qualityDescription(strategy string) string {
	switch strategy {
	case "tdd":
		return heredoc(`
			Test-driven development is the default discipline.
			Write the failing test first, then the minimal implementation.
			All tests must pass before merge.`)
	case "unit":
		return heredoc(`
			Unit tests cover core logic paths.
			Aim for high coverage on business-critical packages.
			Run tests on every commit.`)
	case "integration":
		return heredoc(`
			Integration tests validate component boundaries.
			External dependencies are replaced with test doubles.
			End-to-end smoke tests guard release readiness.`)
	case "minimal":
		return heredoc(`
			Tests focus on critical paths and edge cases.
			Avoid testing implementation details.
			Add tests when bugs are discovered.`)
	default:
		return "Test strategy: " + strategy
	}
}

func writeGuidelinesSection(b *strings.Builder, config GreenfieldConfig) {
	b.WriteString("## Guidelines\n\n")
	b.WriteString(guidelinesForLanguage(config.PrimaryLanguage))
	b.WriteString(guidelinesForDeploy(config.DeployTarget))
}

func guidelinesForLanguage(lang string) string {
	switch lang {
	case "go":
		return heredoc(`
			- Follow effective Go conventions
			- Run go vet and staticcheck before commit
			- Use table-driven tests
			- Keep functions under 60 lines
			`)
	case "python":
		return heredoc(`
			- Follow PEP 8 style guide
			- Use type hints for all public APIs
			- Prefer pytest for testing
			- Keep functions under 50 lines
			`)
	case "typescript":
		return heredoc(`
			- Use strict TypeScript configuration
			- Prefer interfaces over type aliases for objects
			- Use descriptive variable names
			- Keep functions under 50 lines
			`)
	case "rust":
		return heredoc(`
			- Follow Rust API guidelines
			- Run cargo clippy before commit
			- Write doc tests for public items
			- Keep functions under 60 lines
			`)
	case "java":
		return heredoc(`
			- Follow Google Java Style
			- Prefer immutability
			- Use JUnit 5 for testing
			- Keep methods under 40 lines
			`)
	default:
		return "- Follow community conventions for " + lang + "\n"
	}
}

func guidelinesForDeploy(target string) string {
	switch target {
	case "docker":
		return heredoc(`
			- Multi-stage builds for minimal images
			- No root processes in containers
			- Health checks on every service
			`)
	case "kubernetes":
		return heredoc(`
			- Liveness and readiness probes on all services
			- Resource limits on every pod
			- ConfigMaps for environment-specific values
			`)
	case "serverless":
		return heredoc(`
			- Keep cold starts minimal
			- Idempotent handlers for retries
			- Use infrastructure-as-code for deployment
			`)
	case "none":
		return ""
	default:
		return ""
	}
}

// --- Agents rules sections ---

func writeTestRulesSection(b *strings.Builder, config GreenfieldConfig) {
	b.WriteString("### Test Strategy\n\n")
	b.WriteString("- Strategy: " + config.TestStrategy + "\n")
	b.WriteString(testStrategyDirective(config.TestStrategy))
	b.WriteString("\n")
}

func testStrategyDirective(strategy string) string {
	switch strategy {
	case "tdd":
		return "- Write tests before implementation (red-green-refactor)"
	case "unit":
		return "- Cover core logic with focused unit tests"
	case "integration":
		return "- Validate component interactions with integration tests"
	case "minimal":
		return "- Test critical paths; avoid testing implementation details"
	default:
		return "- Test strategy: " + strategy
	}
}

func writeCIRulesSection(b *strings.Builder, config GreenfieldConfig) {
	b.WriteString("### CI\n\n")
	b.WriteString("- System: " + config.CIPreference + "\n")
	b.WriteString(ciDirective(config.CIPreference))
	b.WriteString("\n")
}

func ciDirective(ci string) string {
	switch ci {
	case "github-actions":
		return "- Workflows in .github/workflows/"
	case "gitlab-ci":
		return "- Pipeline defined in .gitlab-ci.yml"
	case "none":
		return "- No CI configured; run quality gates locally before push"
	default:
		return "- CI system: " + ci
	}
}

func writeDeployRulesSection(b *strings.Builder, config GreenfieldConfig) {
	if config.DeployTarget == "none" {
		return
	}
	b.WriteString("### Deployment\n\n")
	b.WriteString("- Target: " + config.DeployTarget + "\n")
	b.WriteString(deployDirective(config.DeployTarget))
	b.WriteString("\n")
}

func deployDirective(target string) string {
	switch target {
	case "docker":
		return "- Build and push container images on tagged releases"
	case "kubernetes":
		return "- Deploy via manifests; rollouts with health checks"
	case "serverless":
		return "- Deploy functions with infrastructure-as-code"
	default:
		return "- Deploy target: " + target
	}
}

func writeLanguageConventions(b *strings.Builder, config GreenfieldConfig) {
	b.WriteString("### Language Conventions\n\n")
	b.WriteString(languageConventionLines(config.PrimaryLanguage))
	b.WriteString("\n")
}

func languageConventionLines(lang string) string {
	switch lang {
	case "go":
		return heredoc(`
			- go vet, staticcheck on every commit
			- Table-driven tests preferred
			- Error wrapping with fmt.Errorf and %w
			- Interfaces defined by consumers`)
	case "python":
		return heredoc(`
			- ruff for linting and formatting
			- Type hints on all public functions
			- pytest with fixtures for test setup`)
	case "typescript":
		return heredoc(`
			- Strict mode enabled
			- ESLint with recommended rules
			- Prefer async/await over raw promises`)
	case "rust":
		return heredoc(`
			- cargo clippy --deny warnings
			- cargo fmt --check on CI
			- Document all public items`)
	case "java":
		return heredoc(`
			- Google Java Format
			- SpotBugs for static analysis
			- JUnit 5 with AssertJ`)
	default:
		return "- Follow community best practices for " + lang
	}
}

// heredoc trims leading newline and common leading whitespace from a multiline
// string literal, producing clean output for template content.
func heredoc(s string) string {
	s = strings.TrimPrefix(s, "\n")
	lines := strings.Split(s, "\n")
	indent := detectIndent(lines)
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, indent)
	}
	return strings.Join(lines, "\n")
}

// detectIndent finds the common leading whitespace across non-empty lines.
func detectIndent(lines []string) string {
	var common string
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		prefix := line[:len(line)-len(trimmed)]
		if common == "" || len(prefix) < len(common) {
			common = prefix
		}
	}
	if common == "" {
		return ""
	}
	return common
}
